package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	settlementkeeper "github.com/shinzonetwork/shinzohub/x/settlement/keeper"
	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

// At block_height 18000, current epoch = 18000 / 180 = 100.
const testEpoch = 100

func heightAtEpoch(epoch uint64) int64 {
	return int64(epoch) * 180
}

func setEpoch(f *fixture, epoch uint64) {
	f.ctx = f.ctx.WithBlockHeight(heightAtEpoch(epoch))
}

// primeCursor sets last_settled_epoch so BeginBlocker won't waste iterations
// fast-skipping empty epochs from genesis to testEpoch. Useful in tests that
// want to assert on the cursor before the active epoch.
func primeCursor(f *fixture, epoch uint64) {
	f.keeper.SetLastSettledEpoch(f.ctx, epoch)
}

// submit is a convenience for the common queue-a-batch flow.
func submit(t *testing.T, f *fixture, msg *types.MsgAccountSettle) *types.MsgAccountSettleResponse {
	t.Helper()
	srv := settlementkeeper.NewMsgServerImpl(f.keeper)
	resp, err := srv.AccountSettle(sdk.WrapSDKContext(f.ctx), msg)
	require.NoError(t, err)
	return resp
}

// boundaryAdvance closes epoch fromEpoch and runs BeginBlocker repeatedly
// until that epoch is fully settled (both debit and credit queues empty AND
// cursor advanced past it). Each BeginBlocker call simulates one block.
func boundaryAdvance(t *testing.T, f *fixture, fromEpoch uint64) {
	t.Helper()
	setEpoch(f, fromEpoch+1)
	for i := 0; i < 32; i++ {
		require.NoError(t, f.keeper.BeginBlocker(f.ctx))
		if f.keeper.GetLastSettledEpoch(f.ctx) >= fromEpoch {
			return
		}
	}
	t.Fatalf("epoch %d not settled after 32 BeginBlocker iterations", fromEpoch)
}

// boundaryTick crosses into epoch fromEpoch+1 and runs BeginBlocker exactly
// once — used by tests that want to observe partial-drain state.
func boundaryTick(t *testing.T, f *fixture, fromEpoch uint64) {
	t.Helper()
	setEpoch(f, fromEpoch+1)
	require.NoError(t, f.keeper.BeginBlocker(f.ctx))
}

// ─── submission (queueing) ─────────────────────────────────────────────────

func TestSettle_QueuesEntry_NoStateChangeYet(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr

	resp := submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "500"},
		},
	})

	require.Equal(t, uint64(testEpoch), resp.Epoch)
	require.Equal(t, uint64(1), resp.PaymentsApplied, "PaymentsApplied here = entry count queued")
	require.Equal(t, "0", resp.TotalCredited, "submission must not credit yet — boundary does that")

	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, hostAddr),
		"settlement balance MUST stay zero until the boundary fires")
	require.Equal(t, 1, f.keeper.PendingCount(f.ctx, testEpoch))
}

func TestSettle_MultipleSubmissionsAccumulateInQueue(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr

	for i := 0; i < 5; i++ {
		submit(t, f, &types.MsgAccountSettle{
			Submitter: addr(0xAA).String(),
			Epoch:     testEpoch,
			Payments: []types.SettlePayment{
				{Did: "did:key:host-1", Amount: "100"},
			},
		})
	}

	require.Equal(t, 5, f.keeper.PendingCount(f.ctx, testEpoch),
		"five submissions queued, none applied yet")
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, hostAddr))
}

func TestSettle_RejectsWrongEpoch(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, 50)

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)

	_, err := srv.AccountSettle(sdk.WrapSDKContext(f.ctx), &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     49,
	})
	require.ErrorContains(t, err, "does not match current epoch")

	_, err = srv.AccountSettle(sdk.WrapSDKContext(f.ctx), &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     51,
	})
	require.ErrorContains(t, err, "does not match current epoch")

	require.Equal(t, 0, f.keeper.PendingCount(f.ctx, 49))
	require.Equal(t, 0, f.keeper.PendingCount(f.ctx, 51))
}

func TestSettle_UnresolvableDIDFailsAtSubmission(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)
	_, err := srv.AccountSettle(sdk.WrapSDKContext(f.ctx), &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:unknown", Amount: "500"},
		},
	})

	require.ErrorContains(t, err, "not registered as host or indexer")
	require.Equal(t, 0, f.keeper.PendingCount(f.ctx, testEpoch),
		"validation failure must not enqueue anything")
}

