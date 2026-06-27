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

	"github.com/shinzonetwork/shinzohub/x/settlement/types"
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

func (k Keeper) Credit(ctx sdk.Context, recipient sdk.AccAddress, amount math.Int) error {
	if recipient.Empty() {
		return fmt.Errorf("recipient is required")
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}

	sb := k.getEntry(ctx, recipient)
	prev := parseAmount(sb.Amount)
	sb.Amount = prev.Add(amount).String()
	k.setEntry(ctx, sb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCredited,
		sdk.NewAttribute(types.AttrKeyAddress, recipient.String()),
		sdk.NewAttribute(types.AttrKeyAmount, amount.String()),
	))

	return nil
}

func (k Keeper) Debit(ctx sdk.Context, holder sdk.AccAddress, amount math.Int) error {
	if holder.Empty() {
		return fmt.Errorf("holder is required")
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}

	sb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return fmt.Errorf("no settlement balance for address %s", holder.String())
	}

	balance := parseAmount(sb.Amount)
	if balance.LT(amount) {
		return fmt.Errorf("insufficient settlement balance for address %s: have %s, want %s",
			holder.String(), balance.String(), amount.String())
	}

	sb.Amount = balance.Sub(amount).String()
	k.setEntry(ctx, sb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeDebited,
		sdk.NewAttribute(types.AttrKeyAddress, holder.String()),
		sdk.NewAttribute(types.AttrKeyAmount, amount.String()),
	))

	return nil
}

func (k Keeper) Claim(ctx sdk.Context, claimer sdk.AccAddress, amount math.Int) error {
	if claimer.Empty() {
		return fmt.Errorf("claimer is required")
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}

	sb, found := k.getEntryIfExists(ctx, claimer)
	if !found {
		return fmt.Errorf("no settlement balance for address %s", claimer.String())
	}

	balance := parseAmount(sb.Amount)
	if balance.LT(amount) {
		return fmt.Errorf("insufficient settlement balance for address %s: have %s, want %s",
			claimer.String(), balance.String(), amount.String())
	}

	coins := sdk.NewCoins(sdk.NewCoin(types.SettlementDenom, amount))

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return fmt.Errorf("mint %s: %w", types.SettlementDenom, err)
	}
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, claimer, coins); err != nil {
		return fmt.Errorf("transfer %s to claimer: %w", types.SettlementDenom, err)
	}

	sb.Amount = balance.Sub(amount).String()
	k.setEntry(ctx, sb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeClaimed,
		sdk.NewAttribute(types.AttrKeyAddress, claimer.String()),
		sdk.NewAttribute(types.AttrKeyAmount, amount.String()),
	))

	return nil
}

func (k Keeper) GetBalance(ctx sdk.Context, holder sdk.AccAddress) math.Int {
	sb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return math.ZeroInt()
	}
	return parseAmount(sb.Amount)
}

func (k Keeper) GetEntry(ctx sdk.Context, holder sdk.AccAddress) (types.SettlementBalance, bool) {
	return k.getEntryIfExists(ctx, holder)
}

func (k Keeper) getEntry(ctx sdk.Context, holder sdk.AccAddress) types.SettlementBalance {
	sb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return types.SettlementBalance{Address: holder.String(), Amount: "0"}
	}
	return sb
}

func (k Keeper) getEntryIfExists(ctx sdk.Context, holder sdk.AccAddress) (types.SettlementBalance, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(balanceKey(holder))
	if len(bz) == 0 {
		return types.SettlementBalance{}, false
	}
	var sb types.SettlementBalance
	if err := k.cdc.Unmarshal(bz, &sb); err != nil {
		return types.SettlementBalance{}, false
	}
	return sb, true
}

func (k Keeper) setEntry(ctx sdk.Context, sb types.SettlementBalance) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&sb)
	if err != nil {
		panic(err)
	}
	store.Set([]byte(types.BalancePrefix+sb.Address), bz)
}

func balanceKey(holder sdk.AccAddress) []byte {
	return []byte(types.BalancePrefix + holder.String())
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
	for _, sb := range gs.Balances {
		k.setEntry(ctx, sb)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	balanceStore := prefix.NewStore(store, []byte(types.BalancePrefix))

	var balances []types.SettlementBalance
	it := balanceStore.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var sb types.SettlementBalance
		if err := k.cdc.Unmarshal(it.Value(), &sb); err != nil {
			panic(err)
		}
		balances = append(balances, sb)
	}

	return &types.GenesisState{Balances: balances}
}
