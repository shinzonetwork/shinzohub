package types

const (
	ModuleName = "host"
	StoreKey   = ModuleName

	// HostPrefix is the KVStore prefix for host records.
	// Key format: host/<bech32_address> → Host proto bytes
	HostPrefix = "host/"

	// HostCountKey stores the total number of registered hosts as uint64.
	HostCountKey = "host_count"

	// AddrDIDPrefix stores the addr→DID mapping for backward compatibility.
	// Key format: addr_did/<address_bytes> → DID bytes
	AddrDIDPrefix = "addr_did/"

	// DIDAddrPrefix stores the DID→addr mapping for backward compatibility.
	// Key format: did_addr/<did_bytes> → address bytes
	DIDAddrPrefix = "did_addr/"
)