func TestSettle_InvalidDebitAddressFailsAtSubmission(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)
	_, err := srv.AccountSettle(sdk.WrapSDKContext(f.ctx), &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: "not-a-bech32", Amount: "100"},
		},
	})

	require.ErrorContains(t, err, "debit[0]")
	require.Equal(t, 0, f.keeper.PendingCount(f.ctx, testEpoch))
}

func TestSettle_PartialBatchFailsAtomically(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	f.host.addrs["did:key:good"] = hostAddr

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)
	_, err := srv.AccountSettle(sdk.WrapSDKContext(f.ctx), &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:good", Amount: "100"},
			{Did: "did:key:bad", Amount: "200"},
		},
	})

	require.Error(t, err)
	require.Equal(t, 0, f.keeper.PendingCount(f.ctx, testEpoch),
		"earlier valid entry must NOT have been queued — pre-validation is atomic per submission")
}

// ─── boundary application ─────────────────────────────────────────────────

func TestBoundary_AppliesSumOfQueuedSubmissions(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	idxAddr := addr(0x22)
	debitAddr := addr(0x33)

	f.host.addrs["did:key:host-1"] = hostAddr
	f.indexer.addrs["did:key:indexer-1"] = idxAddr
	f.qb.balances[debitAddr.String()] = math.NewInt(1_000)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "500"},
		},
	})
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:indexer-1", Amount: "200"},
		},
		Debits: []types.SettleDebit{
			{Address: debitAddr.String(), Amount: "300"},
		},
	})

	// Nothing applied yet.
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, hostAddr))
	require.Equal(t, math.NewInt(1_000), f.qb.balances[debitAddr.String()])

	boundaryAdvance(t, f, testEpoch)

	require.Equal(t, math.NewInt(500), f.keeper.GetBalance(f.ctx, hostAddr))
	require.Equal(t, math.NewInt(200), f.keeper.GetBalance(f.ctx, idxAddr))
	require.Equal(t, math.NewInt(700), f.qb.balances[debitAddr.String()])
	require.Equal(t, uint64(testEpoch), f.keeper.GetLastSettledEpoch(f.ctx))
	require.Equal(t, 0, f.keeper.PendingCount(f.ctx, testEpoch),
		"pending queue must be drained after boundary processing")
}

func TestBoundary_SumsDuplicatePaymentsByRecipient(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr

	// Two separate submissions targeting the same DID.
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "2000"},
		},
	})
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "3000"},
		},
	})

	boundaryAdvance(t, f, testEpoch)

	require.Equal(t, math.NewInt(5000), f.keeper.GetBalance(f.ctx, hostAddr),
		"both submissions to the same recipient must be summed at the boundary")
}

func TestBoundary_DrainToZeroDebit(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	debitAddr := addr(0x77)
	f.qb.balances[debitAddr.String()] = math.NewInt(50)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: debitAddr.String(), Amount: "200"},
		},
	})

	boundaryAdvance(t, f, testEpoch)

	require.True(t, f.qb.balances[debitAddr.String()].IsZero(),
		"drain-to-zero must cap the debit at whatever's available")
}

func TestBoundary_DrainToZero_NoBalance(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	debitAddr := addr(0x77)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: debitAddr.String(), Amount: "200"},
		},
	})

	boundaryAdvance(t, f, testEpoch)
	// No panic, no error — debit attempts skipped silently.
}

func TestBoundary_EmptyEpochEmitsNoActivity(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)
	// No submissions during epoch testEpoch.

	boundaryAdvance(t, f, testEpoch)

	require.Equal(t, uint64(testEpoch), f.keeper.GetLastSettledEpoch(f.ctx),
		"empty epoch still advances the cursor")

	var seenNoActivity bool
	for _, e := range f.ctx.EventManager().Events() {
		if e.Type == types.EventTypeEpochNoActivity {
			seenNoActivity = true
		}
	}
	require.True(t, seenNoActivity, "empty epoch must emit no-activity event")
}

func TestBoundary_EmitsSettledAndChunkEvents(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "500"},
		},
	})

	boundaryAdvance(t, f, testEpoch)

	var sawSettled, sawCreditChunk bool
	for _, e := range f.ctx.EventManager().Events() {
		switch e.Type {
		case types.EventTypeSettled:
			sawSettled = true
			attrs := map[string]string{}
			for _, a := range e.Attributes {
				attrs[a.Key] = a.Value
			}
			require.Equal(t, "100", attrs[types.AttrKeyEpoch])
		case types.EventTypeSettleChunk:
			attrs := map[string]string{}
			for _, a := range e.Attributes {
				attrs[a.Key] = a.Value
			}
			if attrs["queue"] == "credit" {
				sawCreditChunk = true
				require.Equal(t, "500", attrs[types.AttrKeyTotalCredited])
				require.Equal(t, "1", attrs[types.AttrKeyEntryCount])
			}
		}
	}
	require.True(t, sawCreditChunk, "credit chunk event must fire with totals")
	require.True(t, sawSettled, "Settled event must fire once both queues drain")
}

