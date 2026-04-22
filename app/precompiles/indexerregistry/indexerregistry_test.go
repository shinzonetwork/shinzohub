package indexerregistry_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
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
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
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

func (m *mockSourcehubKeeper) SendICASetRelationship(_ sdk.Context, _ string, _ string, _ string) (uint64, string, string, error) {
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
	ctx           sdk.Context
	precompile    *indexerregistry.Precompile
	indexerKeeper indexerkeeper.Keeper
	mockAdmin     *mockAdminKeeper
	mockSourcehub *mockSourcehubKeeper
	stateDB       *mockStateDB
	cdc           codec.Codec
}

func (s *PrecompileTestSuite) simulateIndexerAck(callerAddr []byte) {
	did, found := s.indexerKeeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	s.Require().True(found, "pending indexer did not land in state")
	meta := &sourcehubtypes.SetRelationshipMeta{Did: string(did), Group: "indexer"}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	cb := indexerkeeper.NewAckCallback(s.indexerKeeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)
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
	s.cdc = cdc

	p, err := indexerregistry.NewPrecompile(10000, s.indexerKeeper)
	require.NoError(s.T(), err)
	s.precompile = p
}

func generateNodeIdentityKey(t *testing.T, message []byte) (pubkey, signature []byte) {
	privKey, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	h := sha256.Sum256(message)
	return privKey.PubKey().SerializeUncompressed(), ecdsa.Sign(privKey, h[:]).Serialize()
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

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) TestRegister_Success() {
	message := []byte("test-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
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
	args := []interface{}{nodePub, nodeSig, message, "192.168.1.1:8080", "ethereum", uint64(1)}

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
	message := []byte("test-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, nodeSig, message, "192.168.1.1:8080", "ethereum", uint64(1)}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "indexer not asserted")
}

func (s *PrecompileTestSuite) TestGetSourceChain_Registered() {
	message := []byte("nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddd")
	_, err := s.indexerKeeper.RegisterIndexer(s.ctx, nodePub, nodeSig, message, "192.168.1.1:8080", caller.Bytes(), "ethereum", 1)
	s.Require().NoError(err)
	s.simulateIndexerAck(caller.Bytes())

	method := s.precompile.ABI.Methods["getSourceChain"]
	bz, err := s.precompile.GetSourceChain(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)

	expected := crypto.Keccak256Hash([]byte("ethereum"))
	s.Require().Equal(expected, common.Hash(out[0].([32]byte)))
}

func (s *PrecompileTestSuite) TestRegister_ICAFailure() {
	s.mockSourcehub.err = fmt.Errorf("ICA not ready")

	message := []byte("test-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
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
	args := []interface{}{nodePub, nodeSig, message, "192.168.1.1:8080", "ethereum", uint64(1)}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "ICA not ready")
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
