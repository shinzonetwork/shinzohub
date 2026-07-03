package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

func TestGetCurrentEpoch_DerivedFromBlockHeight(t *testing.T) {
	f := newFixture(t)

	// epoch 100 starts at height 100 * 180 = 18000
	f.ctx = f.ctx.WithBlockHeight(18_000)
	require.Equal(t, uint64(100), f.keeper.GetCurrentEpoch(f.ctx))

	// epoch 100 still — anywhere within the 180-block window
	f.ctx = f.ctx.WithBlockHeight(18_000 + 179)
	require.Equal(t, uint64(100), f.keeper.GetCurrentEpoch(f.ctx))

	// epoch 101 starts at height 18180
	f.ctx = f.ctx.WithBlockHeight(18_180)
	require.Equal(t, uint64(101), f.keeper.GetCurrentEpoch(f.ctx))
}

func TestGetCurrentEpoch_ZeroAtGenesis(t *testing.T) {
	f := newFixture(t)
	f.ctx = f.ctx.WithBlockHeight(0)
	require.Equal(t, uint64(0), f.keeper.GetCurrentEpoch(f.ctx),
		"block height 0 must yield epoch 0, not panic or huge number")
}

func TestGetLastSettledEpoch_DefaultsToZero(t *testing.T) {
	f := newFixture(t)
	require.Equal(t, uint64(0), f.keeper.GetLastSettledEpoch(f.ctx))
}

func TestSetLastSettledEpoch_PersistsAcrossReads(t *testing.T) {
	f := newFixture(t)

	f.keeper.SetLastSettledEpoch(f.ctx, 42)
	require.Equal(t, uint64(42), f.keeper.GetLastSettledEpoch(f.ctx))

	f.keeper.SetLastSettledEpoch(f.ctx, 43)
	require.Equal(t, uint64(43), f.keeper.GetLastSettledEpoch(f.ctx))
}

func TestSetLastSettledEpoch_AcceptsLargeValues(t *testing.T) {
	f := newFixture(t)

	const huge uint64 = 1<<63 + 5
	f.keeper.SetLastSettledEpoch(f.ctx, huge)
	require.Equal(t, huge, f.keeper.GetLastSettledEpoch(f.ctx),
		"epoch is stored as full uint64, must round-trip past int64 max")
}

func TestEpochBlocksIs180(t *testing.T) {
	require.Equal(t, int64(180), types.EpochBlocks,
		"settlement epoch is locked at 180 blocks")
}

func TestGenesis_RoundTripsLastSettledEpoch(t *testing.T) {
	src := newFixture(t)
	require.NoError(t, src.keeper.Credit(src.ctx, addr(1), math.NewInt(100)))
	src.keeper.SetLastSettledEpoch(src.ctx, 7)

	exported := src.keeper.ExportGenesis(src.ctx)
	require.Equal(t, uint64(7), exported.LastSettledEpoch)

	dst := newFixture(t)
	dst.keeper.InitGenesis(dst.ctx, *exported)

	require.Equal(t, uint64(7), dst.keeper.GetLastSettledEpoch(dst.ctx),
		"InitGenesis must restore the epoch cursor")
	require.Equal(t, math.NewInt(100), dst.keeper.GetBalance(dst.ctx, addr(1)),
		"existing balances should still round-trip too")
}

func TestGenesis_LastSettledEpochZeroIsSkipped(t *testing.T) {
	dst := newFixture(t)
	dst.keeper.SetLastSettledEpoch(dst.ctx, 9)

	dst.keeper.InitGenesis(dst.ctx, types.GenesisState{
		Balances:         []types.SettlementBalance{},
		LastSettledEpoch: 0,
	})

	require.Equal(t, uint64(9), dst.keeper.GetLastSettledEpoch(dst.ctx),
		"InitGenesis with epoch=0 should be a no-op (default), not clobber prior state")
}
