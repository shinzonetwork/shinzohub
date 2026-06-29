package querybalance_test

import (
	"context"
	"math/big"
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
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	querybalanceprecompile "github.com/shinzonetwork/shinzohub/app/precompiles/querybalance"
	qbkeeper "github.com/shinzonetwork/shinzohub/x/querybalance/keeper"
	"github.com/shinzonetwork/shinzohub/x/querybalance/types"
)

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

type mockStateDB struct {
	vm.StateDB
	logs []*gethtypes.Log
}

func (m *mockStateDB) AddLog(log *gethtypes.Log) {
	m.logs = append(m.logs, log)
}

type fixture struct {
	t          *testing.T
	ctx        sdk.Context
	keeper     qbkeeper.Keeper
	precompile *querybalanceprecompile.Precompile
	bank       *mockBankKeeper
	stateDB    *mockStateDB
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

	p, err := querybalanceprecompile.NewPrecompile(10_000, k)
	require.NoError(t, err)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, cosmoslog.NewNopLogger())

	return &fixture{
		t:          t,
		ctx:        ctx,
		keeper:     k,
		precompile: p,
		bank:       bank,
		stateDB:    &mockStateDB{},
	}
}

func makeContract(caller common.Address) *vm.Contract {
	return vm.NewContract(
		caller,
		common.HexToAddress(querybalanceprecompile.PrecompileAddress),
		uint256.NewInt(0),
		1_000_000,
		nil,
	)
}

func evmAddr(b byte) common.Address {
	var a common.Address
	for i := range a {
		a[i] = b
	}
	return a
}

func cosmosAddr(b common.Address) sdk.AccAddress {
	return sdk.AccAddress(b.Bytes())
}

// ─── fund ──────────────────────────────────────────────────────────────────

func TestFund_HappyPath(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x11)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fund"]
	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(750_000),
	})
	require.NoError(t, err)

	// Bank moved SHINUSD from caller to module
	require.Len(t, f.bank.moves, 1)
	require.Equal(t, "in", f.bank.moves[0].kind)
	require.Equal(t, cosmosAddr(caller).String(), f.bank.moves[0].from)
	require.Equal(t, types.ModuleName, f.bank.moves[0].to)
	require.Equal(t, types.QueryBalanceDenom, f.bank.moves[0].coins[0].Denom,
		"precompile must transfer SHINUSD, not bond denom")
	require.Equal(t, math.NewInt(750_000), f.bank.moves[0].coins[0].Amount)

	// Keeper ledger reflects the credit
	require.Equal(t, math.NewInt(750_000), f.keeper.GetBalance(f.ctx, cosmosAddr(caller)))
}

func TestFund_RejectsZeroAmount(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x22)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fund"]
	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(0),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "positive")
	require.Empty(t, f.bank.moves, "rejected fund must not touch bank")
}

func TestFund_RejectsNegativeAmount(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x33)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fund"]
	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(-10),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "positive")
}

func TestFund_RejectsNilAmount(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x44)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fund"]
	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, []interface{}{
		(*big.Int)(nil),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid amount")
}

func TestFund_BankFailureBubbles(t *testing.T) {
	f := newFixture(t)
	f.bank.failNextIn = true

	caller := evmAddr(0x55)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fund"]
	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(500),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "transfer to module account")
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, cosmosAddr(caller)),
		"failed bank move must NOT credit balance")
}

func TestFund_AccumulatesAcrossCalls(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x66)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fund"]

	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, []interface{}{big.NewInt(100)})
	require.NoError(t, err)
	_, err = f.precompile.Fund(f.ctx, contract, f.stateDB, &method, []interface{}{big.NewInt(250)})
	require.NoError(t, err)

	require.Equal(t, math.NewInt(350), f.keeper.GetBalance(f.ctx, cosmosAddr(caller)),
		"two funds from same caller accumulate")
}

func TestFund_EmitsFundedEvent(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x77)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fund"]
	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(123_456),
	})
	require.NoError(t, err)

	require.Len(t, f.stateDB.logs, 1)
	log := f.stateDB.logs[0]

	expectedTopic0 := crypto.Keccak256Hash([]byte("Funded(address,address,uint256)"))
	require.Equal(t, expectedTopic0, log.Topics[0])
	require.Equal(t, common.BytesToHash(caller.Bytes()), log.Topics[1], "funder is in topic[1]")
	require.Equal(t, common.BytesToHash(caller.Bytes()), log.Topics[2],
		"self-fund: recipient = funder in topic[2]")
}

// ─── fundFor ───────────────────────────────────────────────────────────────

