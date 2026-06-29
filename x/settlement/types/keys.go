package types

const (
	ModuleName = "settlement"
	StoreKey   = ModuleName

	BalancePrefix = "balance/"

	// LastSettledEpochKey stores the last epoch number that has been settled.
	// Used to reject replays and out-of-order settlement messages.
	LastSettledEpochKey = "last_settled_epoch"

	SettlementDenom = "ushinusd"

	// EpochSeconds is the duration of a settlement epoch in seconds.
	// epoch = floor(block_time_unix / EpochSeconds)
	EpochSeconds int64 = 180
)

const (
	EventTypeCredited = "settlement.credited"
	EventTypeDebited  = "settlement.debited"
	EventTypeClaimed  = "settlement.claimed"

	AttrKeyAddress = "address"
	AttrKeyAmount  = "amount"
)
