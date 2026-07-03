package keeper_test

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/crypto"
	viewbundle "github.com/shinzonetwork/viewbundle-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
	"github.com/shinzonetwork/shinzohub/x/view/keeper"
	"github.com/shinzonetwork/shinzohub/x/view/types"
)

// Records the call and can inject an error to test pending rollback.
type mockSourcehubKeeper struct {
	calls       int
	lastID      string
	lastCreator string
	err         error
}

func (m *mockSourcehubKeeper) RegisterObject(_ sdk.Context, id, requestor string) (uint64, string, string, error) {
	m.calls++
	m.lastID = id
	m.lastCreator = requestor
	if m.err != nil {
		return 0, "", "", m.err
	}
	return 42, "icacontroller-test", "channel-0", nil
}

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	keeper        keeper.Keeper
	mockSourcehub *mockSourcehubKeeper
	cdc           codec.Codec
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	s.mockSourcehub = &mockSourcehubKeeper{}

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	s.cdc = cdc

	s.keeper = keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		s.mockSourcehub,
	)
	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{Height: 7}, false, cosmoslog.NewNopLogger())
}

const (
	sampleAddress = "0x95C3Ad461380cAF7b6Fb53Bc2B1ca42808cd031C"
	sampleCreator = "0x939D585A4370c8Cde88CB4C34Fe751c41cAaff90"
	sampleName    = "View651f73f5"
)

var sampleBundle = []byte("VWL\x01<viewbundle bytes>")

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

func findEvent(ctx sdk.Context, t string) *sdk.Event {
	for _, e := range ctx.EventManager().Events() {
		if e.Type == t {
			ev := e
			return &ev
		}
	}
	return nil
}

func attr(ev *sdk.Event, key string) string {
	for _, a := range ev.Attributes {
		if a.Key == key {
			return a.Value
		}
	}
	return ""
}

// Happy path: pending entry written, sourcehub called, view_pending emitted.
func (s *KeeperTestSuite) TestRegisterView_WritesPendingAndCallsSourcehub() {
	view, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)
	s.Equal(sampleName, view.Name)
	s.Equal(uint64(7), view.Height)

	s.Equal(1, s.mockSourcehub.calls)
	s.Equal(sampleAddress, s.mockSourcehub.lastID)
	s.NotEmpty(s.mockSourcehub.lastCreator, "creator should be converted to bech32 before sourcehub call")

	pending, found, err := s.keeper.GetPendingView(s.ctx, sampleAddress)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal(sampleName, pending.Name)

	_, found, err = s.keeper.GetView(s.ctx, sampleAddress)
	s.Require().NoError(err)
	s.False(found, "final store should be empty until ack")

	ev := findEvent(s.ctx, types.EventTypeViewPending)
	s.Require().NotNil(ev)
	s.Equal(sampleAddress, attr(ev, types.AttrKeyAddress))
	s.Equal(sampleCreator, attr(ev, types.AttrKeyCreator))
	s.Equal(sampleName, attr(ev, types.AttrKeyName))
	s.Equal(base64.StdEncoding.EncodeToString(sampleBundle), attr(ev, types.AttrKeyData))
}

// Sourcehub error → pending entry rolled back so tx reverts cleanly.
func (s *KeeperTestSuite) TestRegisterView_RolledBackWhenSourcehubFails() {
	s.mockSourcehub.err = sdkErr("ICA channel not ready")

	_, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().Error(err)

	_, found, _ := s.keeper.GetPendingView(s.ctx, sampleAddress)
	s.False(found)
	s.Equal(uint64(0), s.keeper.GetViewCount(s.ctx))
}

// Non-hex creator is rejected before any side-effects.
func (s *KeeperTestSuite) TestRegisterView_RejectsMalformedCreator() {
	_, err := s.keeper.RegisterView(s.ctx, sampleName, "not-a-hex-address", sampleAddress, sampleBundle)
	s.Require().Error(err)

	s.Equal(0, s.mockSourcehub.calls)
	_, found, _ := s.keeper.GetPendingView(s.ctx, sampleAddress)
	s.False(found)
}

// Re-registering the same address while PENDING is an idempotent no-op: no second
// ICA fires (a duplicate that fails could delete the pending row and strand the
// view the first ICA registered).
func (s *KeeperTestSuite) TestRegisterView_WhilePending_IsIdempotent() {
	_, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)
	s.Equal(1, s.mockSourcehub.calls)

	view, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)
	s.Equal(sampleName, view.Name)
	s.Equal(1, s.mockSourcehub.calls, "duplicate registration must not fire a second ICA")
}

