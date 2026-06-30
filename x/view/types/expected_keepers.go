package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// Subset of sourcehub keeper used to fire ACP register-object via ICA.
type SourcehubKeeper interface {
	RegisterObject(ctx sdk.Context, id, requestor string) (seq uint64, portID, channelID string, err error)
}
