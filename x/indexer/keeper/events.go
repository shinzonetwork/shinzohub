package keeper

import (
	"encoding/hex"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

func emitAsserted(ctx sdk.Context, row *types.Indexer) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIndexerAsserted,
		sdk.NewAttribute(types.AttrKeySourceChain, row.SourceChain),
		sdk.NewAttribute(types.AttrKeySourceChainID, strconv.FormatUint(row.SourceChainId, 10)),
		sdk.NewAttribute(types.AttrKeyValidatorPubkey, hex.EncodeToString(row.ValidatorPubkey)),
		sdk.NewAttribute(types.AttrKeyOperatorAddress, row.OperatorAddress),
		sdk.NewAttribute(types.AttrKeyPayoutAddress, row.PayoutAddress),
		sdk.NewAttribute(types.AttrKeyNonce, strconv.FormatUint(row.Nonce, 10)),
	))
}

func emitSuperseded(ctx sdk.Context, sourceChainID uint64, validatorPubkey []byte, oldAddr, newAddr string, oldNonce, newNonce uint64) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIndexerSuperseded,
		sdk.NewAttribute(types.AttrKeySourceChainID, strconv.FormatUint(sourceChainID, 10)),
		sdk.NewAttribute(types.AttrKeyValidatorPubkey, hex.EncodeToString(validatorPubkey)),
		sdk.NewAttribute(types.AttrKeyOldOperator, oldAddr),
		sdk.NewAttribute(types.AttrKeyNewOperator, newAddr),
		sdk.NewAttribute(types.AttrKeyOldNonce, strconv.FormatUint(oldNonce, 10)),
		sdk.NewAttribute(types.AttrKeyNewNonce, strconv.FormatUint(newNonce, 10)),
	))
}

func emitPayoutUpdated(ctx sdk.Context, row *types.Indexer) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIndexerPayoutUpdated,
		sdk.NewAttribute(types.AttrKeySourceChainID, strconv.FormatUint(row.SourceChainId, 10)),
		sdk.NewAttribute(types.AttrKeyValidatorPubkey, hex.EncodeToString(row.ValidatorPubkey)),
		sdk.NewAttribute(types.AttrKeyOperatorAddress, row.OperatorAddress),
		sdk.NewAttribute(types.AttrKeyPayoutAddress, row.PayoutAddress),
		sdk.NewAttribute(types.AttrKeyNonce, strconv.FormatUint(row.Nonce, 10)),
	))
}

func emitRevoked(ctx sdk.Context, row *types.Indexer, atNonce uint64) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIndexerRevoked,
		sdk.NewAttribute(types.AttrKeySourceChainID, strconv.FormatUint(row.SourceChainId, 10)),
		sdk.NewAttribute(types.AttrKeyValidatorPubkey, hex.EncodeToString(row.ValidatorPubkey)),
		sdk.NewAttribute(types.AttrKeyOperatorAddress, row.OperatorAddress),
		sdk.NewAttribute(types.AttrKeyNonce, strconv.FormatUint(atNonce, 10)),
	))
}

func emitPending(ctx sdk.Context, row *types.Indexer, did, connectionString string) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIndexerPending,
		sdk.NewAttribute(types.AttrKeySourceChainID, strconv.FormatUint(row.SourceChainId, 10)),
		sdk.NewAttribute(types.AttrKeyValidatorPubkey, hex.EncodeToString(row.ValidatorPubkey)),
		sdk.NewAttribute(types.AttrKeyOperatorAddress, row.OperatorAddress),
		sdk.NewAttribute(types.AttrKeyDID, did),
		sdk.NewAttribute(types.AttrKeyConnectionString, connectionString),
	))
}

func emitRegistered(ctx sdk.Context, row *types.Indexer) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIndexerRegistered,
		sdk.NewAttribute(types.AttrKeySourceChainID, strconv.FormatUint(row.SourceChainId, 10)),
		sdk.NewAttribute(types.AttrKeyValidatorPubkey, hex.EncodeToString(row.ValidatorPubkey)),
		sdk.NewAttribute(types.AttrKeyOperatorAddress, row.OperatorAddress),
		sdk.NewAttribute(types.AttrKeyDID, row.Did),
		sdk.NewAttribute(types.AttrKeyConnectionString, row.ConnectionString),
	))
}

func emitRegistrationFailed(ctx sdk.Context, operatorAddress, attemptedDid, reason string) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIndexerRegistrationFailed,
		sdk.NewAttribute(types.AttrKeyOperatorAddress, operatorAddress),
		sdk.NewAttribute(types.AttrKeyDID, attemptedDid),
		sdk.NewAttribute(types.AttrKeyReason, reason),
	))
}
