package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

func (m msgServer) Claim(goCtx context.Context, msg *types.MsgClaim) (*types.MsgClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	claimer, err := sdk.AccAddressFromBech32(msg.Claimer)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("claimer: %s", err)
	}

	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("amount %q is not an integer", msg.Amount)
	}

	if err := m.Keeper.Claim(ctx, claimer, amount); err != nil {
		return nil, fmt.Errorf("claim: %w", err)
	}

	remaining := m.Keeper.GetBalance(ctx, claimer)
	return &types.MsgClaimResponse{Remaining: remaining.String()}, nil
}

// AccountSettle enqueues one chunk of the current epoch's accounting output.
//
// Execution model — QUEUE + BOUNDARY APPLY:
//   - During epoch N, callers submit any number of MsgAccountSettle txs.
//   - Each tx is pre-validated (DIDs resolve, addresses parse, amounts
//     positive) and then APPENDED to a pending queue keyed by epoch.
//   - At the next epoch boundary, BeginBlocker drains the queue for the
//     just-closed epoch, sums by recipient/address, applies credits and
//     debits in one atomic step, and marks the epoch settled.
//
// This lets the accounting service stream partial results during the epoch
// instead of assembling one giant batch.
//
// Replay protection: cosmos's account-sequence antehandler rejects identical
// signed txs before they reach this handler, so we don't need per-submission
// replay logic. The pending queue admits anything that validates.
//
// The `pools` slice is accepted but ignored — reserved in the wire format
// so the accounting service can include pool data now without a future
// migration.
//
// Caveat — multiple services running identical logic will currently
// double-credit because every valid submission lands in the queue and gets
// summed at the boundary. Safe multi-service operation requires the
// consensus upgrade below.
//
// TODO(consensus-aggregation): for multi-service redundancy without double
// credit, switch the boundary processor from sum-everything to hash-of-batch
// voting:
//   - canonical-encode each pending submission, hash it
//   - plurality wins (≥N matching); apply that batch only
//   - emit no_consensus event if threshold not met
// Prerequisites: allowlist (or stake-based) gating of submitters, and a
// deterministic canonical form so services that compute the same answer
// produce the same hash.
func (m msgServer) AccountSettle(goCtx context.Context, msg *types.MsgAccountSettle) (*types.MsgAccountSettleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Epoch validation — only the current epoch may be settled. Multiple
	//    submissions per epoch are allowed; cosmos's account-sequence
	//    antehandler handles signed-tx replay.
	currentEpoch := m.Keeper.GetCurrentEpoch(ctx)
	if msg.Epoch != currentEpoch {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf(
			"settle epoch %d does not match current epoch %d", msg.Epoch, currentEpoch)
	}

	// 2. Pre-validate + resolve. Only structural checks: addresses parse,
	//    amounts parse as positive integers, DIDs resolve to a registered
	//    host or indexer. We DO NOT check whether the debit address has
	//    enough querybalance — the user has already consumed those queries
	//    off-chain, so the debit must land regardless of current balance.
	//    Drain-to-zero at the boundary handles the shortfall case; future
	//    queries from that user are rejected at the gateway, not here.
	//
	//    We snapshot the DID-to-address resolution at submission time so the
	//    boundary processor doesn't have to redo it (and isn't surprised if
	//    a DID is unregistered between now and the boundary).
	resolvedPayments := make([]types.AddressAmount, 0, len(msg.Payments))
	for i, p := range msg.Payments {
		addr, ok := m.resolveDID(ctx, p.Did)
		if !ok {
			return nil, sdkerrors.ErrInvalidRequest.Wrapf(
				"payment[%d]: DID %q not registered as host or indexer", i, p.Did)
		}
		if _, err := parsePositiveAmount(p.Amount); err != nil {
			return nil, sdkerrors.ErrInvalidRequest.Wrapf("payment[%d]: %s", i, err)
		}
		resolvedPayments = append(resolvedPayments, types.AddressAmount{
			Address: addr.String(),
			Amount:  p.Amount,
		})
	}

	resolvedDebits := make([]types.AddressAmount, 0, len(msg.Debits))
	for i, d := range msg.Debits {
		if _, err := sdk.AccAddressFromBech32(d.Address); err != nil {
			return nil, sdkerrors.ErrInvalidAddress.Wrapf("debit[%d]: %s", i, err)
		}
		if _, err := parsePositiveAmount(d.Amount); err != nil {
			return nil, sdkerrors.ErrInvalidRequest.Wrapf("debit[%d]: %s", i, err)
		}
		resolvedDebits = append(resolvedDebits, types.AddressAmount{
			Address: d.Address,
			Amount:  d.Amount,
		})
	}

	// 3. Enqueue. Split the submission into a debit-only entry and a
	//    credit-only entry, each going to its own queue. Debits are drained
	//    first per block (gateway access depends on them), so isolating them
	//    in their own queue lets the boundary processor prioritize cleanly.
	if len(resolvedDebits) > 0 {
		debitEntry := types.PendingSettleEntry{
			Submitter: msg.Submitter,
			Debits:    resolvedDebits,
		}
		if _, err := m.Keeper.EnqueuePendingDebit(ctx, msg.Epoch, debitEntry); err != nil {
			return nil, fmt.Errorf("enqueue debit: %w", err)
		}
	}
	if len(resolvedPayments) > 0 {
		creditEntry := types.PendingSettleEntry{
			Submitter: msg.Submitter,
			Payments:  resolvedPayments,
		}
		if _, err := m.Keeper.EnqueuePendingCredit(ctx, msg.Epoch, creditEntry); err != nil {
			return nil, fmt.Errorf("enqueue credit: %w", err)
		}
	}

	// 4. Emit queued event so observers can see the submission landed.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSettleQueued,
		sdk.NewAttribute(types.AttrKeyEpoch, strconv.FormatUint(msg.Epoch, 10)),
		sdk.NewAttribute(types.AttrKeySubmitter, msg.Submitter),
		sdk.NewAttribute(types.AttrKeyPaymentsApplied, strconv.Itoa(len(resolvedPayments))),
		sdk.NewAttribute(types.AttrKeyDebitsApplied, strconv.Itoa(len(resolvedDebits))),
	))

	// PaymentsApplied/DebitsApplied here reflect ENTRY counts queued; actual
	// application happens at the epoch boundary and is reported via the
	// Settled event then. TotalCredited/Debited are not knowable yet because
	// debits drain-to-zero against state that may change before the boundary.
	return &types.MsgAccountSettleResponse{
		Epoch:           msg.Epoch,
		PaymentsApplied: uint64(len(resolvedPayments)),
		DebitsApplied:   uint64(len(resolvedDebits)),
		TotalCredited:   "0",
		TotalDebited:    "0",
		PoolsUpdated:    0,
	}, nil
}

