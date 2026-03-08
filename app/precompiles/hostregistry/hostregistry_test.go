package hostregistry_test

import (
	"crypto/ed25519"
	"crypto/rand"
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

func generatePeerKey(t *testing.T, message []byte) (pubkey, signature []byte) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sig := ed25519.Sign(priv, message)
	return []byte(pub), sig
}

func generateNodeIdentityKey(t *testing.T, message []byte) (pubkey, signature []byte) {
	privKey, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	pubkey = privKey.PubKey().SerializeUncompressed()
	h := sha256.Sum256(message)
	signature = ecdsa.Sign(privKey, h[:]).Serialize()
	return pubkey, signature
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

	getPidMethod, ok := s.precompile.ABI.Methods["getPid"]
	s.Require().True(ok)
	s.Require().False(s.precompile.IsTransaction(&getPidMethod))
}

func (s *PrecompileTestSuite) TestRegister_Success() {
	message := []byte("test-nonce")
	peerPub, peerSig := generatePeerKey(s.T(), message)
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{peerPub, peerSig, nodePub, nodeSig, message}

	bz, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().NoError(err)
	s.Require().Nil(bz)

	s.Require().True(s.hostKeeper.IsRegisteredHost(s.ctx, caller.Bytes()))
	s.Require().Len(s.stateDB.logs, 1)
	s.Require().Equal(common.HexToAddress(hostregistry.PrecompileAddress), s.stateDB.logs[0].Address)

	events := s.ctx.EventManager().Events()
	found := false
	for _, e := range events {
		if e.Type == "HostRegistered" {
			found = true
		}
	}
	s.Require().True(found)
}

func (s *PrecompileTestSuite) TestRegister_InvalidPeerKey() {
	message := []byte("test-nonce")
	_, peerSig := generatePeerKey(s.T(), []byte("wrong"))
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)

	peerPub := make([]byte, 32)
	rand.Read(peerPub)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{peerPub, peerSig, nodePub, nodeSig, message}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
}

func (s *PrecompileTestSuite) TestRegister_EmptyArgs() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	contract := makeContract(caller)
	method := s.precompile.ABI.Methods["register"]

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte{}, []byte("sig"), []byte("node"), []byte("nodesig"), []byte("msg"),
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid peerKeyPubkey")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte("peer"), []byte{}, []byte("node"), []byte("nodesig"), []byte("msg"),
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid peerKeySignature")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte("peer"), []byte("sig"), []byte{}, []byte("nodesig"), []byte("msg"),
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid nodeIdentityKeyPubkey")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte("peer"), []byte("sig"), []byte("node"), []byte{}, []byte("msg"),
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid nodeIdentityKeySignature")

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte("peer"), []byte("sig"), []byte("node"), []byte("nodesig"), []byte{},
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid message")
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
	message := []byte("test-nonce")
	peerPub, peerSig := generatePeerKey(s.T(), message)
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	_, _, err := s.hostKeeper.RegisterHost(s.ctx, peerPub, peerSig, nodePub, nodeSig, message, caller.Bytes())
	s.Require().NoError(err)

	method := s.precompile.ABI.Methods["isRegistered"]
	bz, err := s.precompile.IsRegistered(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal(true, out[0].(bool))
}

func (s *PrecompileTestSuite) TestGetDid_NotRegistered() {
	method := s.precompile.ABI.Methods["getDid"]
	addr := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	bz, err := s.precompile.GetDid(s.ctx, &method, []interface{}{addr})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Empty(out[0].([]byte))
}

func (s *PrecompileTestSuite) TestGetDid_Registered() {
	message := []byte("test-nonce")
	peerPub, peerSig := generatePeerKey(s.T(), message)
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	did, _, err := s.hostKeeper.RegisterHost(s.ctx, peerPub, peerSig, nodePub, nodeSig, message, caller.Bytes())
	s.Require().NoError(err)

	method := s.precompile.ABI.Methods["getDid"]
	bz, err := s.precompile.GetDid(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal(did, out[0].([]byte))
}

func (s *PrecompileTestSuite) TestGetPid_NotRegistered() {
	method := s.precompile.ABI.Methods["getPid"]
	addr := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	bz, err := s.precompile.GetPid(s.ctx, &method, []interface{}{addr})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Empty(out[0].([]byte))
}

func (s *PrecompileTestSuite) TestGetPid_Registered() {
	message := []byte("test-nonce")
	peerPub, peerSig := generatePeerKey(s.T(), message)
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")

	_, pid, err := s.hostKeeper.RegisterHost(s.ctx, peerPub, peerSig, nodePub, nodeSig, message, caller.Bytes())
	s.Require().NoError(err)

	method := s.precompile.ABI.Methods["getPid"]
	bz, err := s.precompile.GetPid(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)

	out, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Equal(pid, out[0].([]byte))
}

func (s *PrecompileTestSuite) TestHandleMethod_Dispatch() {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	contract := makeContract(caller)

	isRegMethod := s.precompile.ABI.Methods["isRegistered"]
	args := []interface{}{caller}

	bz, err := s.precompile.HandleMethod(s.ctx, contract, s.stateDB, &isRegMethod, args)
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

	message := []byte("test-nonce")
	peerPub, peerSig := generatePeerKey(s.T(), message)
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddd")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{peerPub, peerSig, nodePub, nodeSig, message}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "ICA not ready")
}

func (s *PrecompileTestSuite) TestRegister_EventLogStructure() {
	message := []byte("test-nonce")
	peerPub, peerSig := generatePeerKey(s.T(), message)
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	caller := common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	contract := makeContract(caller)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{peerPub, peerSig, nodePub, nodeSig, message}

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, args)
	s.Require().NoError(err)

	s.Require().Len(s.stateDB.logs, 1)
	log := s.stateDB.logs[0]
	s.Require().Len(log.Topics, 2)
	s.Require().Equal(common.BytesToHash(caller.Bytes()), log.Topics[1])
	s.Require().NotEmpty(log.Data)
}
