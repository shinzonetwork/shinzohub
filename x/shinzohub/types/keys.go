package types

const (
	// ModuleName defines the module name
	ModuleName = "shinzohub"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_shinzohub"
)

var (
	ParamsKey = []byte("p_shinzohub")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
