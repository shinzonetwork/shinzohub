package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := gs.Validate(); err != nil {
		panic(err)
	}

	// Set all ICA metadata fields
	k.SetControllerConnectionID(ctx, gs.ControllerConnectionId)
	k.SetHostConnectionID(ctx, gs.HostConnectionId)
	k.SetVersion(ctx, gs.Version)
	k.SetEncoding(ctx, gs.Encoding)
	k.SetTxType(ctx, gs.TxType)
}

// ExportGenesis returns the module's exported genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		ControllerConnectionId: k.GetControllerConnectionID(ctx),
		HostConnectionId:       k.GetHostConnectionID(ctx),
		Version:                k.GetVersion(ctx),
		Encoding:               k.GetEncoding(ctx),
		TxType:                 k.GetTxType(ctx),
	}
}
