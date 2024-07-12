package keeper

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// GetAllockups returns all lockups in the store.
// It is only used for exporting all lockups as part of the app state.
func (k Keeper) GetAllLockups(ctx context.Context) []types.Lockup {

	var lockups []types.Lockup

	cb := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		lockups = append(lockups, lockup)
	}

	k.mustIterateLockups(ctx, true, cb)
	k.mustIterateLockups(ctx, false, cb)

	return lockups
}

func (k Keeper) setLockup(ctx context.Context, unlocking bool, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int, creationHeight int64, unbondTime *time.Time, unlockTime *time.Time) {

	key := types.LockupKey(delAddr, valAddr)
	lockup := &types.Lockup{
		ValidatorAddress: valAddr.String(),
		Amount:           amt,
		CreationHeight:   creationHeight,
		UnbondTime:       unbondTime,
		UnlockTime:       unlockTime,
	}
	b := k.cdc.MustMarshal(lockup)
	store := k.lockupStore(ctx, unlocking)
	store.Set(key, b)
}

func (k Keeper) getLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) math.Int {

	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, false)
	b := store.Get(key)
	if b == nil {
		return math.ZeroInt()
	}

	var lockup types.Lockup
	k.cdc.MustUnmarshal(b, &lockup)

	return lockup.Amount
}

func (k Keeper) getUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (
	found bool, amt math.Int, unbondTime time.Time, unlockTime time.Time) {

	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, true)
	b := store.Get(key)
	if b == nil {
		return false, math.ZeroInt(), time.Time{}, time.Time{}
	}

	var lockup types.Lockup
	k.cdc.MustUnmarshal(b, &lockup)

	return true, lockup.Amount, *lockup.UnbondTime, *lockup.UnlockTime
}

func (k Keeper) removeUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {

	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, true)
	store.Delete(key)
}

func (k Keeper) addLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) {

	lockedAmt := k.getLockup(ctx, delAddr, valAddr)
	amt = amt.Add(lockedAmt)

	k.setLockup(ctx, false, delAddr, valAddr, amt, 0, nil, nil)
}

func (k Keeper) subtractLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) error {

	lockedAmt := k.getLockup(ctx, delAddr, valAddr)

	lockedAmt, err := lockedAmt.SafeSub(amt)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from locked amount %s", amt, lockedAmt)
	}

	k.setLockup(ctx, false, delAddr, valAddr, lockedAmt, 0, nil, nil)

	return nil
}

func (k Keeper) TotalAmountByAddr(ctx context.Context, delAddr sdk.AccAddress) math.Int {

	amt := math.ZeroInt()

	cb := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		if delAddr.Equals(delAddr) {
			amt = amt.Add(lockup.Amount)
		}
	}

	k.mustIterateLockups(ctx, false, cb)

	return amt
}

// iterateLockups iterates over all lockups in the store and performs the provided callback function.
// The iterator itself doesn't return an error, but the callback does.
// If the callback returns an error, the iteration stops and the error is returned.
func (k Keeper) iterateLockups(ctx context.Context, unlocking bool,
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) error) error {

	store := k.lockupStore(ctx, unlocking)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {

		var lockup types.Lockup
		k.cdc.MustUnmarshal(iterator.Value(), &lockup)

		delAddr, valAddr := types.LockupKeyToAddresses(iterator.Key())
		err := cb(delAddr, valAddr, lockup)
		if err != nil {
			return errorsmod.Wrapf(err, "%s/%s/, amt: %s", delAddr, valAddr, lockup.Amount)
		}
	}

	return nil
}

// mustIterateLockups iterates over all lockups in the store and performs the provided callback function.
func (k Keeper) mustIterateLockups(ctx context.Context, unlocking bool,
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup)) {

	store := k.lockupStore(ctx, unlocking)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {

		var lockup types.Lockup
		k.cdc.MustUnmarshal(iterator.Value(), &lockup)

		delAddr, valAddr := types.LockupKeyToAddresses(iterator.Key())
		cb(delAddr, valAddr, lockup)
	}
}

func (k Keeper) lockupStore(ctx context.Context, unlocking bool) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	storePrefix := types.KeyPrefix(unlocking)
	return prefix.NewStore(storeAdapter, storePrefix)
}
