package keeper

import (
	_ "embed"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (k Keeper) SetPolicyId(ctx sdk.Context, txType string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.PolicyId), []byte(txType))
}

func (k Keeper) GetPolicyId(ctx sdk.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.PolicyId))
	if bz == nil {
		return ""
	}
	return string(bz)
}

//go:embed policy.yaml
var policy string
