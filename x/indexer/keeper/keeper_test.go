package keeper_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/shinzonetwork/shinzohub/x/indexer/keeper"
	"github.com/shinzonetwork/shinzohub/x/indexer/types"
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

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	keeper        keeper.Keeper
	mockAdmin     *mockAdminKeeper
	mockSourcehub *mockSourcehubKeeper
	cdc           codec.BinaryCodec
}

func (s *KeeperTestSuite) simulateIndexerAck(callerAddr []byte) {
	did, found := s.keeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	s.Require().True(found, "pending indexer did not land in state")
	meta := &sourcehubtypes.SetRelationshipMeta{Did: string(did), Group: "indexer"}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	cb := keeper.NewAckCallback(s.keeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) SetupTest() {
	s.mockAdmin = &mockAdminKeeper{admins: map[string]bool{}}
	s.mockSourcehub = &mockSourcehubKeeper{}

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	s.cdc = cdc
	storeService := runtime.NewKVStoreService(storeKey)

	s.keeper = keeper.NewKeeper(cdc, storeService, s.mockAdmin, s.mockSourcehub)
	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, cosmoslog.NewNopLogger())
}

func generateNodeIdentityKey(t *testing.T, message []byte) (pubkey, signature []byte) {
	privKey, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	h := sha256.Sum256(message)
	return privKey.PubKey().SerializeUncompressed(), ecdsa.Sign(privKey, h[:]).Serialize()
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestRegisterIndexer_Success() {
	message := []byte("test-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	did, err := s.keeper.RegisterIndexer(s.ctx, nodePub, nodeSig, message, "192.168.1.1:8080", callerAddr, "ethereum", 1)
	s.Require().NoError(err)
	s.Require().NotEmpty(did)

	s.simulateIndexerAck(callerAddr)

	bech32Addr := sdk.AccAddress(callerAddr).String()
	indexer, found, err := s.keeper.GetIndexer(s.ctx, bech32Addr)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(bech32Addr, indexer.Address)
	s.Require().Equal(string(did), indexer.Did)
	s.Require().Equal("192.168.1.1:8080", indexer.ConnectionString)
	s.Require().Equal("ethereum", indexer.SourceChain)
	s.Require().Equal(uint64(1), indexer.SourceChainId)

	s.Require().Equal(uint64(1), s.keeper.GetIndexerCount(s.ctx))
}

func (s *KeeperTestSuite) TestRegisterIndexer_InvalidNodeSignature() {
	message := []byte("test-nonce")
	nodePub, _ := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}
	_, wrongSig := generateNodeIdentityKey(s.T(), []byte("wrong"))

	_, err := s.keeper.RegisterIndexer(s.ctx, nodePub, wrongSig, message, "192.168.1.1:8080", callerAddr, "ethereum", 1)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRegisterIndexer_SameAddrDifferentDID_Fails() {
	message := []byte("test-nonce")
	nodePub1, nodeSig1 := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	_, err := s.keeper.RegisterIndexer(s.ctx, nodePub1, nodeSig1, message, "192.168.1.1:8080", callerAddr, "ethereum", 1)
	s.Require().NoError(err)

	nodePub2, nodeSig2 := generateNodeIdentityKey(s.T(), message)
	_, err = s.keeper.RegisterIndexer(s.ctx, nodePub2, nodeSig2, message, "192.168.1.2:8080", callerAddr, "ethereum", 1)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "address already registered")
}

func (s *KeeperTestSuite) TestRegisterIndexer_ICAFailure() {
	s.mockSourcehub.err = fmt.Errorf("ICA not ready")

	message := []byte("test-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	_, err := s.keeper.RegisterIndexer(s.ctx, nodePub, nodeSig, message, "192.168.1.1:8080", callerAddr, "ethereum", 1)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "ICA not ready")
}

func (s *KeeperTestSuite) TestSetIndexer_GetIndexer() {
	indexer := types.Indexer{
		Address:          "shinzo1idx0",
		Did:              "did:key:z0",
		ConnectionString: "10.0.0.1:8080",
		SourceChain:      "ethereum",
		SourceChainId:    1,
	}

	err := s.keeper.SetIndexer(s.ctx, indexer)
	s.Require().NoError(err)

	got, found, err := s.keeper.GetIndexer(s.ctx, "shinzo1idx0")
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(indexer.Address, got.Address)
	s.Require().Equal(indexer.SourceChain, got.SourceChain)
}

func (s *KeeperTestSuite) TestGetIndexerCount_AfterMultiple() {
	for i := 0; i < 3; i++ {
		_ = s.keeper.SetIndexer(s.ctx, types.Indexer{
			Address:          fmt.Sprintf("shinzo1idx%d", i),
			Did:              fmt.Sprintf("did:%d", i),
			ConnectionString: fmt.Sprintf("10.0.0.%d:8080", i),
		})
	}
	s.Require().Equal(uint64(3), s.keeper.GetIndexerCount(s.ctx))
}

func (s *KeeperTestSuite) TestGenesis_InitExportRoundtrip() {
	genesis := types.GenesisState{
		Indexers: []types.Indexer{
			{Address: "shinzo1idx0", Did: "did:0", ConnectionString: "10.0.0.1:8080", SourceChain: "ethereum", SourceChainId: 1},
			{Address: "shinzo1idx1", Did: "did:1", ConnectionString: "10.0.0.2:8080", SourceChain: "polygon", SourceChainId: 137},
		},
		Assertions: []types.IndexerAssertion{
			{ConsensusPubKey: "pk0", DelegateAddress: "shinzo1del0", SourceChain: "ethereum", SourceChainId: 1, AssertionId: "a0"},
		},
	}

	s.keeper.InitGenesis(s.ctx, genesis)

	exported := s.keeper.ExportGenesis(s.ctx)
	s.Require().Len(exported.Indexers, 2)
	s.Require().Equal(uint64(2), s.keeper.GetIndexerCount(s.ctx))
}

func (s *KeeperTestSuite) TestMsgServer_AddIndexerAssertion_NotAdmin() {
	ms := keeper.NewMsgServerImpl(s.keeper)

	_, err := ms.AddIndexerAssertion(s.ctx, &types.MsgIndexerAssertion{
		Signer:            "shinzo1admin",
		DelegateAddress:   "shinzo1delegate",
		SourceChain:       "ethereum",
		SourceChainId:     1,
		AssertionId:       "a0",
		DelegateDigest:    make([]byte, 32),
		DelegateSignature: make([]byte, 65),
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "admin required")
}

func (s *KeeperTestSuite) TestQueryServer_IndexerCount() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.IndexerCount(s.ctx, &types.QueryIndexerCountRequest{})
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), resp.Count)

	_ = s.keeper.SetIndexer(s.ctx, types.Indexer{Address: "shinzo1idx0", Did: "did:0", ConnectionString: "10.0.0.1:8080"})
	resp, err = qs.IndexerCount(s.ctx, &types.QueryIndexerCountRequest{})
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), resp.Count)
}

