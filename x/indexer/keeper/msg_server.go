package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	commoncrypto "github.com/shinzonetwork/shinzohub/x/common/crypto"
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

	if err := commoncrypto.VerifyDelegateSignature(msg.DelegateAddress, msg.DelegateDigest, msg.DelegateSignature); err != nil {
		return nil, sdkerrors.ErrUnauthorized.Wrap(err.Error())
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
			"IndexerAsserted",
			sdk.NewAttribute("signer", msg.Signer),
			sdk.NewAttribute("consensus_pub_key", msg.ConsensusPubKey),
			sdk.NewAttribute("delegate_address", msg.DelegateAddress),
			sdk.NewAttribute("source_chain", msg.SourceChain),
			sdk.NewAttribute("source_chain_id", fmt.Sprintf("%d", msg.SourceChainId)),
			sdk.NewAttribute("assertion_id", msg.AssertionId),
		),
	)

	return &types.MsgIndexerAssertionResponse{}, nil
}
