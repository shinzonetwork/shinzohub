package types

const (
	ModuleName = "querybalance"
	StoreKey   = ModuleName

	// balance/<address> → QueryBalance
	BalancePrefix = "balance/"

	// QueryBalanceDenom is the only denomination the module accepts for
	// funding. Users top up their per-query credit with SHINUSD (the same
	// base unit settlement mints into wallets via Claim). The chain's bond
	// denom is NOT accepted here.
	QueryBalanceDenom = "ushinusd"
)

const (
	EventTypeFunded  = "querybalance.funded"
	EventTypeDebited = "querybalance.debited"

	AttrKeyAddress   = "address"
	AttrKeyFunder    = "funder"
	AttrKeyRecipient = "recipient"
	AttrKeyAmount    = "amount"
)
