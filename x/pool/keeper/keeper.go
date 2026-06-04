package keeper

import (
	"encoding/binary"
	"fmt"
	"strconv"

	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/shinzonetwork/shinzohub/x/pool/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService
	authority    string
	viewKeeper   types.ViewKeeper
	bankKeeper   types.BankKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	viewKeeper types.ViewKeeper,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		viewKeeper:   viewKeeper,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) ViewKeeper() types.ViewKeeper { return k.viewKeeper }
func (k Keeper) BankKeeper() types.BankKeeper { return k.bankKeeper }

// CreatePool persists a new pool entry. Reverts if a pool with the same address
// already exists.
func (k Keeper) CreatePool(
	ctx sdk.Context,
	poolAddress, viewAddress string,
	config types.PoolConfig,
) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := poolKey(poolAddress)

	if len(store.Get(key)) > 0 {
		return fmt.Errorf("pool already exists: %s", poolAddress)
	}

	// Ensure the view exists before allowing a pool to be created for it.
	if _, found, err := k.viewKeeper.GetView(ctx, viewAddress); err != nil {
		return fmt.Errorf("view lookup failed: %w", err)
	} else if !found {
		return fmt.Errorf("view not registered: %s", viewAddress)
	}

	pool := types.Pool{
		PoolAddress: poolAddress,
		ViewAddress: viewAddress,
		Config:      config,
		CreatedAt:   ctx.BlockHeight(),
	}

	bz, err := k.cdc.Marshal(&pool)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	store.Set(poolByViewKey(viewAddress, poolAddress), []byte{})

	k.incrementCount(ctx)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypePoolCreated,
		sdk.NewAttribute(types.AttrKeyPoolAddress, poolAddress),
		sdk.NewAttribute(types.AttrKeyViewAddress, viewAddress),
	))

	return nil
}

// GetPoolDetail returns a pool together with all its hosts and demands.
// Returns (PoolDetail{}, false, nil) if the pool doesn't exist.
func (k Keeper) GetPoolDetail(ctx sdk.Context, poolAddress string) (types.PoolDetail, bool, error) {
	pool, found, err := k.GetPool(ctx, poolAddress)
	if err != nil || !found {
		return types.PoolDetail{}, found, err
	}

	var hosts []types.PoolHostEntry
	if err := k.IterateHosts(ctx, poolAddress, func(addr string, h types.PoolHost) bool {
		hosts = append(hosts, types.PoolHostEntry{
			PoolAddress: poolAddress,
			HostAddress: addr,
			Host:        h,
		})
		return true
	}); err != nil {
		return types.PoolDetail{}, false, err
	}

	var demands []types.PoolDemandEntry
	if err := k.IterateDemands(ctx, poolAddress, func(addr string, d types.PoolDemand) bool {
		demands = append(demands, types.PoolDemandEntry{
			PoolAddress:       poolAddress,
			RegistrantAddress: addr,
			Demand:            d,
		})
		return true
	}); err != nil {
		return types.PoolDetail{}, false, err
	}

	return types.PoolDetail{
		Pool:    pool,
		Hosts:   hosts,
		Demands: demands,
	}, true, nil
}

// GetAllPoolDetails returns every pool denormalized with its hosts and demands.
// Heavier than GetAllPools because it does extra iteration per pool.
func (k Keeper) GetAllPoolDetails(ctx sdk.Context, pageReq *query.PageRequest) ([]types.PoolDetail, *query.PageResponse, error) {
	pools, pageRes, err := k.GetAllPools(ctx, pageReq)
	if err != nil {
		return nil, nil, err
	}

	details := make([]types.PoolDetail, 0, len(pools))
	for _, p := range pools {
		d, _, err := k.GetPoolDetail(ctx, p.PoolAddress)
		if err != nil {
			return nil, nil, err
		}
		details = append(details, d)
	}
	return details, pageRes, nil
}

// GetPoolsForView returns all pool addresses created for a given view.
func (k Keeper) GetPoolsForView(ctx sdk.Context, viewAddress string) ([]string, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	idxStore := prefix.NewStore(store, []byte(types.PoolByViewPrefix+viewAddress+"/"))

	var addrs []string
	it := idxStore.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		addrs = append(addrs, string(it.Key()))
	}
	return addrs, nil
}

