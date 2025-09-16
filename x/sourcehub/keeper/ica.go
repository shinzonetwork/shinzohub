package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (k Keeper) RegisterSourcehubICA(ctx sdk.Context) error {
	return k.IcaCtrlKeeper.RegisterInterchainAccount(
		ctx,
		k.GetControllerConnectionID(ctx),
		types.ModuleAddress.String(),
		k.GetICAMetadata(ctx),
		channeltypes.ORDERED,
	)
}
