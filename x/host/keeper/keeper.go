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
	nodeIdentityKeyPubkey []byte,
	nodeIdentityKeySignature []byte,
	message []byte,
	connectionString string,
	callerAddr []byte,
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
	pendingAddrKey := append([]byte(types.PendingAddrDIDPrefix), callerAddr...)
	for _, key := range [][]byte{addrKey, pendingAddrKey} {
		existingDID := store.Get(key)
		if len(existingDID) > 0 && !bytesEqual(existingDID, didBytes) {
			return nil, fmt.Errorf("address already registered as host with a different DID")
		}
	}

	didKey := append([]byte(types.DIDAddrPrefix), didBytes...)
	pendingDidKey := append([]byte(types.PendingDIDAddrPrefix), didBytes...)
	for _, key := range [][]byte{didKey, pendingDidKey} {
		existingAddr := store.Get(key)
		if len(existingAddr) > 0 && !bytesEqual(existingAddr, callerAddr) {
			return nil, fmt.Errorf("DID already registered as host with a different address")
		}
	}

	bech32Addr := sdk.AccAddress(callerAddr).String()

	host := types.Host{
		Address:          bech32Addr,
		Did:              did,
		ConnectionString: connectionString,
	}
	if err := k.SetPendingHost(ctx, host); err != nil {
		return nil, fmt.Errorf("record pending host: %w", err)
	}
	store.Set(append([]byte(types.PendingAddrDIDPrefix), callerAddr...), didBytes)
	store.Set(append([]byte(types.PendingDIDAddrPrefix), didBytes...), callerAddr)

	if _, _, _, err := k.sourcehubKeeper.SendICASetRelationship(ctx, did, "host", bech32Addr); err != nil {
		_ = k.DeletePendingHost(ctx, bech32Addr)
		store.Delete(append([]byte(types.PendingAddrDIDPrefix), callerAddr...))
		store.Delete(append([]byte(types.PendingDIDAddrPrefix), didBytes...))
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeHostPending,
		sdk.NewAttribute(types.AttrKeyAddress, bech32Addr),
		sdk.NewAttribute(types.AttrKeyDID, did),
	))

	return didBytes, nil
}

func (k Keeper) SetPendingHost(ctx sdk.Context, host types.Host) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&host)
	if err != nil {
		return err
	}
	store.Set([]byte(types.PendingHostPrefix+host.Address), bz)
	return nil
}

func (k Keeper) GetPendingHost(ctx sdk.Context, address string) (types.Host, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.PendingHostPrefix + address))
	if len(bz) == 0 {
		return types.Host{}, false, nil
	}
	var h types.Host
	if err := k.cdc.Unmarshal(bz, &h); err != nil {
		return types.Host{}, false, err
	}
	return h, true, nil
}

func (k Keeper) DeletePendingHost(ctx sdk.Context, address string) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Delete([]byte(types.PendingHostPrefix + address))
	return nil
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

func (k Keeper) GetDIDForPendingAddress(ctx sdk.Context, address []byte) ([]byte, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(append([]byte(types.PendingAddrDIDPrefix), address...))
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
