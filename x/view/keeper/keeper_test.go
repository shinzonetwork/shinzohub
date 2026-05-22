package keeper_test

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/crypto"
	viewbundle "github.com/shinzonetwork/viewbundle-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

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

func mustBuildViewBundle(t require.TestingT, query, sdl string, lenses ...viewbundle.Lens) []byte {
	bundle, err := viewbundle.NewBundler().BundleView(viewbundle.View{
		Query: query,
		Sdl:   sdl,
		Transform: viewbundle.Transform{
			Lenses: lenses,
		},
	})
	require.NoError(t, err)
	return bundle
}

func testLens(wasm []byte, args string) viewbundle.Lens {
	return viewbundle.Lens{
		Path:      base64.StdEncoding.EncodeToString(wasm),
		Arguments: args,
	}
}

func expectedShortLensHash(wasm []byte) string {
	hash := crypto.Keccak256(wasm)
	return "0x" + hex.EncodeToString(hash[:16])
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

func (s *KeeperTestSuite) TestQueryServer_Views_FilterByNameAndCreator() {
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "V1", Creator: "c1"})
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x2", Name: "V2", Creator: "c1"})
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x3", Name: "V2", Creator: "c2"})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 100},
		Name:       "V2",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 2)
	s.Require().Equal("V2", resp.Views[0].Name)
	s.Require().Equal("V2", resp.Views[1].Name)

	resp, err = qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 100},
		Creator:    "c1",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 2)
	s.Require().Equal("c1", resp.Views[0].Creator)
	s.Require().Equal("c1", resp.Views[1].Creator)

	resp, err = qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 100},
		Name:       "V2",
		Creator:    "c2",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Equal("0x3", resp.Views[0].ContractAddress)
}

func (s *KeeperTestSuite) TestQueryServer_Views_IncludeMetadata() {
	viewQuery := "Ethereum__Mainnet__Log { address blockNumber }"
	viewSdl := "type TestRoot @materialized(if: false) { address: String blockNumber: Int }"
	lensWasm := []byte("lens-one")
	lensArgs := `{"token":"0x1"}`
	bundle := mustBuildViewBundle(s.T(), viewQuery, viewSdl, testLens(lensWasm, lensArgs))

	_ = s.keeper.SetView(s.ctx, types.View{
		ContractAddress: "0x1",
		Name:            "TestRoot",
		Creator:         "c1",
		Data:            bundle,
	})

	qs := keeper.NewQueryServerImpl(s.keeper)
	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination:      &query.PageRequest{Limit: 100},
		IncludeMetadata: true,
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Nil(resp.Views[0].Data)
	s.Require().NotNil(resp.Views[0].Metadata)
	s.Require().Equal(viewQuery, resp.Views[0].Metadata.Query)
	s.Require().Equal(viewSdl, resp.Views[0].Metadata.Sdl)
	s.Require().Equal("TestRoot", resp.Views[0].Metadata.RootType)
	s.Require().Empty(resp.Views[0].Metadata.ParseError)
	s.Require().Len(resp.Views[0].Metadata.Lenses, 1)
	s.Require().Equal(uint32(1), resp.Views[0].Metadata.Lenses[0].Id)
	s.Require().Equal(lensArgs, resp.Views[0].Metadata.Lenses[0].Args)
	s.Require().Equal(expectedShortLensHash(lensWasm), resp.Views[0].Metadata.Lenses[0].Hash)
}

