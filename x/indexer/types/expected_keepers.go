package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type AdminKeeper interface {
	IsAdmin(ctx sdk.Context, address string) bool
}

type SourcehubKeeper interface {
	SendICASetRelationship(
		ctx sdk.Context,
		did string,
		group string,
		requestor string,
	) (sequence uint64, portID, channelID string, err error)

	SendICASetAndDeleteRelationship(
		ctx sdk.Context,
		newDid string,
		prevDid string,
		group string,
		requestor string,
	) (sequence uint64, portID, channelID string, err error)
}
