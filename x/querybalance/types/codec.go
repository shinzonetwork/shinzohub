package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// No Msg types — funding happens via precompile, debit is keeper-internal.
}
