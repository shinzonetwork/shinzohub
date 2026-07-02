package types

const (
	ModuleName = "indexer"
	StoreKey   = ModuleName

	IndexerByValidatorPrefix = "indexer/"
	AddrIndexPrefix          = "addr_idx/"
	DIDIndexPrefix           = "did_idx/"
	PendingClaimPrefix       = "pending_claim/"
	IndexerCountKey          = "indexer_count"

	// GroupIndexerName is the ACP relationship group indexers are registered
	// under on sourcehub. It is written when firing the SetRelationship ICA and
	// read back when the ack lands, so both sides must use this constant to stay
	// in sync.
	GroupIndexerName = "indexer"

	// MaxValidatorPubkeyLen bounds the validator pubkey size. The pubkey becomes
	// part of a store key, so an unbounded value would bloat state. The cap is
	// chain-agnostic and generous on purpose — source chains use different key
	// formats (secp256k1 uncompressed = 65B, ed25519 = 32B, BLS = 96B) — so this
	// is an anti-bloat guard, not a format check.
	MaxValidatorPubkeyLen = 128

	// MaxChainSpecificLen bounds the opaque per-chain audit bytes carried on an
	// assertion.
	MaxChainSpecificLen = 4096
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
