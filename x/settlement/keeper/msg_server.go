package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

func (m msgServer) Claim(goCtx context.Context, msg *types.MsgClaim) (*types.MsgClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	claimer, err := sdk.AccAddressFromBech32(msg.Claimer)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("claimer: %s", err)
	}

	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("amount %q is not an integer", msg.Amount)
	}

	if err := m.Keeper.Claim(ctx, claimer, amount); err != nil {
		return nil, fmt.Errorf("claim: %w", err)
	}

	remaining := m.Keeper.GetBalance(ctx, claimer)
	return &types.MsgClaimResponse{Remaining: remaining.String()}, nil
}
