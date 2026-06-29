package keeper

import (
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

type Keeper struct {
	cdc                codec.BinaryCodec
	storeService       storetypes.KVStoreService
	authority          string
	bankKeeper         types.BankKeeper
	hostKeeper         types.HostKeeper
	indexerKeeper      types.IndexerKeeper
	queryBalanceKeeper types.QueryBalanceKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	bankKeeper types.BankKeeper,
	hostKeeper types.HostKeeper,
	indexerKeeper types.IndexerKeeper,
	queryBalanceKeeper types.QueryBalanceKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:                cdc,
		storeService:       storeService,
		bankKeeper:         bankKeeper,
		hostKeeper:         hostKeeper,
		indexerKeeper:      indexerKeeper,
		queryBalanceKeeper: queryBalanceKeeper,
		authority:          authority,
	}
}

// Accessors used by the msg server.
func (k Keeper) HostKeeper() types.HostKeeper                 { return k.hostKeeper }
func (k Keeper) IndexerKeeper() types.IndexerKeeper           { return k.indexerKeeper }
func (k Keeper) QueryBalanceKeeper() types.QueryBalanceKeeper { return k.queryBalanceKeeper }

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) Credit(ctx sdk.Context, recipient sdk.AccAddress, amount math.Int) error {
	if recipient.Empty() {
		return fmt.Errorf("recipient is required")
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}

	sb := k.getEntry(ctx, recipient)
	prev := parseAmount(sb.Amount)
	sb.Amount = prev.Add(amount).String()
	k.setEntry(ctx, sb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCredited,
		sdk.NewAttribute(types.AttrKeyAddress, recipient.String()),
		sdk.NewAttribute(types.AttrKeyAmount, amount.String()),
	))

	return nil
}

func (k Keeper) Debit(ctx sdk.Context, holder sdk.AccAddress, amount math.Int) error {
	if holder.Empty() {
		return fmt.Errorf("holder is required")
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}

	sb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return fmt.Errorf("no settlement balance for address %s", holder.String())
	}

	balance := parseAmount(sb.Amount)
	if balance.LT(amount) {
		return fmt.Errorf("insufficient settlement balance for address %s: have %s, want %s",
			holder.String(), balance.String(), amount.String())
	}

	sb.Amount = balance.Sub(amount).String()
	k.setEntry(ctx, sb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeDebited,
		sdk.NewAttribute(types.AttrKeyAddress, holder.String()),
		sdk.NewAttribute(types.AttrKeyAmount, amount.String()),
	))

	return nil
}

func (k Keeper) Claim(ctx sdk.Context, claimer sdk.AccAddress, amount math.Int) error {
	if claimer.Empty() {
		return fmt.Errorf("claimer is required")
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}

	sb, found := k.getEntryIfExists(ctx, claimer)
	if !found {
		return fmt.Errorf("no settlement balance for address %s", claimer.String())
	}

	balance := parseAmount(sb.Amount)
	if balance.LT(amount) {
		return fmt.Errorf("insufficient settlement balance for address %s: have %s, want %s",
			claimer.String(), balance.String(), amount.String())
	}

	coins := sdk.NewCoins(sdk.NewCoin(types.SettlementDenom, amount))

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return fmt.Errorf("mint %s: %w", types.SettlementDenom, err)
	}
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, claimer, coins); err != nil {
		return fmt.Errorf("transfer %s to claimer: %w", types.SettlementDenom, err)
	}

	sb.Amount = balance.Sub(amount).String()
	k.setEntry(ctx, sb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeClaimed,
		sdk.NewAttribute(types.AttrKeyAddress, claimer.String()),
		sdk.NewAttribute(types.AttrKeyAmount, amount.String()),
	))

	return nil
}

func (k Keeper) GetBalance(ctx sdk.Context, holder sdk.AccAddress) math.Int {
	sb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return math.ZeroInt()
	}
	return parseAmount(sb.Amount)
}

