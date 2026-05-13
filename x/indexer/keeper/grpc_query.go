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

func (q queryServer) Indexers(
	goCtx context.Context,
	req *types.QueryIndexersRequest,
) (*types.QueryIndexersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	indexers, pageRes, err := q.Keeper.IterateIndexers(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryIndexersResponse{
		Indexers:   indexers,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) Indexer(
	goCtx context.Context,
	req *types.QueryIndexerRequest,
) (*types.QueryIndexerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.SourceChainId == 0 || len(req.ValidatorPubkey) == 0 {
		return nil, status.Error(codes.InvalidArgument, "source_chain_id and validator_pubkey are required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	ix, found, err := q.Keeper.GetIndexerByValidator(ctx, req.SourceChainId, req.ValidatorPubkey)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "indexer not found")
	}
	return &types.QueryIndexerResponse{Indexer: ix}, nil
}

func (q queryServer) IndexerByAddress(
	goCtx context.Context,
	req *types.QueryIndexerByAddressRequest,
) (*types.QueryIndexerByAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.OperatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "operator_address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	ix, found, err := q.Keeper.GetIndexerByAddress(ctx, req.OperatorAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "indexer not found")
	}
	return &types.QueryIndexerByAddressResponse{Indexer: ix}, nil
}

func (q queryServer) IndexerCount(
	goCtx context.Context,
	req *types.QueryIndexerCountRequest,
) (*types.QueryIndexerCountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryIndexerCountResponse{Count: q.Keeper.GetIndexerCount(ctx)}, nil
}
