package keeper

import (
	"bytes"
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

type Keeper struct {
	cdc             codec.BinaryCodec
	storeService    storetypes.KVStoreService
	adminKeeper     types.AdminKeeper
	sourcehubKeeper types.SourcehubKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	adminKeeper types.AdminKeeper,
	sourcehubKeeper types.SourcehubKeeper,
) Keeper {
	return Keeper{
		cdc:             cdc,
		storeService:    storeService,
		adminKeeper:     adminKeeper,
		sourcehubKeeper: sourcehubKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) AdminKeeper() types.AdminKeeper            { return k.adminKeeper }
func (k Keeper) SourcehubKeeper() types.SourcehubKeeper    { return k.sourcehubKeeper }

// ─── Key encoding ─────────────────────────────────────────────────────

// indexerRowKey encodes a row's primary key.
// Format: IndexerByValidatorPrefix | 8-byte big-endian source_chain_id | validator_pubkey
func indexerRowKey(sourceChainID uint64, validatorPubkey []byte) []byte {
	out := make([]byte, 0, len(types.IndexerByValidatorPrefix)+8+len(validatorPubkey))
	out = append(out, []byte(types.IndexerByValidatorPrefix)...)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], sourceChainID)
	out = append(out, buf[:]...)
	out = append(out, validatorPubkey...)
	return out
}

// addrIndexKey encodes an operator-address inverse index key.
func addrIndexKey(operatorAddress string) []byte {
	return []byte(types.AddrIndexPrefix + operatorAddress)
}

// encodeAddrIndexValue encodes (source_chain_id, validator_pubkey) for storage as
// the value of an addr_idx entry.
func encodeAddrIndexValue(sourceChainID uint64, validatorPubkey []byte) []byte {
	out := make([]byte, 8+len(validatorPubkey))
	binary.BigEndian.PutUint64(out[:8], sourceChainID)
	copy(out[8:], validatorPubkey)
	return out
}

func decodeAddrIndexValue(v []byte) (uint64, []byte, error) {
	if len(v) < 8 {
		return 0, nil, fmt.Errorf("addr_idx value too short: %d", len(v))
	}
	return binary.BigEndian.Uint64(v[:8]), v[8:], nil
}

// ─── Reads ────────────────────────────────────────────────────────────

func (k Keeper) GetIndexerByValidator(ctx sdk.Context, sourceChainID uint64, validatorPubkey []byte) (types.Indexer, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(indexerRowKey(sourceChainID, validatorPubkey))
	if len(bz) == 0 {
		return types.Indexer{}, false, nil
	}
	var ix types.Indexer
	if err := k.cdc.Unmarshal(bz, &ix); err != nil {
		return types.Indexer{}, false, err
	}
	return ix, true, nil
}

func (k Keeper) GetIndexerByAddress(ctx sdk.Context, operatorAddress string) (types.Indexer, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(addrIndexKey(operatorAddress))
	if len(v) == 0 {
		return types.Indexer{}, false, nil
	}
	chainID, pub, err := decodeAddrIndexValue(v)
	if err != nil {
		return types.Indexer{}, false, err
	}
	return k.GetIndexerByValidator(ctx, chainID, pub)
}

