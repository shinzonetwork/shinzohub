package keeper

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

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
	sourcehubKeeper types.SourcehubKeeper
}

func NewKeeper(cdc codec.BinaryCodec, storeService storetypes.KVStoreService, sourcehubKeeper types.SourcehubKeeper) Keeper {
	return Keeper{cdc: cdc, storeService: storeService, sourcehubKeeper: sourcehubKeeper}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// Stages a pending view and fires the ACP register-object ICA. Finalised in ack_callback.go on SUCCESS.
func (k Keeper) RegisterView(ctx sdk.Context, name, creator, address string, data []byte) (types.View, error) {
	// The view address is deterministic from caller+bundle, so an existing final
	// or pending entry means this exact registration is already complete or in
	// flight. Re-staging would fire a duplicate RegisterObject ICA; if that
	// duplicate's ack fails it would delete the pending row and strand a view the
	// first ICA actually registered. Treat re-registration as an idempotent no-op.
	if existing, found, err := k.GetView(ctx, address); err != nil {
		return types.View{}, err
	} else if found {
		return existing, nil
	}
	if existing, found, err := k.GetPendingView(ctx, address); err != nil {
		return types.View{}, err
	} else if found {
		return existing, nil
	}

	view := types.View{
		Name:    name,
		Creator: creator,
		Address: address,
		Data:    data,
		Height:  uint64(ctx.BlockHeight()),
	}

	if err := k.SetPendingView(ctx, view); err != nil {
		return types.View{}, fmt.Errorf("record pending view: %w", err)
	}

	requestor, err := evmHexToBech32(creator)
	if err != nil {
		_ = k.DeletePendingView(ctx, address)
		return types.View{}, fmt.Errorf("convert creator to bech32: %w", err)
	}
	if _, _, _, err := k.sourcehubKeeper.RegisterObject(ctx, address, requestor); err != nil {
		_ = k.DeletePendingView(ctx, address)
		return types.View{}, fmt.Errorf("register view object via ICA: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeViewPending,
		sdk.NewAttribute(types.AttrKeyAddress, address),
		sdk.NewAttribute(types.AttrKeyCreator, creator),
		sdk.NewAttribute(types.AttrKeyName, name),
		sdk.NewAttribute(types.AttrKeyData, base64.StdEncoding.EncodeToString(data)),
	))

	return view, nil
}

func (k Keeper) SetPendingView(ctx sdk.Context, view types.View) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&view)
	if err != nil {
		return err
	}
	store.Set([]byte(types.PendingViewPrefix+view.Address), bz)
	return nil
}

func (k Keeper) GetPendingView(ctx sdk.Context, address string) (types.View, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.PendingViewPrefix + address))
	if len(bz) == 0 {
		return types.View{}, false, nil
	}
	var v types.View
	if err := k.cdc.Unmarshal(bz, &v); err != nil {
		return types.View{}, false, err
	}
	return v, true, nil
}

func (k Keeper) DeletePendingView(ctx sdk.Context, address string) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Delete([]byte(types.PendingViewPrefix + address))
	return nil
}

func (k Keeper) SetView(ctx sdk.Context, view types.View) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := []byte(types.ViewPrefix + view.Address)

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

func (k Keeper) GetView(ctx sdk.Context, address string) (types.View, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.ViewPrefix + address))
	if len(bz) == 0 {
		return types.View{}, false, nil
	}
	var view types.View
	if err := k.cdc.Unmarshal(bz, &view); err != nil {
		return types.View{}, false, err
	}
	return view, true, nil
}

func (k Keeper) GetAllViews(ctx sdk.Context, pageReq *query.PageRequest) ([]types.View, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	viewStore := prefix.NewStore(store, []byte(types.ViewPrefix))

	var views []types.View
	pageRes, err := query.Paginate(viewStore, pageReq, func(_, value []byte) error {
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

func (k Keeper) FilterViews(
	ctx sdk.Context,
	pageReq *query.PageRequest,
	onResult func(view types.View, accumulate bool) (bool, error),
) (*query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	viewStore := prefix.NewStore(store, []byte(types.ViewPrefix))

	return query.FilteredPaginate(viewStore, pageReq, func(_, value []byte, accumulate bool) (bool, error) {
		var view types.View
		if err := k.cdc.Unmarshal(value, &view); err != nil {
			return false, err
		}
		return onResult(view, accumulate)
	})
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
	views, _, _ := k.GetAllViews(ctx, &query.PageRequest{Limit: 10_000_000})
	return &types.GenesisState{Views: views}
}

// Sourcehub expects the ACP requestor as bech32, but the keeper holds EVM hex.
func evmHexToBech32(addr string) (string, error) {
	raw, err := hex.DecodeString(strings.TrimPrefix(addr, "0x"))
	if err != nil {
		return "", err
	}
	if len(raw) != 20 {
		return "", fmt.Errorf("expected 20 bytes, got %d", len(raw))
	}
	return sdk.AccAddress(raw).String(), nil
}
