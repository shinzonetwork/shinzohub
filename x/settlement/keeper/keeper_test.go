package keeper_test

import (
	"context"
	"testing"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	settlementkeeper "github.com/shinzonetwork/shinzohub/x/settlement/keeper"
	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

type bankMove struct {
	kind  string
	from  string
	to    string
	coins sdk.Coins
}

type mockBankKeeper struct {
	moves          []bankMove
	failNextMint   bool
	failNextSend   bool
}

func (m *mockBankKeeper) MintCoins(_ context.Context, mod string, amt sdk.Coins) error {
	if m.failNextMint {
		m.failNextMint = false
		return errMock
	}
	m.moves = append(m.moves, bankMove{kind: "mint", to: mod, coins: amt})
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, mod string, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "burn", from: mod, coins: amt})
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, mod string, to sdk.AccAddress, amt sdk.Coins) error {
	if m.failNextSend {
		m.failNextSend = false
		return errMock
	}
	m.moves = append(m.moves, bankMove{kind: "out", from: mod, to: to.String(), coins: amt})
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, from sdk.AccAddress, mod string, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "in", from: from.String(), to: mod, coins: amt})
	return nil
}

var errMock = &mockErr{msg: "mock failure"}

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }

type fixture struct {
	t      *testing.T
	ctx    sdk.Context
	keeper settlementkeeper.Keeper
	bank   *mockBankKeeper
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bank := &mockBankKeeper{}

	k := settlementkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		bank,
		"authority",
	)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, cosmoslog.NewNopLogger())

	return &fixture{t: t, ctx: ctx, keeper: k, bank: bank}
}

func addr(b byte) sdk.AccAddress {
	out := make([]byte, 20)
	for i := range out {
		out[i] = b
	}
	return out
}

func TestCredit_AddsToBalance(t *testing.T) {
	f := newFixture(t)
	a := addr(1)

	require.NoError(t, f.keeper.Credit(f.ctx, a, math.NewInt(100)))

	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, a))
	require.Empty(t, f.bank.moves, "credit must not move tokens, only update the ledger")
}

func TestCredit_Accumulates(t *testing.T) {
	f := newFixture(t)
	target := addr(9)

	require.NoError(t, f.keeper.Credit(f.ctx, target, math.NewInt(50)))
	require.NoError(t, f.keeper.Credit(f.ctx, target, math.NewInt(30)))

	require.Equal(t, math.NewInt(80), f.keeper.GetBalance(f.ctx, target))
}

func TestCredit_RejectsEmptyRecipient(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Credit(f.ctx, sdk.AccAddress{}, math.NewInt(100))
	require.ErrorContains(t, err, "recipient is required")
}

func TestCredit_RejectsNonPositiveAmount(t *testing.T) {
	f := newFixture(t)

	require.ErrorContains(t, f.keeper.Credit(f.ctx, addr(1), math.ZeroInt()), "positive")
	require.ErrorContains(t, f.keeper.Credit(f.ctx, addr(1), math.NewInt(-1)), "positive")
}

func TestDebit_DeductsFromBalance(t *testing.T) {
	f := newFixture(t)
	target := addr(9)
	require.NoError(t, f.keeper.Credit(f.ctx, target, math.NewInt(500)))

	require.NoError(t, f.keeper.Debit(f.ctx, target, math.NewInt(200)))
	require.Equal(t, math.NewInt(300), f.keeper.GetBalance(f.ctx, target))
}

func TestDebit_RejectsInsufficient(t *testing.T) {
	f := newFixture(t)
	target := addr(9)
	require.NoError(t, f.keeper.Credit(f.ctx, target, math.NewInt(50)))

	err := f.keeper.Debit(f.ctx, target, math.NewInt(100))
	require.ErrorContains(t, err, "insufficient")
	require.Equal(t, math.NewInt(50), f.keeper.GetBalance(f.ctx, target), "failed debit must not change balance")
}

func TestDebit_RejectsUnknownAddress(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Debit(f.ctx, addr(7), math.NewInt(1))
	require.ErrorContains(t, err, "no settlement balance")
}

func TestDebit_RejectsEmptyHolder(t *testing.T) {
	f := newFixture(t)
	err := f.keeper.Debit(f.ctx, sdk.AccAddress{}, math.NewInt(1))
	require.ErrorContains(t, err, "holder is required")
}

func TestDebit_RejectsNonPositiveAmount(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, f.keeper.Credit(f.ctx, addr(1), math.NewInt(10)))

	require.ErrorContains(t, f.keeper.Debit(f.ctx, addr(1), math.ZeroInt()), "positive")
	require.ErrorContains(t, f.keeper.Debit(f.ctx, addr(1), math.NewInt(-5)), "positive")
}