func TestBoundary_AdvancesAcrossMultipleEpochsAtOnce(t *testing.T) {
	f := newFixture(t)

	// Submit during epoch 100.
	setEpoch(f, testEpoch)
	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "100"},
		},
	})

	// Jump straight to epoch 105 without firing BeginBlocker in between.
	setEpoch(f, testEpoch+5)
	require.NoError(t, f.keeper.BeginBlocker(f.ctx))

	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, hostAddr),
		"epoch 100's pending entries must have been processed during catch-up")
	require.Equal(t, uint64(testEpoch+4), f.keeper.GetLastSettledEpoch(f.ctx),
		"cursor must catch up to currentEpoch-1, even across multiple gaps")
}

func TestBoundary_DoesNotApplyForCurrentEpoch(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "500"},
		},
	})

	// BeginBlocker runs during the same epoch — should NOT close it.
	require.NoError(t, f.keeper.BeginBlocker(f.ctx))

	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, hostAddr),
		"BeginBlocker must not drain the queue for the current (still-open) epoch")
	require.Equal(t, 1, f.keeper.PendingCount(f.ctx, testEpoch))
}

func TestBoundary_RepeatedRunIsNoOp(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "500"},
		},
	})

	boundaryAdvance(t, f, testEpoch)
	require.Equal(t, math.NewInt(500), f.keeper.GetBalance(f.ctx, hostAddr))

	// Run BeginBlocker again in the SAME epoch — no double-credit.
	require.NoError(t, f.keeper.BeginBlocker(f.ctx))
	require.Equal(t, math.NewInt(500), f.keeper.GetBalance(f.ctx, hostAddr),
		"repeated BeginBlocker must not double-apply")
}

func TestBoundary_LinearEpochAdvance(t *testing.T) {
	f := newFixture(t)

	setEpoch(f, testEpoch)
	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "100"},
		},
	})
	boundaryAdvance(t, f, testEpoch)
	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, hostAddr))

	// Now in epoch testEpoch+1; submit another batch.
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch + 1,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "200"},
		},
	})
	boundaryAdvance(t, f, testEpoch+1)
	require.Equal(t, math.NewInt(300), f.keeper.GetBalance(f.ctx, hostAddr),
		"second epoch's settlement adds to the running balance")
	require.Equal(t, uint64(testEpoch+1), f.keeper.GetLastSettledEpoch(f.ctx))
}

func TestSettle_AppliesPoolStatsAtSubmissionTime(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	f.pool.existing["pool-A"] = struct{}{}
	f.pool.existing["pool-B"] = struct{}{}

	resp := submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Pools: []types.PoolUpdate{
			{PoolAddress: "pool-A", Price: "9995", Utilization: 32, Rewards: "1000000", NumberOfQueries: 10},
			{PoolAddress: "pool-B", Price: "1005", Utilization: 68, Rewards: "500000", NumberOfQueries: 5},
		},
	})

	require.Equal(t, uint64(2), resp.PoolsUpdated)
	require.Len(t, f.pool.updates, 2, "pool keeper must be called once per pool update at submission time")
	require.Equal(t, "pool-A", f.pool.updates[0].addr)
	require.Equal(t, uint64(32), f.pool.updates[0].utilization)
	require.Equal(t, uint64(10), f.pool.updates[0].queries)
	require.Equal(t, "1000000", f.pool.updates[0].rewards.String())
	require.Equal(t, uint64(testEpoch), f.pool.updates[0].epoch)
}

func TestSettle_RejectsNonexistentPool(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)
	_, err := srv.AccountSettle(sdk.WrapSDKContext(f.ctx), &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Pools: []types.PoolUpdate{
			{PoolAddress: "ghost", Price: "100", Utilization: 50, Rewards: "0", NumberOfQueries: 1},
		},
	})

	require.ErrorContains(t, err, "pool ghost not found")
	require.Len(t, f.pool.updates, 0, "pool not found must fail atomically — no partial updates")
}