func (k Keeper) GetPool(ctx sdk.Context, poolAddress string) (types.Pool, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(poolKey(poolAddress))
	if len(bz) == 0 {
		return types.Pool{}, false, nil
	}
	var pool types.Pool
	if err := k.cdc.Unmarshal(bz, &pool); err != nil {
		return types.Pool{}, false, err
	}
	return pool, true, nil
}

func (k Keeper) PoolExists(ctx sdk.Context, poolAddress string) bool {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return len(store.Get(poolKey(poolAddress))) > 0
}

func (k Keeper) GetAllPools(ctx sdk.Context, pageReq *query.PageRequest) ([]types.Pool, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	poolStore := prefix.NewStore(store, []byte(types.PoolPrefix))

	var pools []types.Pool
	pageRes, err := query.Paginate(poolStore, pageReq, func(_, value []byte) error {
		var p types.Pool
		if err := k.cdc.Unmarshal(value, &p); err != nil {
			return err
		}
		pools = append(pools, p)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return pools, pageRes, nil
}

func (k Keeper) GetPoolCount(ctx sdk.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.PoolCountKey))
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) incrementCount(ctx sdk.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	count := k.GetPoolCount(ctx) + 1
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set([]byte(types.PoolCountKey), bz)
}

// AddHost adds a host as a member of the pool. The ask starts at "0" until
// the host submits one via SetHostAsk. Reverts if the host is already a member.
func (k Keeper) AddHost(ctx sdk.Context, poolAddress, hostAddress string) error {
	if !k.PoolExists(ctx, poolAddress) {
		return fmt.Errorf("pool not found: %s", poolAddress)
	}

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := hostKey(poolAddress, hostAddress)

	if len(store.Get(key)) > 0 {
		return fmt.Errorf("host already in pool: %s", hostAddress)
	}

	h := types.PoolHost{
		Ask:      "0",
		JoinedAt: ctx.BlockHeight(),
	}
	bz, err := k.cdc.Marshal(&h)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeHostJoined,
		sdk.NewAttribute(types.AttrKeyPoolAddress, poolAddress),
		sdk.NewAttribute(types.AttrKeyHostAddress, hostAddress),
	))

	return nil
}

// SetHostAsk updates the host's ask price. Host must already be a member.
func (k Keeper) SetHostAsk(ctx sdk.Context, poolAddress, hostAddress, ask string) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := hostKey(poolAddress, hostAddress)

	bz := store.Get(key)
	if len(bz) == 0 {
		return fmt.Errorf("host not in pool: %s", hostAddress)
	}

	var h types.PoolHost
	if err := k.cdc.Unmarshal(bz, &h); err != nil {
		return err
	}
	h.Ask = ask

	bz, err := k.cdc.Marshal(&h)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAskSubmitted,
		sdk.NewAttribute(types.AttrKeyPoolAddress, poolAddress),
		sdk.NewAttribute(types.AttrKeyHostAddress, hostAddress),
		sdk.NewAttribute(types.AttrKeyAsk, ask),
	))

	return nil
}

func (k Keeper) GetHost(ctx sdk.Context, poolAddress, hostAddress string) (types.PoolHost, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(hostKey(poolAddress, hostAddress))
	if len(bz) == 0 {
		return types.PoolHost{}, false, nil
	}
	var h types.PoolHost
	if err := k.cdc.Unmarshal(bz, &h); err != nil {
		return types.PoolHost{}, false, err
	}
	return h, true, nil
}

func (k Keeper) RemoveHost(ctx sdk.Context, poolAddress, hostAddress string) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := hostKey(poolAddress, hostAddress)
	if len(store.Get(key)) == 0 {
		return fmt.Errorf("host not in pool: %s", hostAddress)
	}
	store.Delete(key)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeHostLeft,
		sdk.NewAttribute(types.AttrKeyPoolAddress, poolAddress),
		sdk.NewAttribute(types.AttrKeyHostAddress, hostAddress),
	))

	return nil
}

// IterateHosts walks every host in a pool. Return false from the callback to stop.
func (k Keeper) IterateHosts(
	ctx sdk.Context,
	poolAddress string,
	cb func(hostAddress string, h types.PoolHost) bool,
) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	hostStore := prefix.NewStore(store, []byte(types.PoolHostPrefix+poolAddress+"/"))

	it := hostStore.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var h types.PoolHost
		if err := k.cdc.Unmarshal(it.Value(), &h); err != nil {
			return err
		}
		if !cb(string(it.Key()), h) {
			break
		}
	}
	return nil
}

