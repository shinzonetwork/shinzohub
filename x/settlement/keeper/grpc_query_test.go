package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	settlementkeeper "github.com/shinzonetwork/shinzohub/x/settlement/keeper"
	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

func TestQueryBalance_ReturnsAmountForKnownAddress(t *testing.T) {
	f := newFixture(t)
	a := addr(1)
	require.NoError(t, f.keeper.Credit(f.ctx, a, math.NewInt(750_000)))

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	resp, err := qs.Balance(sdk.WrapSDKContext(f.ctx), &types.QueryBalanceRequest{
		Address: a.String(),
	})

	require.NoError(t, err)
	require.Equal(t, "750000", resp.Amount)
}

func TestQueryBalance_ReturnsZeroForUnknownAddress(t *testing.T) {
	f := newFixture(t)
	a := addr(42)

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	resp, err := qs.Balance(sdk.WrapSDKContext(f.ctx), &types.QueryBalanceRequest{
		Address: a.String(),
	})

	require.NoError(t, err)
	require.Equal(t, "0", resp.Amount, "unknown address must read as zero, not error")
}

func TestQueryBalance_RejectsEmptyAddress(t *testing.T) {
	f := newFixture(t)

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	_, err := qs.Balance(sdk.WrapSDKContext(f.ctx), &types.QueryBalanceRequest{Address: ""})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestQueryBalance_RejectsInvalidBech32(t *testing.T) {
	f := newFixture(t)

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	_, err := qs.Balance(sdk.WrapSDKContext(f.ctx), &types.QueryBalanceRequest{
		Address: "not-a-bech32-address",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestQueryBalance_RejectsNilRequest(t *testing.T) {
	f := newFixture(t)

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	_, err := qs.Balance(sdk.WrapSDKContext(f.ctx), nil)

	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestQueryBalances_EmptyWhenLedgerEmpty(t *testing.T) {
	f := newFixture(t)

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	resp, err := qs.Balances(sdk.WrapSDKContext(f.ctx), &types.QueryBalancesRequest{})

	require.NoError(t, err)
	require.Empty(t, resp.Balances)
}

func TestQueryBalances_ReturnsAllEntries(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, f.keeper.Credit(f.ctx, addr(1), math.NewInt(100)))
	require.NoError(t, f.keeper.Credit(f.ctx, addr(2), math.NewInt(200)))
	require.NoError(t, f.keeper.Credit(f.ctx, addr(3), math.NewInt(300)))

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	resp, err := qs.Balances(sdk.WrapSDKContext(f.ctx), &types.QueryBalancesRequest{})

	require.NoError(t, err)
	require.Len(t, resp.Balances, 3)

	amounts := map[string]string{}
	for _, b := range resp.Balances {
		amounts[b.Address] = b.Amount
	}
	require.Equal(t, "100", amounts[addr(1).String()])
	require.Equal(t, "200", amounts[addr(2).String()])
	require.Equal(t, "300", amounts[addr(3).String()])
}

func TestQueryBalances_RespectsPagination(t *testing.T) {
	f := newFixture(t)
	for i := byte(1); i <= 10; i++ {
		require.NoError(t, f.keeper.Credit(f.ctx, addr(i), math.NewInt(int64(i)*10)))
	}

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)

	first, err := qs.Balances(sdk.WrapSDKContext(f.ctx), &types.QueryBalancesRequest{
		Pagination: &query.PageRequest{Limit: 4, CountTotal: true},
	})
	require.NoError(t, err)
	require.Len(t, first.Balances, 4)
	require.NotNil(t, first.Pagination)
	require.Equal(t, uint64(10), first.Pagination.Total)
	require.NotNil(t, first.Pagination.NextKey)

	second, err := qs.Balances(sdk.WrapSDKContext(f.ctx), &types.QueryBalancesRequest{
		Pagination: &query.PageRequest{Limit: 4, Key: first.Pagination.NextKey},
	})
	require.NoError(t, err)
	require.Len(t, second.Balances, 4)

	seen := map[string]struct{}{}
	for _, b := range first.Balances {
		seen[b.Address] = struct{}{}
	}
	for _, b := range second.Balances {
		_, dup := seen[b.Address]
		require.False(t, dup, "pagination yielded the same entry twice")
	}
}

func TestQueryBalances_RejectsNilRequest(t *testing.T) {
	f := newFixture(t)

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	_, err := qs.Balances(sdk.WrapSDKContext(f.ctx), nil)

	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestQueryBalance_ReflectsClaimAndCredit(t *testing.T) {
	f := newFixture(t)
	a := addr(5)
	require.NoError(t, f.keeper.Credit(f.ctx, a, math.NewInt(1_000)))

	qs := settlementkeeper.NewQueryServerImpl(f.keeper)
	resp1, err := qs.Balance(sdk.WrapSDKContext(f.ctx), &types.QueryBalanceRequest{Address: a.String()})
	require.NoError(t, err)
	require.Equal(t, "1000", resp1.Amount)

	require.NoError(t, f.keeper.Claim(f.ctx, a, math.NewInt(400)))

	resp2, err := qs.Balance(sdk.WrapSDKContext(f.ctx), &types.QueryBalanceRequest{Address: a.String()})
	require.NoError(t, err)
	require.Equal(t, "600", resp2.Amount, "query must reflect post-claim balance")
}
