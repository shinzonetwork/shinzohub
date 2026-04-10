package keeper

import (
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

func (k Keeper) RegisterIndexer(
	ctx sdk.Context,
	nodeIdentityKeyPubkey []byte,
	nodeIdentityKeySignature []byte,
	message []byte,
	connectionString string,
	callerAddr []byte,
	sourceChain string,
	sourceChainId uint64,
) ([]byte, error) {
	if err := commoncrypto.VerifyNodeIdentityKeySignature(nodeIdentityKeyPubkey, message, nodeIdentityKeySignature); err != nil {
		return nil, err
	}

	did, err := commoncrypto.DeriveDID(nodeIdentityKeyPubkey)
	if err != nil {
		return nil, err
	}

	didBytes := []byte(did)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	addrKey := append([]byte(types.AddrDIDPrefix), callerAddr...)
	existingDID := store.Get(addrKey)
	if len(existingDID) > 0 {
		if !bytesEqual(existingDID, didBytes) {
			return nil, fmt.Errorf("address already registered as indexer with a different DID")
		}
	}

	didKey := append([]byte(types.DIDAddrPrefix), didBytes...)
	existingAddr := store.Get(didKey)
	if len(existingAddr) > 0 {
		if !bytesEqual(existingAddr, callerAddr) {
			return nil, fmt.Errorf("DID already registered as indexer with a different address")
		}
	}

	if err := k.sourcehubKeeper.SendICASetRelationship(ctx, did, "indexer"); err != nil {
		return nil, err
	}

	store.Set(addrKey, didBytes)
	store.Set(didKey, callerAddr)

	bech32Addr := sdk.AccAddress(callerAddr).String()
	if err := k.SetIndexer(ctx, types.Indexer{
		Address:          bech32Addr,
		Did:              did,
		ConnectionString: connectionString,
		SourceChain:      sourceChain,
		SourceChainId:    sourceChainId,
	}); err != nil {
		return nil, fmt.Errorf("failed to index indexer: %w", err)
	}

	return didBytes, nil
}

func assertionKey(delegate, sourceChain string, sourceChainId uint64) []byte {
	suffix := fmt.Sprintf("%s:%s:%d", delegate, sourceChain, sourceChainId)
	return append([]byte(types.AssertionPrefix), []byte(suffix)...)
}

func (k Keeper) SetIndexerAssertion(ctx sdk.Context, a types.IndexerAssertion) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&a)
	if err != nil {
		return err
	}
	store.Set(assertionKey(a.DelegateAddress, a.SourceChain, a.SourceChainId), bz)
	return nil
}

func (k Keeper) GetIndexerAssertion(ctx sdk.Context, delegate, sourceChain string, sourceChainId uint64) (types.IndexerAssertion, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(assertionKey(delegate, sourceChain, sourceChainId))
	if len(bz) == 0 {
		return types.IndexerAssertion{}, false, nil
	}
	var a types.IndexerAssertion
	if err := k.cdc.Unmarshal(bz, &a); err != nil {
		return types.IndexerAssertion{}, false, err
	}
	return a, true, nil
}

func (k Keeper) GetAssertionsByDelegate(ctx sdk.Context, delegate string) ([]types.IndexerAssertion, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	assertionStore := prefix.NewStore(store, []byte(types.AssertionPrefix+delegate+":"))

	var assertions []types.IndexerAssertion
	iter := assertionStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var a types.IndexerAssertion
		if err := k.cdc.Unmarshal(iter.Value(), &a); err != nil {
			return nil, err
		}
		assertions = append(assertions, a)
	}
	return assertions, nil
}

func (k Keeper) SetIndexer(ctx sdk.Context, indexer types.Indexer) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := []byte(types.IndexerPrefix + indexer.Address)

	isNew := len(store.Get(key)) == 0

	bz, err := k.cdc.Marshal(&indexer)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	if isNew {
		k.incrementCount(ctx)
	}
	return nil
}

func (k Keeper) GetIndexer(ctx sdk.Context, address string) (types.Indexer, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := []byte(types.IndexerPrefix + address)

	bz := store.Get(key)
	if len(bz) == 0 {
		return types.Indexer{}, false, nil
	}

	var indexer types.Indexer
	if err := k.cdc.Unmarshal(bz, &indexer); err != nil {
		return types.Indexer{}, false, err
	}
	return indexer, true, nil
}

func (k Keeper) GetAllIndexers(ctx sdk.Context, pageReq *query.PageRequest) ([]types.Indexer, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	indexerStore := prefix.NewStore(store, []byte(types.IndexerPrefix))

	var indexers []types.Indexer
	pageRes, err := query.Paginate(indexerStore, pageReq, func(key, value []byte) error {
		var indexer types.Indexer
		if err := k.cdc.Unmarshal(value, &indexer); err != nil {
			return err
		}
		indexers = append(indexers, indexer)
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

func (k Keeper) incrementCount(ctx sdk.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	count := k.GetIndexerCount(ctx) + 1
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set([]byte(types.IndexerCountKey), bz)
}

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	for _, indexer := range gs.Indexers {
		_ = k.SetIndexer(ctx, indexer)
	}
	for _, a := range gs.Assertions {
		_ = k.SetIndexerAssertion(ctx, a)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	indexers, _, _ := k.GetAllIndexers(ctx, &query.PageRequest{Limit: uint64(10000000)})
	return &types.GenesisState{
		Indexers:   indexers,
		Assertions: []types.IndexerAssertion{}, // TODO: iterate assertion prefix for full export
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
