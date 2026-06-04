package keeper_test

import (
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
	"github.com/stretchr/testify/require"

	poolkeeper "github.com/shinzonetwork/shinzohub/x/pool/keeper"
	"github.com/shinzonetwork/shinzohub/x/pool/types"
	viewtypes "github.com/shinzonetwork/shinzohub/x/view/types"
)

// fixture wires up a fresh keeper backed by an in-memory store and a mock view
// keeper. Use registerView() to make CreatePool happy.
type fixture struct {
	t      *testing.T
	ctx    sdk.Context
	keeper poolkeeper.Keeper
	views  *mockViewKeeper
}

type mockViewKeeper struct {
	views map[string]viewtypes.View
}

func (m *mockViewKeeper) GetView(_ sdk.Context, addr string) (viewtypes.View, bool, error) {
	v, ok := m.views[addr]
	return v, ok, nil
}

func (m *mockViewKeeper) registerView(addr string) {
	m.views[addr] = viewtypes.View{Address: addr, Name: "test-view"}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	views := &mockViewKeeper{views: map[string]viewtypes.View{}}

	k := poolkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		views,
		nil, // bankKeeper isn't used by the keeper itself
		"",
	)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, cosmoslog.NewNopLogger())

	return &fixture{t: t, ctx: ctx, keeper: k, views: views}
}

// Common config so we don't repeat literals everywhere.
func cfg() types.PoolConfig { return types.PoolConfig{WindowSize: 200_000} }

func TestCreatePool_PersistsAndIncrementsCount(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")

	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	p, found, err := f.keeper.GetPool(f.ctx, "0xpool")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "0xview", p.ViewAddress)
	require.Equal(t, uint64(200_000), p.Config.WindowSize)
	require.Equal(t, uint64(1), f.keeper.GetPoolCount(f.ctx))
}

func TestCreatePool_RejectsDuplicate(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")

	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))
	err := f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg())
	require.ErrorContains(t, err, "already exists")
}

func TestCreatePool_RejectsUnknownView(t *testing.T) {
	f := newFixture(t)
	// no view registered

	err := f.keeper.CreatePool(f.ctx, "0xpool", "0xview-doesnt-exist", cfg())
	require.ErrorContains(t, err, "not registered")
}

func TestCreatePool_EmitsEvent(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")

	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	events := f.ctx.EventManager().Events()
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	require.Equal(t, types.EventTypePoolCreated, last.Type)
}

func TestPoolExists(t *testing.T) {
	f := newFixture(t)
	require.False(t, f.keeper.PoolExists(f.ctx, "0xpool"))

	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))
	require.True(t, f.keeper.PoolExists(f.ctx, "0xpool"))
}

func TestGetPoolsForView_ListsAllForOneView(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")

	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpoolA", "0xview", cfg()))
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpoolB", "0xview", types.PoolConfig{WindowSize: 100_000}))

	pools, err := f.keeper.GetPoolsForView(f.ctx, "0xview")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"0xpoolA", "0xpoolB"}, pools)
}

func TestGetPoolsForView_EmptyForUnknown(t *testing.T) {
	f := newFixture(t)
	pools, err := f.keeper.GetPoolsForView(f.ctx, "0xnope")
	require.NoError(t, err)
	require.Empty(t, pools)
}

func TestAddHost_AppearsInIteration(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	require.NoError(t, f.keeper.AddHost(f.ctx, "0xpool", "0xhost"))

	h, found, err := f.keeper.GetHost(f.ctx, "0xpool", "0xhost")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "0", h.Ask) // default until SetHostAsk
}

func TestAddHost_RejectsDuplicate(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	require.NoError(t, f.keeper.AddHost(f.ctx, "0xpool", "0xhost"))
	err := f.keeper.AddHost(f.ctx, "0xpool", "0xhost")
	require.ErrorContains(t, err, "already in pool")
}

func TestAddHost_RejectsUnknownPool(t *testing.T) {
	f := newFixture(t)
	err := f.keeper.AddHost(f.ctx, "0xnope", "0xhost")
	require.ErrorContains(t, err, "pool not found")
}

func TestSetHostAsk_UpdatesPrice(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))
	require.NoError(t, f.keeper.AddHost(f.ctx, "0xpool", "0xhost"))

	require.NoError(t, f.keeper.SetHostAsk(f.ctx, "0xpool", "0xhost", "1234"))

	h, _, err := f.keeper.GetHost(f.ctx, "0xpool", "0xhost")
	require.NoError(t, err)
	require.Equal(t, "1234", h.Ask)
}