func (k Keeper) GetEntry(ctx sdk.Context, holder sdk.AccAddress) (types.SettlementBalance, bool) {
	return k.getEntryIfExists(ctx, holder)
}

func (k Keeper) GetAllBalances(
	ctx sdk.Context,
	pageReq *query.PageRequest,
) ([]types.SettlementBalance, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	balanceStore := prefix.NewStore(store, []byte(types.BalancePrefix))

	var balances []types.SettlementBalance
	pageRes, err := query.Paginate(balanceStore, pageReq, func(_, value []byte) error {
		var sb types.SettlementBalance
		if err := k.cdc.Unmarshal(value, &sb); err != nil {
			return err
		}
		balances = append(balances, sb)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return balances, pageRes, nil
}

func (k Keeper) getEntry(ctx sdk.Context, holder sdk.AccAddress) types.SettlementBalance {
	sb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return types.SettlementBalance{Address: holder.String(), Amount: "0"}
	}
	return sb
}

func (k Keeper) getEntryIfExists(ctx sdk.Context, holder sdk.AccAddress) (types.SettlementBalance, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(balanceKey(holder))
	if len(bz) == 0 {
		return types.SettlementBalance{}, false
	}
	var sb types.SettlementBalance
	if err := k.cdc.Unmarshal(bz, &sb); err != nil {
		return types.SettlementBalance{}, false
	}
	return sb, true
}

func (k Keeper) setEntry(ctx sdk.Context, sb types.SettlementBalance) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&sb)
	if err != nil {
		panic(err)
	}
	store.Set([]byte(types.BalancePrefix+sb.Address), bz)
}

func balanceKey(holder sdk.AccAddress) []byte {
	return []byte(types.BalancePrefix + holder.String())
}

func parseAmount(s string) math.Int {
	if s == "" {
		return math.ZeroInt()
	}
	v, ok := math.NewIntFromString(s)
	if !ok {
		return math.ZeroInt()
	}
	return v
}

// GetCurrentEpoch returns the epoch number derived from the block timestamp.
// epoch = floor(block_time_unix / EpochSeconds). Returns 0 if BlockTime is unset.
func (k Keeper) GetCurrentEpoch(ctx sdk.Context) uint64 {
	t := ctx.BlockTime().Unix()
	if t <= 0 {
		return 0
	}
	return uint64(t / types.EpochSeconds)
}

func (k Keeper) GetLastSettledEpoch(ctx sdk.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.LastSettledEpochKey))
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetLastSettledEpoch(ctx sdk.Context, epoch uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	var bz [8]byte
	binary.BigEndian.PutUint64(bz[:], epoch)
	store.Set([]byte(types.LastSettledEpochKey), bz[:])
}

// EnqueuePendingDebit appends a debit-only entry to the epoch's debit queue
// AND updates the per-address pending-debit-total index. Returns the entry's
// sequence number.
func (k Keeper) EnqueuePendingDebit(ctx sdk.Context, epoch uint64, entry types.PendingSettleEntry) (uint64, error) {
	for _, d := range entry.Debits {
		amt, ok := math.NewIntFromString(d.Amount)
		if !ok || !amt.IsPositive() {
			continue
		}
		k.addPendingDebitTotal(ctx, d.Address, amt)
	}
	return k.enqueuePendingTo(ctx, types.PendingDebitPrefix, types.PendingDebitCounterKey, epoch, entry)
}

// EnqueuePendingCredit appends a credit-only entry to the epoch's credit queue.
func (k Keeper) EnqueuePendingCredit(ctx sdk.Context, epoch uint64, entry types.PendingSettleEntry) (uint64, error) {
	return k.enqueuePendingTo(ctx, types.PendingCreditPrefix, types.PendingCreditCounterKey, epoch, entry)
}

