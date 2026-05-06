package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// HostKeeper defines the interface for the host module's keeper
// that the view module needs to validate host registration.
type HostKeeper interface {
	IsRegisteredHost(ctx sdk.Context, address []byte) bool
}

// SourcehubKeeper defines the interface for the sourcehub module's keeper
// that the view module needs for ACP object registration.
type SourcehubKeeper interface {
	RegisterObject(ctx sdk.Context, id string, requestor string) (sequence uint64, portID, channelID string, err error)
}
