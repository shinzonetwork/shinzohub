package viewregistry_test

import (
	"fmt"
	"math/big"
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
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	viewbundle "github.com/shinzonetwork/viewbundle-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/shinzonetwork/shinzohub/app/precompiles/viewregistry"
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
	viewkeeper "github.com/shinzonetwork/shinzohub/x/view/keeper"
	viewtypes "github.com/shinzonetwork/shinzohub/x/view/types"
)

const (
	statusNone       uint8 = 0
	statusPending    uint8 = 1
	statusRegistered uint8 = 2
)

// Records the ICA call and can inject an error to test revert behavior.
type mockSourcehubKeeper struct {
	calls int
	err   error
}

func (m *mockSourcehubKeeper) RegisterObject(_ sdk.Context, _, _ string) (uint64, string, string, error) {
	m.calls++
	if m.err != nil {
		return 0, "", "", m.err
	}
	return 1, "icacontroller-test", "channel-0", nil
}

// Captures every log the precompile emits.
type mockStateDB struct {
	vm.StateDB
	logs []*gethtypes.Log
}

func (m *mockStateDB) AddLog(log *gethtypes.Log) { m.logs = append(m.logs, log) }

type PrecompileTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	precompile    *viewregistry.Precompile
	viewKeeper    viewkeeper.Keeper
	mockSourcehub *mockSourcehubKeeper
	stateDB       *mockStateDB
	cdc           codec.Codec
}

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) SetupTest() {
	s.mockSourcehub = &mockSourcehubKeeper{}

	storeKey := storetypes.NewKVStoreKey(viewtypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	s.cdc = cdc

	s.viewKeeper = viewkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		s.mockSourcehub,
	)
	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{Height: 11}, false, cosmoslog.NewNopLogger())
	s.stateDB = &mockStateDB{}

	p, err := viewregistry.NewPrecompile(10_000, s.viewKeeper)
	require.NoError(s.T(), err)
	s.precompile = p
}

// Minimal valid viewbundle whose SDL declares `type <name>`.
func buildBundle(name string) []byte {
	h := viewbundle.DecodedHeader{
		Header: viewbundle.Header{
			Query: "Log { address }",
			Sdl:   fmt.Sprintf("type %s @materialized(if: false) { x: String }", name),
		},
	}
	bz, err := viewbundle.EncodeHeader(h)
	if err != nil {
		panic(err)
	}
	return bz
}

func makeContract(caller common.Address) *vm.Contract {
	return vm.NewContract(
		caller,
		common.HexToAddress(viewregistry.PrecompileAddress),
		uint256.NewInt(0),
		1_000_000,
		nil,
	)
}

// Calls register and returns the synthesized viewAddr.
func (s *PrecompileTestSuite) register(caller common.Address, bundle []byte) common.Address {
	contract := makeContract(caller)
	method := s.precompile.ABI.Methods["register"]
	out, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{bundle})
	s.Require().NoError(err)

	values, err := method.Outputs.Unpack(out)
	s.Require().NoError(err)
	return values[0].(common.Address)
}

type viewOut struct {
	ViewAddress common.Address
	Name        string
	Creator     string
	Height      uint64
	Status      uint8
}

func (s *PrecompileTestSuite) getView(addr common.Address) viewOut {
	method := s.precompile.ABI.Methods["getView"]
	out, err := s.precompile.GetView(s.ctx, &method, []interface{}{addr})
	s.Require().NoError(err)

	values, err := method.Outputs.Unpack(out)
	s.Require().NoError(err)
	raw := values[0].(struct {
		ViewAddress common.Address `json:"viewAddress"`
		Name        string         `json:"name"`
		Creator     string         `json:"creator"`
		Height      uint64         `json:"height"`
		Status      uint8          `json:"status"`
	})
	return viewOut(raw)
}

// Simulates the IBC ack landing so tests can observe pending → registered.
func (s *PrecompileTestSuite) fireAck(viewAddr common.Address, status sourcehubtypes.RequestStatus) {
	meta := &sourcehubtypes.RegisterObjectMeta{
		ResourceName: sourcehubtypes.ViewResourceName,
		ObjectId:     viewAddr.Hex(),
	}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	err = viewkeeper.NewAckCallback(s.viewKeeper).OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_REGISTER_OBJECT,
		Meta:   metaBz,
		Status: status,
	})
	s.Require().NoError(err)
}

