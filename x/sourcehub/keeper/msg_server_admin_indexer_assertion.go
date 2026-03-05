package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (m msgServer) AdminIndexerAssertion(
	goCtx context.Context,
	msg *types.MsgAdminIndexerAssertion,
) (*types.MsgAdminIndexerAssertionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.Keeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}

	if err := m.Keeper.SetIndexerAssertion(ctx, types.IndexerAssertion{
		ConsensusPubKey: msg.ConsensusPubKey,
		DelegateAddress: msg.DelegateAddress,
		SourceChain:     msg.SourceChain,
		SourceChainId:   msg.SourceChainId,
		AssertionId:     msg.AssertionId,
	}); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"AdminIndexerAsserted",
			sdk.NewAttribute("signer", msg.Signer),
			sdk.NewAttribute("consensus_pub_key", msg.ConsensusPubKey),
			sdk.NewAttribute("delegate_address", msg.DelegateAddress),
			sdk.NewAttribute("source_chain", msg.SourceChain),
			sdk.NewAttribute("source_chain_id", fmt.Sprintf("%d", msg.SourceChainId)),
			sdk.NewAttribute("assertion_id", msg.AssertionId),
		),
	)

	return &types.MsgAdminIndexerAssertionResponse{}, nil
}
