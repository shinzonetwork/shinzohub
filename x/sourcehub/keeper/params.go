package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (k Keeper) GetParams(ctx sdk.Context) (p types.Params, e error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return types.Params{}, fmt.Errorf("failed to get admin: %w", err)
	}

	return params, nil
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	if err := k.Params.Set(ctx, params); err != nil {
		k.Logger(ctx).Error("failed to set params", "error", err)
		panic(err)
	}
}
