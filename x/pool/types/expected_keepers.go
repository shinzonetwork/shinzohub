package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	viewtypes "github.com/shinzonetwork/shinzohub/x/view/types"
)

// ViewKeeper is the subset of the view keeper the pool module uses to
// validate that a referenced view exists before creating a pool for it.
type ViewKeeper interface {
	GetView(ctx sdk.Context, viewAddress string) (viewtypes.View, bool, error)
}

// BankKeeper is the subset of the bank keeper the pool module uses for
// bond custody (transferring NZO into and out of the pool module account).
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
}
