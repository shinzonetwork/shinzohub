package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

type queryServer struct {
	Keeper
}

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func (q queryServer) Balance(goCtx context.Context, req *types.QueryBalanceRequest) (*types.QueryBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	holder, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid bech32 address")
	}

	balance := q.Keeper.GetBalance(ctx, holder)
	return &types.QueryBalanceResponse{Amount: balance.String()}, nil
}

func (q queryServer) Balances(goCtx context.Context, req *types.QueryBalancesRequest) (*types.QueryBalancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	balances, pageRes, err := q.Keeper.GetAllBalances(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryBalancesResponse{
		Balances:   balances,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) EffectiveBalance(goCtx context.Context, req *types.QueryEffectiveBalanceRequest) (*types.QueryEffectiveBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	holder, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid bech32 address")
	}

	actual := q.Keeper.QueryBalanceKeeper().GetBalance(ctx, holder)
	pending := q.Keeper.GetPendingDebitTotal(ctx, holder)
	effective := q.Keeper.GetEffectiveBalance(ctx, holder)

	return &types.QueryEffectiveBalanceResponse{
		Address:      holder.String(),
		Actual:       actual.String(),
		PendingDebit: pending.String(),
		Effective:    effective.String(),
	}, nil
}