func TestSetHostAsk_RejectsUnknownHost(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	err := f.keeper.SetHostAsk(f.ctx, "0xpool", "0xstranger", "100")
	require.ErrorContains(t, err, "host not in pool")
}

func TestRemoveHost_DropsTheEntry(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))
	require.NoError(t, f.keeper.AddHost(f.ctx, "0xpool", "0xhost"))

	require.NoError(t, f.keeper.RemoveHost(f.ctx, "0xpool", "0xhost"))

	_, found, err := f.keeper.GetHost(f.ctx, "0xpool", "0xhost")
	require.NoError(t, err)
	require.False(t, found)
}

// ---------- Demand ----------

func TestRegisterDemand_PersistsEntry(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	d := types.PoolDemand{
		Bond:      "100",
		PricePref: "200",
		Binding:   true,
		ExpiresAt: 1_000,
	}
	require.NoError(t, f.keeper.RegisterDemand(f.ctx, "0xpool", "0xdev", d))

	got, found, err := f.keeper.GetDemand(f.ctx, "0xpool", "0xdev")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, d, got)
}

func TestRegisterDemand_RejectsDuplicate(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	d := types.PoolDemand{Bond: "100", ExpiresAt: 1_000}
	require.NoError(t, f.keeper.RegisterDemand(f.ctx, "0xpool", "0xdev", d))

	err := f.keeper.RegisterDemand(f.ctx, "0xpool", "0xdev", d)
	require.ErrorContains(t, err, "already registered")
}

func TestRegisterDemand_RejectsUnknownPool(t *testing.T) {
	f := newFixture(t)
	err := f.keeper.RegisterDemand(f.ctx, "0xnope", "0xdev", types.PoolDemand{Bond: "1"})
	require.ErrorContains(t, err, "pool not found")
}

func TestGetPoolDetail_BundlesEverything(t *testing.T) {
	f := newFixture(t)
	f.views.registerView("0xview")
	require.NoError(t, f.keeper.CreatePool(f.ctx, "0xpool", "0xview", cfg()))

	require.NoError(t, f.keeper.AddHost(f.ctx, "0xpool", "0xhost"))
	require.NoError(t, f.keeper.RegisterDemand(f.ctx, "0xpool", "0xdev",
		types.PoolDemand{Bond: "500", ExpiresAt: 1_000}))

	detail, found, err := f.keeper.GetPoolDetail(f.ctx, "0xpool")
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, "0xpool", detail.Pool.PoolAddress)
	require.Len(t, detail.Hosts, 1)
	require.Equal(t, "0xhost", detail.Hosts[0].HostAddress)
	require.Len(t, detail.Demands, 1)
	require.Equal(t, "0xdev", detail.Demands[0].RegistrantAddress)
	require.Equal(t, "500", detail.Demands[0].Demand.Bond)
}

func TestGetPoolDetail_MissingPoolReturnsFalse(t *testing.T) {
	f := newFixture(t)
	_, found, err := f.keeper.GetPoolDetail(f.ctx, "0xnope")
	require.NoError(t, err)
	require.False(t, found)
}

func TestGenesis_RoundTrip(t *testing.T) {
	src := newFixture(t)
	src.views.registerView("0xview")
	require.NoError(t, src.keeper.CreatePool(src.ctx, "0xpool", "0xview", cfg()))
	require.NoError(t, src.keeper.AddHost(src.ctx, "0xpool", "0xhost"))
	require.NoError(t, src.keeper.SetHostAsk(src.ctx, "0xpool", "0xhost", "999"))
	require.NoError(t, src.keeper.RegisterDemand(src.ctx, "0xpool", "0xdev",
		types.PoolDemand{Bond: "100", ExpiresAt: 1_000}))

	exported := src.keeper.ExportGenesis(src.ctx)

	// Import into a fresh keeper and confirm the state matches.
	dst := newFixture(t)
	dst.keeper.InitGenesis(dst.ctx, *exported)

	p, found, _ := dst.keeper.GetPool(dst.ctx, "0xpool")
	require.True(t, found)
	require.Equal(t, "0xview", p.ViewAddress)

	h, found, _ := dst.keeper.GetHost(dst.ctx, "0xpool", "0xhost")
	require.True(t, found)
	require.Equal(t, "999", h.Ask)

	d, found, _ := dst.keeper.GetDemand(dst.ctx, "0xpool", "0xdev")
	require.True(t, found)
	require.Equal(t, "100", d.Bond)
}
