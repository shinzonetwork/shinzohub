package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

// BeginBlocker incrementally drains the pending-settlement queues with debit
// priority and a per-block work cap.
//
// Per block, for the oldest closed epoch with work:
//   - If the debit queue is non-empty, drain up to MaxDebitsPerBlock debits.
//   - Else if the credit queue is non-empty, drain up to MaxCreditsPerBlock
//     credits.
//   - Else (both empty), advance the cursor and emit either Settled (if any
//     work was applied for this epoch in a prior block) or EpochNoActivity.
//
// At most ONE chunk-drain (debit OR credit) is processed per block. Empty
// closed epochs are fast-skipped in the same block since they cost only a
// cursor write each.
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	currentEpoch := k.GetCurrentEpoch(ctx)
	if currentEpoch == 0 {
		return nil
	}

	chunkDrained := false
	for {
		nextEpoch := k.GetLastSettledEpoch(ctx) + 1
		if nextEpoch >= currentEpoch {
			return nil
		}

		debits := k.PendingDebitCount(ctx, nextEpoch)
		credits := k.PendingCreditCount(ctx, nextEpoch)

		if debits == 0 && credits == 0 {
			// Closed epoch with nothing in either queue — fast-skip.
			ctx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeEpochNoActivity,
				sdk.NewAttribute(types.AttrKeyEpoch, strconv.FormatUint(nextEpoch, 10)),
			))
			k.SetLastSettledEpoch(ctx, nextEpoch)
			continue
		}

		// Non-empty epoch needing drain.
		if chunkDrained {
			return nil
		}

		// DEBIT PRIORITY: if any debits are queued for this epoch, those run
		// first this block. Credits only drain once their epoch's debit queue
		// is empty.
		if debits > 0 {
			if err := k.ProcessPendingDebitChunk(ctx, nextEpoch, types.MaxDebitsPerBlock); err != nil {
				return err
			}
		} else {
			if err := k.ProcessPendingCreditChunk(ctx, nextEpoch, types.MaxCreditsPerBlock); err != nil {
				return err
			}
		}
		chunkDrained = true

		// If both queues are now empty for this epoch, advance the cursor and
		// loop to handle the next epoch. Otherwise stop — more drain blocks
		// coming.
		if k.PendingCount(ctx, nextEpoch) == 0 {
			k.emitSettled(ctx, nextEpoch)
			k.SetLastSettledEpoch(ctx, nextEpoch)
			continue
		}
		return nil
	}
}

// ProcessPendingDebitChunk drains up to `limit` debit entries from the given
// epoch and applies them drain-to-zero against querybalance.
func (k Keeper) ProcessPendingDebitChunk(ctx sdk.Context, epoch uint64, limit int) error {
	debitByAddr := map[string]math.Int{}

	drained := k.DrainPendingDebits(ctx, epoch, limit, func(entry types.PendingSettleEntry) {
		for _, d := range entry.Debits {
			amt, ok := math.NewIntFromString(d.Amount)
			if !ok || !amt.IsPositive() {
				continue
			}
			if existing, ok := debitByAddr[d.Address]; ok {
				debitByAddr[d.Address] = existing.Add(amt)
			} else {
				debitByAddr[d.Address] = amt
			}
			// Decrement the per-address pending total — this line is no
			// longer queued. The decrement uses the QUEUED amount, not the
			// drain-to-zero "taken" amount: the index tracks "what's in the
			// queue", not "what got applied".
			k.subPendingDebitTotal(ctx, d.Address, amt)
		}
	})

	// Apply debits — drain-to-zero against the live querybalance.
	//
	// The accounting service is reporting queries the user ALREADY consumed
	// off-chain. The debit is final from its perspective; the chain takes
	// whatever's available and stops, even if that means zero. A user whose
	// balance has been drained will see "insufficient" at the gateway on
	// their next query — that's where access control lives, not here.
	totalDebited := math.ZeroInt()
	for _, key := range sortedAddrKeys(debitByAddr) {
		addr, err := sdk.AccAddressFromBech32(key)
		if err != nil {
			continue
		}
		taken, err := k.applyDebit(ctx, addr, debitByAddr[key])
		if err != nil {
			// Never return from BeginBlocker for a single bad entry — that would
			// halt the chain (and, since the entry was already drained, retry
			// identically forever). Skip it and keep the boundary making progress.
			k.Logger(ctx).Error("settlement: skipping undrainable debit", "address", key, "err", err)
			continue
		}
		totalDebited = totalDebited.Add(taken)
	}

	remainingDebits := k.PendingDebitCount(ctx, epoch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSettleChunk,
		sdk.NewAttribute(types.AttrKeyEpoch, strconv.FormatUint(epoch, 10)),
		sdk.NewAttribute("queue", "debit"),
		sdk.NewAttribute(types.AttrKeyEntryCount, strconv.Itoa(drained)),
		sdk.NewAttribute(types.AttrKeyRemainingCount, strconv.Itoa(remainingDebits)),
		sdk.NewAttribute(types.AttrKeyDebitsApplied, strconv.Itoa(len(debitByAddr))),
		sdk.NewAttribute(types.AttrKeyTotalDebited, totalDebited.String()),
	))

	return nil
}

