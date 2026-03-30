package hostregistry_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	dbm "github.com/cosmos/cosmos-db"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/shinzonetwork/shinzohub/app/precompiles/hostregistry"
	hostkeeper "github.com/shinzonetwork/shinzohub/x/host/keeper"
	hosttypes "github.com/shinzonetwork/shinzohub/x/host/types"
)

type mockSourcehubKeeper struct {
	err error
}

func (m *mockSourcehubKeeper) SendICASetRelationship(_ sdk.Context, _ string, _ string) error {
	return m.err
}

type mockStateDB struct {
	vm.StateDB
	logs []*gethtypes.Log
}

func (m *mockStateDB) AddLog(log *gethtypes.Log) {
	m.logs = append(m.logs, log)
}

type PrecompileTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	precompile    *hostregistry.Precompile
	hostKeeper    hostkeeper.Keeper
	mockSourcehub *mockSourcehubKeeper
	stateDB       *mockStateDB
}

func (s *PrecompileTestSuite) SetupTest() {
	s.mockSourcehub = &mockSourcehubKeeper{}

	storeKey := storetypes.NewKVStoreKey(hosttypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	storeService := runtime.NewKVStoreService(storeKey)

	s.hostKeeper = hostkeeper.NewKeeper(cdc, storeService, s.mockSourcehub, "authority")
	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, cosmoslog.NewNopLogger())
	s.stateDB = &mockStateDB{}

	p, err := hostregistry.NewPrecompile(10000, s.hostKeeper)
	require.NoError(s.T(), err)
	s.precompile = p
}

func makeContract(caller common.Address) *vm.Contract {
	return vm.NewContract(
		caller,
		common.HexToAddress(hostregistry.PrecompileAddress),
		uint256.NewInt(0),
		1000000,
		nil,
	)
}

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) TestAddress() {
	s.Require().Equal(
		common.HexToAddress("0x0000000000000000000000000000000000000211"),
		s.precompile.Address(),
	)
}

func (s *PrecompileTestSuite) TestRequiredGas() {
	s.Require().Equal(uint64(10000), s.precompile.RequiredGas(nil))
}

func (s *PrecompileTestSuite) TestIsTransaction() {
	registerMethod, ok := s.precompile.ABI.Methods["register"]
	s.Require().True(ok)
	s.Require().True(s.precompile.IsTransaction(&registerMethod))

	isRegisteredMethod, ok := s.precompile.ABI.Methods["isRegistered"]
	s.Require().True(ok)
	s.Require().False(s.precompile.IsTransaction(&isRegisteredMethod))

	getDidMethod, ok := s.precompile.ABI.Methods["getDid"]
	s.Require().True(ok)
	s.Require().False(s.precompile.IsTransaction(&getDidMethod))

	getCSMethod, ok := s.precompile.ABI.Methods["getConnectionString"]
	s.Require().True(ok)
	s.Require().False(s.precompile.IsTransaction(&getCSMethod))
}

func (s *PrecompileTestSuite) TestRegister_Success() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{"192.168.1.1:8080"}

	bz, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().NoError(err)
	s.Require().Nil(bz)

	s.Require().True(s.hostKeeper.IsRegisteredHost(s.ctx, caller.Bytes()))
	s.Require().Len(s.stateDB.logs, 1)

	events := s.ctx.EventManager().Events()
	found := false
	for _, e := range events {
		if e.Type == "HostRegistered" {
			found = true
		}
	}
	s.Require().True(found)
}

func (s *PrecompileTestSuite) TestRegister_EmptyArgs() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)
	method := s.precompile.ABI.Methods["register"]

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{""})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid connectionString")
}

func (s *PrecompileTestSuite) TestIsRegistered_False() {
	method := s.precompile.ABI.Methods["isRegistered"]
	addr := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	bz, err := s.precompile.IsRegistered(s.ctx, &method, []interface{}{addr})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal(false, out[0].(bool))
}

func (s *PrecompileTestSuite) TestIsRegistered_True() {
	caller := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	_, err := s.hostKeeper.RegisterHost(s.ctx, "192.168.1.1:8080", caller.Bytes())
	s.Require().NoError(err)

	method := s.precompile.ABI.Methods["isRegistered"]
	bz, err := s.precompile.IsRegistered(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal(true, out[0].(bool))
}

func (s *PrecompileTestSuite) TestGetConnectionString_NotRegistered() {
	method := s.precompile.ABI.Methods["getConnectionString"]
	addr := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	bz, err := s.precompile.GetConnectionString(s.ctx, &method, []interface{}{addr})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Empty(out[0].(string))
}

func (s *PrecompileTestSuite) TestGetConnectionString_Registered() {
	caller := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")

	_, err := s.hostKeeper.RegisterHost(s.ctx, "192.168.1.1:8080", caller.Bytes())
	s.Require().NoError(err)

	method := s.precompile.ABI.Methods["getConnectionString"]
	bz, err := s.precompile.GetConnectionString(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal("192.168.1.1:8080", out[0].(string))
}

func (s *PrecompileTestSuite) TestHandleMethod_Dispatch() {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	contract := makeContract(caller)

	isRegMethod := s.precompile.ABI.Methods["isRegistered"]
	bz, err := s.precompile.HandleMethod(s.ctx, contract, s.stateDB, &isRegMethod, []interface{}{caller})
	s.Require().NoError(err)

	out, err := isRegMethod.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal(false, out[0].(bool))
}

func (s *PrecompileTestSuite) TestHandleMethod_UnknownMethod() {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	contract := makeContract(caller)

	fakeMethod := s.precompile.ABI.Methods["isRegistered"]
	fakeMethod.Name = "nonExistentMethod"

	_, err := s.precompile.HandleMethod(s.ctx, contract, s.stateDB, &fakeMethod, nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nonExistentMethod")
}

func (s *PrecompileTestSuite) TestRegister_ICAFailure() {
	s.mockSourcehub.err = fmt.Errorf("ICA not ready")

	caller := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddd")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{"192.168.1.1:8080"}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "ICA not ready")
}
