package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shinzonetwork/shinzohub/x/host/types"
)

type queryServer struct {
	Keeper
}

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func (q queryServer) Hosts(goCtx context.Context, req *types.QueryHostsRequest) (*types.QueryHostsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	hosts, pageRes, err := q.Keeper.GetAllHosts(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryHostsResponse{
		Hosts:      hosts,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) Host(goCtx context.Context, req *types.QueryHostRequest) (*types.QueryHostResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	host, found, err := q.Keeper.GetHost(ctx, req.Address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "host not found")
	}

	return &types.QueryHostResponse{Host: host}, nil
}

func (q queryServer) HostCount(goCtx context.Context, req *types.QueryHostCountRequest) (*types.QueryHostCountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	count := q.Keeper.GetHostCount(ctx)

	return &types.QueryHostCountResponse{Count: count}, nil
}