// applyDebit runs one address's drain-to-zero debit and returns the amount
// actually taken. It converts a panic out of the querybalance keeper into an
// error: querybalance fails loud on corrupt state, which is correct in a tx or
// query context (baseapp recovers, the tx reverts) but must NOT escape here —
// this runs inside BeginBlocker, where an unrecovered panic halts the chain and,
// because the entry was already drained, retries identically forever. The reads
// happen before any write, so recovering leaves no partial state behind.
func (k Keeper) applyDebit(ctx sdk.Context, addr sdk.AccAddress, requested math.Int) (taken math.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			taken = math.ZeroInt()
			err = fmt.Errorf("querybalance drain panicked: %v", r)
		}
	}()

	balance := k.queryBalanceKeeper.GetBalance(ctx, addr)
	taken = requested
	if balance.LT(taken) {
		taken = balance
	}
	if !taken.IsPositive() {
		return math.ZeroInt(), nil
	}
	if err := k.queryBalanceKeeper.Debit(ctx, addr, taken); err != nil {
		return math.ZeroInt(), err
	}
	return taken, nil
}

// ProcessPendingCreditChunk drains up to `limit` credit entries from the
// given epoch and applies them to settlement balances.
func (k Keeper) ProcessPendingCreditChunk(ctx sdk.Context, epoch uint64, limit int) error {
	creditByAddr := map[string]math.Int{}

	drained := k.DrainPendingCredits(ctx, epoch, limit, func(entry types.PendingSettleEntry) {
		for _, p := range entry.Payments {
			amt, ok := math.NewIntFromString(p.Amount)
			if !ok || !amt.IsPositive() {
				continue
			}
			if existing, ok := creditByAddr[p.Address]; ok {
				creditByAddr[p.Address] = existing.Add(amt)
			} else {
				creditByAddr[p.Address] = amt
			}
		}
	})

	totalCredited := math.ZeroInt()
	for _, key := range sortedAddrKeys(creditByAddr) {
		addr, err := sdk.AccAddressFromBech32(key)
		if err != nil {
			continue
		}
		amt := creditByAddr[key]
		if err := k.Credit(ctx, addr, amt); err != nil {
			// See ProcessPendingDebitChunk: never halt the chain over one bad
			// entry — skip it and continue draining the boundary.
			k.Logger(ctx).Error("settlement: skipping unapplyable credit", "address", key, "err", err)
			continue
		}
		totalCredited = totalCredited.Add(amt)
	}

	remainingCredits := k.PendingCreditCount(ctx, epoch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSettleChunk,
		sdk.NewAttribute(types.AttrKeyEpoch, strconv.FormatUint(epoch, 10)),
		sdk.NewAttribute("queue", "credit"),
		sdk.NewAttribute(types.AttrKeyEntryCount, strconv.Itoa(drained)),
		sdk.NewAttribute(types.AttrKeyRemainingCount, strconv.Itoa(remainingCredits)),
		sdk.NewAttribute(types.AttrKeyPaymentsApplied, strconv.Itoa(len(creditByAddr))),
		sdk.NewAttribute(types.AttrKeyTotalCredited, totalCredited.String()),
	))

	return nil
}

// emitSettled is fired by BeginBlocker once both queues for an epoch are
// drained. Totals are not included — observers can aggregate ChunkApplied
// events for that epoch if they want grand totals.
func (k Keeper) emitSettled(ctx sdk.Context, epoch uint64) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSettled,
		sdk.NewAttribute(types.AttrKeyEpoch, strconv.FormatUint(epoch, 10)),
		sdk.NewAttribute(types.AttrKeyPoolsUpdated, "0"),
	))
}
