package types

const (
	ModuleName = "host"
	StoreKey   = ModuleName

	// HostPrefix is the KVStore prefix for host records.
	// Key format: host/<bech32_address> → Host proto bytes
	HostPrefix = "host/"

	// HostCountKey stores the total number of registered hosts as uint64.
	HostCountKey = "host_count"

	// AddrDIDPrefix stores the addr→DID mapping.
	// Key format: addr_did/<bech32_address> → DID string bytes
	AddrDIDPrefix = "addr_did/"

	// DIDAddrPrefix stores the DID→addr mapping.
	// Key format: did_addr/<did> → bech32 address string bytes
	DIDAddrPrefix = "did_addr/"

	PendingHostPrefix    = "pending_host/"
	PendingAddrDIDPrefix = "pending_addr_did/"
	PendingDIDAddrPrefix = "pending_did_addr/"
)

const (
	EventTypeHostPending              = "host.host_pending"
	EventTypeHostRegistered           = "host.host_registered"
	EventTypeHostRegistrationFailed   = "host.host_registration_failed"
	EventTypeHostRegistrationTimedOut = "host.host_registration_timed_out"

	AttrKeyAddress = "address"
	AttrKeyDID     = "did"
	AttrKeyError   = "error"
)
