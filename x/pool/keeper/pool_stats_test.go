package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestPoolStats_DefaultsToZeroForNewPool(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	stats := f.keeper.GetPoolStats(f.ctx, "0xpool")
	require.Equal(t, "0xpool", stats.PoolAddress)
	require.Equal(t, uint64(0), stats.Utilization)
	require.Equal(t, uint64(0), stats.TotalQueries)
	require.Equal(t, "0", stats.TotalRewards)
	require.Equal(t, uint64(0), stats.LastUpdatedEpoch)
}

func TestUpdatePoolStats_FirstUpdate(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	err := f.keeper.UpdatePoolStats(
		f.ctx, "0xpool",
		math.NewInt(9995),     // price
		32,                    // utilization
		10,                    // addQueries
		math.NewInt(1_000_000), // addRewards
		100,                   // epoch
	)
	require.NoError(t, err)

	stats := f.keeper.GetPoolStats(f.ctx, "0xpool")
	require.Equal(t, "9995", stats.Price)
	require.Equal(t, uint64(32), stats.Utilization)
	require.Equal(t, uint64(10), stats.TotalQueries)
	require.Equal(t, "1000000", stats.TotalRewards)
	require.Equal(t, uint64(100), stats.LastUpdatedEpoch)
}

func TestUpdatePoolStats_AccumulatesQueriesAndRewards(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	require.NoError(t, f.keeper.UpdatePoolStats(f.ctx, "0xpool", math.NewInt(100), 30, 10, math.NewInt(500), 100))
	require.NoError(t, f.keeper.UpdatePoolStats(f.ctx, "0xpool", math.NewInt(100), 30, 25, math.NewInt(1500), 100))

	stats := f.keeper.GetPoolStats(f.ctx, "0xpool")
	require.Equal(t, uint64(35), stats.TotalQueries, "queries accumulate across updates")
	require.Equal(t, "2000", stats.TotalRewards, "rewards accumulate across updates")
}

func TestUpdatePoolStats_PriceAndUtilizationAreLastWins(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	require.NoError(t, f.keeper.UpdatePoolStats(f.ctx, "0xpool", math.NewInt(100), 30, 0, math.ZeroInt(), 100))
	require.NoError(t, f.keeper.UpdatePoolStats(f.ctx, "0xpool", math.NewInt(200), 50, 0, math.ZeroInt(), 100))

	stats := f.keeper.GetPoolStats(f.ctx, "0xpool")
	require.Equal(t, "200", stats.Price, "later price overwrites earlier")
	require.Equal(t, uint64(50), stats.Utilization, "later utilization overwrites earlier")
}

func TestUpdatePoolStats_RejectsUnknownPool(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.UpdatePoolStats(f.ctx, "0xpool-ghost", math.NewInt(100), 30, 10, math.ZeroInt(), 100)
	require.ErrorContains(t, err, "pool 0xpool-ghost not found")
}

func TestPoolDetail_IncludesStats(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))
	require.NoError(t, f.keeper.UpdatePoolStats(f.ctx, "0xpool", math.NewInt(9995), 32, 10, math.NewInt(1_000_000), 100))

	detail, found, err := f.keeper.GetPoolDetail(f.ctx, "0xpool")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "9995", detail.Stats.Price)
	require.Equal(t, uint64(32), detail.Stats.Utilization)
	require.Equal(t, uint64(10), detail.Stats.TotalQueries)
}
