package keeper

import (
	"encoding/binary"
	"fmt"
	"net/url"

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

func validateEndpointAddress(endpointAddress string) error {
	u, err := url.Parse(endpointAddress)
	if err != nil {
		return fmt.Errorf("%w: %w", types.ErrInvalidEndpointAddress, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https, got %q", types.ErrInvalidEndpointAddress, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: missing host", types.ErrInvalidEndpointAddress)
	}
	return nil
}

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
	endpointAddress string,
	caller sdk.AccAddress,
) ([]byte, error) {
	if err := commoncrypto.VerifyNodeIdentityKeySignature(nodeIdentityKeyPubkey, message, nodeIdentityKeySignature); err != nil {
		return nil, err
	}

	if err := validateEndpointAddress(endpointAddress); err != nil {
		return nil, err
	}

	did, err := commoncrypto.DeriveDID(nodeIdentityKeyPubkey)
	if err != nil {
		return nil, err
	}

	bech32Addr := caller.String()
	bech32Bytes := []byte(bech32Addr)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	addrKey := addrIndexKey(types.AddrDIDPrefix, bech32Addr)
	pendingAddrKey := addrIndexKey(types.PendingAddrDIDPrefix, bech32Addr)
	for _, key := range [][]byte{addrKey, pendingAddrKey} {
		existingDID := store.Get(key)
		if len(existingDID) > 0 && string(existingDID) != did {
			return nil, types.ErrAddressRegisteredDifferentDID
		}
	}

	didKey := didIndexKey(types.DIDAddrPrefix, did)
	pendingDidKey := didIndexKey(types.PendingDIDAddrPrefix, did)
	for _, key := range [][]byte{didKey, pendingDidKey} {
		existingAddr := store.Get(key)
		if len(existingAddr) > 0 && string(existingAddr) != bech32Addr {
			return nil, types.ErrDIDRegisteredDifferentAddress
		}
	}

	host := types.Host{
		Address:          bech32Addr,
		Did:              did,
		ConnectionString: connectionString,
		EndpointAddress:  endpointAddress,
	}
	if err := k.SetPendingHost(ctx, host); err != nil {
		return nil, fmt.Errorf("record pending host: %w", err)
	}
	store.Set(pendingAddrKey, []byte(did))
	store.Set(pendingDidKey, bech32Bytes)

	if _, _, _, err := k.sourcehubKeeper.SendICASetRelationship(ctx, did, "host", bech32Addr); err != nil {
		_ = k.DeletePendingHost(ctx, bech32Addr)
		store.Delete(pendingAddrKey)
		store.Delete(pendingDidKey)
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeHostPending,
		sdk.NewAttribute(types.AttrKeyAddress, bech32Addr),
		sdk.NewAttribute(types.AttrKeyDID, did),
	))

	return []byte(did), nil
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

func (k Keeper) IsRegisteredHost(ctx sdk.Context, addr sdk.AccAddress) bool {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(addrIndexKey(types.AddrDIDPrefix, addr.String()))
	return len(v) > 0
}

func (k Keeper) GetDIDForAddress(ctx sdk.Context, addr sdk.AccAddress) (string, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(addrIndexKey(types.AddrDIDPrefix, addr.String()))
	if len(v) == 0 {
		return "", false
	}
	return string(v), true
}

func (k Keeper) GetDIDForPendingAddress(ctx sdk.Context, addr sdk.AccAddress) (string, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(addrIndexKey(types.PendingAddrDIDPrefix, addr.String()))
	if len(v) == 0 {
		return "", false
	}
	return string(v), true
}

func (k Keeper) GetAddressForDID(ctx sdk.Context, did string) (sdk.AccAddress, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(didIndexKey(types.DIDAddrPrefix, did))
	if len(v) == 0 {
		return nil, false
	}
	addr, err := sdk.AccAddressFromBech32(string(v))
	if err != nil {
		return nil, false
	}
	return addr, true
}

func (k Keeper) GetPendingAddressForDID(ctx sdk.Context, did string) (sdk.AccAddress, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(didIndexKey(types.PendingDIDAddrPrefix, did))
	if len(v) == 0 {
		return nil, false
	}
	addr, err := sdk.AccAddressFromBech32(string(v))
	if err != nil {
		return nil, false
	}
	return addr, true
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

func (k Keeper) FilterHosts(
	ctx sdk.Context,
	pageReq *query.PageRequest,
	onResult func(host types.Host, accumulate bool) (bool, error),
) (*query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	hostStore := prefix.NewStore(store, []byte(types.HostPrefix))

	return query.FilteredPaginate(hostStore, pageReq, func(_, value []byte, accumulate bool) (bool, error) {
		var host types.Host
		if err := k.cdc.Unmarshal(value, &host); err != nil {
			return false, err
		}
		return onResult(host, accumulate)
	})
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

func addrIndexKey(prefix, bech32Addr string) []byte {
	return []byte(prefix + bech32Addr)
}

func didIndexKey(prefix, did string) []byte {
	return []byte(prefix + did)
}
