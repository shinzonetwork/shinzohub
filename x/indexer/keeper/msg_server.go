package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

func (m msgServer) AddIndexerAssertion(
	goCtx context.Context,
	msg *types.MsgIndexerAssertion,
) (*types.MsgIndexerAssertionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if !m.Keeper.adminKeeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}
	if err := m.Keeper.UpsertAssertion(ctx, msg); err != nil {
		return nil, err
	}
	return &types.MsgIndexerAssertionResponse{}, nil
}

func (m msgServer) SetPayout(
	goCtx context.Context,
	msg *types.MsgSetPayout,
) (*types.MsgSetPayoutResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if !m.Keeper.adminKeeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}
	if err := m.Keeper.SetPayout(ctx, msg); err != nil {
		return nil, err
	}
	return &types.MsgSetPayoutResponse{}, nil
}

func (m msgServer) RevokeIndexer(
	goCtx context.Context,
	msg *types.MsgRevokeIndexer,
) (*types.MsgRevokeIndexerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if !m.Keeper.adminKeeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}
	if err := m.Keeper.RevokeIndexer(ctx, msg); err != nil {
		return nil, err
	}
	return &types.MsgRevokeIndexerResponse{}, nil
}