func TestClaim_MintsAndTransfers(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	require.NoError(t, f.keeper.Credit(f.ctx, claimer, math.NewInt(1_000_000)))

	require.NoError(t, f.keeper.Claim(f.ctx, claimer, math.NewInt(750_000)))

	require.Equal(t, math.NewInt(250_000), f.keeper.GetBalance(f.ctx, claimer),
		"pending balance should decrease by claim amount")

	require.Len(t, f.bank.moves, 2, "claim must mint and then transfer")

	mint := f.bank.moves[0]
	require.Equal(t, "mint", mint.kind)
	require.Equal(t, types.ModuleName, mint.to)
	require.Equal(t, types.SettlementDenom, mint.coins[0].Denom)
	require.Equal(t, math.NewInt(750_000), mint.coins[0].Amount)

	send := f.bank.moves[1]
	require.Equal(t, "out", send.kind)
	require.Equal(t, types.ModuleName, send.from)
	require.Equal(t, claimer.String(), send.to)
	require.Equal(t, types.SettlementDenom, send.coins[0].Denom)
	require.Equal(t, math.NewInt(750_000), send.coins[0].Amount)
}

func TestClaim_RejectsInsufficient(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	require.NoError(t, f.keeper.Credit(f.ctx, claimer, math.NewInt(40)))

	err := f.keeper.Claim(f.ctx, claimer, math.NewInt(100))
	require.ErrorContains(t, err, "insufficient")
	require.Equal(t, math.NewInt(40), f.keeper.GetBalance(f.ctx, claimer),
		"failed claim must not change pending balance")
	require.Empty(t, f.bank.moves, "failed claim must not mint or transfer")
}

func TestClaim_RejectsUnknownAddress(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Claim(f.ctx, addr(7), math.NewInt(1))
	require.ErrorContains(t, err, "no settlement balance")
	require.Empty(t, f.bank.moves)
}

func TestClaim_RejectsEmptyClaimer(t *testing.T) {
	f := newFixture(t)
	err := f.keeper.Claim(f.ctx, sdk.AccAddress{}, math.NewInt(1))
	require.ErrorContains(t, err, "claimer is required")
}

func TestClaim_RejectsNonPositiveAmount(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, f.keeper.Credit(f.ctx, addr(1), math.NewInt(100)))

	require.ErrorContains(t, f.keeper.Claim(f.ctx, addr(1), math.ZeroInt()), "positive")
	require.ErrorContains(t, f.keeper.Claim(f.ctx, addr(1), math.NewInt(-1)), "positive")
}

func TestClaim_MintFailureBubbles(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	require.NoError(t, f.keeper.Credit(f.ctx, claimer, math.NewInt(100)))
	f.bank.failNextMint = true

	err := f.keeper.Claim(f.ctx, claimer, math.NewInt(100))
	require.ErrorContains(t, err, "mint")
	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, claimer),
		"failed mint must leave pending balance untouched")
}

func TestClaim_TransferFailureBubbles(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	require.NoError(t, f.keeper.Credit(f.ctx, claimer, math.NewInt(100)))
	f.bank.failNextSend = true

	err := f.keeper.Claim(f.ctx, claimer, math.NewInt(100))
	require.ErrorContains(t, err, "transfer")
	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, claimer),
		"failed transfer must leave pending balance untouched")
}

func TestGetBalance_ZeroForUnknown(t *testing.T) {
	f := newFixture(t)
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, addr(42)))
}

func TestGetEntry_ReturnsFalseForUnknown(t *testing.T) {
	f := newFixture(t)

	_, found := f.keeper.GetEntry(f.ctx, addr(42))
	require.False(t, found)
}

func TestGetEntry_ReturnsTrueAfterCredit(t *testing.T) {
	f := newFixture(t)
	a := addr(5)
	require.NoError(t, f.keeper.Credit(f.ctx, a, math.NewInt(123)))

	entry, found := f.keeper.GetEntry(f.ctx, a)
	require.True(t, found)
	require.Equal(t, a.String(), entry.Address)
	require.Equal(t, "123", entry.Amount)
}

func TestGenesis_RoundTrip(t *testing.T) {
	src := newFixture(t)
	require.NoError(t, src.keeper.Credit(src.ctx, addr(10), math.NewInt(100)))
	require.NoError(t, src.keeper.Credit(src.ctx, addr(20), math.NewInt(250)))

	exported := src.keeper.ExportGenesis(src.ctx)

	dst := newFixture(t)
	dst.keeper.InitGenesis(dst.ctx, *exported)

	require.Equal(t, math.NewInt(100), dst.keeper.GetBalance(dst.ctx, addr(10)))
	require.Equal(t, math.NewInt(250), dst.keeper.GetBalance(dst.ctx, addr(20)))
}

func TestEvents_EmittedOnEachOperation(t *testing.T) {
	f := newFixture(t)
	a := addr(1)

	require.NoError(t, f.keeper.Credit(f.ctx, a, math.NewInt(100)))
	require.NoError(t, f.keeper.Debit(f.ctx, a, math.NewInt(20)))
	require.NoError(t, f.keeper.Claim(f.ctx, a, math.NewInt(30)))

	events := f.ctx.EventManager().Events()
	var seenCredit, seenDebit, seenClaim bool
	for _, e := range events {
		switch e.Type {
		case types.EventTypeCredited:
			seenCredit = true
		case types.EventTypeDebited:
			seenDebit = true
		case types.EventTypeClaimed:
			seenClaim = true
		}
	}
	require.True(t, seenCredit, "credit event missing")
	require.True(t, seenDebit, "debit event missing")
	require.True(t, seenClaim, "claim event missing")
}
