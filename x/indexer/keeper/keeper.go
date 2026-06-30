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

	commoncrypto "github.com/shinzonetwork/shinzohub/x/common/crypto"
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

func (k Keeper) AdminKeeper() types.AdminKeeper         { return k.adminKeeper }
func (k Keeper) SourcehubKeeper() types.SourcehubKeeper { return k.sourcehubKeeper }

func indexerRowKey(sourceChainID uint64, validatorPubkey []byte) []byte {
	out := make([]byte, 0, len(types.IndexerByValidatorPrefix)+8+len(validatorPubkey))
	out = append(out, []byte(types.IndexerByValidatorPrefix)...)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], sourceChainID)
	out = append(out, buf[:]...)
	out = append(out, validatorPubkey...)
	return out
}

func addrIndexKey(operatorAddress string) []byte {
	return []byte(types.AddrIndexPrefix + operatorAddress)
}

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

func (k Keeper) IterateIndexers(ctx sdk.Context, sourceChainID uint64, pageReq *query.PageRequest) ([]types.Indexer, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	iterPrefix := []byte(types.IndexerByValidatorPrefix)
	if sourceChainID != 0 {
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], sourceChainID)
		iterPrefix = append(iterPrefix, buf[:]...)
	}
	indexerStore := prefix.NewStore(store, iterPrefix)

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

func (k Keeper) UpsertAssertion(ctx sdk.Context, msg *types.MsgIndexerAssertion) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	rowKey := indexerRowKey(msg.SourceChainId, msg.ValidatorPubkey)
	existingBz := store.Get(rowKey)

	var (
		existing        types.Indexer
		isUpdate        = len(existingBz) > 0
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
		row.Registered = existing.Registered
		row.Did = existing.Did
		row.ConnectionString = existing.ConnectionString
	}

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
	k.deletePendingClaimsByOperator(ctx, row.OperatorAddress)
	k.decrementCount(ctx)

	emitRevoked(ctx, &row, msg.Nonce)
	return nil
}

// deletePendingClaimsByOperator removes any in-flight pending claims belonging to
// operatorAddress. Pending claims are keyed by DID, not operator, so a claim from
// an in-flight registration ICA cannot be deleted by key alone — we scan the
// prefix and drop every entry for this operator. This keeps revoke deterministic:
// without it an orphaned claim would linger in state until (and only if) its stale
// ack eventually arrives.
func (k Keeper) deletePendingClaimsByOperator(ctx sdk.Context, operatorAddress string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	claimStore := prefix.NewStore(store, []byte(types.PendingClaimPrefix))

	var staleDIDs [][]byte
	it := claimStore.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var claim types.PendingClaim
		if err := k.cdc.Unmarshal(it.Value(), &claim); err != nil {
			continue
		}
		if claim.OperatorAddress == operatorAddress {
			staleDIDs = append(staleDIDs, append([]byte(nil), it.Key()...))
		}
	}

	for _, did := range staleDIDs {
		claimStore.Delete(did)
	}
}

type RegisterResult struct {
	Did           string
	SourceChain   string
	SourceChainID uint64
	Pending       bool
}

func (k Keeper) RegisterIndexer(
	ctx sdk.Context,
	operatorAddress string,
	nodeIdentityKeyPubkey []byte,
	nodeIdentityKeySignature []byte,
	message []byte,
	connectionString string,
) (RegisterResult, error) {
	if err := commoncrypto.VerifyNodeIdentityKeySignature(nodeIdentityKeyPubkey, message, nodeIdentityKeySignature); err != nil {
		return RegisterResult{}, err
	}

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(addrIndexKey(operatorAddress))
	if len(v) == 0 {
		return RegisterResult{}, fmt.Errorf("indexer not asserted for address %s", operatorAddress)
	}
	chainID, pub, decErr := decodeAddrIndexValue(v)
	if decErr != nil {
		return RegisterResult{}, decErr
	}
	rowBz := store.Get(indexerRowKey(chainID, pub))
	if len(rowBz) == 0 {
		return RegisterResult{}, fmt.Errorf("addr_idx points at missing row")
	}
	var row types.Indexer
	if err := k.cdc.Unmarshal(rowBz, &row); err != nil {
		return RegisterResult{}, err
	}

	did, err := commoncrypto.DeriveDID(nodeIdentityKeyPubkey)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("derive did: %w", err)
	}

	out := RegisterResult{
		Did:           did,
		SourceChain:   row.SourceChain,
		SourceChainID: row.SourceChainId,
	}

	if row.Registered && row.Did == did && row.ConnectionString == connectionString {
		out.Pending = false
		return out, nil
	}

	claim := &types.PendingClaim{
		OperatorAddress:  operatorAddress,
		ConnectionString: connectionString,
	}
	claimBz, mErr := k.cdc.Marshal(claim)
	if mErr != nil {
		return RegisterResult{}, fmt.Errorf("marshal pending claim: %w", mErr)
	}
	store.Set(pendingClaimKey(did), claimBz)

	if row.Did == "" {
		if _, _, _, err := k.sourcehubKeeper.SendICASetRelationship(ctx, did, types.GroupIndexerName, operatorAddress); err != nil {
			store.Delete(pendingClaimKey(did))
			return RegisterResult{}, err
		}
	} else {
		if _, _, _, err := k.sourcehubKeeper.SendICASetAndDeleteRelationship(ctx, did, row.Did, types.GroupIndexerName, operatorAddress); err != nil {
			store.Delete(pendingClaimKey(did))
			return RegisterResult{}, err
		}
	}

	emitPending(ctx, &row, did, connectionString)
	out.Pending = true
	return out, nil
}

func pendingClaimKey(did string) []byte {
	return []byte(types.PendingClaimPrefix + did)
}

func (k Keeper) SetPendingClaim(ctx sdk.Context, did string, claim types.PendingClaim) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&claim)
	if err != nil {
		return err
	}
	store.Set(pendingClaimKey(did), bz)
	return nil
}

func (k Keeper) GetPendingClaim(ctx sdk.Context, did string) (types.PendingClaim, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(pendingClaimKey(did))
	if len(bz) == 0 {
		return types.PendingClaim{}, false, nil
	}
	var c types.PendingClaim
	if err := k.cdc.Unmarshal(bz, &c); err != nil {
		return types.PendingClaim{}, false, err
	}
	return c, true, nil
}

func (k Keeper) DeletePendingClaim(ctx sdk.Context, did string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Delete(pendingClaimKey(did))
}

func (k Keeper) ApplyRegistration(
	ctx sdk.Context,
	operatorAddress string,
	did string,
	connectionString string,
) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	v := store.Get(addrIndexKey(operatorAddress))
	if len(v) == 0 {
		return nil
	}
	chainID, pub, err := decodeAddrIndexValue(v)
	if err != nil {
		return err
	}

	rowKey := indexerRowKey(chainID, pub)
	bz := store.Get(rowKey)
	if len(bz) == 0 {
		return nil
	}
	var row types.Indexer
	if err := k.cdc.Unmarshal(bz, &row); err != nil {
		return err
	}

	row.Did = did
	row.ConnectionString = connectionString
	row.Registered = true

	out, err := k.cdc.Marshal(&row)
	if err != nil {
		return err
	}
	store.Set(rowKey, out)

	emitRegistered(ctx, &row)
	return nil
}

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
	indexers, _, _ := k.IterateIndexers(ctx, 0, &query.PageRequest{Limit: 10_000_000})
	return &types.GenesisState{Indexers: indexers}
}
