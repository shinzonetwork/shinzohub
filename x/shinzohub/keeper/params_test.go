package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "shinzohub/testutil/keeper"
	"shinzohub/x/shinzohub/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.ShinzohubKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
