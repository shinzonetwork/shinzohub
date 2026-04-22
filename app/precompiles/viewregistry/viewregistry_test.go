package viewregistry_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	dbm "github.com/cosmos/cosmos-db"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/shinzonetwork/shinzohub/app/precompiles/viewregistry"
	viewkeeper "github.com/shinzonetwork/shinzohub/x/view/keeper"
	viewtypes "github.com/shinzonetwork/shinzohub/x/view/types"
)

type mockHostKeeper struct{}

func (m *mockHostKeeper) IsRegisteredHost(_ sdk.Context, _ []byte) bool {
	return false
}

type mockSourcehubKeeper struct {
	err error
}

func (m *mockSourcehubKeeper) RegisterObject(_ sdk.Context, _ string, _ string) (uint64, string, string, error) {
	return 0, "", "", m.err
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
	ctx        sdk.Context
	precompile *viewregistry.Precompile
	viewKeeper viewkeeper.Keeper
	mockSH     *mockSourcehubKeeper
	stateDB    *mockStateDB
}

func (s *PrecompileTestSuite) SetupTest() {
	s.mockSH = &mockSourcehubKeeper{}
	s.stateDB = &mockStateDB{}

	storeKey := storetypes.NewKVStoreKey(viewtypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	storeService := runtime.NewKVStoreService(storeKey)

	s.viewKeeper = viewkeeper.NewKeeper(
		cdc,
		storeService,
		&mockHostKeeper{},
		s.mockSH,
		"authority",
	)

	p, err := viewregistry.NewPrecompile(50000, s.viewKeeper)
	require.NoError(s.T(), err)
	s.precompile = p

	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{Height: 42}, false, cosmoslog.NewNopLogger())
}

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) TestAddress() {
	s.Require().Equal(common.HexToAddress("0x0000000000000000000000000000000000000210"), s.precompile.Address())
}

func (s *PrecompileTestSuite) TestRequiredGas() {
	s.Require().Equal(uint64(50000), s.precompile.RequiredGas(nil))
}

func (s *PrecompileTestSuite) TestIsTransaction() {
	registerMethod := s.precompile.ABI.Methods["register"]
	s.Require().True(s.precompile.IsTransaction(&registerMethod))

	registerWithPricingMethod := s.precompile.ABI.Methods["registerWithPricing"]
	s.Require().True(s.precompile.IsTransaction(&registerWithPricingMethod))

	getViewMethod := s.precompile.ABI.Methods["getView"]
	s.Require().False(s.precompile.IsTransaction(&getViewMethod))
}

func (s *PrecompileTestSuite) TestGetView_NotFound() {
	method := s.precompile.ABI.Methods["getView"]
	addr := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	bz, err := s.precompile.GetView(s.ctx, &method, []interface{}{addr})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal("", out[0].(string))
}

func (s *PrecompileTestSuite) TestGetView_Found() {
	contractAddr := "0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF"
	err := s.viewKeeper.SetView(s.ctx, viewtypes.View{
		Name:            "TestView",
		Creator:         "cosmos1creator",
		ContractAddress: contractAddr,
	})
	s.Require().NoError(err)

	method := s.precompile.ABI.Methods["getView"]
	bz, err := s.precompile.GetView(s.ctx, &method, []interface{}{common.HexToAddress(contractAddr)})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal("cosmos1creator", out[0].(string))
}

func (s *PrecompileTestSuite) TestHandleMethod_GetView() {
	method := s.precompile.ABI.Methods["getView"]
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")

	bz, err := s.precompile.HandleMethod(s.ctx, nil, nil, s.stateDB, &method, []interface{}{addr})
	s.Require().NoError(err)
	s.Require().NotNil(bz)
}

func (s *PrecompileTestSuite) TestHandleMethod_UnknownMethod() {
	fakeMethod := s.precompile.ABI.Methods["getView"]
	fakeMethod.Name = "doesNotExist"

	_, err := s.precompile.HandleMethod(s.ctx, nil, nil, s.stateDB, &fakeMethod, nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "doesNotExist")
}

func (s *PrecompileTestSuite) TestRegister_InvalidDataType() {
	method := s.precompile.ABI.Methods["register"]

	_, err := s.precompile.Register(s.ctx, nil, nil, s.stateDB, &method, []interface{}{"not-bytes"})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid data")
}

func (s *PrecompileTestSuite) TestRegister_InvalidViewbundle() {
	method := s.precompile.ABI.Methods["register"]

	contract := vm.NewContract(
		common.HexToAddress("0xCALLER"),
		common.HexToAddress("0x210"),
		nil,
		1000000,
		nil,
	)

	_, err := s.precompile.Register(s.ctx, nil, contract, s.stateDB, &method, []interface{}{[]byte("garbage")})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "viewbundle")
}

func (s *PrecompileTestSuite) TestRegisterWithPricing_InvalidDataType() {
	method := s.precompile.ABI.Methods["registerWithPricing"]

	_, err := s.precompile.RegisterWithPricing(s.ctx, nil, nil, s.stateDB, &method, []interface{}{"not-bytes", common.Address{}})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid data")
}

func (s *PrecompileTestSuite) TestRegisterWithPricing_InvalidPricingType() {
	method := s.precompile.ABI.Methods["registerWithPricing"]

	_, err := s.precompile.RegisterWithPricing(s.ctx, nil, nil, s.stateDB, &method, []interface{}{[]byte("data"), "not-an-address"})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid pricing")
}

func (s *PrecompileTestSuite) TestGetView_MultipleViews() {
	for i := 0; i < 3; i++ {
		addr := fmt.Sprintf("0x%040d", i)
		_ = s.viewKeeper.SetView(s.ctx, viewtypes.View{
			ContractAddress: common.HexToAddress(addr).Hex(),
			Name:            fmt.Sprintf("V%d", i),
			Creator:         fmt.Sprintf("cosmos1creator%d", i),
		})
	}

	method := s.precompile.ABI.Methods["getView"]
	addr0 := common.HexToAddress(fmt.Sprintf("0x%040d", 0))
	bz, err := s.precompile.GetView(s.ctx, &method, []interface{}{addr0})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal("cosmos1creator0", out[0].(string))
}
