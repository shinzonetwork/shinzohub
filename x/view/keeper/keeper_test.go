package keeper_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	dbm "github.com/cosmos/cosmos-db"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
	"github.com/shinzonetwork/shinzohub/x/view/keeper"
	"github.com/shinzonetwork/shinzohub/x/view/types"
)

type mockHostKeeper struct{}

func (m *mockHostKeeper) IsRegisteredHost(_ sdk.Context, _ []byte) bool {
	return false
}

type mockSourcehubKeeper struct {
	called bool
	lastID string
	err    error
}

func (m *mockSourcehubKeeper) RegisterObject(_ sdk.Context, id string, _ string) (uint64, string, string, error) {
	m.called = true
	m.lastID = id
	return 0, "", "", m.err
}

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	keeper        keeper.Keeper
	mockHost      *mockHostKeeper
	mockSourcehub *mockSourcehubKeeper
	cdc           codec.BinaryCodec
}

func (s *KeeperTestSuite) simulateViewAck(viewId string) {
	meta := &sourcehubtypes.RegisterObjectMeta{ResourceName: sourcehubtypes.ViewResourceName, ObjectId: viewId}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	cb := keeper.NewAckCallback(s.keeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_REGISTER_OBJECT,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) SetupTest() {
	s.mockHost = &mockHostKeeper{}
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

	s.keeper = keeper.NewKeeper(
		cdc,
		storeService,
		s.mockHost,
		s.mockSourcehub,
		"authority",
	)

	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{Height: 10}, false, cosmoslog.NewNopLogger())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestRegisterView_Success() {
	err := s.keeper.RegisterView(s.ctx, "TestView_0xabc", "TestView", "cosmos1creator", "0xabc", []byte("viewdata"))
	s.Require().NoError(err)
	s.Require().True(s.mockSourcehub.called)
	s.Require().Equal("TestView_0xabc", s.mockSourcehub.lastID)

	_, canonicalFound, _ := s.keeper.GetView(s.ctx, "0xabc")
	s.Require().False(canonicalFound)
	pending, pendingFound, err := s.keeper.GetPendingView(s.ctx, "TestView_0xabc")
	s.Require().NoError(err)
	s.Require().True(pendingFound)
	s.Require().Equal("TestView", pending.Name)

	s.simulateViewAck("TestView_0xabc")

	view, found, err := s.keeper.GetView(s.ctx, "0xabc")
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal("TestView", view.Name)
	s.Require().Equal("cosmos1creator", view.Creator)
	s.Require().Equal("0xabc", view.ContractAddress)
	s.Require().Equal([]byte("viewdata"), view.Data)
	s.Require().Equal(uint64(10), view.Height)
}

func (s *KeeperTestSuite) TestRegisterView_SourcehubFailure() {
	s.mockSourcehub.err = fmt.Errorf("sourcehub down")
	err := s.keeper.RegisterView(s.ctx, "id1", "View1", "cosmos1x", "0x1", []byte("data"))
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "sourcehub down")

	_, found, _ := s.keeper.GetView(s.ctx, "0x1")
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestSetView_GetView() {
	view := types.View{
		Name:            "MyView",
		Creator:         "cosmos1abc",
		ContractAddress: "0xdeadbeef",
		Data:            []byte("payload"),
		Height:          5,
	}
	err := s.keeper.SetView(s.ctx, view)
	s.Require().NoError(err)

	got, found, err := s.keeper.GetView(s.ctx, "0xdeadbeef")
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(view.Name, got.Name)
	s.Require().Equal(view.Creator, got.Creator)
	s.Require().Equal(view.Data, got.Data)
}

func (s *KeeperTestSuite) TestGetView_NotFound() {
	_, found, err := s.keeper.GetView(s.ctx, "0xnotexist")
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestGetViewByAddress_Alias() {
	err := s.keeper.SetView(s.ctx, types.View{
		Name:            "V1",
		Creator:         "cosmos1x",
		ContractAddress: "0x111",
	})
	s.Require().NoError(err)

	v1, f1, e1 := s.keeper.GetView(s.ctx, "0x111")
	v2, f2, e2 := s.keeper.GetViewByAddress(s.ctx, "0x111")
	s.Require().NoError(e1)
	s.Require().NoError(e2)
	s.Require().Equal(f1, f2)
	s.Require().Equal(v1, v2)
}

func (s *KeeperTestSuite) TestSetView_UpdateDoesNotIncrementCount() {
	err := s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "V1"})
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), s.keeper.GetViewCount(s.ctx))

	err = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "V1Updated"})
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), s.keeper.GetViewCount(s.ctx))
}

func (s *KeeperTestSuite) TestGetViewCount_Empty() {
	s.Require().Equal(uint64(0), s.keeper.GetViewCount(s.ctx))
}

