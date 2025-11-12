package keeper

import (
	"strconv"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func RoleToString(role uint8) string {
	switch role {
	case types.RoleIndexer:
		return "indexer"
	case types.RoleHost:
		return "host"
	default:
		return "unknown"
	}
}

func addrRoleKey(address []byte, role uint8) []byte {
	// "addr_role:" + <address-bytes> + ":" + <role-int>
	return append(
		append(
			append([]byte(types.AddrRolePrefix), address...),
			':',
		),
		[]byte(strconv.FormatUint(uint64(role), 10))...,
	)
}

func didRoleKey(did []byte, role uint8) []byte {
	// "did_role:" + <did-bytes> + ":" + <role-int>
	return append(
		append(
			append([]byte(types.DIDRolePrefix), did...),
			':',
		),
		[]byte(strconv.FormatUint(uint64(role), 10))...,
	)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
