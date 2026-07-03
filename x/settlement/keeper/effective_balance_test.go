package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	settlementkeeper "github.com/shinzonetwork/shinzohub/x/settlement/keeper"
	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

// ─── index maintenance ─────────────────────────────────────────────────────

func TestPendingDebitIndex_IncrementsOnEnqueue(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	a := addr(0x11)
	require.Equal(t, math.ZeroInt(), f.keeper.GetPendingDebitTotal(f.ctx, a))

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "300"},
		},
	})

	require.Equal(t, math.NewInt(300), f.keeper.GetPendingDebitTotal(f.ctx, a))
}

func TestPendingDebitIndex_AccumulatesAcrossSubmissions(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	a := addr(0x11)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "100"},
		},
	})
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "250"},
		},
	})

	require.Equal(t, math.NewInt(350), f.keeper.GetPendingDebitTotal(f.ctx, a),
		"index must sum across submissions, not overwrite")
}

func TestPendingDebitIndex_AccumulatesAcrossEpochs(t *testing.T) {
	f := newFixture(t)
	primeCursor(f, testEpoch-1)

	a := addr(0x11)
	f.qb.balances[a.String()] = math.NewInt(10_000)

	setEpoch(f, testEpoch)
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "100"},
		},
	})

	// Cross into epoch testEpoch+1 — but DON'T run BeginBlocker yet, so
	// epoch testEpoch's queue is still un-drained.
	setEpoch(f, testEpoch+1)
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch + 1,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "200"},
		},
	})

	require.Equal(t, math.NewInt(300), f.keeper.GetPendingDebitTotal(f.ctx, a),
		"index must reflect debits queued in BOTH unsettled epochs")
}

func TestPendingDebitIndex_DecrementsOnDrain(t *testing.T) {
	f := newFixture(t)
	primeCursor(f, testEpoch-1)
	setEpoch(f, testEpoch)

	a := addr(0x11)
	f.qb.balances[a.String()] = math.NewInt(1_000)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "300"},
		},
	})
	require.Equal(t, math.NewInt(300), f.keeper.GetPendingDebitTotal(f.ctx, a))

	boundaryAdvance(t, f, testEpoch)

	require.Equal(t, math.ZeroInt(), f.keeper.GetPendingDebitTotal(f.ctx, a),
		"index must drop to zero once the queued debit is drained")
	require.Equal(t, math.NewInt(700), f.qb.balances[a.String()])
}

func TestPendingDebitIndex_DecrementsByQueuedAmountNotTakenAmount(t *testing.T) {
	// Drain-to-zero semantics: queue says 500, balance is 50.
	// Index must drop by the QUEUED 500, not the TAKEN 50 — the queue is
	// drained regardless of how much was actually applied.
	f := newFixture(t)
	primeCursor(f, testEpoch-1)
	setEpoch(f, testEpoch)

	a := addr(0x11)
	f.qb.balances[a.String()] = math.NewInt(50)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "500"},
		},
	})
	require.Equal(t, math.NewInt(500), f.keeper.GetPendingDebitTotal(f.ctx, a))

	boundaryAdvance(t, f, testEpoch)

	require.Equal(t, math.ZeroInt(), f.keeper.GetPendingDebitTotal(f.ctx, a),
		"queued amount fully cleared from index, regardless of drain-to-zero outcome")
	require.True(t, f.qb.balances[a.String()].IsZero(),
		"querybalance drained to zero (took the 50 that was there)")
}

// ─── effective balance keeper method ───────────────────────────────────────

func TestGetEffectiveBalance_NoPending(t *testing.T) {
	f := newFixture(t)
	a := addr(0x11)
	f.qb.balances[a.String()] = math.NewInt(1_000)

	eff := f.keeper.GetEffectiveBalance(f.ctx, a)
	require.Equal(t, math.NewInt(1_000), eff,
		"no pending → effective == actual")
}

func TestGetEffectiveBalance_SubtractsPending(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	a := addr(0x11)
	f.qb.balances[a.String()] = math.NewInt(1_000)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "400"},
		},
	})

	require.Equal(t, math.NewInt(600), f.keeper.GetEffectiveBalance(f.ctx, a))
}

func TestGetEffectiveBalance_ClampsAtZero(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	a := addr(0x11)
	f.qb.balances[a.String()] = math.NewInt(100)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "500"},
		},
	})

	require.True(t, f.keeper.GetEffectiveBalance(f.ctx, a).IsZero(),
		"pending exceeds actual → effective is zero, never negative")
}

func TestGetEffectiveBalance_OtherUsersUnaffected(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	a := addr(0x11)
	b := addr(0x22)
	f.qb.balances[a.String()] = math.NewInt(1_000)
	f.qb.balances[b.String()] = math.NewInt(1_000)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "400"},
		},
	})

	require.Equal(t, math.NewInt(600), f.keeper.GetEffectiveBalance(f.ctx, a))
	require.Equal(t, math.NewInt(1_000), f.keeper.GetEffectiveBalance(f.ctx, b),
		"pending for A must not affect B")
}

// ─── EffectiveBalance RPC handler ─────────────────────────────────────────

func TestQueryEffectiveBalance_HappyPath(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	a := addr(0x11)
	f.qb.balances[a.String()] = math.NewInt(1_000)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: a.String(), Amount: "300"},
		},
	})

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	resp, err := qs.EffectiveBalance(sdk.WrapSDKContext(f.ctx), &types.QueryEffectiveBalanceRequest{
		Address: a.String(),
	})
	require.NoError(t, err)
	require.Equal(t, a.String(), resp.Address)
	require.Equal(t, "1000", resp.Actual)
	require.Equal(t, "300", resp.PendingDebit)
	require.Equal(t, "700", resp.Effective)
}

func TestQueryEffectiveBalance_RejectsEmptyAddress(t *testing.T) {
	f := newFixture(t)
	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	_, err := qs.EffectiveBalance(sdk.WrapSDKContext(f.ctx), &types.QueryEffectiveBalanceRequest{})
	require.Error(t, err)
}

func TestQueryEffectiveBalance_RejectsInvalidBech32(t *testing.T) {
	f := newFixture(t)
	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	_, err := qs.EffectiveBalance(sdk.WrapSDKContext(f.ctx), &types.QueryEffectiveBalanceRequest{
		Address: "not-a-bech32",
	})
	require.Error(t, err)
}

func TestQueryEffectiveBalance_ZeroBalance_NoPending(t *testing.T) {
	f := newFixture(t)
	a := addr(0x42)

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	resp, err := qs.EffectiveBalance(sdk.WrapSDKContext(f.ctx), &types.QueryEffectiveBalanceRequest{
		Address: a.String(),
	})
	require.NoError(t, err)
	require.Equal(t, "0", resp.Actual)
	require.Equal(t, "0", resp.PendingDebit)
	require.Equal(t, "0", resp.Effective)
}