func (k Keeper) enqueuePendingTo(ctx sdk.Context, queuePrefix, counterKey string, epoch uint64, entry types.PendingSettleEntry) (uint64, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	seq := k.nextPendingSeq(ctx, counterKey)
	bz, err := k.cdc.Marshal(&entry)
	if err != nil {
		return 0, fmt.Errorf("marshal pending entry: %w", err)
	}
	store.Set(pendingEntryKey(queuePrefix, epoch, seq), bz)
	return seq, nil
}

// DrainPendingDebits drains up to `limit` debit entries from the epoch's
// debit queue. See DrainPending for behavior.
func (k Keeper) DrainPendingDebits(ctx sdk.Context, epoch uint64, limit int, fn func(types.PendingSettleEntry)) int {
	return k.drainPendingFrom(ctx, types.PendingDebitPrefix, epoch, limit, fn)
}

// DrainPendingCredits drains up to `limit` credit entries from the epoch's
// credit queue.
func (k Keeper) DrainPendingCredits(ctx sdk.Context, epoch uint64, limit int, fn func(types.PendingSettleEntry)) int {
	return k.drainPendingFrom(ctx, types.PendingCreditPrefix, epoch, limit, fn)
}

func (k Keeper) drainPendingFrom(ctx sdk.Context, queuePrefix string, epoch uint64, limit int, fn func(types.PendingSettleEntry)) int {
	rootStore := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	pendingStore := prefix.NewStore(rootStore, pendingEpochPrefix(queuePrefix, epoch))

	type keyed struct {
		key   []byte
		entry types.PendingSettleEntry
	}
	var batch []keyed

	it := pendingStore.Iterator(nil, nil)
	for ; it.Valid(); it.Next() {
		if limit > 0 && len(batch) >= limit {
			break
		}
		var entry types.PendingSettleEntry
		if err := k.cdc.Unmarshal(it.Value(), &entry); err != nil {
			it.Close()
			panic(fmt.Errorf("unmarshal pending settle entry: %w", err))
		}
		keyCopy := make([]byte, len(it.Key()))
		copy(keyCopy, it.Key())
		batch = append(batch, keyed{key: keyCopy, entry: entry})
	}
	it.Close()

	for _, kv := range batch {
		fn(kv.entry)
		pendingStore.Delete(kv.key)
	}
	return len(batch)
}

// PendingDebitCount / PendingCreditCount / PendingCount expose queue depth.
// PendingCount is the union — used by BeginBlocker to know if an epoch is
// fully settled.
func (k Keeper) PendingDebitCount(ctx sdk.Context, epoch uint64) int {
	return k.countQueue(ctx, types.PendingDebitPrefix, epoch)
}

func (k Keeper) PendingCreditCount(ctx sdk.Context, epoch uint64) int {
	return k.countQueue(ctx, types.PendingCreditPrefix, epoch)
}

func (k Keeper) PendingCount(ctx sdk.Context, epoch uint64) int {
	return k.PendingDebitCount(ctx, epoch) + k.PendingCreditCount(ctx, epoch)
}

func (k Keeper) countQueue(ctx sdk.Context, queuePrefix string, epoch uint64) int {
	rootStore := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	pendingStore := prefix.NewStore(rootStore, pendingEpochPrefix(queuePrefix, epoch))

	it := pendingStore.Iterator(nil, nil)
	defer it.Close()
	n := 0
	for ; it.Valid(); it.Next() {
		n++
	}
	return n
}

func (k Keeper) nextPendingSeq(ctx sdk.Context, counterKey string) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(counterKey))
	var next uint64
	if len(bz) == 8 {
		next = binary.BigEndian.Uint64(bz)
	}
	var out [8]byte
	binary.BigEndian.PutUint64(out[:], next+1)
	store.Set([]byte(counterKey), out[:])
	return next
}

// ─── pending debit total index ────────────────────────────────────────────
//
// pending_debit_total/<bech32_address> → math.Int as a string. Maintained on
// every EnqueuePendingDebit (+=) and every ProcessPendingDebitChunk (-=) so
// the EffectiveBalance query can compute querybalance - pending_debit in O(1).