func (k Keeper) IterateIndexers(ctx sdk.Context, pageReq *query.PageRequest) ([]types.Indexer, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	indexerStore := prefix.NewStore(store, []byte(types.IndexerByValidatorPrefix))

	var indexers []types.Indexer
	pageRes, err := query.Paginate(indexerStore, pageReq, func(_, value []byte) error {
		var ix types.Indexer
		if err := k.cdc.Unmarshal(value, &ix); err != nil {
			return err
		}
		indexers = append(indexers, ix)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return indexers, pageRes, nil
}

func (k Keeper) GetIndexerCount(ctx sdk.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.IndexerCountKey))
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// ─── Writes ───────────────────────────────────────────────────────────

// UpsertAssertion handles a MsgIndexerAssertion. Creates or updates the row
// keyed by (source_chain_id, validator_pubkey). Enforces nonce monotonicity,
// rejects operator-address collisions across validators, and resets
// operator-side fields when the operator changes.
func (k Keeper) UpsertAssertion(ctx sdk.Context, msg *types.MsgIndexerAssertion) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	rowKey := indexerRowKey(msg.SourceChainId, msg.ValidatorPubkey)
	existingBz := store.Get(rowKey)

	var (
		existing       types.Indexer
		isUpdate       = len(existingBz) > 0
		operatorChanged = true
	)
	if isUpdate {
		if err := k.cdc.Unmarshal(existingBz, &existing); err != nil {
			return fmt.Errorf("decode existing indexer: %w", err)
		}
		if msg.Nonce <= existing.Nonce {
			return fmt.Errorf("nonce %d not strictly greater than existing %d", msg.Nonce, existing.Nonce)
		}
		operatorChanged = existing.OperatorAddress != msg.OperatorAddress
	}

	// Collision check: if addr_idx for this operator points to a different
	// (source_chain_id, validator_pubkey), reject.
	existingAddrVal := store.Get(addrIndexKey(msg.OperatorAddress))
	if len(existingAddrVal) > 0 {
		existingChainID, existingPub, err := decodeAddrIndexValue(existingAddrVal)
		if err != nil {
			return fmt.Errorf("decode addr_idx value: %w", err)
		}
		if existingChainID != msg.SourceChainId || !bytes.Equal(existingPub, msg.ValidatorPubkey) {
			return fmt.Errorf("operator address %s already in use by another validator", msg.OperatorAddress)
		}
	}

	// Build new row.
	row := types.Indexer{
		SourceChain:        msg.SourceChain,
		SourceChainId:      msg.SourceChainId,
		ValidatorPubkey:    msg.ValidatorPubkey,
		AssertionAuthority: msg.AssertionAuthority,
		Nonce:              msg.Nonce,
		ChainSpecific:      msg.ChainSpecific,
		OperatorAddress:    msg.OperatorAddress,
		PayoutAddress:      msg.PayoutAddress,
	}
	if isUpdate && !operatorChanged {
		// Idempotent path: preserve operator-side fields.
		row.Registered = existing.Registered
		row.Did = existing.Did
		row.ConnectionString = existing.ConnectionString
	}
	// If operator changed, operator-side fields are zero-valued — new operator must register.

	// If operator changed, drop the old addr_idx and emit the supersession event.
	if isUpdate && operatorChanged {
		store.Delete(addrIndexKey(existing.OperatorAddress))
		emitSuperseded(ctx, msg.SourceChainId, msg.ValidatorPubkey, existing.OperatorAddress, msg.OperatorAddress, existing.Nonce, msg.Nonce)
	}

	bz, err := k.cdc.Marshal(&row)
	if err != nil {
		return fmt.Errorf("marshal indexer: %w", err)
	}
	store.Set(rowKey, bz)
	store.Set(addrIndexKey(msg.OperatorAddress), encodeAddrIndexValue(msg.SourceChainId, msg.ValidatorPubkey))

	if !isUpdate {
		k.incrementCount(ctx)
	}

	emitAsserted(ctx, &row)
	return nil
}

// SetPayout handles a MsgSetPayout. Updates only payout_address and nonce on
// an existing row; operator-side fields are untouched.
func (k Keeper) SetPayout(ctx sdk.Context, msg *types.MsgSetPayout) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	rowKey := indexerRowKey(msg.SourceChainId, msg.ValidatorPubkey)

	bz := store.Get(rowKey)
	if len(bz) == 0 {
		return fmt.Errorf("indexer not found for chain %d validator", msg.SourceChainId)
	}
	var row types.Indexer
	if err := k.cdc.Unmarshal(bz, &row); err != nil {
		return fmt.Errorf("decode indexer: %w", err)
	}
	if msg.Nonce <= row.Nonce {
		return fmt.Errorf("nonce %d not strictly greater than existing %d", msg.Nonce, row.Nonce)
	}

	row.PayoutAddress = msg.PayoutAddress
	row.Nonce = msg.Nonce

	out, err := k.cdc.Marshal(&row)
	if err != nil {
		return fmt.Errorf("marshal indexer: %w", err)
	}
	store.Set(rowKey, out)

	emitPayoutUpdated(ctx, &row)
	return nil
}

