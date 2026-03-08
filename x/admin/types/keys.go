package types

import "cosmossdk.io/collections"

const (
	ModuleName = "admin"
	StoreKey   = ModuleName
)

var KeyPrefixParams = collections.NewPrefix(0)
