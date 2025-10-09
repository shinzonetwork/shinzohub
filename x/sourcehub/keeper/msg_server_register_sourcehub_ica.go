package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

// RegisterSourcehubICA handles MsgRegisterSourcehubICA
func (m msgServer) RegisterSourcehubICA(goCtx context.Context, msg *types.MsgRegisterSourcehubICA) (*types.MsgRegisterSourcehubICAResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.Keeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}

	m.Keeper.SetControllerConnectionID(ctx, msg.HostConnectionId)
	m.Keeper.SetHostConnectionID(ctx, msg.HostConnectionId)

	if err := m.Keeper.IcaCtrlKeeper.RegisterInterchainAccount(
		ctx,
		m.Keeper.GetControllerConnectionID(ctx),
		types.ModuleAddress.String(),
		m.Keeper.GetICAMetadata(ctx),
		channeltypes.ORDERED,
	); err != nil {
		return nil, err
	}

	return &types.MsgRegisterSourcehubICAResponse{}, nil
}
