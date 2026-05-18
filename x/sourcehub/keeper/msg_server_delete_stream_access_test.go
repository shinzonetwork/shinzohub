package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/keeper"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

// stubAdminKeeper lets a test control whether the signer is treated as
// admin without standing up the real admin module.
type stubAdminKeeper struct {
	isAdmin bool
}

func (s stubAdminKeeper) IsAdmin(_ sdk.Context, _ string) bool { return s.isAdmin }

func setupKeeperWithAdmin(t *testing.T, isAdmin bool) (keeper.Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	storeService := runtime.NewKVStoreService(storeKey)

	k := keeper.NewKeeper(cdc, storeService, nil, stubAdminKeeper{isAdmin: isAdmin})
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, cosmoslog.NewNopLogger())
	return k, ctx
}

// testSigner is a valid Bech32 address under the SDK default config
// (cosmos prefix). The production binary registers a "shinzo" prefix in
// cmd init; this test package does not run that init, so we use the
// default to satisfy AccAddressFromBech32.
var testSigner = sdk.AccAddress(make([]byte, 20)).String()

const (
	testStreamID = "0xc5d55f9a4e8788abaaf74d4772c2a4afe60a23a3"
	testDID      = "did:key:zQ3sSubscriber"
)

// DeleteStreamAccess must reject a non-admin signer before doing any
// ICA work. The happy path requires a live ICA channel and is
// exercised end-to-end against a deployed environment, not here.
func TestMsgServer_DeleteStreamAccess_NonAdminDenied(t *testing.T) {
	k, ctx := setupKeeperWithAdmin(t, false)
	srv := keeper.NewMsgServerImpl(k)

	msg := &types.MsgDeleteStreamAccess{
		Signer:   testSigner,
		Resource: types.Resource_RESOURCE_VIEW,
		StreamId: testStreamID,
		Did:      testDID,
	}

	resp, err := srv.DeleteStreamAccess(ctx, msg)
	require.Error(t, err)
	require.True(t, sdkerrors.ErrUnauthorized.Is(err), "expected ErrUnauthorized, got %v", err)
	require.Nil(t, resp)
}

// When the admin check passes, the handler proceeds past admin and
// fails on the next gate (no connection ID set). The specific error
// proves the admin branch did not short-circuit and the ICA wiring is
// the next prerequisite the handler enforces.
func TestMsgServer_DeleteStreamAccess_AdminProceedsToICAStage(t *testing.T) {
	k, ctx := setupKeeperWithAdmin(t, true)
	srv := keeper.NewMsgServerImpl(k)

	msg := &types.MsgDeleteStreamAccess{
		Signer:   testSigner,
		Resource: types.Resource_RESOURCE_VIEW,
		StreamId: testStreamID,
		Did:      testDID,
	}

	_, err := srv.DeleteStreamAccess(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no connection ID set in module state")
}
