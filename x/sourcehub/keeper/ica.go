package keeper

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (k Keeper) SetControllerConnectionID(ctx sdk.Context, connectionID string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.KeyConnectionID), []byte(connectionID))
}

func (k Keeper) GetControllerConnectionID(ctx sdk.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.KeyConnectionID))
	if bz == nil {
		return ""
	}
	return string(bz)
}

func (k Keeper) SetHostConnectionID(ctx sdk.Context, hostConnectionID string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.KeyHostConnectionID), []byte(hostConnectionID))
}

func (k Keeper) GetHostConnectionID(ctx sdk.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.KeyHostConnectionID))
	if bz == nil {
		return ""
	}
	return string(bz)
}

func (k Keeper) SetVersion(ctx sdk.Context, version string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.KeyVersion), []byte(version))
}

func (k Keeper) GetVersion(ctx sdk.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.KeyVersion))
	if bz == nil {
		return ""
	}
	return string(bz)
}

func (k Keeper) SetEncoding(ctx sdk.Context, encoding string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.KeyEncoding), []byte(encoding))
}

func (k Keeper) GetEncoding(ctx sdk.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.KeyEncoding))
	if bz == nil {
		return ""
	}
	return string(bz)
}

func (k Keeper) SetTxType(ctx sdk.Context, txType string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.KeyTxType), []byte(txType))
}

func (k Keeper) GetTxType(ctx sdk.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.KeyTxType))
	if bz == nil {
		return ""
	}
	return string(bz)
}

func (k Keeper) GetICAMetadata(ctx sdk.Context) string {
	controllerConnID := k.GetControllerConnectionID(ctx)
	hostConnID := k.GetHostConnectionID(ctx)
	version := k.GetVersion(ctx)
	encoding := k.GetEncoding(ctx)
	txType := k.GetTxType(ctx)

	meta := map[string]string{
		"version":                  version,
		"controller_connection_id": controllerConnID,
		"host_connection_id":       hostConnID,
		"encoding":                 encoding,
		"tx_type":                  txType,
	}

	bz, err := json.Marshal(meta)
	if err != nil {
		return ""
	}

	return string(bz)
}
