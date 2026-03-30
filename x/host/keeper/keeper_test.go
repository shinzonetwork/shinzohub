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
	dbm "github.com/cosmos/cosmos-db"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/shinzonetwork/shinzohub/x/host/keeper"
	"github.com/shinzonetwork/shinzohub/x/host/types"
)

type mockSourcehubKeeper struct {
	called    bool
	lastDID   string
	lastGroup string
	err       error
}

func (m *mockSourcehubKeeper) SendICASetRelationship(ctx sdk.Context, did string, group string) error {
	m.called = true
	m.lastDID = did
	m.lastGroup = group
	return m.err
}

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	keeper        keeper.Keeper
	mockSourcehub *mockSourcehubKeeper
	cdc           codec.BinaryCodec
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

	storeService := runtime.NewKVStoreService(storeKey)

	s.keeper = keeper.NewKeeper(
		cdc,
		storeService,
		s.mockSourcehub,
		"authority",
	)

	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, cosmoslog.NewNopLogger())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestRegisterHost_Success() {
	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	did, err := s.keeper.RegisterHost(s.ctx, "192.168.1.1:8080", callerAddr)
	s.Require().NoError(err)
	s.Require().NotEmpty(did)

	s.Require().True(s.mockSourcehub.called)
	s.Require().Equal("host", s.mockSourcehub.lastGroup)

	s.Require().True(s.keeper.IsRegisteredHost(s.ctx, callerAddr))

	gotDID, found := s.keeper.GetDIDForAddress(s.ctx, callerAddr)
	s.Require().True(found)
	s.Require().Equal(did, gotDID)

	bech32Addr := sdk.AccAddress(callerAddr).String()
	host, found, err := s.keeper.GetHost(s.ctx, bech32Addr)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(bech32Addr, host.Address)
	s.Require().Equal(string(did), host.Did)
	s.Require().Equal("192.168.1.1:8080", host.ConnectionString)

	s.Require().Equal(uint64(1), s.keeper.GetHostCount(s.ctx))
}

func (s *KeeperTestSuite) TestRegisterHost_DuplicateSameKeys_Idempotent() {
	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	_, err := s.keeper.RegisterHost(s.ctx, "192.168.1.1:8080", callerAddr)
	s.Require().NoError(err)

	_, err = s.keeper.RegisterHost(s.ctx, "192.168.1.2:9090", callerAddr)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestRegisterHost_ICAFailure_Propagates() {
	s.mockSourcehub.err = fmt.Errorf("ICA channel not open")

	callerAddr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	_, err := s.keeper.RegisterHost(s.ctx, "192.168.1.1:8080", callerAddr)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "ICA channel not open")
}

func (s *KeeperTestSuite) TestIsRegisteredHost_NotRegistered() {
	callerAddr := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd}
	s.Require().False(s.keeper.IsRegisteredHost(s.ctx, callerAddr))
}

func (s *KeeperTestSuite) TestGetDIDForAddress_NotFound() {
	callerAddr := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd}
	_, found := s.keeper.GetDIDForAddress(s.ctx, callerAddr)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestSetHost_GetHost() {
	host := types.Host{
		Address:          "shinzo1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tpw",
		Did:              "did:key:z6Mk...",
		ConnectionString: "10.0.0.1:8080",
	}

	err := s.keeper.SetHost(s.ctx, host)
	s.Require().NoError(err)

	got, found, err := s.keeper.GetHost(s.ctx, host.Address)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(host.Address, got.Address)
	s.Require().Equal(host.Did, got.Did)
	s.Require().Equal(host.ConnectionString, got.ConnectionString)
}

func (s *KeeperTestSuite) TestGetHost_NotFound() {
	_, found, err := s.keeper.GetHost(s.ctx, "shinzo1nonexistent")
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestGetHostCount_Empty() {
	s.Require().Equal(uint64(0), s.keeper.GetHostCount(s.ctx))
}

func (s *KeeperTestSuite) TestGetHostCount_AfterMultipleHosts() {
	for i := 0; i < 3; i++ {
		err := s.keeper.SetHost(s.ctx, types.Host{
			Address:          fmt.Sprintf("shinzo1host%d", i),
			Did:              fmt.Sprintf("did:key:z%d", i),
			ConnectionString: fmt.Sprintf("10.0.0.%d:8080", i),
		})
		s.Require().NoError(err)
	}
	s.Require().Equal(uint64(3), s.keeper.GetHostCount(s.ctx))
}

func (s *KeeperTestSuite) TestSetHost_UpdateDoesNotIncrementCount() {
	host := types.Host{
		Address:          "shinzo1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tpw",
		Did:              "did:key:z6Mk...",
		ConnectionString: "10.0.0.1:8080",
	}

	err := s.keeper.SetHost(s.ctx, host)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), s.keeper.GetHostCount(s.ctx))

	host.ConnectionString = "10.0.0.2:9090"
	err = s.keeper.SetHost(s.ctx, host)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), s.keeper.GetHostCount(s.ctx))
}