// RevokeIndexer handles a MsgRevokeIndexer. Deletes the row and the inverse
// index entry.
func (k Keeper) RevokeIndexer(ctx sdk.Context, msg *types.MsgRevokeIndexer) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	rowKey := indexerRowKey(msg.SourceChainId, msg.ValidatorPubkey)

	bz := store.Get(rowKey)
	if len(bz) == 0 {
		return fmt.Errorf("indexer not found for chain %d validator", msg.SourceChainId)
	}
	var row types.Indexer
	if err := k.cdc.Unmarshal(bz, &row); err != nil {
		return fmt.Errorf("decode indexer: %w", err)
	}
	if msg.Nonce <= row.Nonce {
		return fmt.Errorf("nonce %d not strictly greater than existing %d", msg.Nonce, row.Nonce)
	}

	store.Delete(rowKey)
	store.Delete(addrIndexKey(row.OperatorAddress))
	k.decrementCount(ctx)

	emitRevoked(ctx, &row, msg.Nonce)
	return nil
}

// CompleteRegistration flips an indexer row to registered. Called by the EVM
// precompile after verifying the operator's possession of the node identity
// key. Returns the updated row plus the previous DID on the row (empty if the
// row was not yet registered). The caller uses the previous DID to decide
// whether a new ICA SetRelationship is needed: if it differs from the new DID,
// the caller should fire it (and optionally clean up the old DID's
// relationship).
func (k Keeper) CompleteRegistration(
	ctx sdk.Context,
	operatorAddress string,
	did string,
	connectionString string,
) (row types.Indexer, prevDid string, err error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	v := store.Get(addrIndexKey(operatorAddress))
	if len(v) == 0 {
		return types.Indexer{}, "", fmt.Errorf("indexer not asserted for address %s", operatorAddress)
	}
	chainID, pub, decErr := decodeAddrIndexValue(v)
	if decErr != nil {
		return types.Indexer{}, "", decErr
	}

	rowKey := indexerRowKey(chainID, pub)
	bz := store.Get(rowKey)
	if len(bz) == 0 {
		return types.Indexer{}, "", fmt.Errorf("addr_idx points at missing row")
	}
	if err := k.cdc.Unmarshal(bz, &row); err != nil {
		return types.Indexer{}, "", err
	}

	prevDid = row.Did
	row.Registered = true
	row.Did = did
	row.ConnectionString = connectionString

	out, mErr := k.cdc.Marshal(&row)
	if mErr != nil {
		return types.Indexer{}, "", fmt.Errorf("marshal indexer: %w", mErr)
	}
	store.Set(rowKey, out)

	emitRegistered(ctx, &row)
	return row, prevDid, nil
}

// ─── Count helpers ────────────────────────────────────────────────────

func (k Keeper) incrementCount(ctx sdk.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	count := k.GetIndexerCount(ctx) + 1
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], count)
	store.Set([]byte(types.IndexerCountKey), buf[:])
}

func (k Keeper) decrementCount(ctx sdk.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	count := k.GetIndexerCount(ctx)
	if count == 0 {
		return
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], count-1)
	store.Set([]byte(types.IndexerCountKey), buf[:])
}

// ─── Genesis ──────────────────────────────────────────────────────────

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	for i := range gs.Indexers {
		row := gs.Indexers[i]
		bz, err := k.cdc.Marshal(&row)
		if err != nil {
			panic(fmt.Errorf("genesis marshal indexer: %w", err))
		}
		store.Set(indexerRowKey(row.SourceChainId, row.ValidatorPubkey), bz)
		store.Set(addrIndexKey(row.OperatorAddress), encodeAddrIndexValue(row.SourceChainId, row.ValidatorPubkey))
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(len(gs.Indexers)))
	store.Set([]byte(types.IndexerCountKey), buf[:])
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	indexers, _, _ := k.IterateIndexers(ctx, &query.PageRequest{Limit: 10_000_000})
	return &types.GenesisState{Indexers: indexers}
}
