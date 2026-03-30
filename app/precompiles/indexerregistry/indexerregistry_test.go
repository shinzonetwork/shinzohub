package indexerregistry_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/shinzonetwork/shinzohub/app/precompiles/indexerregistry"
	indexerkeeper "github.com/shinzonetwork/shinzohub/x/indexer/keeper"
	indexertypes "github.com/shinzonetwork/shinzohub/x/indexer/types"
)

type mockAdminKeeper struct {
	admins map[string]bool
}

func (m *mockAdminKeeper) IsAdmin(_ sdk.Context, address string) bool {
	return m.admins[address]
}

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
	precompile    *indexerregistry.Precompile
	indexerKeeper indexerkeeper.Keeper
	mockAdmin     *mockAdminKeeper
	mockSourcehub *mockSourcehubKeeper
	stateDB       *mockStateDB
}

func (s *PrecompileTestSuite) SetupTest() {
	s.mockAdmin = &mockAdminKeeper{admins: map[string]bool{}}
	s.mockSourcehub = &mockSourcehubKeeper{}

	storeKey := storetypes.NewKVStoreKey(indexertypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	storeService := runtime.NewKVStoreService(storeKey)

	s.indexerKeeper = indexerkeeper.NewKeeper(cdc, storeService, s.mockAdmin, s.mockSourcehub)
	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, cosmoslog.NewNopLogger())
	s.stateDB = &mockStateDB{}

	p, err := indexerregistry.NewPrecompile(10000, s.indexerKeeper)
	require.NoError(s.T(), err)
	s.precompile = p
}

func makeContract(caller common.Address) *vm.Contract {
	return vm.NewContract(
		caller,
		common.HexToAddress(indexerregistry.PrecompileAddress),
		uint256.NewInt(0),
		1000000,
		nil,
	)
}

func (s *PrecompileTestSuite) registerIndexer(caller common.Address, connStr, sourceChain string, sourceChainId uint64) []byte {
	did, err := s.indexerKeeper.RegisterIndexer(s.ctx, connStr, caller.Bytes(), sourceChain, sourceChainId)
	s.Require().NoError(err)
	return did
}

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) TestAddress() {
	s.Require().Equal(
		common.HexToAddress("0x0000000000000000000000000000000000000212"),
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

	isRegMethod, ok := s.precompile.ABI.Methods["isRegistered"]
	s.Require().True(ok)
	s.Require().False(s.precompile.IsTransaction(&isRegMethod))

	getDidMethod, ok := s.precompile.ABI.Methods["getDid"]
	s.Require().True(ok)
	s.Require().False(s.precompile.IsTransaction(&getDidMethod))

	getCSMethod, ok := s.precompile.ABI.Methods["getConnectionString"]
	s.Require().True(ok)
	s.Require().False(s.precompile.IsTransaction(&getCSMethod))

	getSCMethod, ok := s.precompile.ABI.Methods["getSourceChain"]
	s.Require().True(ok)
	s.Require().False(s.precompile.IsTransaction(&getSCMethod))
}

func (s *PrecompileTestSuite) TestRegister_Success() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)

	delegate := sdk.AccAddress(caller.Bytes()).String()
	_ = s.indexerKeeper.SetIndexerAssertion(s.ctx, indexertypes.IndexerAssertion{
		ConsensusPubKey: "pk",
		DelegateAddress: delegate,
		SourceChain:     "ethereum",
		SourceChainId:   1,
		AssertionId:     "a0",
	})

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{"192.168.1.1:8080", "ethereum", uint64(1)}

	bz, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().NoError(err)
	s.Require().Nil(bz)

	s.Require().Len(s.stateDB.logs, 1)

	events := s.ctx.EventManager().Events()
	found := false
	for _, e := range events {
		if e.Type == "IndexerRegistered" {
			found = true
		}
	}
	s.Require().True(found)
}

func (s *PrecompileTestSuite) TestRegister_NotAsserted() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{"192.168.1.1:8080", "ethereum", uint64(1)}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "indexer not asserted")
}

func (s *PrecompileTestSuite) TestRegister_EmptyArgs() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)
	method := s.precompile.ABI.Methods["register"]

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		"", "ethereum", uint64(1),
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid connectionString")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		"192.168.1.1:8080", "", uint64(1),
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid sourceChain")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		"192.168.1.1:8080", "ethereum", uint64(0),
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid sourceChainId")
}

func (s *PrecompileTestSuite) TestIsRegistered_True() {
	caller := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	s.registerIndexer(caller, "192.168.1.1:8080", "ethereum", 1)

	method := s.precompile.ABI.Methods["isRegistered"]
	bz, err := s.precompile.IsRegistered(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal(true, out[0].(bool))
}

func (s *PrecompileTestSuite) TestGetConnectionString_Registered() {
	caller := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")
	s.registerIndexer(caller, "192.168.1.1:8080", "ethereum", 1)

	method := s.precompile.ABI.Methods["getConnectionString"]
	bz, err := s.precompile.GetConnectionString(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal("192.168.1.1:8080", out[0].(string))
}

func (s *PrecompileTestSuite) TestGetSourceChain_Registered() {
	caller := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddd")
	s.registerIndexer(caller, "192.168.1.1:8080", "ethereum", 1)

	method := s.precompile.ABI.Methods["getSourceChain"]
	bz, err := s.precompile.GetSourceChain(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)

	expected := crypto.Keccak256Hash([]byte("ethereum"))
	s.Require().Equal(expected, common.Hash(out[0].([32]byte)))
}

func (s *PrecompileTestSuite) TestHandleMethod_UnknownMethod() {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	contract := makeContract(caller)

	fakeMethod := s.precompile.ABI.Methods["isRegistered"]
	fakeMethod.Name = "nonExistent"

	_, err := s.precompile.HandleMethod(s.ctx, contract, s.stateDB, &fakeMethod, nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nonExistent")
}

func (s *PrecompileTestSuite) TestRegister_ICAFailure() {
	s.mockSourcehub.err = fmt.Errorf("ICA not ready")

	caller := common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	contract := makeContract(caller)

	delegate := sdk.AccAddress(caller.Bytes()).String()
	_ = s.indexerKeeper.SetIndexerAssertion(s.ctx, indexertypes.IndexerAssertion{
		ConsensusPubKey: "pk",
		DelegateAddress: delegate,
		SourceChain:     "ethereum",
		SourceChainId:   1,
		AssertionId:     "a0",
	})

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{"192.168.1.1:8080", "ethereum", uint64(1)}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "ICA not ready")
}