func (s *KeeperTestSuite) TestGetAllHosts_Empty() {
	hosts, _, err := s.keeper.GetAllHosts(s.ctx, nil)
	s.Require().NoError(err)
	s.Require().Empty(hosts)
}

func (s *KeeperTestSuite) TestGetAllHosts_ReturnsAll() {
	for i := 0; i < 5; i++ {
		err := s.keeper.SetHost(s.ctx, types.Host{
			Address:          fmt.Sprintf("shinzo1host%d", i),
			Did:              fmt.Sprintf("did:key:z%d", i),
			ConnectionString: fmt.Sprintf("10.0.0.%d:8080", i),
		})
		s.Require().NoError(err)
	}

	hosts, _, err := s.keeper.GetAllHosts(s.ctx, nil)
	s.Require().NoError(err)
	s.Require().Len(hosts, 5)
}

func (s *KeeperTestSuite) TestGenesis_InitExportRoundtrip() {
	genesis := types.GenesisState{
		Hosts: []types.Host{
			{Address: "shinzo1host0", Did: "did:key:z0", ConnectionString: "10.0.0.1:8080"},
			{Address: "shinzo1host1", Did: "did:key:z1", ConnectionString: "10.0.0.2:8080"},
		},
	}

	s.keeper.InitGenesis(s.ctx, genesis)

	exported := s.keeper.ExportGenesis(s.ctx)
	s.Require().Len(exported.Hosts, 2)

	addrSet := make(map[string]bool)
	for _, h := range exported.Hosts {
		addrSet[h.Address] = true
	}
	s.Require().True(addrSet["shinzo1host0"])
	s.Require().True(addrSet["shinzo1host1"])
}

func (s *KeeperTestSuite) TestGenesis_InitSetsCount() {
	genesis := types.GenesisState{
		Hosts: []types.Host{
			{Address: "shinzo1host0", Did: "did:key:z0", ConnectionString: "10.0.0.1:8080"},
			{Address: "shinzo1host1", Did: "did:key:z1", ConnectionString: "10.0.0.2:8080"},
			{Address: "shinzo1host2", Did: "did:key:z2", ConnectionString: "10.0.0.3:8080"},
		},
	}

	s.keeper.InitGenesis(s.ctx, genesis)
	s.Require().Equal(uint64(3), s.keeper.GetHostCount(s.ctx))
}

func (s *KeeperTestSuite) TestQueryServer_HostCount() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	resp, err := qs.HostCount(s.ctx, &types.QueryHostCountRequest{})
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), resp.Count)

	_ = s.keeper.SetHost(s.ctx, types.Host{Address: "shinzo1host0", Did: "did:0", ConnectionString: "10.0.0.1:8080"})

	resp, err = qs.HostCount(s.ctx, &types.QueryHostCountRequest{})
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), resp.Count)
}

func (s *KeeperTestSuite) TestQueryServer_Host_Found() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	_ = s.keeper.SetHost(s.ctx, types.Host{Address: "shinzo1host0", Did: "did:0", ConnectionString: "10.0.0.1:8080"})

	resp, err := qs.Host(s.ctx, &types.QueryHostRequest{Address: "shinzo1host0"})
	s.Require().NoError(err)
	s.Require().Equal("shinzo1host0", resp.Host.Address)
}

func (s *KeeperTestSuite) TestQueryServer_Host_NotFound() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	_, err := qs.Host(s.ctx, &types.QueryHostRequest{Address: "shinzo1nonexistent"})
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestQueryServer_Hosts_Paginated() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	for i := 0; i < 5; i++ {
		_ = s.keeper.SetHost(s.ctx, types.Host{
			Address:          fmt.Sprintf("shinzo1host%d", i),
			Did:              fmt.Sprintf("did:%d", i),
			ConnectionString: fmt.Sprintf("10.0.0.%d:8080", i),
		})
	}

	resp, err := qs.Hosts(s.ctx, &types.QueryHostsRequest{})
	s.Require().NoError(err)
	s.Require().Len(resp.Hosts, 5)
}

func (s *KeeperTestSuite) TestQueryServer_NilRequest() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	_, err := qs.Hosts(s.ctx, nil)
	s.Require().Error(err)

	_, err = qs.Host(s.ctx, nil)
	s.Require().Error(err)

	_, err = qs.HostCount(s.ctx, nil)
	s.Require().Error(err)
}
