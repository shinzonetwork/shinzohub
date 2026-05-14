package types

const (
	ModuleName = "indexer"
	StoreKey   = ModuleName

	IndexerByValidatorPrefix = "indexer/"
	AddrIndexPrefix          = "addr_idx/"
	PendingClaimPrefix       = "pending_claim/"
	IndexerCountKey          = "indexer_count"
)

const (
	EventTypeIndexerAsserted           = "indexer.indexer_asserted"
	EventTypeIndexerSuperseded         = "indexer.indexer_superseded"
	EventTypeIndexerPayoutUpdated      = "indexer.indexer_payout_updated"
	EventTypeIndexerRevoked            = "indexer.indexer_revoked"
	EventTypeIndexerPending            = "indexer.indexer_pending"
	EventTypeIndexerRegistered         = "indexer.indexer_registered"
	EventTypeIndexerRegistrationFailed = "indexer.indexer_registration_failed"

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
	AttrKeyReason           = "reason"
)
