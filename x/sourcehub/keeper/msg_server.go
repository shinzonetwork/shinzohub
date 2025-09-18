package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

// msgServer is the concrete implementation of the MsgServer interface
type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

// RegisterSourcehubICA handles MsgRegisterSourcehubICA
func (m msgServer) RegisterSourcehubICA(goCtx context.Context, msg *types.MsgRegisterSourcehubICA) (*types.MsgRegisterSourcehubICAResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	m.Keeper.SetControllerConnectionID(ctx, msg.HostConnectionId)
	m.Keeper.SetHostConnectionID(ctx, msg.HostConnectionId)

	if err := m.Keeper.RegisterSourcehubICA(ctx); err != nil {
		return nil, err
	}

	return &types.MsgRegisterSourcehubICAResponse{}, nil
}

func (m msgServer) RequestStreamAccess(goCtx context.Context, msg *types.MsgRequestStreamAccess) (*types.MsgRequestStreamAccessResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.RequestStreamAccess(ctx, msg.StreamId, msg.Did); err != nil {
		return nil, err
	}

	return &types.MsgRequestStreamAccessResponse{}, nil
}
