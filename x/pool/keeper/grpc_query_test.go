package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	poolkeeper "github.com/shinzonetwork/shinzohub/x/pool/keeper"
	"github.com/shinzonetwork/shinzohub/x/pool/types"
)

// TestPoolsForView checks the by-view query returns every pool registered for the
// view, scoped to that view, with the correct per-pool is_active and window so a
// caller can pick an active pool to serve.
func TestPoolsForView(t *testing.T) {
	f := newFixture(t)
	q := poolkeeper.NewQueryServerImpl(f.keeper)

	const view = "0xview1"
	f.views.registerView(view)

	// Active pool: window 100, at the 3-host activation threshold.
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpoolA", view, types.PoolConfig{WindowSize: 100}))
	for _, h := range []string{"0xh1", "0xh2", "0xh3"} {
		require.NoError(t, f.keeper.AddHost(f.ctx, "0xpoolA", h))
	}
	// Pending pool for the same view: window 200, one host short of active.
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpoolB", view, types.PoolConfig{WindowSize: 200}))
	for _, h := range []string{"0xh4", "0xh5"} {
		require.NoError(t, f.keeper.AddHost(f.ctx, "0xpoolB", h))
	}
	// A pool on another view must not leak into the result.
	f.views.registerView("0xview2")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpoolC", "0xview2", types.PoolConfig{WindowSize: 100}))

	resp, err := q.PoolsForView(f.ctx, &types.QueryPoolsForViewRequest{ViewAddress: view})
	require.NoError(t, err)
	require.Len(t, resp.Details, 2)

	byPool := make(map[string]types.PoolDetail, len(resp.Details))
	for _, d := range resp.Details {
		byPool[d.Pool.PoolAddress] = d
	}

	a, ok := byPool["0xpoolA"]
	require.True(t, ok)
	require.True(t, a.IsActive)
	require.Equal(t, uint64(100), a.Pool.Config.WindowSize)

	b, ok := byPool["0xpoolB"]
	require.True(t, ok)
	require.False(t, b.IsActive)
	require.Equal(t, uint64(200), b.Pool.Config.WindowSize)
}

func TestPoolsForView_Validation(t *testing.T) {
	f := newFixture(t)
	q := poolkeeper.NewQueryServerImpl(f.keeper)

	_, err := q.PoolsForView(f.ctx, &types.QueryPoolsForViewRequest{})
	require.Error(t, err)

	// A view with no pools is a valid, empty result, not an error.
	resp, err := q.PoolsForView(f.ctx, &types.QueryPoolsForViewRequest{ViewAddress: "0xnopools"})
	require.NoError(t, err)
	require.Empty(t, resp.Details)
}
