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
)
