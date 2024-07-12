package keeper_test

import (
	"testing"

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
