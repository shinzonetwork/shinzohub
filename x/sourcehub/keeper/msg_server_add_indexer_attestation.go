package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

// AddIndexerAttestation records an indexer's attestation on SourceHub and
// emits an event. Actual role registration still happens in RegisterEntity.
func (m msgServer) AddIndexerAttestation(
	goCtx context.Context,
	msg *types.MsgIndexerAttestation,
) (*types.MsgIndexerAttestationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.Keeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}

	// Verify the delegate signed the digest with their source-chain key.
	if err := verifyDelegateSignature(msg.DelegateAddress, msg.DelegateDigest, msg.DelegateSignature); err != nil {
		return nil, sdkerrors.ErrUnauthorized.Wrap(err.Error())
	}

	// Persist/update the indexer attestation keyed by delegate address.
	if err := m.Keeper.SetIndexerAttestation(ctx, types.IndexerAttestation{
		ConsensusPubKey: msg.ConsensusPubKey,
		DelegateAddress: msg.DelegateAddress,
		SourceChain:     msg.SourceChain,
		SourceChainId:   msg.SourceChainId,
		AttestationId:   msg.AttestationId,
	}); err != nil {
		return nil, err
	}

	// Emit a structured event so off-chain services can react.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"IndexerAttested",
			sdk.NewAttribute("signer", msg.Signer),
			sdk.NewAttribute("consensus_pub_key", msg.ConsensusPubKey),
			sdk.NewAttribute("delegate_address", msg.DelegateAddress),
			sdk.NewAttribute("source_chain", msg.SourceChain),
			sdk.NewAttribute("source_chain_id", fmt.Sprintf("%d", msg.SourceChainId)),
			sdk.NewAttribute("attestation_id", msg.AttestationId),
		),
	)

	return &types.MsgIndexerAttestationResponse{}, nil
}
