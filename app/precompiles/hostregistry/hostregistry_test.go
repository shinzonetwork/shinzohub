package hostregistry_test

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

	"github.com/shinzonetwork/shinzohub/app/precompiles/hostregistry"
	hostkeeper "github.com/shinzonetwork/shinzohub/x/host/keeper"
	hosttypes "github.com/shinzonetwork/shinzohub/x/host/types"
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

type mockSourcehubKeeper struct {
	err      error
	checkErr error
	called   bool
}

func (m *mockSourcehubKeeper) SendICASetRelationship(_ sdk.Context, _ string, _ string, _ string) (uint64, string, string, error) {
	m.called = true
	return 0, "", "", m.err
}

func (m *mockSourcehubKeeper) CheckICAReady(_ sdk.Context) error {
	return m.checkErr
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
	cdc           codec.Codec
}

func (s *PrecompileTestSuite) simulateHostAck(callerAddr []byte) {
	did, found := s.hostKeeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	s.Require().True(found, "pending host did not land in state")
	meta := &sourcehubtypes.SetRelationshipMeta{Did: string(did), Group: "host"}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	cb := hostkeeper.NewAckCallback(s.hostKeeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)
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
	s.cdc = cdc

	p, err := hostregistry.NewPrecompile(10000, s.hostKeeper, s.mockSourcehub)
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
		common.HexToAddress(hostregistry.PrecompileAddress),
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

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, nodeSig, message, "192.168.1.1:8080", "https://192.168.1.1:8443/api/v0/graphql"}

	bz, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().NoError(err)
	s.Require().Nil(bz)

	s.Require().False(s.hostKeeper.IsRegisteredHost(s.ctx, caller.Bytes()))
	s.simulateHostAck(caller.Bytes())
	s.Require().True(s.hostKeeper.IsRegisteredHost(s.ctx, caller.Bytes()))
	s.Require().Len(s.stateDB.logs, 1)
}

func (s *PrecompileTestSuite) TestRegister_ICANotReady() {
	s.mockSourcehub.checkErr = fmt.Errorf("no active ICA channel for portID X on connection Y")

	message := []byte("ica-not-ready-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0xaabbccddaabbccddaabbccddaabbccddaabbccdd")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, nodeSig, message, "192.168.1.1:8080", "https://192.168.1.1:8443/api/v0/graphql"}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "no active ICA channel")

	s.Require().Empty(s.stateDB.logs)
	s.Require().False(s.mockSourcehub.called)
	s.Require().False(s.hostKeeper.IsRegisteredHost(s.ctx, caller.Bytes()))
}

func (s *PrecompileTestSuite) TestRegister_EmptyArgs() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)
	method := s.precompile.ABI.Methods["register"]

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte{}, []byte("sig"), []byte("msg"), "conn",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid nodeIdentityKeyPubkey")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte("node"), []byte{}, []byte("msg"), "conn",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid nodeIdentityKeySignature")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte("node"), []byte("sig"), []byte{}, "conn",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid message")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte("node"), []byte("sig"), []byte("msg"), "",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid connectionString")
}

func (s *PrecompileTestSuite) TestRegister_ICAFailure() {
	s.mockSourcehub.err = fmt.Errorf("ICA not ready")

	message := []byte("test-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddd")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, nodeSig, message, "192.168.1.1:8080", "https://192.168.1.1:8443/api/v0/graphql"}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "ICA not ready")
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
