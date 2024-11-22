package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/app"
	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func init() {
	app.SetConfig(true)
}

func setupKeeper(t *testing.T) (*keeper.Keeper, sdk.Context) {
	k, ctx := keepertest.TierKeeper(t)
	return &k, ctx
}

func TestSetAndGetLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1000)
	creationHeight := int64(10)
	unbondTime := time.Now().Add(1 * time.Hour)
	unlockTime := time.Now().Add(2 * time.Hour)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.SetLockup(ctx, false, delAddr, valAddr, amount, creationHeight, &unbondTime, &unlockTime)

	store := k.GetAllLockups(ctx)
	require.Len(t, store, 1)

	lockup := store[0]
	require.Equal(t, delAddr.String(), lockup.DelegatorAddress)
	require.Equal(t, valAddr.String(), lockup.ValidatorAddress)
	require.Equal(t, amount, lockup.Amount)
	require.Equal(t, creationHeight, lockup.CreationHeight)
	require.Equal(t, unbondTime.UTC(), lockup.UnbondTime.UTC())
	require.Equal(t, unlockTime.UTC(), lockup.UnlockTime.UTC())
}

func TestAddLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.AddLockup(ctx, delAddr, valAddr, amount)

	lockup := k.GetLockup(ctx, delAddr, valAddr)
	require.Equal(t, amount, lockup)
}

func TestSubtractLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.AddLockup(ctx, delAddr, valAddr, amount)

	err = k.SubtractLockup(ctx, delAddr, valAddr, math.NewInt(500))
	require.NoError(t, err)

	lockup := k.GetLockup(ctx, delAddr, valAddr)
	require.Equal(t, math.NewInt(500), lockup)
}

func TestGetAllLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr1, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.Nil(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.SetLockup(ctx, false, delAddr1, valAddr1, amount1, 1, nil, nil)
	k.SetLockup(ctx, false, delAddr2, valAddr2, amount2, 2, nil, nil)

	lockups := k.GetAllLockups(ctx)
	require.Len(t, lockups, 2)

	require.Equal(t, delAddr1.String(), lockups[0].DelegatorAddress)
	require.Equal(t, valAddr1.String(), lockups[0].ValidatorAddress)
	require.Equal(t, delAddr2.String(), lockups[1].DelegatorAddress)
	require.Equal(t, valAddr2.String(), lockups[1].ValidatorAddress)
}

func TestIterateLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.AddLockup(ctx, delAddr, valAddr, amount)

	count := 0
	k.MustIterateLockups(ctx, false, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		require.Equal(t, "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9", delAddr.String())
		require.Equal(t, "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm", valAddr.String())
		require.Equal(t, amount, lockup.Amount)
		count++
	})

	require.Equal(t, 1, count)
}
