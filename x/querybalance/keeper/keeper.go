package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/querybalance/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService
	authority    string
	bankKeeper   types.BankKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) Fund(
	ctx sdk.Context,
	funder sdk.AccAddress,
	did string,
	amount sdk.Coins,
) error {
	if did == "" {
		return fmt.Errorf("did is required")
	}
	if !amount.IsValid() || amount.IsZero() {
		return fmt.Errorf("amount must be a positive coin")
	}
	if len(amount) != 1 {
		return fmt.Errorf("fund accepts a single coin denomination, got %d", len(amount))
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, funder, types.ModuleName, amount,
	); err != nil {
		return fmt.Errorf("transfer to module account: %w", err)
	}

	qb := k.getEntry(ctx, did)
	prev := parseAmount(qb.Amount)
	qb.Amount = prev.Add(amount[0].Amount).String()
	k.setEntry(ctx, qb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeFunded,
		sdk.NewAttribute(types.AttrKeyDID, did),
		sdk.NewAttribute(types.AttrKeyFunder, funder.String()),
		sdk.NewAttribute(types.AttrKeyAmount, amount[0].Amount.String()),
	))

	return nil
}

func (k Keeper) Debit(ctx sdk.Context, did string, amount math.Int) error {
	if did == "" {
		return fmt.Errorf("did is required")
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}

	qb, found := k.getEntryIfExists(ctx, did)
	if !found {
		return fmt.Errorf("no balance for did %s", did)
	}

	balance := parseAmount(qb.Amount)
	if balance.LT(amount) {
		return fmt.Errorf("insufficient balance for did %s: have %s, want %s",
			did, balance.String(), amount.String())
	}

	qb.Amount = balance.Sub(amount).String()
	k.setEntry(ctx, qb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeDebited,
		sdk.NewAttribute(types.AttrKeyDID, did),
		sdk.NewAttribute(types.AttrKeyAmount, amount.String()),
	))

	return nil
}

func (k Keeper) GetBalance(ctx sdk.Context, did string) math.Int {
	qb, found := k.getEntryIfExists(ctx, did)
	if !found {
		return math.ZeroInt()
	}
	return parseAmount(qb.Amount)
}

func (k Keeper) GetEntry(ctx sdk.Context, did string) (types.QueryBalance, bool) {
	return k.getEntryIfExists(ctx, did)
}

func (k Keeper) getEntry(ctx sdk.Context, did string) types.QueryBalance {
	qb, found := k.getEntryIfExists(ctx, did)
	if !found {
		return types.QueryBalance{Did: did, Amount: "0"}
	}
	return qb
}

func (k Keeper) getEntryIfExists(ctx sdk.Context, did string) (types.QueryBalance, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(balanceKey(did))
	if len(bz) == 0 {
		return types.QueryBalance{}, false
	}
	var qb types.QueryBalance
	if err := k.cdc.Unmarshal(bz, &qb); err != nil {
		return types.QueryBalance{}, false
	}
	return qb, true
}

func (k Keeper) setEntry(ctx sdk.Context, qb types.QueryBalance) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&qb)
	if err != nil {
		panic(err)
	}
	store.Set(balanceKey(qb.Did), bz)
}

func balanceKey(did string) []byte {
	return []byte(types.BalancePrefix + did)
}

func parseAmount(s string) math.Int {
	if s == "" {
		return math.ZeroInt()
	}
	v, ok := math.NewIntFromString(s)
	if !ok {
		return math.ZeroInt()
	}
	return v
}

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	for _, qb := range gs.Balances {
		k.setEntry(ctx, qb)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	balanceStore := prefix.NewStore(store, []byte(types.BalancePrefix))

	var balances []types.QueryBalance
	it := balanceStore.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var qb types.QueryBalance
		if err := k.cdc.Unmarshal(it.Value(), &qb); err != nil {
			panic(err)
		}
		balances = append(balances, qb)
	}

	return &types.GenesisState{Balances: balances}
}