// GetPendingDebitTotal returns the sum of all debits currently queued (across
// any unsettled epochs) against the given address. Returns 0 if none.
func (k Keeper) GetPendingDebitTotal(ctx sdk.Context, addr sdk.AccAddress) math.Int {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(pendingDebitTotalKey(addr.String()))
	if len(bz) == 0 {
		return math.ZeroInt()
	}
	v, ok := math.NewIntFromString(string(bz))
	if !ok {
		return math.ZeroInt()
	}
	return v
}

// GetEffectiveBalance returns max(0, querybalance - pending_debit_total) for
// the address — what the gateway should treat as actually spendable.
func (k Keeper) GetEffectiveBalance(ctx sdk.Context, addr sdk.AccAddress) math.Int {
	actual := k.queryBalanceKeeper.GetBalance(ctx, addr)
	pending := k.GetPendingDebitTotal(ctx, addr)
	if actual.LTE(pending) {
		return math.ZeroInt()
	}
	return actual.Sub(pending)
}

func (k Keeper) addPendingDebitTotal(ctx sdk.Context, bech32Addr string, delta math.Int) {
	if !delta.IsPositive() {
		return
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := pendingDebitTotalKey(bech32Addr)
	cur := math.ZeroInt()
	if bz := store.Get(key); len(bz) > 0 {
		if v, ok := math.NewIntFromString(string(bz)); ok {
			cur = v
		}
	}
	store.Set(key, []byte(cur.Add(delta).String()))
}

func (k Keeper) subPendingDebitTotal(ctx sdk.Context, bech32Addr string, delta math.Int) {
	if !delta.IsPositive() {
		return
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := pendingDebitTotalKey(bech32Addr)
	cur := math.ZeroInt()
	if bz := store.Get(key); len(bz) > 0 {
		if v, ok := math.NewIntFromString(string(bz)); ok {
			cur = v
		}
	}
	next := cur.Sub(delta)
	if !next.IsPositive() {
		// Either zero or negative (defensive); delete the row to keep state lean.
		store.Delete(key)
		return
	}
	store.Set(key, []byte(next.String()))
}

func pendingDebitTotalKey(bech32Addr string) []byte {
	return []byte(types.PendingDebitTotalPrefix + bech32Addr)
}

func pendingEpochPrefix(queuePrefix string, epoch uint64) []byte {
	p := []byte(queuePrefix)
	var epochBuf [8]byte
	binary.BigEndian.PutUint64(epochBuf[:], epoch)
	out := make([]byte, 0, len(p)+8+1)
	out = append(out, p...)
	out = append(out, epochBuf[:]...)
	out = append(out, '/')
	return out
}

func pendingEntryKey(queuePrefix string, epoch, seq uint64) []byte {
	p := pendingEpochPrefix(queuePrefix, epoch)
	var seqBuf [8]byte
	binary.BigEndian.PutUint64(seqBuf[:], seq)
	out := make([]byte, 0, len(p)+8)
	out = append(out, p...)
	out = append(out, seqBuf[:]...)
	return out
}

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	for _, sb := range gs.Balances {
		k.setEntry(ctx, sb)
	}
	if gs.LastSettledEpoch > 0 {
		k.SetLastSettledEpoch(ctx, gs.LastSettledEpoch)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	balanceStore := prefix.NewStore(store, []byte(types.BalancePrefix))

	var balances []types.SettlementBalance
	it := balanceStore.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var sb types.SettlementBalance
		if err := k.cdc.Unmarshal(it.Value(), &sb); err != nil {
			panic(err)
		}
		balances = append(balances, sb)
	}

	return &types.GenesisState{
		Balances:         balances,
		LastSettledEpoch: k.GetLastSettledEpoch(ctx),
	}
}
