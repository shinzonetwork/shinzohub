package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// AdminKeeper defines the interface for the admin module's keeper.
type AdminKeeper interface {
	IsAdmin(ctx sdk.Context, address string) bool
}

// SourcehubKeeper defines the interface for the sourcehub module's keeper
// that the indexer module needs for ICA/ACP operations.
type SourcehubKeeper interface {
	SendICASetRelationship(ctx sdk.Context, did string, group string, requestor string) (sequence uint64, portID, channelID string, err error)
}
