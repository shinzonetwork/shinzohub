package types

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "tier"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for the module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the module
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_tier"

	// LockupKeyPrefix is the prefix to retrieve all Lockup
	LockupKeyPrefix = "Lockup/"

	// UnlockingLockupKeyPrefix is the prefix to retrieve all UnlockingLockup
	UnlockingLockupKeyPrefix = "UnlockingLockup/"
)

var (
	ParamsKey = []byte("p_tier")
)

func KeyPrefix(unlocking bool) []byte {
	if unlocking {
		return []byte(UnlockingLockupKeyPrefix)
	}
	return []byte(LockupKeyPrefix)
}

// LockupKey returns the store key to retrieve a Lockup from the index fields
func LockupKey(delAddr sdk.AccAddress, valAddr sdk.ValAddress) []byte {

	// Calculate the size of the buffer in advance
	size := len(delAddr.Bytes()) + 1 + len(valAddr.Bytes()) + 1
	buf := make([]byte, 0, size)

	// Append bytes to the buffer
	buf = append(buf, delAddr.Bytes()...)
	buf = append(buf, '/')
	buf = append(buf, valAddr.Bytes()...)
	buf = append(buf, '/')

	return buf
}

func LockupKeyToAddresses(key []byte) (sdk.AccAddress, sdk.ValAddress) {

	// Find the positions of the delimiters
	parts := bytes.Split(key, []byte{'/'})
	if len(parts) != 3 {
		panic("expected format in delAddr/valAddr/")
	}

	// Reconstruct the addresses
	delAddr := sdk.AccAddress(parts[0])
	valAddr := sdk.ValAddress(parts[1])

	return delAddr, valAddr
}