// Happy path: returns (viewAddr, name), emits ViewCreated log, view is PENDING.
func (s *PrecompileTestSuite) TestRegister_HappyPath_ViewIsPending() {
	caller := common.HexToAddress("0xabCDef1234567890abcdEF1234567890ABCDEF12")
	bundle := buildBundle("ViewHappy")

	viewAddr := s.register(caller, bundle)
	s.Require().NotEqual(common.Address{}, viewAddr)

	s.Require().Len(s.stateDB.logs, 1)
	log := s.stateDB.logs[0]
	s.Equal(common.HexToAddress(viewregistry.PrecompileAddress), log.Address)
	s.Equal(common.BytesToHash(viewAddr.Bytes()), log.Topics[1])
	s.Equal(common.BytesToHash(caller.Bytes()), log.Topics[2])

	s.Equal(1, s.mockSourcehub.calls)
	got := s.getView(viewAddr)
	s.Equal(statusPending, got.Status)
	s.Equal("ViewHappy", got.Name)
	s.Equal(caller.Hex(), got.Creator, "creator must be EVM hex, not bech32")
	s.Equal(uint64(11), got.Height)
}

// After SUCCESS ack: status flips to REGISTERED, view appears in listViews/count.
func (s *PrecompileTestSuite) TestRegister_AfterAck_ViewIsRegistered() {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	viewAddr := s.register(caller, buildBundle("ViewAcked"))

	s.fireAck(viewAddr, sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS)

	got := s.getView(viewAddr)
	s.Equal(statusRegistered, got.Status)

	// listViews + viewCount must reflect the freshly promoted view.
	listMethod := s.precompile.ABI.Methods["listViews"]
	bz, err := s.precompile.ListViews(s.ctx, &listMethod, []interface{}{
		big.NewInt(0), big.NewInt(10),
	})
	s.Require().NoError(err)
	values, err := listMethod.Outputs.Unpack(bz)
	s.Require().NoError(err)
	tuples := values[0].([]struct {
		ViewAddress common.Address `json:"viewAddress"`
		Name        string         `json:"name"`
		Creator     string         `json:"creator"`
		Height      uint64         `json:"height"`
		Status      uint8          `json:"status"`
	})
	s.Require().Len(tuples, 1)
	s.Equal(viewAddr, tuples[0].ViewAddress)

	countMethod := s.precompile.ABI.Methods["viewCount"]
	bz, err = s.precompile.ViewCount(s.ctx, &countMethod)
	s.Require().NoError(err)
	values, err = countMethod.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Equal(uint64(1), values[0].(*big.Int).Uint64())
}

// Unknown address returns status NONE so callers can tell it apart from pending.
func (s *PrecompileTestSuite) TestGetView_UnknownAddress_ReturnsNone() {
	got := s.getView(common.HexToAddress("0xdeadbeef00000000000000000000000000000000"))
	s.Equal(statusNone, got.Status)
	s.Empty(got.Name)
}

// listViews hides pending views; they remain reachable via getView only.
func (s *PrecompileTestSuite) TestListViews_ExcludesPending() {
	caller := common.HexToAddress("0x2222222222222222222222222222222222222222")
	pendingAddr := s.register(caller, buildBundle("ViewPending"))
	registeredAddr := s.register(caller, buildBundle("ViewRegistered"))
	s.fireAck(registeredAddr, sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS)

	s.Equal(statusPending, s.getView(pendingAddr).Status)

	method := s.precompile.ABI.Methods["listViews"]
	bz, err := s.precompile.ListViews(s.ctx, &method, []interface{}{
		big.NewInt(0), big.NewInt(10),
	})
	s.Require().NoError(err)
	values, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	tuples := values[0].([]struct {
		ViewAddress common.Address `json:"viewAddress"`
		Name        string         `json:"name"`
		Creator     string         `json:"creator"`
		Height      uint64         `json:"height"`
		Status      uint8          `json:"status"`
	})
	s.Require().Len(tuples, 1)
	s.Equal(registeredAddr, tuples[0].ViewAddress)
}

