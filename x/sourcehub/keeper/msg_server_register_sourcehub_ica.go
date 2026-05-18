package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (m msgServer) RegisterSourcehubICA(goCtx context.Context, msg *types.MsgRegisterSourcehubICA) (*types.MsgRegisterSourcehubICAResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.adminKeeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}

	m.SetControllerConnectionID(ctx, msg.ControllerConnectionId)
	m.SetHostConnectionID(ctx, msg.HostConnectionId)

	if err := m.IcaCtrlKeeper.RegisterInterchainAccount(
		ctx,
		m.GetControllerConnectionID(ctx),
		types.ModuleAddress.String(),
		m.GetICAMetadata(ctx),
		channeltypes.ORDERED,
	); err != nil {
		return nil, err
	}

	return &types.MsgRegisterSourcehubICAResponse{}, nil
}
