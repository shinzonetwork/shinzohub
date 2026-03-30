package keeper_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/shinzonetwork/shinzohub/x/indexer/keeper"
	"github.com/shinzonetwork/shinzohub/x/indexer/types"
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

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	keeper        keeper.Keeper
	mockAdmin     *mockAdminKeeper
	mockSourcehub *mockSourcehubKeeper
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
	storeService := runtime.NewKVStoreService(storeKey)

	s.keeper = keeper.NewKeeper(cdc, storeService, s.mockAdmin, s.mockSourcehub)
	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, cosmoslog.NewNopLogger())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestRegisterIndexer_Success() {
	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	did, err := s.keeper.RegisterIndexer(s.ctx, "192.168.1.1:8080", callerAddr, "ethereum", 1)
	s.Require().NoError(err)
	s.Require().NotEmpty(did)

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

func (s *KeeperTestSuite) TestRegisterIndexer_Idempotent() {
	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	_, err := s.keeper.RegisterIndexer(s.ctx, "192.168.1.1:8080", callerAddr, "ethereum", 1)
	s.Require().NoError(err)

	_, err = s.keeper.RegisterIndexer(s.ctx, "192.168.1.2:9090", callerAddr, "ethereum", 1)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestRegisterIndexer_ICAFailure() {
	s.mockSourcehub.err = fmt.Errorf("ICA not ready")

	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	_, err := s.keeper.RegisterIndexer(s.ctx, "192.168.1.1:8080", callerAddr, "ethereum", 1)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "ICA not ready")
}

func (s *KeeperTestSuite) TestSetIndexerAssertion_GetIndexerAssertion() {
	assertion := types.IndexerAssertion{
		ConsensusPubKey: "pubkey123",
		DelegateAddress: "shinzo1delegate",
		SourceChain:     "ethereum",
		SourceChainId:   1,
		AssertionId:     "assert-001",
	}

	err := s.keeper.SetIndexerAssertion(s.ctx, assertion)
	s.Require().NoError(err)

	got, found, err := s.keeper.GetIndexerAssertion(s.ctx, "shinzo1delegate", "ethereum", 1)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(assertion.ConsensusPubKey, got.ConsensusPubKey)
	s.Require().Equal(assertion.DelegateAddress, got.DelegateAddress)
	s.Require().Equal(assertion.AssertionId, got.AssertionId)
}

func (s *KeeperTestSuite) TestGetIndexerAssertion_NotFound() {
	_, found, err := s.keeper.GetIndexerAssertion(s.ctx, "shinzo1nobody", "ethereum", 1)
	s.Require().NoError(err)
	s.Require().False(found)
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

func (s *KeeperTestSuite) TestGetIndexer_NotFound() {
	_, found, err := s.keeper.GetIndexer(s.ctx, "shinzo1nonexistent")
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestGetIndexerCount_Empty() {
	s.Require().Equal(uint64(0), s.keeper.GetIndexerCount(s.ctx))
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

func (s *KeeperTestSuite) TestSetIndexer_UpdateDoesNotIncrementCount() {
	indexer := types.Indexer{Address: "shinzo1idx0", Did: "did:0", ConnectionString: "10.0.0.1:8080"}
	_ = s.keeper.SetIndexer(s.ctx, indexer)
	s.Require().Equal(uint64(1), s.keeper.GetIndexerCount(s.ctx))

	indexer.ConnectionString = "10.0.0.2:9090"
	_ = s.keeper.SetIndexer(s.ctx, indexer)
	s.Require().Equal(uint64(1), s.keeper.GetIndexerCount(s.ctx))
}

func (s *KeeperTestSuite) TestGetAllIndexers_Empty() {
	indexers, _, err := s.keeper.GetAllIndexers(s.ctx, nil)
	s.Require().NoError(err)
	s.Require().Empty(indexers)
}

func (s *KeeperTestSuite) TestGetAllIndexers_ReturnsAll() {
	for i := 0; i < 5; i++ {
		_ = s.keeper.SetIndexer(s.ctx, types.Indexer{
			Address:          fmt.Sprintf("shinzo1idx%d", i),
			Did:              fmt.Sprintf("did:%d", i),
			ConnectionString: fmt.Sprintf("10.0.0.%d:8080", i),
		})
	}
	indexers, _, err := s.keeper.GetAllIndexers(s.ctx, nil)
	s.Require().NoError(err)
	s.Require().Len(indexers, 5)
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

	got, found, err := s.keeper.GetIndexerAssertion(s.ctx, "shinzo1del0", "ethereum", 1)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal("a0", got.AssertionId)
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

func (s *KeeperTestSuite) TestQueryServer_Indexer_Found() {
	qs := keeper.NewQueryServerImpl(s.keeper)
	_ = s.keeper.SetIndexer(s.ctx, types.Indexer{Address: "shinzo1idx0", Did: "did:0", ConnectionString: "10.0.0.1:8080"})

	resp, err := qs.Indexer(s.ctx, &types.QueryIndexerRequest{Address: "shinzo1idx0"})
	s.Require().NoError(err)
	s.Require().Equal("shinzo1idx0", resp.Indexer.Address)
}

func (s *KeeperTestSuite) TestQueryServer_Indexer_NotFound() {
	qs := keeper.NewQueryServerImpl(s.keeper)
	_, err := qs.Indexer(s.ctx, &types.QueryIndexerRequest{Address: "shinzo1nonexistent"})
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestQueryServer_Indexers() {
	qs := keeper.NewQueryServerImpl(s.keeper)
	for i := 0; i < 3; i++ {
		_ = s.keeper.SetIndexer(s.ctx, types.Indexer{
			Address:          fmt.Sprintf("shinzo1idx%d", i),
			Did:              fmt.Sprintf("did:%d", i),
			ConnectionString: fmt.Sprintf("10.0.0.%d:8080", i),
		})
	}
	resp, err := qs.Indexers(s.ctx, &types.QueryIndexersRequest{})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 3)
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
