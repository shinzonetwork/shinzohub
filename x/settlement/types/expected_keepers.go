package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

// HostKeeper resolves DIDs registered as hosts to their payout address.
type HostKeeper interface {
	GetAddressForDID(ctx sdk.Context, did string) (sdk.AccAddress, bool)
}

// IndexerKeeper resolves DIDs registered as indexers to their payout address.
type IndexerKeeper interface {
	GetAddressForDID(ctx sdk.Context, did string) (sdk.AccAddress, bool)
}

// QueryBalanceKeeper is the interface settlement uses to drain user query
// balances at epoch end. Debit must reject over-spend; settlement caps the
// requested amount at GetBalance to avoid errors (drain-to-zero policy).
type QueryBalanceKeeper interface {
	GetBalance(ctx sdk.Context, holder sdk.AccAddress) math.Int
	Debit(ctx sdk.Context, holder sdk.AccAddress, amount math.Int) error
}
