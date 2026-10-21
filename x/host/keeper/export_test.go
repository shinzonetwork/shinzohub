package keeper

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/host/types"
)

// StagePendingHost writes a pending host and its pending indexes, the state
// RegisterHost wrote before ACP was bypassed. Tests that cover the ack callback
// stage it here.
func (k Keeper) StagePendingHost(ctx sdk.Context, host types.Host) error {
	if err := k.SetPendingHost(ctx, host); err != nil {
		return err
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(addrIndexKey(types.PendingAddrDIDPrefix, host.Address), []byte(host.Did))
	store.Set(didIndexKey(types.PendingDIDAddrPrefix, host.Did), []byte(host.Address))
	return nil
}
