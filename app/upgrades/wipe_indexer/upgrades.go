// Package wipe_indexer contains the upgrade handler that resets the indexer
// module store after the validator-keyed schema refactor. Existing rows from
// the old schema are no longer decodable; existing clusters must re-attest
// every indexer under the new flow.
package wipe_indexer

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/shinzonetwork/shinzohub/app/upgrades"
	indexertypes "github.com/shinzonetwork/shinzohub/x/indexer/types"
)

const UpgradeName = "wipe_indexer"

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		store := sdkCtx.KVStore(ak.GetStoreKey(indexertypes.StoreKey))

		// Collect first, delete second — iterator can't be mutated while open.
		var keys [][]byte
		iter := store.Iterator(nil, nil)
		for ; iter.Valid(); iter.Next() {
			keys = append(keys, append([]byte{}, iter.Key()...))
		}
		iter.Close()
		for _, k := range keys {
			store.Delete(k)
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
