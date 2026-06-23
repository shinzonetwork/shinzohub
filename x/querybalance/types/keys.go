package types

const (
	ModuleName = "querybalance"
	StoreKey   = ModuleName

	// balance/<address> → QueryBalance
	BalancePrefix = "balance/"
)

const (
	EventTypeFunded  = "querybalance.funded"
	EventTypeDebited = "querybalance.debited"

	AttrKeyAddress   = "address"
	AttrKeyFunder    = "funder"
	AttrKeyRecipient = "recipient"
	AttrKeyAmount    = "amount"
)
