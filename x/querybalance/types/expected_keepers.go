package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper is the subset of the bank keeper this module uses for fund
// custody — sending funder NZO into the module account on Fund and back out
// during settlement (the settlement-time outflow lives on the pool keeper's
// side, not here).
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
}

// StakingKeeper is used purely to look up the chain's bond denom so we don't
// have to hardcode it in this module.
type StakingKeeper interface {
	BondDenom(ctx context.Context) (string, error)
}
