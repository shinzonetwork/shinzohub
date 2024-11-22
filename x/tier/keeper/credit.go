package keeper

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// mintCredit mints a coin and sends it to the specified address.
func (k Keeper) mintCredit(ctx context.Context, addr sdk.AccAddress, amt math.Int) error {
	coins := sdk.NewCoins(sdk.NewCoin(appparams.CreditDenom, amt))
	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return errorsmod.Wrap(err, "mint coins")
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
	if err != nil {
		return errorsmod.Wrap(err, "send coins from module to account")
	}

	return nil
}

// proratedCredit calculates the credits earned on the lockingAmt.
func (k Keeper) proratedCredit(ctx context.Context, delAddr sdk.AccAddress, lockingAmt math.Int) math.Int {
	// Calculate the reward credits earned on the new lock.
	rates := k.GetParams(ctx).RewardRates
	lockedAmt := k.TotalAmountByAddr(ctx, delAddr)
	credit := calculateCredit(rates, lockedAmt, lockingAmt)

	// Pro-rate the credit based on the time elapsed in the current epoch.
	epochInfo := k.epochsKeeper.GetEpochInfo(ctx, types.EpochIdentifier)
	sinceCurrentEpoch := time.Since(epochInfo.CurrentEpochStartTime).Milliseconds()
	epochDuration := epochInfo.Duration.Milliseconds()

	// TODO: is this check necessary?
	// Under what condition can sinceCurrentEpoch be greater than epochDuration?
	// What happens if the chain is paused for a long time?
	if sinceCurrentEpoch < epochDuration {
		credit = credit.MulRaw(sinceCurrentEpoch).QuoRaw(epochDuration)
	}

	return credit
}

// burnAllCredits burns all the reward credits in the system.
// It is called at the end of each epoch.
func (k Keeper) burnAllCredits(ctx context.Context) error {
	// Note that we can't simply iterate through the lockup records because credits
	// are transferrable and can be stored in accounts that are not tracked by lockups.
	// Instead, we iterate through all the balances to find and burn the credits.
	var err error

	cb := func(addr sdk.AccAddress, coin sdk.Coin) (stop bool) {
		if coin.Denom != appparams.CreditDenom {
			return false
		}

		coins := sdk.NewCoins(coin)

		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins)
		if err != nil {
			err = errorsmod.Wrapf(err, "send %s from %s to module", coins, addr)
			return true
		}

		err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
		if err != nil {
			err = errorsmod.Wrapf(err, "burn %s", coins)
			return true
		}

		return false
	}

	k.bankKeeper.IterateAllBalances(ctx, cb)

	return err
}

// resetAllCredits resets all the credits in the system.
func (k Keeper) resetAllCredits(ctx context.Context) error {
	// Reward to a delegator is calculated based on the total locked amount
	// to all validators. Since each lockup entry only records locked amount
	// for a single validator, we need to iterate through all the lockups to
	// calculate the total locked amount for each delegator.
	lockedAmts := make(map[string]math.Int)

	cb := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		amt, ok := lockedAmts[delAddr.String()]
		if !ok {
			amt = math.ZeroInt()
		}
		lockedAmts[delAddr.String()] = amt.Add(lockup.Amount)
	}

	k.MustIterateLockups(ctx, false, cb)

	rates := k.GetParams(ctx).RewardRates

	for delStrAddr, amt := range lockedAmts {
		delAddr := sdk.MustAccAddressFromBech32(delStrAddr)
		credit := calculateCredit(rates, math.ZeroInt(), amt)
		err := k.mintCredit(ctx, delAddr, credit)
		if err != nil {
			return errorsmod.Wrapf(err, "mint %s to %s", credit, delAddr)
		}
	}

	return nil
}

// calculateCredit calculates the reward earned on the lockingAmt.
// lockingAmt is stacked up on top of the lockedAmt to earn at the
// highest eligible reward.
func calculateCredit(rateList []types.Rate, lockedAmt, lockingAmt math.Int) math.Int {
	credit := math.ZeroInt()
	stakedAmt := lockedAmt.Add(lockingAmt)

	// Iterate from the highest reward rate to the lowest.
	for _, r := range rateList {
		// Continue if the total lock does not reach the current rate requirement.
		if stakedAmt.LT(r.Amount) {
			continue
		}

		lower := math.MaxInt(r.Amount, lockedAmt)
		diff := stakedAmt.Sub(lower)

		diffDec := math.LegacyNewDecFromInt(diff)
		rateDec := math.LegacyNewDec(r.Rate)

		amt := diffDec.Mul(rateDec).Quo(math.LegacyNewDec(100))
		credit = credit.Add(amt.TruncateInt())

		// Subtract the lock that has been rewarded.
		stakedAmt = stakedAmt.Sub(diff)
		lockingAmt = lockingAmt.Sub(diff)

		// Break if all the new lock has been rewarded.
		if lockingAmt.IsZero() {
			break
		}
	}

	return credit
}
