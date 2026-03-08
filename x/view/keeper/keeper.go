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

	"github.com/shinzonetwork/shinzohub/x/view/types"
)

type Keeper struct {
	cdc             codec.BinaryCodec
	storeService    storetypes.KVStoreService
	authority       string
	hostKeeper      types.HostKeeper
	sourcehubKeeper types.SourcehubKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	hostKeeper types.HostKeeper,
	sourcehubKeeper types.SourcehubKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:             cdc,
		storeService:    storeService,
		hostKeeper:      hostKeeper,
		sourcehubKeeper: sourcehubKeeper,
		authority:       authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) RegisterView(ctx sdk.Context, viewId, name, creator, contractAddress string, data []byte) error {
	if err := k.sourcehubKeeper.RegisterObject(ctx, viewId); err != nil {
		return fmt.Errorf("failed to register view object: %w", err)
	}

	if err := k.SetView(ctx, types.View{
		Name:            name,
		Creator:         creator,
		ContractAddress: contractAddress,
		Data:            data,
		Height: uint64(ctx.BlockHeight()),
	}); err != nil {
		return err
	}

	return nil
}

func (k Keeper) SetView(ctx sdk.Context, view types.View) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := []byte(types.ViewPrefix + view.ContractAddress)

	isNew := len(store.Get(key)) == 0

	bz, err := k.cdc.Marshal(&view)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	if isNew {
		k.incrementCount(ctx)
	}
	return nil
}

func (k Keeper) GetView(ctx sdk.Context, contractAddress string) (types.View, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := []byte(types.ViewPrefix + contractAddress)

	bz := store.Get(key)
	if len(bz) == 0 {
		return types.View{}, false, nil
	}

	var view types.View
	if err := k.cdc.Unmarshal(bz, &view); err != nil {
		return types.View{}, false, err
	}
	return view, true, nil
}

func (k Keeper) GetViewByAddress(ctx sdk.Context, contractAddress string) (types.View, bool, error) {
	return k.GetView(ctx, contractAddress)
}

func (k Keeper) GetAllViews(ctx sdk.Context, pageReq *query.PageRequest) ([]types.View, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	viewStore := prefix.NewStore(store, []byte(types.ViewPrefix))

	var views []types.View
	pageRes, err := query.Paginate(viewStore, pageReq, func(key, value []byte) error {
		var view types.View
		if err := k.cdc.Unmarshal(value, &view); err != nil {
			return err
		}
		views = append(views, view)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return views, pageRes, nil
}

func (k Keeper) GetViewCount(ctx sdk.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.ViewCountKey))
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) incrementCount(ctx sdk.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	count := k.GetViewCount(ctx) + 1
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set([]byte(types.ViewCountKey), bz)
}

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	for _, view := range gs.Views {
		_ = k.SetView(ctx, view)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	views, _, _ := k.GetAllViews(ctx, &query.PageRequest{Limit: uint64(10000000)})
	return &types.GenesisState{Views: views}
}
