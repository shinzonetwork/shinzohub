package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// No Msg types — host registration is done via precompile.
	// Only query types are used, and they don't need interface registration.
}
