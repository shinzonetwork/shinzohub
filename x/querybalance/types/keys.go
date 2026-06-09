package types

const (
	ModuleName = "querybalance"
	StoreKey   = ModuleName

	// balance/<did> → QueryBalance
	BalancePrefix = "balance/"
)

const (
	EventTypeFunded    = "querybalance.funded"
	EventTypeDebited   = "querybalance.debited"
	EventTypeClaimed   = "querybalance.claimed"
	EventTypeWithdrawn = "querybalance.withdrawn"

	AttrKeyDID     = "did"
	AttrKeyFunder  = "funder"
	AttrKeyAmount  = "amount"
	AttrKeyOwner   = "owner"
	AttrKeyClaimer = "claimer"
)
