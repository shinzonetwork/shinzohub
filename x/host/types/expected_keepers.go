package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// SourcehubKeeper defines the interface for the sourcehub module's keeper
// that the host module needs for ICA/ACP operations.
type SourcehubKeeper interface {
	SendICASetRelationship(ctx sdk.Context, did string, group string) error
}
