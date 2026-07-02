package querybalance_test

import (
	"context"
	"sync"
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
	"github.com/ethereum/go-ethereum/core/tracing"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/shinzonetwork/shinzohub/app"
	"github.com/shinzonetwork/shinzohub/app/precompiles/querybalance"
	qbkeeper "github.com/shinzonetwork/shinzohub/x/querybalance/keeper"
	qbtypes "github.com/shinzonetwork/shinzohub/x/querybalance/types"
)

// The precompile denominates msg.value using the EVM coin's extended denom, which
// is a process-global configured during app setup. Configure it once for the
// whole test binary.
var evmCoinOnce sync.Once

func configureEVMCoin(t *testing.T) {
	t.Helper()
	evmCoinOnce.Do(func() {
		require.NoError(t, app.EVMAppOptions(app.ChainID18Decimals))
	})
}

// mockBankKeeper records every module transfer so tests can assert *who* was
// charged. The escrow must come from the precompile account (which holds the
// EVM-parked msg.value), never from the funder — pulling from the funder again
// would double-charge them.
type mockBankKeeper struct {
	moves []bankMove
}

type bankMove struct {
	from  string
	to    string
	coins sdk.Coins
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, from sdk.AccAddress, mod string, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{from: from.String(), to: mod, coins: amt})
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, _, _ string, _ sdk.Coins) error {
	return nil
}

type mockStateDB struct {
	vm.StateDB
	logs   []*gethtypes.Log
	subbed []subCall
}

type subCall struct {
	addr   common.Address
	amount *uint256.Int
}

func (m *mockStateDB) AddLog(log *gethtypes.Log) {
	m.logs = append(m.logs, log)
}

func (m *mockStateDB) SubBalance(addr common.Address, amount *uint256.Int, _ tracing.BalanceChangeReason) uint256.Int {
	m.subbed = append(m.subbed, subCall{addr: addr, amount: amount})
	return uint256.Int{}
}

type fixture struct {
	ctx        sdk.Context
	keeper     qbkeeper.Keeper
	bank       *mockBankKeeper
	stateDB    *mockStateDB
	precompile *querybalance.Precompile
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	configureEVMCoin(t)

	storeKey := storetypes.NewKVStoreKey(qbtypes.StoreKey)
	db := dbm.NewMemDB()
	cms := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bank := &mockBankKeeper{}
	k := qbkeeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bank, "authority")

	p, err := querybalance.NewPrecompile(0, k)
	require.NoError(t, err)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, cosmoslog.NewNopLogger())
	return &fixture{ctx: ctx, keeper: k, bank: bank, stateDB: &mockStateDB{}, precompile: p}
}

func makeContract(caller common.Address, value *uint256.Int) *vm.Contract {
	return vm.NewContract(
		caller,
		common.HexToAddress(querybalance.PrecompileAddress),
		value,
		1_000_000,
		nil,
	)
}

// fund() escrows msg.value from the precompile account (not the caller) and
// credits the caller's own query balance.
func TestFund_EscrowsFromPrecompileNotCaller(t *testing.T) {
	f := newFixture(t)
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	precompileAddr := common.HexToAddress(querybalance.PrecompileAddress)

	method := f.precompile.ABI.Methods["fund"]
	contract := makeContract(caller, uint256.NewInt(100))

	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, nil)
	require.NoError(t, err)

	// Exactly one escrow, sourced from the precompile account — NOT the caller.
	require.Len(t, f.bank.moves, 1)
	require.Equal(t, sdk.AccAddress(precompileAddr.Bytes()).String(), f.bank.moves[0].from)
	require.NotEqual(t, sdk.AccAddress(caller.Bytes()).String(), f.bank.moves[0].from,
		"escrow must not be pulled from the caller (double-charge)")
	require.Equal(t, qbtypes.ModuleName, f.bank.moves[0].to)

	// Ledger credits the caller's balance with the full value.
	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, sdk.AccAddress(caller.Bytes())))

	// StateDB reconciled: the precompile's EVM-visible balance is decremented by
	// the escrowed value.
	require.Len(t, f.stateDB.subbed, 1)
	require.Equal(t, precompileAddr, f.stateDB.subbed[0].addr)
	require.Equal(t, uint256.NewInt(100), f.stateDB.subbed[0].amount)

	// A Funded log is emitted.
	require.Len(t, f.stateDB.logs, 1)
}

// fundFor(recipient) credits the recipient while still escrowing from the
// precompile account.
func TestFundFor_CreditsRecipient(t *testing.T) {
	f := newFixture(t)
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	precompileAddr := common.HexToAddress(querybalance.PrecompileAddress)

	method := f.precompile.ABI.Methods["fundFor"]
	contract := makeContract(caller, uint256.NewInt(250))

	_, err := f.precompile.FundFor(f.ctx, contract, f.stateDB, &method, []interface{}{recipient})
	require.NoError(t, err)

	require.Len(t, f.bank.moves, 1)
	require.Equal(t, sdk.AccAddress(precompileAddr.Bytes()).String(), f.bank.moves[0].from)

	require.Equal(t, math.NewInt(250), f.keeper.GetBalance(f.ctx, sdk.AccAddress(recipient.Bytes())))
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, sdk.AccAddress(caller.Bytes())),
		"funder must not be credited when funding for someone else")
}

// A zero-value call is rejected before any state changes.
func TestFund_RejectsZeroValue(t *testing.T) {
	f := newFixture(t)
	caller := common.HexToAddress("0x3333333333333333333333333333333333333333")

	method := f.precompile.ABI.Methods["fund"]
	contract := makeContract(caller, uint256.NewInt(0))

	_, err := f.precompile.Fund(f.ctx, contract, f.stateDB, &method, nil)
	require.ErrorContains(t, err, "non-zero")
	require.Empty(t, f.bank.moves)
	require.Empty(t, f.stateDB.subbed)
}

// fundFor rejects the zero recipient.
func TestFundFor_RejectsZeroRecipient(t *testing.T) {
	f := newFixture(t)
	caller := common.HexToAddress("0x4444444444444444444444444444444444444444")

	method := f.precompile.ABI.Methods["fundFor"]
	contract := makeContract(caller, uint256.NewInt(100))

	_, err := f.precompile.FundFor(f.ctx, contract, f.stateDB, &method, []interface{}{common.Address{}})
	require.ErrorContains(t, err, "invalid recipient")
	require.Empty(t, f.bank.moves)
}
