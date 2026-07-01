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

	qbkeeper "github.com/shinzonetwork/shinzohub/x/querybalance/keeper"
	"github.com/shinzonetwork/shinzohub/x/querybalance/types"
)

const testDenom = "anzo"

type bankMove struct {
	kind  string
	from  string
	to    string
	coins sdk.Coins
}

type mockBankKeeper struct {
	moves      []bankMove
	failNextIn bool
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, from sdk.AccAddress, mod string, amt sdk.Coins) error {
	if m.failNextIn {
		m.failNextIn = false
		return errMock
	}
	m.moves = append(m.moves, bankMove{kind: "in", from: from.String(), to: mod, coins: amt})
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, mod string, to sdk.AccAddress, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "out", from: mod, to: to.String(), coins: amt})
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, from, to string, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "modmod", from: from, to: to, coins: amt})
	return nil
}

var errMock = &mockErr{msg: "mock failure"}

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }

type fixture struct {
	t      *testing.T
	ctx    sdk.Context
	keeper qbkeeper.Keeper
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

	k := qbkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		bank,
		"authority",
	)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, cosmoslog.NewNopLogger())

	return &fixture{t: t, ctx: ctx, keeper: k, bank: bank}
}

func nzo(amount int64) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(amount)))
}

func addr(b byte) sdk.AccAddress {
	out := make([]byte, 20)
	for i := range out {
		out[i] = b
	}
	return out
}

func TestFund_Self(t *testing.T) {
	f := newFixture(t)
	a := addr(1)

	require.NoError(t, f.keeper.Fund(f.ctx, a, a, nzo(100)))

	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, a))
	require.Len(t, f.bank.moves, 1)
	require.Equal(t, "in", f.bank.moves[0].kind)
	require.Equal(t, a.String(), f.bank.moves[0].from)
}

func TestFund_Sponsor(t *testing.T) {
	f := newFixture(t)
	sponsor, recipient := addr(1), addr(2)

	require.NoError(t, f.keeper.Fund(f.ctx, sponsor, recipient, nzo(100)))

	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, recipient))
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, sponsor))
	require.Equal(t, sponsor.String(), f.bank.moves[0].from)
}

func TestFund_AccumulatesAcrossCalls(t *testing.T) {
	f := newFixture(t)
	target := addr(9)

	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), target, nzo(50)))
	require.NoError(t, f.keeper.Fund(f.ctx, addr(2), target, nzo(30)))

	require.Equal(t, math.NewInt(80), f.keeper.GetBalance(f.ctx, target))
}

func TestFund_RejectsEmptyFunder(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Fund(f.ctx, sdk.AccAddress{}, addr(1), nzo(100))
	require.ErrorContains(t, err, "funder is required")
}

func TestFund_RejectsEmptyRecipient(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Fund(f.ctx, addr(1), sdk.AccAddress{}, nzo(100))
	require.ErrorContains(t, err, "recipient is required")
}

func TestFund_RejectsZeroAmount(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Fund(f.ctx, addr(1), addr(2), sdk.Coins{})
	require.ErrorContains(t, err, "positive coin")
}

func TestFund_RejectsMultipleDenoms(t *testing.T) {
	f := newFixture(t)
	mixed := sdk.NewCoins(
		sdk.NewCoin(testDenom, math.NewInt(10)),
		sdk.NewCoin("other", math.NewInt(10)),
	)

	err := f.keeper.Fund(f.ctx, addr(1), addr(2), mixed)
	require.ErrorContains(t, err, "single coin denomination")
}

func TestFund_BankFailureBubbles(t *testing.T) {
	f := newFixture(t)
	f.bank.failNextIn = true

	err := f.keeper.Fund(f.ctx, addr(1), addr(2), nzo(100))
	require.ErrorContains(t, err, "transfer to module account")
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, addr(2)))
}

func TestDebit_DeductsFromBalance(t *testing.T) {
	f := newFixture(t)
	target := addr(9)
	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), target, nzo(500)))

	require.NoError(t, f.keeper.Debit(f.ctx, target, math.NewInt(200)))
	require.Equal(t, math.NewInt(300), f.keeper.GetBalance(f.ctx, target))
}

func TestDebit_RejectsInsufficient(t *testing.T) {
	f := newFixture(t)
	target := addr(9)
	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), target, nzo(50)))

	err := f.keeper.Debit(f.ctx, target, math.NewInt(100))
	require.ErrorContains(t, err, "insufficient balance")
}

func TestDebit_RejectsUnknownAddress(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Debit(f.ctx, addr(7), math.NewInt(1))
	require.ErrorContains(t, err, "no balance")
}

func TestGetBalance_ZeroForUnknown(t *testing.T) {
	f := newFixture(t)
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, addr(42)))
}

func TestGenesis_RoundTrip(t *testing.T) {
	src := newFixture(t)
	require.NoError(t, src.keeper.Fund(src.ctx, addr(1), addr(10), nzo(100)))
	require.NoError(t, src.keeper.Fund(src.ctx, addr(2), addr(20), nzo(250)))

	exported := src.keeper.ExportGenesis(src.ctx)

	dst := newFixture(t)
	dst.keeper.InitGenesis(dst.ctx, *exported)

	require.Equal(t, math.NewInt(100), dst.keeper.GetBalance(dst.ctx, addr(10)))
	require.Equal(t, math.NewInt(250), dst.keeper.GetBalance(dst.ctx, addr(20)))
}
