package keeper

import (
	"context"
	"strings"

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
	hosts := make([]types.Host, 0)
	pageRes, err := q.Keeper.FilterHosts(ctx, req.Pagination, func(host types.Host, accumulate bool) (bool, error) {
		if req.Did != "" && host.Did != req.Did {
			return false, nil
		}
		if req.ConnectionString != "" && !strings.Contains(host.ConnectionString, req.ConnectionString) {
			return false, nil
		}

		if accumulate {
			hosts = append(hosts, host)
		}
		return true, nil
	})
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
