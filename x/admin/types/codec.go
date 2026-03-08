package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// No Msg types — admin is configured via genesis params.
	_ = registry
}
