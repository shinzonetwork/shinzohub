package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.Params(ctx, &types.QueryParamsRequest{})

	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestLockupQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	delAddr := sdk.AccAddress("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	valAddr := sdk.ValAddress("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	amount := math.NewInt(1000)

	keeper.AddLockup(ctx, delAddr, valAddr, amount)

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.Lockup(ctx, &types.LockupRequest{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
	})

	require.NoError(t, err)
	require.Equal(t, &types.LockupResponse{
		Lockup: types.Lockup{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
			Amount:           amount,
		},
	}, response)
}

func TestLockupsQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	delAddr := sdk.AccAddress("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	valAddr := sdk.ValAddress("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	keeper.AddLockup(ctx, delAddr, valAddr, amount1)
	keeper.AddLockup(ctx, delAddr, valAddr, amount2)

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.Lockups(ctx, &types.LockupsRequest{
		DelegatorAddress: delAddr.String(),
	})

	require.NoError(t, err)
	require.Len(t, response.Lockup, 1)
	require.Equal(t, amount1.Add(amount2), response.Lockup[0].Amount)
}

func TestUnlockingLockupQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	delAddr := sdk.AccAddress("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	valAddr := sdk.ValAddress("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	amount := math.NewInt(1000)

	unbondTime := time.Now().Add(24 * time.Hour).UTC()
	unlockTime := time.Now().Add(48 * time.Hour).UTC()

	keeper.SetLockup(ctx, true, delAddr, valAddr, amount, 1, &unbondTime, &unlockTime)

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.UnlockingLockup(ctx, &types.UnlockingLockupRequest{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
	})

	require.NoError(t, err)
	require.Equal(t, &types.UnlockingLockupResponse{
		Lockup: types.Lockup{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
			Amount:           amount,
			UnbondTime:       &unbondTime,
			UnlockTime:       &unlockTime,
		},
	}, response)
}

func TestUnlockingLockupsQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	delAddr := sdk.AccAddress("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	valAddr := sdk.ValAddress("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	unbondTime1 := time.Now().Add(24 * time.Hour).UTC()
	unlockTime1 := time.Now().Add(48 * time.Hour).UTC()
	unbondTime2 := time.Now().Add(36 * time.Hour).UTC()
	unlockTime2 := time.Now().Add(72 * time.Hour).UTC()

	keeper.SetLockup(ctx, true, delAddr, valAddr, amount1, 1, &unbondTime1, &unlockTime1)
	keeper.SetLockup(ctx, true, delAddr, valAddr, amount2, 2, &unbondTime2, &unlockTime2)

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.UnlockingLockups(ctx, &types.UnlockingLockupsRequest{
		DelegatorAddress: delAddr.String(),
	})

	require.NoError(t, err)
	require.Len(t, response.Lockup, 1)
	// TODO: at the moment, SetLockup() overrides existing lockup.
	// This needs to be changed in favor of storing unlocking lockups separately.
	// After the change, this test should be updated to use new SetLockup logic
	// and the response.Lockup length check above should return 2 as expected.
}
