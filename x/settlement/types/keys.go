package types

const (
	ModuleName = "settlement"
	StoreKey   = ModuleName

	BalancePrefix = "balance/"

	// PendingDebitPrefix and PendingCreditPrefix hold queued submissions
	// waiting for the epoch boundary. Each is keyed by
	//   <prefix>/<epoch_be_uint64>/<seq_be_uint64> → PendingSettleEntry bytes
	// Debits are drained first per block (gateway access control depends on
	// them landing fast); credits only drain after their epoch's debits are
	// gone.
	PendingDebitPrefix  = "pending_debit/"
	PendingCreditPrefix = "pending_credit/"

	// PendingDebitCounterKey / PendingCreditCounterKey are independent
	// monotonic uint64s; one per queue.
	PendingDebitCounterKey  = "pending_debit_counter"
	PendingCreditCounterKey = "pending_credit_counter"

	// PendingDebitTotalPrefix indexes the running sum of debits queued for
	// each address across all unsettled epochs. Maintained on enqueue (++)
	// and on drain (--). Lets the EffectiveBalance query return
	// querybalance - pending_debit in O(1) without iterating the queue.
	// Key format: pending_debit_total/<bech32_address> → math.Int string.
	PendingDebitTotalPrefix = "pending_debit_total/"

	// LastSettledEpochKey records the most recent epoch whose pending queue
	// has been drained by BeginBlocker. Used by the boundary processor to
	// know whether work is owed.
	LastSettledEpochKey = "last_settled_epoch"

	SettlementDenom = "ushinzo"

	// EpochBlocks is the duration of a settlement epoch in blocks.
	// epoch = floor(block_height / EpochBlocks). Block-height-derived so
	// fresh genesis starts at epoch 0 and the boundary processor never has
	// to catch up "pre-history" epochs.
	EpochBlocks int64 = 180

	// MaxDebitsPerBlock caps the debit chunk drained per block. Gateway
	// access control reads EffectiveBalance, which already subtracts the
	// per-address pending-debit total in O(1) — so on-chain drain doesn't
	// need to be fast for correctness. Drain is pure state housekeeping;
	// keep it small for predictable per-block cost.
	MaxDebitsPerBlock = 50

	// MaxCreditsPerBlock caps the credit chunk drained per block. Credits
	// only matter for claim; no urgency, smallest cap.
	MaxCreditsPerBlock = 50
)

const (
	EventTypeCredited        = "settlement.credited"
	EventTypeDebited         = "settlement.debited"
	EventTypeClaimed         = "settlement.claimed"
	EventTypeSettleQueued    = "settlement.queued"
	EventTypeSettleChunk     = "settlement.chunk_applied"
	EventTypeSettled         = "settlement.settled"
	EventTypeEpochNoActivity = "settlement.epoch_no_activity"

	AttrKeyAddress         = "address"
	AttrKeyAmount          = "amount"
	AttrKeyEpoch           = "epoch"
	AttrKeySubmitter       = "submitter"
	AttrKeySeq             = "seq"
	AttrKeyEntryCount      = "entry_count"
	AttrKeyRemainingCount  = "remaining_count"
	AttrKeyPaymentsApplied = "payments_applied"
	AttrKeyDebitsApplied   = "debits_applied"
	AttrKeyTotalCredited   = "total_credited"
	AttrKeyTotalDebited    = "total_debited"
	AttrKeyPoolsUpdated    = "pools_updated"
)
