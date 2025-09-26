package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

type ICAControllerKeeper interface {
	RegisterInterchainAccount(
		ctx sdk.Context,
		connectionID string,
		owner string,
		version string,
		ordering channeltypes.Order,
	) error

	SendTx(
		ctx sdk.Context,
		connectionID string,
		portID string,
		icaPacketData icatypes.InterchainAccountPacketData,
		timeoutTimestamp uint64,
	) (uint64, error)

	GetInterchainAccountAddress(
		ctx sdk.Context,
		connectionID string,
		owner string,
	) (string, bool)
}
