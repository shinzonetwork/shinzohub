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
	"github.com/cosmos/cosmos-sdk/types/query"

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

// Fund moves NZO (ushinzo) from funder's wallet into the querybalance
// module account and credits the recipient's query balance by amount. Denom
// is fixed to types.QueryBalanceDenom — the funder must already hold NZO
// (via settlement claim, bridge, or transfer).
func (k Keeper) Fund(
	ctx sdk.Context,
	funder sdk.AccAddress,
	recipient sdk.AccAddress,
	amount math.Int,
) error {
	if funder.Empty() {
		return fmt.Errorf("funder is required")
	}
	if recipient.Empty() {
		return fmt.Errorf("recipient is required")
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}

	coins := sdk.NewCoins(sdk.NewCoin(types.QueryBalanceDenom, amount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, funder, types.ModuleName, coins,
	); err != nil {
		return fmt.Errorf("transfer to module account: %w", err)
	}

	qb := k.getEntry(ctx, recipient)
	prev := parseAmount(qb.Amount)
	// SafeAdd returns an error instead of panicking on 256-bit overflow. A panic
	// here would escape the precompile's gas-error recovery and crash the node,
	// so surface it as a normal error that reverts the tx.
	sum, err := prev.SafeAdd(amount)
	if err != nil {
		return fmt.Errorf("credit %s: balance overflow: %w", recipient.String(), err)
	}
	qb.Amount = sum.String()
	k.setEntry(ctx, qb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeFunded,
		sdk.NewAttribute(types.AttrKeyFunder, funder.String()),
		sdk.NewAttribute(types.AttrKeyRecipient, recipient.String()),
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

	qb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return fmt.Errorf("no balance for address %s", holder.String())
	}

	balance := parseAmount(qb.Amount)
	if balance.LT(amount) {
		return fmt.Errorf("insufficient balance for address %s: have %s, want %s",
			holder.String(), balance.String(), amount.String())
	}

	qb.Amount = balance.Sub(amount).String()
	k.setEntry(ctx, qb)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeDebited,
		sdk.NewAttribute(types.AttrKeyAddress, holder.String()),
		sdk.NewAttribute(types.AttrKeyAmount, amount.String()),
	))

	return nil
}

func (k Keeper) GetBalance(ctx sdk.Context, holder sdk.AccAddress) math.Int {
	qb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return math.ZeroInt()
	}
	return parseAmount(qb.Amount)
}

func (k Keeper) GetEntry(ctx sdk.Context, holder sdk.AccAddress) (types.QueryBalance, bool) {
	return k.getEntryIfExists(ctx, holder)
}

func (k Keeper) getEntry(ctx sdk.Context, holder sdk.AccAddress) types.QueryBalance {
	qb, found := k.getEntryIfExists(ctx, holder)
	if !found {
		return types.QueryBalance{Address: holder.String(), Amount: "0"}
	}
	return qb
}

func (k Keeper) getEntryIfExists(ctx sdk.Context, holder sdk.AccAddress) (types.QueryBalance, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(balanceKey(holder))
	if len(bz) == 0 {
		return types.QueryBalance{}, false
	}
	var qb types.QueryBalance
	if err := k.cdc.Unmarshal(bz, &qb); err != nil {
		// A decode failure means the stored bytes are corrupt; surfacing it as
		// "not found" would silently zero out a real balance, so fail loudly.
		panic(fmt.Errorf("decode query balance for %s: %w", holder.String(), err))
	}
	return qb, true
}

func (k Keeper) setEntry(ctx sdk.Context, qb types.QueryBalance) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&qb)
	if err != nil {
		panic(err)
	}
	holder, err := sdk.AccAddressFromBech32(qb.Address)
	if err != nil {
		panic(fmt.Errorf("query balance has invalid address %q: %w", qb.Address, err))
	}
	store.Set(balanceKey(holder), bz)
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
		// Stored amounts are always written via math.Int.String(), so a
		// non-numeric value means the state is corrupt. Silently returning zero
		// would make the next Fund overwrite and permanently lose the real
		// balance, so fail loudly instead.
		panic(fmt.Errorf("corrupt query balance amount %q", s))
	}
	return v
}

func (k Keeper) GetAllBalances(
	ctx sdk.Context,
	pageReq *query.PageRequest,
) ([]types.QueryBalance, *query.PageResponse, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	balanceStore := prefix.NewStore(store, []byte(types.BalancePrefix))

	var balances []types.QueryBalance
	pageRes, err := query.Paginate(balanceStore, pageReq, func(_, value []byte) error {
		var qb types.QueryBalance
		if err := k.cdc.Unmarshal(value, &qb); err != nil {
			return err
		}
		balances = append(balances, qb)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return balances, pageRes, nil
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