func (s *KeeperTestSuite) TestQueryServer_Views_MetadataFilters() {
	firstQuery := "Ethereum__Mainnet__Log { address targetField }"
	firstSdl := "type FirstRoot @materialized(if: false) { targetField: String }"
	firstWasm := []byte("first-lens")
	firstArgs := `{"filter":"needle"}`
	firstBundle := mustBuildViewBundle(s.T(), firstQuery, firstSdl, testLens(firstWasm, firstArgs))

	secondQuery := "Ethereum__Mainnet__Block { number }"
	secondSdl := "type SecondRoot @materialized(if: false) { number: Int }"
	secondBundle := mustBuildViewBundle(s.T(), secondQuery, secondSdl, testLens([]byte("second-lens"), `{"filter":"other"}`))

	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "FirstRoot", Data: firstBundle})
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x2", Name: "SecondRoot", Data: secondBundle})

	qs := keeper.NewQueryServerImpl(s.keeper)
	assertSingleMatch := func(req *types.QueryViewsRequest) {
		req.Pagination = &query.PageRequest{Limit: 100}
		resp, err := qs.Views(s.ctx, req)
		s.Require().NoError(err)
		s.Require().Len(resp.Views, 1)
		s.Require().Equal("0x1", resp.Views[0].ContractAddress)
		s.Require().Nil(resp.Views[0].Metadata)
	}

	assertSingleMatch(&types.QueryViewsRequest{MetadataRootType: "FirstRoot"})
	assertSingleMatch(&types.QueryViewsRequest{MetadataLensHash: expectedShortLensHash(firstWasm)})
	assertSingleMatch(&types.QueryViewsRequest{MetadataQueryContains: "targetField"})
	assertSingleMatch(&types.QueryViewsRequest{MetadataSdlContains: "targetField"})
	assertSingleMatch(&types.QueryViewsRequest{MetadataLensArgsContains: "needle"})
}

func (s *KeeperTestSuite) TestQueryServer_Views_MetadataFilterCanIncludeMetadata() {
	viewQuery := "Ethereum__Mainnet__Log { address }"
	viewSdl := "type FilterRoot @materialized(if: false) { address: String }"
	bundle := mustBuildViewBundle(s.T(), viewQuery, viewSdl, testLens([]byte("lens"), "{}"))

	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "FilterRoot", Data: bundle})

	qs := keeper.NewQueryServerImpl(s.keeper)
	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination:       &query.PageRequest{Limit: 100},
		IncludeMetadata:  true,
		MetadataRootType: "FilterRoot",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().NotNil(resp.Views[0].Metadata)
	s.Require().Equal("FilterRoot", resp.Views[0].Metadata.RootType)
}

func (s *KeeperTestSuite) TestQueryServer_Views_MalformedMetadata() {
	validBundle := mustBuildViewBundle(
		s.T(),
		"Ethereum__Mainnet__Log { address }",
		"type ValidRoot @materialized(if: false) { address: String }",
		testLens([]byte("valid-lens"), "{}"),
	)
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x1", Name: "Broken", Data: []byte("not-a-viewbundle")})
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0x2", Name: "Valid", Data: validBundle})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 100},
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 2)

	resp, err = qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination:      &query.PageRequest{Limit: 100},
		IncludeMetadata: true,
		Name:            "Broken",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().NotNil(resp.Views[0].Metadata)
	s.Require().NotEmpty(resp.Views[0].Metadata.ParseError)

	resp, err = qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination:       &query.PageRequest{Limit: 100},
		MetadataRootType: "ValidRoot",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Equal("0x2", resp.Views[0].ContractAddress)
}

func (s *KeeperTestSuite) TestQueryServer_View_Found() {
	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0xabc", Name: "V1", Creator: "cosmos1x"})

	qs := keeper.NewQueryServerImpl(s.keeper)
	resp, err := qs.View(s.ctx, &types.QueryViewRequest{ContractAddress: "0xabc"})
	s.Require().NoError(err)
	s.Require().Equal("V1", resp.View.Name)
}

func (s *KeeperTestSuite) TestQueryServer_View_IncludeMetadata() {
	viewQuery := "Ethereum__Mainnet__Log { address }"
	viewSdl := "type SingleRoot @materialized(if: false) { address: String }"
	bundle := mustBuildViewBundle(s.T(), viewQuery, viewSdl, testLens([]byte("single-lens"), "{}"))

	_ = s.keeper.SetView(s.ctx, types.View{ContractAddress: "0xabc", Name: "SingleRoot", Data: bundle})

	qs := keeper.NewQueryServerImpl(s.keeper)
	resp, err := qs.View(s.ctx, &types.QueryViewRequest{
		ContractAddress: "0xabc",
		IncludeMetadata: true,
	})
	s.Require().NoError(err)
	s.Require().Nil(resp.View.Data)
	s.Require().NotNil(resp.View.Metadata)
	s.Require().Equal("SingleRoot", resp.View.Metadata.RootType)
	s.Require().Equal(viewQuery, resp.View.Metadata.Query)
	s.Require().Equal(viewSdl, resp.View.Metadata.Sdl)
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