// Garbage bytes → decode error, no log, no sourcehub call.
func (s *PrecompileTestSuite) TestRegister_InvalidBundle_Errors() {
	caller := common.HexToAddress("0x3333333333333333333333333333333333333333")
	contract := makeContract(caller)
	method := s.precompile.ABI.Methods["register"]

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		[]byte("not-a-valid-viewbundle"),
	})
	s.Require().Error(err)
	s.Empty(s.stateDB.logs)
	s.Equal(0, s.mockSourcehub.calls)
}

// SDL without `type X` is rejected — there's no name to register under.
func (s *PrecompileTestSuite) TestRegister_SDLMissingTypeName_Errors() {
	h := viewbundle.DecodedHeader{
		Header: viewbundle.Header{
			Query: "Log { address }",
			Sdl:   "scalar Time",
		},
	}
	bz, err := viewbundle.EncodeHeader(h)
	s.Require().NoError(err)

	caller := common.HexToAddress("0x4444444444444444444444444444444444444444")
	contract := makeContract(caller)
	method := s.precompile.ABI.Methods["register"]

	_, err = s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{bz})
	s.Require().Error(err)
	s.Contains(err.Error(), "SDL missing type name")
	s.Empty(s.stateDB.logs)
}

// Sourcehub ICA error surfaces to the EVM caller; no log emitted.
func (s *PrecompileTestSuite) TestRegister_SourcehubFailure_RevertsCleanly() {
	s.mockSourcehub.err = fmt.Errorf("no active ICA channel")

	caller := common.HexToAddress("0x5555555555555555555555555555555555555555")
	contract := makeContract(caller)
	method := s.precompile.ABI.Methods["register"]

	_, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{
		buildBundle("ViewICAFail"),
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "no active ICA channel")
	s.Empty(s.stateDB.logs)
}

// viewAddr = keccak256("shinzo.view.v1" || caller || bundle)[12:] — different
// caller or different bundle ⇒ different address.
func (s *PrecompileTestSuite) TestViewAddress_IsDeterministicPerCallerAndBundle() {
	bundle := buildBundle("ViewDet")
	callerA := common.HexToAddress("0x6666666666666666666666666666666666666666")
	callerB := common.HexToAddress("0x7777777777777777777777777777777777777777")

	addrA := s.register(callerA, bundle)
	addrB := s.register(callerB, bundle)
	s.NotEqual(addrA, addrB)

	addrA2 := s.register(callerA, buildBundle("ViewDifferentSDL"))
	s.NotEqual(addrA, addrA2)
}

// The SDL name is taken from the real type declaration, not from a `type <name>`
// mentioned in a leading comment.
func (s *PrecompileTestSuite) TestRegister_SDLNameIgnoresLeadingComment() {
	h := viewbundle.DecodedHeader{
		Header: viewbundle.Header{
			Query: "Log { address }",
			Sdl:   "# please rename the type Legacy later\ntype RealView @materialized(if: false) { x: String }",
		},
	}
	bz, err := viewbundle.EncodeHeader(h)
	s.Require().NoError(err)

	contract := makeContract(common.HexToAddress("0x8888888888888888888888888888888888888888"))
	method := s.precompile.ABI.Methods["register"]
	out, err := s.precompile.Register(s.ctx, contract, s.stateDB, &method, []interface{}{bz})
	s.Require().NoError(err)

	values, err := method.Outputs.Unpack(out)
	s.Require().NoError(err)
	s.Equal("RealView", values[1].(string))
}

// listViews(_, 0) returns an empty page, not the cosmos default page size.
func (s *PrecompileTestSuite) TestListViews_ZeroLimit_ReturnsEmpty() {
	caller := common.HexToAddress("0x9999999999999999999999999999999999999999")
	viewAddr := s.register(caller, buildBundle("ViewZeroLimit"))
	s.fireAck(viewAddr, sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS)

	method := s.precompile.ABI.Methods["listViews"]
	bz, err := s.precompile.ListViews(s.ctx, &method, []interface{}{
		big.NewInt(0), big.NewInt(0),
	})
	s.Require().NoError(err)
	values, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	tuples := values[0].([]struct {
		ViewAddress common.Address `json:"viewAddress"`
		Name        string         `json:"name"`
		Creator     string         `json:"creator"`
		Height      uint64         `json:"height"`
		Status      uint8          `json:"status"`
	})
	s.Require().Len(tuples, 0)
}