// Re-registering an already-REGISTERED address is a no-op: returns the existing
// view, fires no second ICA, leaves the count and pending store untouched.
func (s *KeeperTestSuite) TestRegisterView_AfterRegistered_IsNoOp() {
	_, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)
	s.fireAck(sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS, "")
	s.Require().Equal(uint64(1), s.keeper.GetViewCount(s.ctx))

	view, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)
	s.Equal(sampleName, view.Name)
	s.Equal(1, s.mockSourcehub.calls, "re-register of a final view must not fire a second ICA")
	s.Equal(uint64(1), s.keeper.GetViewCount(s.ctx), "count must not change")

	_, found, _ := s.keeper.GetPendingView(s.ctx, sampleAddress)
	s.False(found, "re-register must not write a stray pending row")
}

// SUCCESS ack promotes pending → final, bumps count, fires view_registered.
func (s *KeeperTestSuite) TestAckSuccess_PromotesPendingToFinal() {
	_, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)

	s.fireAck(sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS, "")

	_, found, _ := s.keeper.GetPendingView(s.ctx, sampleAddress)
	s.False(found)

	v, found, err := s.keeper.GetView(s.ctx, sampleAddress)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal(sampleName, v.Name)
	s.Equal(sampleCreator, v.Creator)
	s.Equal(uint64(1), s.keeper.GetViewCount(s.ctx))

	ev := findEvent(s.ctx, types.EventTypeViewRegistered)
	s.Require().NotNil(ev)
	s.Equal(sampleAddress, attr(ev, types.AttrKeyAddress))
	s.Equal(sampleCreator, attr(ev, types.AttrKeyCreator))
	s.Equal(sampleName, attr(ev, types.AttrKeyName))
}

// FAILURE ack drops pending and emits view_registration_failed with the error.
func (s *KeeperTestSuite) TestAckFailure_DropsPending() {
	_, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)

	s.fireAck(sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE, "policy rejected")

	_, found, _ := s.keeper.GetPendingView(s.ctx, sampleAddress)
	s.False(found)
	_, found, _ = s.keeper.GetView(s.ctx, sampleAddress)
	s.False(found)
	s.Equal(uint64(0), s.keeper.GetViewCount(s.ctx))

	ev := findEvent(s.ctx, types.EventTypeViewRegistrationFailed)
	s.Require().NotNil(ev)
	s.Equal("policy rejected", attr(ev, types.AttrKeyError))
}

// TIMEOUT uses a distinct event so subscribers can tell it apart from failure.
func (s *KeeperTestSuite) TestAckTimeout_DropsPending() {
	_, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)

	s.fireAck(sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT, "packet timed out")

	_, found, _ := s.keeper.GetPendingView(s.ctx, sampleAddress)
	s.False(found)

	s.NotNil(findEvent(s.ctx, types.EventTypeViewRegistrationTimedOut))
	s.Nil(findEvent(s.ctx, types.EventTypeViewRegistrationFailed))
}

// REGISTER_OBJECT acks for other resources (e.g. "host") must be no-ops.
func (s *KeeperTestSuite) TestAck_IgnoresOtherResources() {
	_, err := s.keeper.RegisterView(s.ctx, sampleName, sampleCreator, sampleAddress, sampleBundle)
	s.Require().NoError(err)

	meta := &sourcehubtypes.RegisterObjectMeta{ResourceName: "host", ObjectId: "irrelevant"}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)

	err = keeper.NewAckCallback(s.keeper).OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_REGISTER_OBJECT,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)

	_, found, _ := s.keeper.GetPendingView(s.ctx, sampleAddress)
	s.True(found)
}

// Late/replay ack with no pending entry is a silent no-op.
func (s *KeeperTestSuite) TestAck_NoPendingView_IsNoop() {
	err := s.fireAckFor("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS, "")
	s.Require().NoError(err)
}

// List/count reflect only finalized views — pending is an internal staging.
func (s *KeeperTestSuite) TestListAndCount_ExcludePending() {
	addrA := "0xaaaa000000000000000000000000000000000000"
	_, err := s.keeper.RegisterView(s.ctx, "ViewA", sampleCreator, addrA, sampleBundle)
	s.Require().NoError(err)
	s.fireAckFor(addrA, sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS, "")

	addrB := "0xbbbb000000000000000000000000000000000000"
	_, err = s.keeper.RegisterView(s.ctx, "ViewB", sampleCreator, addrB, sampleBundle)
	s.Require().NoError(err)

	views, _, err := s.keeper.GetAllViews(s.ctx, &query.PageRequest{Limit: 100})
	s.Require().NoError(err)
	s.Require().Len(views, 1)
	s.Equal("ViewA", views[0].Name)

	s.Equal(uint64(1), s.keeper.GetViewCount(s.ctx))
}

// Genesis InitGenesis ↔ ExportGenesis is a faithful round-trip.
func (s *KeeperTestSuite) TestGenesis_RoundTrip() {
	in := &types.GenesisState{
		Views: []types.View{
			{Name: "A", Address: "0xaaaa000000000000000000000000000000000000", Creator: sampleCreator, Height: 1},
			{Name: "B", Address: "0xbbbb000000000000000000000000000000000000", Creator: sampleCreator, Height: 2},
		},
	}
	s.keeper.InitGenesis(s.ctx, *in)
	out := s.keeper.ExportGenesis(s.ctx)
	s.Require().Len(out.Views, 2)
	s.Equal(uint64(2), s.keeper.GetViewCount(s.ctx))
}

func (s *KeeperTestSuite) TestQueryServer_Views() {
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "V1", Creator: "c1", Height: 5})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x2", Name: "V2", Creator: "c2", Height: 15})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 100},
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 2)
}