func (s *KeeperTestSuite) TestGetViewCount_AfterMultiple() {
	for i := 0; i < 5; i++ {
		err := s.keeper.SetView(s.ctx, types.View{
			ContractAddress: fmt.Sprintf("0x%d", i),
			Name:            fmt.Sprintf("V%d", i),
		})
		s.Require().NoError(err)
	}
	s.Require().Equal(uint64(5), s.keeper.GetViewCount(s.ctx))
}

func (s *KeeperTestSuite) TestGetAllViews_Empty() {
	views, pageRes, err := s.keeper.GetAllViews(s.ctx, &query.PageRequest{})
	s.Require().NoError(err)
	s.Require().NotNil(pageRes)
	s.Require().Empty(views)
}

func (s *KeeperTestSuite) TestGetAllViews_ReturnsAll() {
	for i := 0; i < 3; i++ {
		_ = s.keeper.SetView(s.ctx, types.View{
			ContractAddress: fmt.Sprintf("0x%d", i),
			Name:            fmt.Sprintf("View%d", i),
			Creator:         "cosmos1x",
		})
	}

	views, _, err := s.keeper.GetAllViews(s.ctx, &query.PageRequest{Limit: 100})
	s.Require().NoError(err)
	s.Require().Len(views, 3)
}

func (s *KeeperTestSuite) TestGetAllViews_Pagination() {
	for i := 0; i < 5; i++ {
		_ = s.keeper.SetView(s.ctx, types.View{
			ContractAddress: fmt.Sprintf("0x%d", i),
			Name:            fmt.Sprintf("View%d", i),
		})
	}

	views, pageRes, err := s.keeper.GetAllViews(s.ctx, &query.PageRequest{Limit: 2})
	s.Require().NoError(err)
	s.Require().Len(views, 2)
	s.Require().NotNil(pageRes.NextKey)

	views2, _, err := s.keeper.GetAllViews(s.ctx, &query.PageRequest{Key: pageRes.NextKey, Limit: 10})
	s.Require().NoError(err)
	s.Require().Len(views2, 3)
}

func (s *KeeperTestSuite) TestGenesis_InitExportRoundtrip() {
	gs := types.GenesisState{
		Views: []types.View{
			{Name: "V1", Creator: "cosmos1a", ContractAddress: "0x1", Data: []byte("d1"), Height: 1},
			{Name: "V2", Creator: "cosmos1b", ContractAddress: "0x2", Data: []byte("d2"), Height: 2},
		},
	}

	s.keeper.InitGenesis(s.ctx, gs)
	s.Require().Equal(uint64(2), s.keeper.GetViewCount(s.ctx))

	exported := s.keeper.ExportGenesis(s.ctx)
	s.Require().Len(exported.Views, 2)
}

func (s *KeeperTestSuite) TestQueryServer_Views() {
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "V1", Creator: "c1", Height: 5})
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x2", Name: "V2", Creator: "c2", Height: 15})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 100},
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 2)
}

func (s *KeeperTestSuite) TestQueryServer_Views_SinceBlock() {
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "V1", Height: 5})
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x2", Name: "V2", Height: 15})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 100},
		SinceBlock: 10,
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Equal("V2", resp.Views[0].Name)
}

func (s *KeeperTestSuite) TestQueryServer_Views_ExcludeData() {
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "V1", Data: []byte("bigdata")})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination:  &query.PageRequest{Limit: 100},
		IncludeData: false,
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Nil(resp.Views[0].Data)
}

func (s *KeeperTestSuite) TestQueryServer_Views_IncludeData() {
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "V1", Data: []byte("bigdata")})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination:  &query.PageRequest{Limit: 100},
		IncludeData: true,
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Equal([]byte("bigdata"), resp.Views[0].Data)
}

func (s *KeeperTestSuite) TestQueryServer_View_Found() {
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0xabc", Name: "V1", Creator: "cosmos1x"})

	qs := keeper.NewQueryServerImpl(s.keeper)
	resp, err := qs.View(s.ctx, &types.QueryViewRequest{ContractAddress: "0xabc"})
	s.Require().NoError(err)
	s.Require().Equal("V1", resp.View.Name)
}

func (s *KeeperTestSuite) TestQueryServer_View_NotFound() {
	qs := keeper.NewQueryServerImpl(s.keeper)
	_, err := qs.View(s.ctx, &types.QueryViewRequest{ContractAddress: "0xnotexist"})
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestQueryServer_ViewCount() {
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1"})
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x2"})

	qs := keeper.NewQueryServerImpl(s.keeper)
	resp, err := qs.ViewCount(s.ctx, &types.QueryViewCountRequest{})
	s.Require().NoError(err)
	s.Require().Equal(uint64(2), resp.Count)
}

func (s *KeeperTestSuite) TestQueryServer_NilRequests() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	_, err := qs.Views(s.ctx, nil)
	s.Require().Error(err)

	_, err = qs.View(s.ctx, nil)
	s.Require().Error(err)

	_, err = qs.ViewCount(s.ctx, nil)
	s.Require().Error(err)
}
