package types

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	ModuleName   = "sourcehub"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

var (
	// Module account address
	ModuleAddress = authtypes.NewModuleAddress(ModuleName)
)

const (
	// Prefix for metadata values
	KeyConnectionID     = "controller_connection_id"
	KeyHostConnectionID = "host_connection_id"
	KeyVersion          = "version"
	KeyEncoding         = "encoding"
	KeyTxType           = "tx_type"
)
