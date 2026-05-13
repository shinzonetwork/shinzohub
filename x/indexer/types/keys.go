package types

const (
	ModuleName = "indexer"
	StoreKey   = ModuleName

	// IndexerByValidatorPrefix is the primary store prefix for indexer rows.
	// Key format: indexer/<source_chain_id>/<validator_pubkey> → Indexer proto bytes
	IndexerByValidatorPrefix = "indexer/"

	// AddrIndexPrefix is the inverse index from operator bech32 address to
	// the validator row key.
	// Key format: addr_idx/<operator_address> → "<source_chain_id>/<hex(validator_pubkey)>"
	AddrIndexPrefix = "addr_idx/"

	// IndexerCountKey stores the total number of indexer rows.
	IndexerCountKey = "indexer_count"
)

const (
	EventTypeIndexerAsserted      = "indexer.indexer_asserted"
	EventTypeIndexerSuperseded    = "indexer.indexer_superseded"
	EventTypeIndexerPayoutUpdated = "indexer.indexer_payout_updated"
	EventTypeIndexerRevoked       = "indexer.indexer_revoked"
	EventTypeIndexerRegistered    = "indexer.indexer_registered"

	AttrKeySourceChain      = "source_chain"
	AttrKeySourceChainID    = "source_chain_id"
	AttrKeyValidatorPubkey  = "validator_pubkey"
	AttrKeyOperatorAddress  = "operator_address"
	AttrKeyOldOperator      = "old_operator_address"
	AttrKeyNewOperator      = "new_operator_address"
	AttrKeyPayoutAddress    = "payout_address"
	AttrKeyNonce            = "nonce"
	AttrKeyOldNonce         = "old_nonce"
	AttrKeyNewNonce         = "new_nonce"
	AttrKeyDID              = "did"
	AttrKeyConnectionString = "connection_string"
)
