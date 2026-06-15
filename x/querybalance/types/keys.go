package types

const (
	ModuleName = "querybalance"
	StoreKey   = ModuleName

	// balance/<did> → QueryBalance
	BalancePrefix = "balance/"
)

const (
	EventTypeFunded  = "querybalance.funded"
	EventTypeDebited = "querybalance.debited"

	AttrKeyDID    = "did"
	AttrKeyFunder = "funder"
	AttrKeyAmount = "amount"
)
