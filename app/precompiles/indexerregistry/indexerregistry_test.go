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
	icaCalled bool
	icaErr    error
	checkErr  error
}

func (m *mockSourcehubKeeper) SendICASetRelationship(_ sdk.Context, _, _, _ string) (uint64, string, string, error) {
	m.icaCalled = true
	return 0, "", "", m.icaErr
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

	p, err := indexerregistry.NewPrecompile(10000, s.indexerKeeper, s.mockSourcehub)
	require.NoError(s.T(), err)
	s.precompile = p
}

func generateNodeIdentityKey(t *testing.T, message []byte) (pubkey, signature []byte) {
	privKey, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	h := sha256.Sum256(message)
	return privKey.PubKey().SerializeCompressed(), ecdsa.Sign(privKey, h[:]).Serialize()
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

// assertOperator inserts a baseline indexer assertion for the given operator
// address against (ethereum, 1, validator-A).
func (s *PrecompileTestSuite) assertOperator(operatorBech32 string) {
	admin := "shinzo1admin000000000000000000000000000000000"
	s.mockAdmin.admins[admin] = true
	err := s.indexerKeeper.UpsertAssertion(s.ctx, &indexertypes.MsgIndexerAssertion{
		Signer:             admin,
		SourceChain:        "ethereum",
		SourceChainId:      1,
		ValidatorPubkey:    []byte("validator-A"),
		AssertionAuthority: []byte("withdrawal-W"),
		Nonce:              1,
		OperatorAddress:    operatorBech32,
		PayoutAddress:      operatorBech32,
	})
	s.Require().NoError(err)
}

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

// ─── tests ────────────────────────────────────────────────────────────

func (s *PrecompileTestSuite) TestRegister_Success() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	s.assertOperator(sdk.AccAddress(caller.Bytes()).String())

	message := []byte("op-nonce-1")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, nodeSig, message, "https://idx-1:9090"}

	bz, err := s.precompile.Register(s.ctx, makeContract(caller), s.stateDB, &method, args)
	s.Require().NoError(err)
	s.Require().Nil(bz)

	s.Require().True(s.mockSourcehub.icaCalled, "ICA SetRelationship not fired on first register")
	s.Require().Len(s.stateDB.logs, 1)

	row, found, err := s.indexerKeeper.GetIndexerByAddress(s.ctx, sdk.AccAddress(caller.Bytes()).String())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().True(row.Registered)
	s.Require().NotEmpty(row.Did)
	s.Require().Equal("https://idx-1:9090", row.ConnectionString)
}

func (s *PrecompileTestSuite) TestRegister_IdempotentDoesNotReissueICA() {
	caller := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	s.assertOperator(sdk.AccAddress(caller.Bytes()).String())

	message := []byte("op-nonce-1")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, nodeSig, message, "https://idx-1:9090"}

	_, err := s.precompile.Register(s.ctx, makeContract(caller), s.stateDB, &method, args)
	s.Require().NoError(err)
	s.Require().True(s.mockSourcehub.icaCalled)

	// Reset the flag and call again with a refreshed connection string.
	s.mockSourcehub.icaCalled = false
	args2 := []interface{}{nodePub, nodeSig, message, "https://idx-1:9091"}
	_, err = s.precompile.Register(s.ctx, makeContract(caller), s.stateDB, &method, args2)
	s.Require().NoError(err)
	s.Require().False(s.mockSourcehub.icaCalled, "ICA should not fire on a re-register")

	row, _, _ := s.indexerKeeper.GetIndexerByAddress(s.ctx, sdk.AccAddress(caller.Bytes()).String())
	s.Require().Equal("https://idx-1:9091", row.ConnectionString)
}

func (s *PrecompileTestSuite) TestRegister_RevertsWithoutAssertion() {
	caller := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	message := []byte("op-nonce-1")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, nodeSig, message, "https://idx:9090"}

	_, err := s.precompile.Register(s.ctx, makeContract(caller), s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not asserted")
	s.Require().False(s.mockSourcehub.icaCalled)
}

func (s *PrecompileTestSuite) TestRegister_RevertsOnICANotReady() {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	s.assertOperator(sdk.AccAddress(caller.Bytes()).String())
	s.mockSourcehub.checkErr = fmt.Errorf("no policy ID set in module state")

	message := []byte("op-nonce-1")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, nodeSig, message, "https://idx:9090"}

	_, err := s.precompile.Register(s.ctx, makeContract(caller), s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "no policy ID set")
	s.Require().False(s.mockSourcehub.icaCalled)
}

func (s *PrecompileTestSuite) TestRegister_RevertsOnBadNodeKeySignature() {
	caller := common.HexToAddress("0x2222222222222222222222222222222222222222")
	s.assertOperator(sdk.AccAddress(caller.Bytes()).String())

	message := []byte("op-nonce-1")
	nodePub, _ := generateNodeIdentityKey(s.T(), message)
	_, otherSig := generateNodeIdentityKey(s.T(), message) // signature from a different key

	method := s.precompile.ABI.Methods["register"]
	args := []interface{}{nodePub, otherSig, message, "https://idx:9090"}

	_, err := s.precompile.Register(s.ctx, makeContract(caller), s.stateDB, &method, args)
	s.Require().Error(err)
	s.Require().False(s.mockSourcehub.icaCalled)
}

func (s *PrecompileTestSuite) TestIsRegistered_FalseUntilRegister() {
	caller := common.HexToAddress("0x3333333333333333333333333333333333333333")
	s.assertOperator(sdk.AccAddress(caller.Bytes()).String())

	method := s.precompile.ABI.Methods["isRegistered"]
	bz, err := s.precompile.IsRegistered(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)
	unpacked, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().False(unpacked[0].(bool))

	// After register, IsRegistered flips to true.
	message := []byte("op-nonce-1")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	regMethod := s.precompile.ABI.Methods["register"]
	_, err = s.precompile.Register(s.ctx, makeContract(caller), s.stateDB, &regMethod, []interface{}{nodePub, nodeSig, message, "https://x"})
	s.Require().NoError(err)

	bz, err = s.precompile.IsRegistered(s.ctx, &method, []interface{}{caller})
	s.Require().NoError(err)
	unpacked, err = method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().True(unpacked[0].(bool))
}
