package shinzohub_test

import (
	"testing"

	keepertest "shinzohub/testutil/keeper"
	"shinzohub/testutil/nullify"
	shinzohub "shinzohub/x/shinzohub/module"
	"shinzohub/x/shinzohub/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ShinzohubKeeper(t)
	shinzohub.InitGenesis(ctx, k, genesisState)
	got := shinzohub.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
