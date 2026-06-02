package keeper_test

import (
	"encoding/base64"
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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/shinzonetwork/shinzohub/x/view/keeper"
	"github.com/shinzonetwork/shinzohub/x/view/types"
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
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
