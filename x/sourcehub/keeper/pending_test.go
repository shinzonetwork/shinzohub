package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/keeper"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	storeService := runtime.NewKVStoreService(storeKey)

	k := keeper.NewKeeper(cdc, storeService, nil, nil)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, cosmoslog.NewNopLogger())
	return k, ctx
}

func TestPendingRequest_SetGet(t *testing.T) {
	k, ctx := setupKeeper(t)

	req := keeper.NewPendingICARequest(
		"icacontroller-sourcehub", "channel-0", 42,
		types.RequestKind_REQUEST_KIND_REGISTER_OBJECT,
		"shinzo1creator", ctx.BlockTime(),
		[]byte("meta"),
	)
	require.NoError(t, k.SetPendingRequest(ctx, req))

	got, found, err := k.GetPendingRequest(ctx, "icacontroller-sourcehub", "channel-0", 42)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_PENDING, got.Status)
	require.Equal(t, types.RequestKind_REQUEST_KIND_REGISTER_OBJECT, got.Kind)
	require.Equal(t, "shinzo1creator", got.Requestor)
	require.Equal(t, []byte("meta"), got.Meta)
}

func TestPendingRequest_GetMissing(t *testing.T) {
	k, ctx := setupKeeper(t)
	_, found, err := k.GetPendingRequest(ctx, "p", "c", 7)
	require.NoError(t, err)
	require.False(t, found)
}

func TestResolvePendingRequest_Success(t *testing.T) {
	k, ctx := setupKeeper(t)

	req := keeper.NewPendingICARequest("p", "c", 1, types.RequestKind_REQUEST_KIND_SET_RELATIONSHIP, "addr", ctx.BlockTime(), nil)
	require.NoError(t, k.SetPendingRequest(ctx, req))

	responses := [][]byte{[]byte("r1"), []byte("r2")}
	resolved, found, err := k.ResolvePendingRequest(ctx, "p", "c", 1, types.RequestStatus_REQUEST_STATUS_SUCCESS, "", responses)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_SUCCESS, resolved.Status)
	require.Empty(t, resolved.Error)
	require.Equal(t, responses, resolved.MsgResponses)

	got, _, _ := k.GetPendingRequest(ctx, "p", "c", 1)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_SUCCESS, got.Status)
}

func TestResolvePendingRequest_Failure(t *testing.T) {
	k, ctx := setupKeeper(t)

	req := keeper.NewPendingICARequest("p", "c", 1, types.RequestKind_REQUEST_KIND_SET_RELATIONSHIP, "addr", ctx.BlockTime(), nil)
	require.NoError(t, k.SetPendingRequest(ctx, req))

	resolved, found, err := k.ResolvePendingRequest(ctx, "p", "c", 1, types.RequestStatus_REQUEST_STATUS_FAILURE, "policy not found", nil)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_FAILURE, resolved.Status)
	require.Equal(t, "policy not found", resolved.Error)
}

func TestResolvePendingRequest_Missing(t *testing.T) {
	k, ctx := setupKeeper(t)
	_, found, err := k.ResolvePendingRequest(ctx, "p", "c", 99, types.RequestStatus_REQUEST_STATUS_SUCCESS, "", nil)
	require.NoError(t, err)
	require.False(t, found)
}

func TestRegisterAckCallback_FanOut(t *testing.T) {
	k, _ := setupKeeper(t)

	var calls []string
	cb1 := fakeCallback{name: "one", log: &calls}
	cb2 := fakeCallback{name: "two", log: &calls}

	k.RegisterAckCallback(types.RequestKind_REQUEST_KIND_SET_RELATIONSHIP, cb1)
	k.RegisterAckCallback(types.RequestKind_REQUEST_KIND_SET_RELATIONSHIP, cb2)

	out := k.GetAckCallbacks(types.RequestKind_REQUEST_KIND_SET_RELATIONSHIP)
	require.Len(t, out, 2)
	for _, cb := range out {
		_ = cb.OnPacketAck(sdk.Context{}, types.PendingICARequest{})
	}
	require.Equal(t, []string{"one", "two"}, calls)
}

type fakeCallback struct {
	name string
	log  *[]string
}

func (f fakeCallback) OnPacketAck(_ sdk.Context, _ types.PendingICARequest) error {
	*f.log = append(*f.log, f.name)
	return nil
}
