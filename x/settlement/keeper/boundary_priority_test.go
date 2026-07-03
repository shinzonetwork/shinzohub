package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

// TestBoundary_DebitsApplyBeforeCredits proves that a single block fires the
// debit chunk first; the credit chunk has to wait for the next block.
func TestBoundary_DebitsApplyBeforeCredits(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)
	primeCursor(f, testEpoch-1)

	hostAddr := addr(0x11)
	debitAddr := addr(0x22)

	f.host.addrs["did:key:host-1"] = hostAddr
	f.qb.balances[debitAddr.String()] = math.NewInt(1_000)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "500"},
		},
		Debits: []types.SettleDebit{
			{Address: debitAddr.String(), Amount: "300"},
		},
	})

	require.Equal(t, 1, f.keeper.PendingDebitCount(f.ctx, testEpoch))
	require.Equal(t, 1, f.keeper.PendingCreditCount(f.ctx, testEpoch))

	// One block at the boundary — should drain the debit queue, NOT credits.
	boundaryTick(t, f, testEpoch)

	require.Equal(t, math.NewInt(700), f.qb.balances[debitAddr.String()],
		"debit applied in the first boundary block")
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, hostAddr),
		"credit still queued — not yet applied")
	require.Equal(t, 0, f.keeper.PendingDebitCount(f.ctx, testEpoch))
	require.Equal(t, 1, f.keeper.PendingCreditCount(f.ctx, testEpoch))
	require.Equal(t, uint64(testEpoch-1), f.keeper.GetLastSettledEpoch(f.ctx),
		"cursor must NOT advance past the still-pending epoch")

	// Second block — credits drain.
	require.NoError(t, f.keeper.BeginBlocker(f.ctx))

	require.Equal(t, math.NewInt(500), f.keeper.GetBalance(f.ctx, hostAddr))
	require.Equal(t, uint64(testEpoch), f.keeper.GetLastSettledEpoch(f.ctx),
		"cursor advances now that both queues are empty")
}

// TestBoundary_DebitsRespectPerBlockCap drains > MaxDebitsPerBlock and proves
// it takes multiple blocks; credits wait the whole time.
func TestBoundary_DebitsRespectPerBlockCap(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	f.host.addrs["did:key:host-1"] = hostAddr

	// Queue (MaxDebitsPerBlock + 50) debit entries and 1 credit entry.
	overflow := types.MaxDebitsPerBlock + 50
	for i := 0; i < overflow; i++ {
		debitAddr := addr(byte(0x30 + (i % 200)))
		f.qb.balances[debitAddr.String()] = math.NewInt(10)
		submit(t, f, &types.MsgAccountSettle{
			Submitter: addr(0xAA).String(),
			Epoch:     testEpoch,
			Debits: []types.SettleDebit{
				{Address: debitAddr.String(), Amount: "5"},
			},
		})
	}
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "1000"},
		},
	})

	require.Equal(t, overflow, f.keeper.PendingDebitCount(f.ctx, testEpoch))
	require.Equal(t, 1, f.keeper.PendingCreditCount(f.ctx, testEpoch))

	// First block: should drain a full chunk of debits and not touch credits.
	boundaryTick(t, f, testEpoch)
	require.Equal(t, 50, f.keeper.PendingDebitCount(f.ctx, testEpoch),
		"first block drained exactly MaxDebitsPerBlock debits")
	require.Equal(t, 1, f.keeper.PendingCreditCount(f.ctx, testEpoch),
		"credit queue untouched while debits remain")
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, hostAddr))

	// Second block: drain remaining 50 debits.
	require.NoError(t, f.keeper.BeginBlocker(f.ctx))
	require.Equal(t, 0, f.keeper.PendingDebitCount(f.ctx, testEpoch))
	require.Equal(t, 1, f.keeper.PendingCreditCount(f.ctx, testEpoch),
		"credits STILL queued — only one chunk drains per block")
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, hostAddr))

	// Third block: drain the credit.
	require.NoError(t, f.keeper.BeginBlocker(f.ctx))
	require.Equal(t, math.NewInt(1_000), f.keeper.GetBalance(f.ctx, hostAddr))
	require.Equal(t, uint64(testEpoch), f.keeper.GetLastSettledEpoch(f.ctx))
}

