package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/keeper"
	"github.com/sourcenetwork/sourcehub/x/acp/testutil"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var DefaultTs = MustDateTimeToProto("2024-01-01 00:00:00")

var _ context.Context = (*TestCtx)(nil)

type ActorType int

const (
	Actor_ED25519   ActorType = iota
	Actor_SECP256K1 ActorType = iota
)

type TestState struct {
	PolicyId      string
	PolicyCreator string
}

type TestCtx struct {
	Ctx   context.Context
	T     *testing.T
	State TestState
	// Signer for Txs while running tests under Bearer or Signed Auth modes
	TxSigner *TestActor
	// Timestamp used to generate Msgs in Test
	Timestamp     time.Time
	TokenIssueTs  time.Time
	Executor      MsgExecutor
	Strategy      AuthenticationStrategy
	AccountKeeper *testutil.AccountKeeperStub
	ActorType     ActorType
}

func NewTestCtx(t *testing.T) *TestCtx {
	baseCtx, srv, accKeeper := setupMsgServer(t)
	root := MustNewSourceHubActorFromName("root")
	accKeeper.NewAccount(root.PubKey)
	ctx := &TestCtx{
		Ctx:           baseCtx,
		T:             t,
		TxSigner:      root,
		Timestamp:     time.Date(2024, 6, 21, 12, 10, 00, 0, time.UTC),
		TokenIssueTs:  time.Date(2024, 6, 21, 12, 00, 00, 0, time.UTC),
		Executor:      &KeeperExecutor{k: srv},
		Strategy:      BearerToken,
		AccountKeeper: accKeeper,
		ActorType:     Actor_ED25519, //TODO parametrize
	}
	ctx.GetSourceHubAccount("root")
	return ctx
}

func (c *TestCtx) Deadline() (deadline time.Time, ok bool) { return c.Ctx.Deadline() }
func (c *TestCtx) Done() <-chan struct{}                   { return c.Ctx.Done() }
func (c *TestCtx) Err() error                              { return c.Ctx.Err() }
func (c *TestCtx) Value(key any) any                       { return c.Ctx.Value(key) }

// GetActor gets or create an account with the given alias
func (c *TestCtx) GetActor(alias string) *TestActor {
	switch c.ActorType {
	case Actor_ED25519:
		return MustNewED25519ActorFromName(alias)
	case Actor_SECP256K1:
		return MustNewSourceHubActorFromName(alias)
	default:
		panic(fmt.Sprintf("invalid actor type: %v", c.ActorType))
	}
}

func (c *TestCtx) GetSourceHubAccount(alias string) *TestActor {
	acc := MustNewSourceHubActorFromName(alias)
	c.AccountKeeper.NewAccount(acc.PubKey)
	return acc
}

func setupMsgServer(t *testing.T) (sdk.Context, types.MsgServer, *testutil.AccountKeeperStub) {
	ctx, k, accK := setupKeeper(t)
	return ctx, keeper.NewMsgServerImpl(k), accK
}

func setupKeeper(t *testing.T) (sdk.Context, keeper.Keeper, *testutil.AccountKeeperStub) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	accKeeper := &testutil.AccountKeeperStub{}
	accKeeper.GenAccount()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		accKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return ctx, k, accKeeper
}
