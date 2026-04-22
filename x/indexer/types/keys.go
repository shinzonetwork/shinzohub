package types

const (
	ModuleName = "indexer"
	StoreKey   = ModuleName

	// IndexerPrefix is the KVStore prefix for indexer records.
	// Key format: indexer/<bech32_address> → Indexer proto bytes
	IndexerPrefix = "indexer/"

	// IndexerCountKey stores the total number of registered indexers.
	IndexerCountKey = "indexer_count"

	// AssertionPrefix is the KVStore prefix for indexer assertions.
	// Key format: assertion/<delegate>:<sourceChain>:<sourceChainId> → IndexerAssertion proto bytes
	AssertionPrefix = "assertion/"

	// AddrDIDPrefix stores the addr→DID mapping.
	AddrDIDPrefix = "addr_did/"

	// DIDAddrPrefix stores the DID→addr mapping.
	DIDAddrPrefix = "did_addr/"

	PendingIndexerPrefix = "pending_indexer/"
	PendingAddrDIDPrefix = "pending_addr_did/"
	PendingDIDAddrPrefix = "pending_did_addr/"
)

const (
	EventTypeIndexerPending              = "indexer.indexer_pending"
	EventTypeIndexerRegistered           = "indexer.indexer_registered"
	EventTypeIndexerRegistrationFailed   = "indexer.indexer_registration_failed"
	EventTypeIndexerRegistrationTimedOut = "indexer.indexer_registration_timed_out"

	AttrKeyAddress = "address"
	AttrKeyDID     = "did"
	AttrKeyError   = "error"
)