func (m msgServer) resolveDID(ctx sdk.Context, did string) (sdk.AccAddress, bool) {
	if addr, ok := m.Keeper.hostKeeper.GetAddressForDID(ctx, did); ok {
		return addr, true
	}
	if addr, ok := m.Keeper.indexerKeeper.GetAddressForDID(ctx, did); ok {
		return addr, true
	}
	return nil, false
}

func parsePositiveAmount(s string) (math.Int, error) {
	amt, ok := math.NewIntFromString(s)
	if !ok {
		return math.Int{}, fmt.Errorf("amount %q is not an integer", s)
	}
	if !amt.IsPositive() {
		return math.Int{}, fmt.Errorf("amount %q is not positive", s)
	}
	return amt, nil
}

type addrSum struct {
	addr   sdk.AccAddress
	amount math.Int
}

func sumByAddr(entries []addrSum) map[string]math.Int {
	out := map[string]math.Int{}
	for _, e := range entries {
		k := e.addr.String()
		if existing, ok := out[k]; ok {
			out[k] = existing.Add(e.amount)
		} else {
			out[k] = e.amount
		}
	}
	return out
}

// sortedAddrKeys returns the keys of `m` in sorted order so iteration is
// deterministic across runs — cosmos enforces deterministic state mutations.
func sortedAddrKeys(m map[string]math.Int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
