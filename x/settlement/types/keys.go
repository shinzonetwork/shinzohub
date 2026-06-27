package types

const (
	ModuleName = "settlement"
	StoreKey   = ModuleName

	BalancePrefix = "balance/"

	SettlementDenom = "ushinusd"
)

const (
	EventTypeCredited = "settlement.credited"
	EventTypeDebited  = "settlement.debited"
	EventTypeClaimed  = "settlement.claimed"

	AttrKeyAddress = "address"
	AttrKeyAmount  = "amount"
)
