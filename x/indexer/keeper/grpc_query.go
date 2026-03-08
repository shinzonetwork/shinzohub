package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

type queryServer struct {
	Keeper
}

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func (q queryServer) Indexers(goCtx context.Context, req *types.QueryIndexersRequest) (*types.QueryIndexersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	indexers, pageRes, err := q.Keeper.GetAllIndexers(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryIndexersResponse{
		Indexers:   indexers,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) Indexer(goCtx context.Context, req *types.QueryIndexerRequest) (*types.QueryIndexerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	indexer, found, err := q.Keeper.GetIndexer(ctx, req.Address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "indexer not found")
	}

	return &types.QueryIndexerResponse{Indexer: indexer}, nil
}

func (q queryServer) IndexerCount(goCtx context.Context, req *types.QueryIndexerCountRequest) (*types.QueryIndexerCountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	count := q.Keeper.GetIndexerCount(ctx)

	return &types.QueryIndexerCountResponse{Count: count}, nil
}