func TestFundFor_CreditsRecipientNotCaller(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x88)
	recipient := evmAddr(0x99)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fundFor"]
	_, err := f.precompile.FundFor(f.ctx, contract, f.stateDB, &method, []interface{}{
		recipient,
		big.NewInt(500),
	})
	require.NoError(t, err)

	require.Equal(t, math.NewInt(500), f.keeper.GetBalance(f.ctx, cosmosAddr(recipient)),
		"recipient's balance must be credited")
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, cosmosAddr(caller)),
		"caller's own balance stays zero")

	require.Equal(t, cosmosAddr(caller).String(), f.bank.moves[0].from,
		"funds come from caller, not recipient")
}

func TestFundFor_RejectsZeroRecipient(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0xAA)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fundFor"]
	_, err := f.precompile.FundFor(f.ctx, contract, f.stateDB, &method, []interface{}{
		common.Address{},
		big.NewInt(100),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid recipient")
	require.Empty(t, f.bank.moves)
}

func TestFundFor_RejectsZeroAmount(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0xBB)
	recipient := evmAddr(0xCC)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fundFor"]
	_, err := f.precompile.FundFor(f.ctx, contract, f.stateDB, &method, []interface{}{
		recipient,
		big.NewInt(0),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "positive")
}

func TestFundFor_EmitsFundedEvent(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0xDD)
	recipient := evmAddr(0xEE)
	contract := makeContract(caller)

	method := f.precompile.ABI.Methods["fundFor"]
	_, err := f.precompile.FundFor(f.ctx, contract, f.stateDB, &method, []interface{}{
		recipient,
		big.NewInt(777),
	})
	require.NoError(t, err)

	require.Len(t, f.stateDB.logs, 1)
	log := f.stateDB.logs[0]
	require.Equal(t, common.BytesToHash(caller.Bytes()), log.Topics[1],
		"funder is caller")
	require.Equal(t, common.BytesToHash(recipient.Bytes()), log.Topics[2],
		"recipient is the addr passed in")
}

// ─── balanceOf ─────────────────────────────────────────────────────────────

func TestBalanceOf_ReturnsBalance(t *testing.T) {
	f := newFixture(t)
	holder := evmAddr(0x12)
	contract := makeContract(holder)

	fundMethod := f.precompile.ABI.Methods["fund"]
	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &fundMethod, []interface{}{
		big.NewInt(1_500),
	})
	require.NoError(t, err)

	balMethod := f.precompile.ABI.Methods["balanceOf"]
	bz, err := f.precompile.BalanceOf(f.ctx, &balMethod, []interface{}{holder})
	require.NoError(t, err)

	out, err := balMethod.Outputs.Unpack(bz)
	require.NoError(t, err)
	bal := out[0].(*big.Int)
	require.Equal(t, big.NewInt(1_500).String(), bal.String())
}

func TestBalanceOf_ReturnsZeroForUnknown(t *testing.T) {
	f := newFixture(t)
	holder := evmAddr(0x42)

	method := f.precompile.ABI.Methods["balanceOf"]
	bz, err := f.precompile.BalanceOf(f.ctx, &method, []interface{}{holder})
	require.NoError(t, err)

	out, _ := method.Outputs.Unpack(bz)
	bal := out[0].(*big.Int)
	require.Equal(t, big.NewInt(0).String(), bal.String())
}

// ─── precompile shape ─────────────────────────────────────────────────────

func TestPrecompile_IsTransactionFlags(t *testing.T) {
	f := newFixture(t)

	fundMethod := f.precompile.ABI.Methods["fund"]
	fundForMethod := f.precompile.ABI.Methods["fundFor"]
	balMethod := f.precompile.ABI.Methods["balanceOf"]

	require.True(t, f.precompile.IsTransaction(&fundMethod), "fund must be tx")
	require.True(t, f.precompile.IsTransaction(&fundForMethod), "fundFor must be tx")
	require.False(t, f.precompile.IsTransaction(&balMethod), "balanceOf must be view")
}

func TestPrecompile_AddressIsFixed(t *testing.T) {
	f := newFixture(t)
	require.Equal(t,
		common.HexToAddress(querybalanceprecompile.PrecompileAddress),
		f.precompile.Address(),
	)
}

func TestPrecompile_AbiIsNonPayable(t *testing.T) {
	f := newFixture(t)

	// After the refactor, neither fund nor fundFor should accept msg.value.
	// We verify via the ABI's StateMutability tag rather than trying to
	// construct an actual EVM call with msg.value > 0.
	require.Equal(t, "nonpayable", f.precompile.ABI.Methods["fund"].StateMutability)
	require.Equal(t, "nonpayable", f.precompile.ABI.Methods["fundFor"].StateMutability)
}