// RegisterDemand persists a demand entry. The caller (precompile) is responsible
// for moving the bond into the pool module account before calling this.
// Reverts if the registrant already has an open demand on this pool.
func (k Keeper) RegisterDemand(
	ctx sdk.Context,
	poolAddress, registrantAddress string,
	demand types.PoolDemand,
) error {
	if !k.PoolExists(ctx, poolAddress) {
		return fmt.Errorf("pool not found: %s", poolAddress)
	}

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := demandKey(poolAddress, registrantAddress)

	if len(store.Get(key)) > 0 {
		return fmt.Errorf("demand already registered: %s", registrantAddress)
	}

	bz, err := k.cdc.Marshal(&demand)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeDemandRegistered,
		sdk.NewAttribute(types.AttrKeyPoolAddress, poolAddress),
		sdk.NewAttribute(types.AttrKeyRegistrantAddress, registrantAddress),
		sdk.NewAttribute(types.AttrKeyBond, demand.Bond),
		sdk.NewAttribute("expires_at", strconv.FormatInt(demand.ExpiresAt, 10)),
	))

	return nil
}

func (k Keeper) GetDemand(ctx sdk.Context, poolAddress, registrantAddress string) (types.PoolDemand, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(demandKey(poolAddress, registrantAddress))
	if len(bz) == 0 {
		return types.PoolDemand{}, false, nil
	}
	var d types.PoolDemand
	if err := k.cdc.Unmarshal(bz, &d); err != nil {
		return types.PoolDemand{}, false, err
	}
	return d, true, nil
}

func (k Keeper) IterateDemands(
	ctx sdk.Context,
	poolAddress string,
	cb func(registrantAddress string, d types.PoolDemand) bool,
) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	dmdStore := prefix.NewStore(store, []byte(types.PoolDemandPrefix+poolAddress+"/"))

	it := dmdStore.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var d types.PoolDemand
		if err := k.cdc.Unmarshal(it.Value(), &d); err != nil {
			return err
		}
		if !cb(string(it.Key()), d) {
			break
		}
	}
	return nil
}

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	for _, p := range gs.Pools {
		bz, err := k.cdc.Marshal(&p)
		if err != nil {
			panic(err)
		}
		store.Set(poolKey(p.PoolAddress), bz)
	}

	for _, e := range gs.Hosts {
		bz, err := k.cdc.Marshal(&e.Host)
		if err != nil {
			panic(err)
		}
		store.Set(hostKey(e.PoolAddress, e.HostAddress), bz)
	}

	for _, e := range gs.Demands {
		bz, err := k.cdc.Marshal(&e.Demand)
		if err != nil {
			panic(err)
		}
		store.Set(demandKey(e.PoolAddress, e.RegistrantAddress), bz)
	}

	if n := uint64(len(gs.Pools)); n > 0 {
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, n)
		store.Set([]byte(types.PoolCountKey), bz)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	pools, _, _ := k.GetAllPools(ctx, &query.PageRequest{Limit: 10_000_000})

	gs := &types.GenesisState{Pools: pools}

	for _, p := range pools {
		_ = k.IterateHosts(ctx, p.PoolAddress, func(addr string, h types.PoolHost) bool {
			gs.Hosts = append(gs.Hosts, types.PoolHostEntry{
				PoolAddress: p.PoolAddress,
				HostAddress: addr,
				Host:        h,
			})
			return true
		})
		_ = k.IterateDemands(ctx, p.PoolAddress, func(addr string, d types.PoolDemand) bool {
			gs.Demands = append(gs.Demands, types.PoolDemandEntry{
				PoolAddress:       p.PoolAddress,
				RegistrantAddress: addr,
				Demand:            d,
			})
			return true
		})
	}

	return gs
}

func poolKey(poolAddress string) []byte {
	return []byte(types.PoolPrefix + poolAddress)
}

func poolByViewKey(viewAddress, poolAddress string) []byte {
	return []byte(types.PoolByViewPrefix + viewAddress + "/" + poolAddress)
}

func hostKey(poolAddress, hostAddress string) []byte {
	return []byte(types.PoolHostPrefix + poolAddress + "/" + hostAddress)
}

func demandKey(poolAddress, registrantAddress string) []byte {
	return []byte(types.PoolDemandPrefix + poolAddress + "/" + registrantAddress)
}
