package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/admin/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService
	authority    string
	Params       collections.Item[types.Params]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	authority string,
) Keeper {
	_, err := sdk.AccAddressFromBech32(authority)
	if err != nil {
		panic(err)
	}

	sb := collections.NewSchemaBuilder(storeService)

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		Params:       collections.NewItem(sb, types.KeyPrefixParams, "params", codec.CollValue[types.Params](cdc)),
	}
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) IsAdmin(ctx sdk.Context, address string) bool {
	for _, admin := range k.GetAdmins(ctx) {
		if admin == address {
			return true
		}
	}
	return false
}

func (k Keeper) GetAdmins(ctx sdk.Context) []string {
	p, err := k.GetParams(ctx)
	if err != nil {
		return []string{k.authority}
	}

	if p.Admin == "" || p.Admin == k.authority {
		return []string{k.authority}
	}

	return []string{p.Admin, k.authority}
}

func (k Keeper) GetParams(ctx sdk.Context) (types.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return types.Params{}, fmt.Errorf("failed to get admin params: %w", err)
	}
	return params, nil
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	if err := k.Params.Set(ctx, params); err != nil {
		k.Logger(ctx).Error("failed to set params", "error", err)
		panic(err)
	}
}

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	k.SetParams(ctx, gs.Params)
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	p, _ := k.GetParams(ctx)
	return &types.GenesisState{Params: p}
}