// TestBoundary_CreditsRespectPerBlockCap drains > MaxCreditsPerBlock credits
// when no debits are queued; proves multi-block credit drain.
func TestBoundary_CreditsRespectPerBlockCap(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)
	primeCursor(f, testEpoch-1)

	// Need (MaxCreditsPerBlock + N) DISTINCT DIDs, each registered.
	overflow := types.MaxCreditsPerBlock + 30
	hostAddrs := make([]sdk.AccAddress, overflow)
	for i := 0; i < overflow; i++ {
		hostAddrs[i] = addr(byte(i + 1))
		did := fmt.Sprintf("did:key:host-%d", i)
		f.host.addrs[did] = hostAddrs[i]
		submit(t, f, &types.MsgAccountSettle{
			Submitter: addr(0xAA).String(),
			Epoch:     testEpoch,
			Payments: []types.SettlePayment{
				{Did: did, Amount: "10"},
			},
		})
	}

	require.Equal(t, overflow, f.keeper.PendingCreditCount(f.ctx, testEpoch))

	// First block: drain MaxCreditsPerBlock credits.
	boundaryTick(t, f, testEpoch)
	require.Equal(t, 30, f.keeper.PendingCreditCount(f.ctx, testEpoch))
	require.Equal(t, uint64(testEpoch-1), f.keeper.GetLastSettledEpoch(f.ctx),
		"cursor stays put while credit queue isn't empty")

	// Second block: drain the remaining 30 credits.
	require.NoError(t, f.keeper.BeginBlocker(f.ctx))
	require.Equal(t, 0, f.keeper.PendingCreditCount(f.ctx, testEpoch))
	require.Equal(t, uint64(testEpoch), f.keeper.GetLastSettledEpoch(f.ctx),
		"cursor advances now that both queues are empty")
}

// TestBoundary_OnlyOneChunkPerBlock proves the "at most one chunk drain per
// block" rule: even if both queues have work, a single block drains one.
func TestBoundary_OnlyOneChunkPerBlock(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	hostAddr := addr(0x11)
	debitAddr := addr(0x22)
	f.host.addrs["did:key:host-1"] = hostAddr
	f.qb.balances[debitAddr.String()] = math.NewInt(100)

	// Mixed-content single submission → one debit-queue entry + one credit-queue entry.
	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Payments: []types.SettlePayment{
			{Did: "did:key:host-1", Amount: "500"},
		},
		Debits: []types.SettleDebit{
			{Address: debitAddr.String(), Amount: "50"},
		},
	})

	// Both queues have 1 entry. One block drains debits only.
	boundaryTick(t, f, testEpoch)

	require.Equal(t, 0, f.keeper.PendingDebitCount(f.ctx, testEpoch))
	require.Equal(t, 1, f.keeper.PendingCreditCount(f.ctx, testEpoch),
		"single chunk drained per block — credits remain")
}

// TestBoundary_DebitOnlySubmissionSettlesInOneBlock confirms that a submission
// with only debits and no credits closes in a single block.
func TestBoundary_DebitOnlySubmissionSettlesInOneBlock(t *testing.T) {
	f := newFixture(t)
	setEpoch(f, testEpoch)

	debitAddr := addr(0x22)
	f.qb.balances[debitAddr.String()] = math.NewInt(100)

	submit(t, f, &types.MsgAccountSettle{
		Submitter: addr(0xAA).String(),
		Epoch:     testEpoch,
		Debits: []types.SettleDebit{
			{Address: debitAddr.String(), Amount: "50"},
		},
	})

	boundaryTick(t, f, testEpoch)

	require.Equal(t, math.NewInt(50), f.qb.balances[debitAddr.String()])
	require.Equal(t, uint64(testEpoch), f.keeper.GetLastSettledEpoch(f.ctx),
		"epoch settled in one block — no credit queue work was needed")
}