func (s *KeeperTestSuite) TestQueryServer_IndexersFilters() {
	qs := keeper.NewQueryServerImpl(s.keeper)
	indexers := []types.Indexer{
		{Address: "shinzo1idx0", Did: "did:key:z0", ConnectionString: "10.0.0.1:8080"},
		{Address: "shinzo1idx1", Did: "did:key:z1", ConnectionString: "10.0.0.2:8080"},
		{Address: "shinzo1idx2", Did: "did:key:z2", ConnectionString: "wss://example.com/indexer"},
	}
	for _, indexer := range indexers {
		s.Require().NoError(s.keeper.SetIndexer(s.ctx, indexer))
	}

	resp, err := qs.Indexers(s.ctx, &types.QueryIndexersRequest{Did: "did:key:z1"})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 1)
	s.Require().Equal("shinzo1idx1", resp.Indexers[0].Address)

	resp, err = qs.Indexers(s.ctx, &types.QueryIndexersRequest{ConnectionString: "10.0.0."})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 2)
	s.Require().Equal("shinzo1idx0", resp.Indexers[0].Address)
	s.Require().Equal("shinzo1idx1", resp.Indexers[1].Address)

	resp, err = qs.Indexers(s.ctx, &types.QueryIndexersRequest{
		Did:              "did:key:z1",
		ConnectionString: "example.com",
	})
	s.Require().NoError(err)
	s.Require().Empty(resp.Indexers)
}

func (s *KeeperTestSuite) TestQueryServer_IndexersFilterBeforePagination() {
	qs := keeper.NewQueryServerImpl(s.keeper)
	indexers := []types.Indexer{
		{Address: "shinzo1a", Did: "did:key:za", ConnectionString: "alpha"},
		{Address: "shinzo1b", Did: "did:key:zb", ConnectionString: "needle-1"},
		{Address: "shinzo1c", Did: "did:key:zc", ConnectionString: "needle-2"},
	}
	for _, indexer := range indexers {
		s.Require().NoError(s.keeper.SetIndexer(s.ctx, indexer))
	}

	resp, err := qs.Indexers(s.ctx, &types.QueryIndexersRequest{
		Pagination:       &query.PageRequest{Limit: 1},
		ConnectionString: "needle",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 1)
	s.Require().Equal("shinzo1b", resp.Indexers[0].Address)
	s.Require().NotEmpty(resp.Pagination.NextKey)

	resp, err = qs.Indexers(s.ctx, &types.QueryIndexersRequest{
		Pagination:       &query.PageRequest{Key: resp.Pagination.NextKey, Limit: 1},
		ConnectionString: "needle",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 1)
	s.Require().Equal("shinzo1c", resp.Indexers[0].Address)
	s.Require().Empty(resp.Pagination.NextKey)
}

func (s *KeeperTestSuite) TestQueryServer_NilRequests() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	_, err := qs.Indexers(s.ctx, nil)
	s.Require().Error(err)

	_, err = qs.Indexer(s.ctx, nil)
	s.Require().Error(err)

	_, err = qs.IndexerCount(s.ctx, nil)
	s.Require().Error(err)
}
