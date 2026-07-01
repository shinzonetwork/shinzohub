package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shinzonetwork/shinzohub/x/pool/types"
)

type queryServer struct {
	Keeper
}

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func (q queryServer) Pool(goCtx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.PoolAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "pool_address is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	pool, found, err := q.Keeper.GetPool(ctx, req.PoolAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "pool not found")
	}

	return &types.QueryPoolResponse{Pool: pool}, nil
}

func (q queryServer) Pools(goCtx context.Context, req *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	pools, pageRes, err := q.Keeper.GetAllPools(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryPoolsResponse{
		Pools:      pools,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) Detail(goCtx context.Context, req *types.QueryDetailRequest) (*types.QueryDetailResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.PoolAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "pool_address is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	detail, found, err := q.Keeper.GetPoolDetail(ctx, req.PoolAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "pool not found")
	}

	return &types.QueryDetailResponse{Detail: detail}, nil
}

func (q queryServer) Details(goCtx context.Context, req *types.QueryDetailsRequest) (*types.QueryDetailsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	details, pageRes, err := q.Keeper.GetAllPoolDetails(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDetailsResponse{
		Details:    details,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) Hosts(goCtx context.Context, req *types.QueryHostsRequest) (*types.QueryHostsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.PoolAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "pool_address is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	hosts, pageRes, err := q.Keeper.GetAllHosts(ctx, req.PoolAddress, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryHostsResponse{
		Hosts:      hosts,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) Demands(goCtx context.Context, req *types.QueryDemandsRequest) (*types.QueryDemandsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.PoolAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "pool_address is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	demands, pageRes, err := q.Keeper.GetAllDemands(ctx, req.PoolAddress, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDemandsResponse{
		Demands:    demands,
		Pagination: pageRes,
	}, nil
}