func (s *KeeperTestSuite) TestQueryServer_Views_SinceBlock() {
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "V1", Height: 5})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x2", Name: "V2", Height: 15})

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
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "V1", Data: []byte("bigdata")})

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
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "V1", Data: []byte("bigdata")})

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
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "V1", Creator: "c1"})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x2", Name: "V2", Creator: "c1"})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x3", Name: "V2", Creator: "c2"})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 100},
		Name:       "v2",
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
		Name:       "2",
		Creator:    "c2",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Equal("0x3", resp.Views[0].Address)
}

func (s *KeeperTestSuite) TestQueryServer_Views_FilterBeforePagination() {
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "Alpha"})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x2", Name: "NeedleOne"})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x3", Name: "NeedleTwo"})

	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Limit: 1},
		Name:       "needle",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Equal("NeedleOne", resp.Views[0].Name)
	s.Require().NotEmpty(resp.Pagination.NextKey)

	resp, err = qs.Views(s.ctx, &types.QueryViewsRequest{
		Pagination: &query.PageRequest{Key: resp.Pagination.NextKey, Limit: 1},
		Name:       "needle",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Views, 1)
	s.Require().Equal("NeedleTwo", resp.Views[0].Name)
	s.Require().Empty(resp.Pagination.NextKey)
}

func (s *KeeperTestSuite) TestQueryServer_Views_IncludeMetadata() {
	viewQuery := "Ethereum__Mainnet__Log { address blockNumber }"
	viewSdl := "type TestRoot @materialized(if: false) { address: String blockNumber: Int }"
	lensWasm := []byte("lens-one")
	lensArgs := `{"token":"0x1"}`
	bundle := mustBuildViewBundle(s.T(), viewQuery, viewSdl, testLens(lensWasm, lensArgs))

	_ = s.keeper.SetView(s.ctx, types.View{
		Address: "0x1",
		Name:    "TestRoot",
		Creator: "c1",
		Data:    bundle,
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

	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "FirstRoot", Data: firstBundle})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x2", Name: "SecondRoot", Data: secondBundle})

	qs := keeper.NewQueryServerImpl(s.keeper)
	assertSingleMatch := func(req *types.QueryViewsRequest) {
		req.Pagination = &query.PageRequest{Limit: 100}
		resp, err := qs.Views(s.ctx, req)
		s.Require().NoError(err)
		s.Require().Len(resp.Views, 1)
		s.Require().Equal("0x1", resp.Views[0].Address)
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

	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "FilterRoot", Data: bundle})

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
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1", Name: "Broken", Data: []byte("not-a-viewbundle")})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x2", Name: "Valid", Data: validBundle})

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
	s.Require().Equal("0x2", resp.Views[0].Address)
}

func (s *KeeperTestSuite) TestQueryServer_View_Found() {
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0xabc", Name: "V1", Creator: "cosmos1x"})

	qs := keeper.NewQueryServerImpl(s.keeper)
	resp, err := qs.View(s.ctx, &types.QueryViewRequest{ContractAddress: "0xabc"})
	s.Require().NoError(err)
	s.Require().Equal("V1", resp.View.Name)
}

func (s *KeeperTestSuite) TestQueryServer_View_IncludeMetadata() {
	viewQuery := "Ethereum__Mainnet__Log { address }"
	viewSdl := "type SingleRoot @materialized(if: false) { address: String }"
	bundle := mustBuildViewBundle(s.T(), viewQuery, viewSdl, testLens([]byte("single-lens"), "{}"))

	_ = s.keeper.SetView(s.ctx, types.View{Address: "0xabc", Name: "SingleRoot", Data: bundle})

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
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x1"})
	_ = s.keeper.SetView(s.ctx, types.View{Address: "0x2"})

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

func (s *KeeperTestSuite) fireAck(status sourcehubtypes.RequestStatus, errMsg string) {
	s.Require().NoError(s.fireAckFor(sampleAddress, status, errMsg))
}

func (s *KeeperTestSuite) fireAckFor(address string, status sourcehubtypes.RequestStatus, errMsg string) error {
	meta := &sourcehubtypes.RegisterObjectMeta{
		ResourceName: sourcehubtypes.ViewResourceName,
		ObjectId:     address,
	}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	return keeper.NewAckCallback(s.keeper).OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_REGISTER_OBJECT,
		Meta:   metaBz,
		Status: status,
		Error:  errMsg,
	})
}

func sdkErr(msg string) error { return &simpleErr{msg} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }
