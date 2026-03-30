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

	"github.com/shinzonetwork/shinzohub/x/host/types"
)

type Keeper struct {
	cdc             codec.BinaryCodec
	storeService    storetypes.KVStoreService
	authority       string
	sourcehubKeeper types.SourcehubKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	sourcehubKeeper types.SourcehubKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:             cdc,
		storeService:    storeService,
		sourcehubKeeper: sourcehubKeeper,
		authority:       authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) RegisterHost(
	ctx sdk.Context,
	connectionString string,
	callerAddr []byte,
) ([]byte, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	// Use caller address as DID.
	did := sdk.AccAddress(callerAddr).String()
	didBytes := []byte(did)

	// Check for existing registration.
	addrKey := append([]byte(types.AddrDIDPrefix), callerAddr...)
	existingDID := store.Get(addrKey)
	if len(existingDID) > 0 {
		if !bytesEqual(existingDID, didBytes) {
			return nil, fmt.Errorf("address already registered as host with a different DID")
		}
	}

	didKey := append([]byte(types.DIDAddrPrefix), didBytes...)
	existingAddr := store.Get(didKey)
	if len(existingAddr) > 0 {
		if !bytesEqual(existingAddr, callerAddr) {
			return nil, fmt.Errorf("DID already registered as host with a different address")
		}
	}

	// Send ICA transaction for ACP relationship.
	if err := k.sourcehubKeeper.SendICASetRelationship(ctx, did, "host"); err != nil {
		return nil, err
	}

	// Store addr→DID and DID→addr mappings.
	store.Set(addrKey, didBytes)
	store.Set(didKey, callerAddr)

	// Store the indexed host record.
	bech32Addr := sdk.AccAddress(callerAddr).String()
	if err := k.SetHost(ctx, types.Host{
		Address:          bech32Addr,
		Did:              did,
		ConnectionString: connectionString,
	}); err != nil {
		return nil, fmt.Errorf("failed to index host: %w", err)
	}

	return didBytes, nil
}

func (k Keeper) IsRegisteredHost(ctx sdk.Context, address []byte) bool {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	addrKey := append([]byte(types.AddrDIDPrefix), address...)
	v := store.Get(addrKey)
	return len(v) > 0
}

func (k Keeper) GetDIDForAddress(ctx sdk.Context, address []byte) ([]byte, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	addrKey := append([]byte(types.AddrDIDPrefix), address...)
	v := store.Get(addrKey)
	if len(v) == 0 {
		return nil, false
	}
	return v, true
}

func (k Keeper) SetHost(ctx sdk.Context, host types.Host) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := []byte(types.HostPrefix + host.Address)

	isNew := len(store.Get(key)) == 0

	bz, err := k.cdc.Marshal(&host)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	if isNew {
		k.incrementCount(ctx)
	}
	return nil
}

func (k Keeper) GetHost(ctx sdk.Context, address string) (types.Host, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := []byte(types.HostPrefix + address)

	bz := store.Get(key)
	if len(bz) == 0 {
		return types.Host{}, false, nil
	}

	var host types.Host
	if err := k.cdc.Unmarshal(bz, &host); err != nil {
		return types.Host{}, false, err
	}
	return host, true, nil
}

func (k Keeper) GetAllHosts(ctx sdk.Context, pageReq *query.PageRequest) ([]types.Host, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	hostStore := prefix.NewStore(store, []byte(types.HostPrefix))

	var hosts []types.Host
	pageRes, err := query.Paginate(hostStore, pageReq, func(key, value []byte) error {
		var host types.Host
		if err := k.cdc.Unmarshal(value, &host); err != nil {
			return err
		}
		hosts = append(hosts, host)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return hosts, pageRes, nil
}

func (k Keeper) GetHostCount(ctx sdk.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.HostCountKey))
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) incrementCount(ctx sdk.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	count := k.GetHostCount(ctx) + 1
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set([]byte(types.HostCountKey), bz)
}

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	for _, host := range gs.Hosts {
		_ = k.SetHost(ctx, host)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	hosts, _, _ := k.GetAllHosts(ctx, &query.PageRequest{Limit: uint64(10000000)})
	return &types.GenesisState{Hosts: hosts}
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
