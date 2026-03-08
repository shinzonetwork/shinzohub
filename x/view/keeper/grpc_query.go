package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shinzonetwork/shinzohub/x/view/types"
)

type queryServer struct {
	Keeper
}

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func (q queryServer) Views(goCtx context.Context, req *types.QueryViewsRequest) (*types.QueryViewsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	views, pageRes, err := q.Keeper.GetAllViews(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	filtered := make([]types.View, 0, len(views))
	for _, v := range views {
		if req.SinceBlock > 0 && v.Height < req.SinceBlock {
			continue
		}
		if !req.IncludeData {
			v.Data = nil
		}
		filtered = append(filtered, v)
	}

	return &types.QueryViewsResponse{
		Views:      filtered,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) View(goCtx context.Context, req *types.QueryViewRequest) (*types.QueryViewResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	view, found, err := q.Keeper.GetView(ctx, req.ContractAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "view not found")
	}

	if !req.IncludeData {
		view.Data = nil
	}

	return &types.QueryViewResponse{View: view}, nil
}

func (q queryServer) ViewCount(goCtx context.Context, req *types.QueryViewCountRequest) (*types.QueryViewCountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	count := q.Keeper.GetViewCount(ctx)

	return &types.QueryViewCountResponse{Count: count}, nil
}
