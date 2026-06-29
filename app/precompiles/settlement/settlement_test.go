package settlement_test

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

	settlementprecompile "github.com/shinzonetwork/shinzohub/app/precompiles/settlement"
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
	moves []bankMove
}

func (m *mockBankKeeper) MintCoins(_ context.Context, mod string, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "mint", to: mod, coins: amt})
	return nil
}
func (m *mockBankKeeper) BurnCoins(_ context.Context, mod string, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "burn", from: mod, coins: amt})
	return nil
}
func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, mod string, to sdk.AccAddress, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "out", from: mod, to: to.String(), coins: amt})
	return nil
}
func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, from sdk.AccAddress, mod string, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "in", from: from.String(), to: mod, coins: amt})
	return nil
}

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
	keeper     settlementkeeper.Keeper
	precompile *settlementprecompile.Precompile
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

	k := settlementkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		bank,
		"authority",
	)

	p, err := settlementprecompile.NewPrecompile(10_000, k)
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
		common.HexToAddress(settlementprecompile.PrecompileAddress),
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

func TestClaim_HappyPath(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x11)
	require.NoError(t, f.keeper.Credit(f.ctx, cosmosAddr(caller), math.NewInt(1_000_000)))

	method := f.precompile.ABI.Methods["claim"]
	contract := makeContract(caller)

	bz, err := f.precompile.Claim(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(750_000),
	})

	require.NoError(t, err)

	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	require.Len(t, out, 1)
	remaining := out[0].(*big.Int)
	require.Equal(t, big.NewInt(250_000).String(), remaining.String())

	require.Equal(t, math.NewInt(250_000), f.keeper.GetBalance(f.ctx, cosmosAddr(caller)))

	require.Len(t, f.bank.moves, 2)
	require.Equal(t, "mint", f.bank.moves[0].kind)
	require.Equal(t, "out", f.bank.moves[1].kind)
	require.Equal(t, cosmosAddr(caller).String(), f.bank.moves[1].to)
}

func TestClaim_RejectsAmountAboveBalance(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x22)
	require.NoError(t, f.keeper.Credit(f.ctx, cosmosAddr(caller), math.NewInt(100)))

	method := f.precompile.ABI.Methods["claim"]
	contract := makeContract(caller)

	_, err := f.precompile.Claim(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(500),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient settlement balance")
	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, cosmosAddr(caller)),
		"failed precompile claim must leave pending balance untouched")
	require.Empty(t, f.bank.moves, "failed precompile claim must not move tokens")
	require.Empty(t, f.stateDB.logs, "failed precompile claim must not emit event")
}

func TestClaim_RejectsCallerWithNoBalance(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x33)

	method := f.precompile.ABI.Methods["claim"]
	contract := makeContract(caller)

	_, err := f.precompile.Claim(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(1),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient settlement balance")
	require.Empty(t, f.bank.moves)
}

func TestClaim_RejectsZeroAmount(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x44)
	require.NoError(t, f.keeper.Credit(f.ctx, cosmosAddr(caller), math.NewInt(100)))

	method := f.precompile.ABI.Methods["claim"]
	contract := makeContract(caller)

	_, err := f.precompile.Claim(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(0),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "positive")
	require.Empty(t, f.bank.moves)
}

func TestClaim_RejectsNegativeAmount(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x55)
	require.NoError(t, f.keeper.Credit(f.ctx, cosmosAddr(caller), math.NewInt(100)))

	method := f.precompile.ABI.Methods["claim"]
	contract := makeContract(caller)

	_, err := f.precompile.Claim(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(-5),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "positive")
	require.Empty(t, f.bank.moves)
}

func TestClaim_EmitsClaimedEvent(t *testing.T) {
	f := newFixture(t)
	caller := evmAddr(0x66)
	require.NoError(t, f.keeper.Credit(f.ctx, cosmosAddr(caller), math.NewInt(1_000)))

	method := f.precompile.ABI.Methods["claim"]
	contract := makeContract(caller)

	_, err := f.precompile.Claim(f.ctx, contract, f.stateDB, &method, []interface{}{
		big.NewInt(400),
	})
	require.NoError(t, err)

	require.Len(t, f.stateDB.logs, 1)
	log := f.stateDB.logs[0]

	expectedTopic0 := crypto.Keccak256Hash([]byte("Claimed(address,uint256,uint256)"))
	require.Equal(t, expectedTopic0, log.Topics[0])

	expectedClaimer := common.BytesToHash(caller.Bytes())
	require.Equal(t, expectedClaimer, log.Topics[1], "claimer must be in indexed topic")

	require.Equal(t, common.HexToAddress(settlementprecompile.PrecompileAddress), log.Address)
}

func TestBalanceOf_ReturnsPendingBalance(t *testing.T) {
	f := newFixture(t)
	holder := evmAddr(0x77)
	require.NoError(t, f.keeper.Credit(f.ctx, cosmosAddr(holder), math.NewInt(1_234_567)))

	method := f.precompile.ABI.Methods["balanceOf"]
	bz, err := f.precompile.BalanceOf(f.ctx, &method, []interface{}{holder})
	require.NoError(t, err)

	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	balance := out[0].(*big.Int)
	require.Equal(t, big.NewInt(1_234_567).String(), balance.String())
}

func TestBalanceOf_ReturnsZeroForUnknown(t *testing.T) {
	f := newFixture(t)
	holder := evmAddr(0x88)

	method := f.precompile.ABI.Methods["balanceOf"]
	bz, err := f.precompile.BalanceOf(f.ctx, &method, []interface{}{holder})
	require.NoError(t, err)

	out, _ := method.Outputs.Unpack(bz)
	balance := out[0].(*big.Int)
	require.Equal(t, big.NewInt(0).String(), balance.String())
}

func TestBalanceOf_ReflectsClaim(t *testing.T) {
	f := newFixture(t)
	holder := evmAddr(0x99)
	require.NoError(t, f.keeper.Credit(f.ctx, cosmosAddr(holder), math.NewInt(500)))

	balMethod := f.precompile.ABI.Methods["balanceOf"]
	claimMethod := f.precompile.ABI.Methods["claim"]
	contract := makeContract(holder)

	bz, _ := f.precompile.BalanceOf(f.ctx, &balMethod, []interface{}{holder})
	before, _ := balMethod.Outputs.Unpack(bz)
	require.Equal(t, big.NewInt(500).String(), before[0].(*big.Int).String())

	_, err := f.precompile.Claim(f.ctx, contract, f.stateDB, &claimMethod, []interface{}{
		big.NewInt(200),
	})
	require.NoError(t, err)

	bz, _ = f.precompile.BalanceOf(f.ctx, &balMethod, []interface{}{holder})
	after, _ := balMethod.Outputs.Unpack(bz)
	require.Equal(t, big.NewInt(300).String(), after[0].(*big.Int).String(),
		"balanceOf must reflect the post-claim pending amount")
}

func TestPrecompile_IsTransactionFlags(t *testing.T) {
	f := newFixture(t)

	claimMethod := f.precompile.ABI.Methods["claim"]
	balMethod := f.precompile.ABI.Methods["balanceOf"]

	require.True(t, f.precompile.IsTransaction(&claimMethod), "claim must be tx")
	require.False(t, f.precompile.IsTransaction(&balMethod), "balanceOf must be view")
}

func TestPrecompile_AddressIsFixed(t *testing.T) {
	f := newFixture(t)
	require.Equal(t,
		common.HexToAddress(settlementprecompile.PrecompileAddress),
		f.precompile.Address(),
	)
}
