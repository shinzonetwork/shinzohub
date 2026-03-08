package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// No Msg types — view registration is done via precompile.
}
