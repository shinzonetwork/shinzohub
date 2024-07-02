package test

import (
	"context"
	"fmt"

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
	"github.com/sourcenetwork/sourcehub/x/acp/types"

	"github.com/sourcenetwork/sourcehub/x/acp/keeper"
	"github.com/sourcenetwork/sourcehub/x/acp/testutil"
)

type KeeperExecutor struct {
	k              types.MsgServer
	accountCreator *testutil.AccountKeeperStub
}

func (e *KeeperExecutor) BearerPolicyCmd(ctx *TestCtx, msg *types.MsgBearerPolicyCmd) (*types.MsgBearerPolicyCmdResponse, error) {
	return e.k.BearerPolicyCmd(ctx, msg)
}

func (e *KeeperExecutor) SignedPolicyCmd(ctx *TestCtx, msg *types.MsgSignedPolicyCmd) (*types.MsgSignedPolicyCmdResponse, error) {
	return e.k.SignedPolicyCmd(ctx, msg)
}

func (e *KeeperExecutor) DirectPolicyCmd(ctx *TestCtx, msg *types.MsgDirectPolicyCmd) (*types.MsgDirectPolicyCmdResponse, error) {
	return e.k.DirectPolicyCmd(ctx, msg)
}

func (e *KeeperExecutor) CreatePolicy(ctx *TestCtx, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error) {
	return e.k.CreatePolicy(ctx, msg)
}

func (e *KeeperExecutor) GetOrCreateAccountFromActor(_ context.Context, actor *TestActor) (sdk.AccountI, error) {
	return e.accountCreator.NewAccount(actor.PubKey), nil
}

func NewExecutor(strategy ExecutorStrategy) (context.Context, MsgExecutor, error) {
	switch strategy {
	case Keeper:
		ctx, exec, err := newKeeperExecutor()
		return ctx, exec, err
	case SDK:
		panic("sdk executor not implemented")
	case CLI:
		panic("sdk executor not implemented")
	default:
		return nil, nil, fmt.Errorf("invalid executor strategy: %v", strategy)
	}
}

func newKeeperExecutor() (context.Context, MsgExecutor, error) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create keeper executor: %v", err)
	}

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

	msgServer := keeper.NewMsgServerImpl(k)
	executor := &KeeperExecutor{
		k:              msgServer,
		accountCreator: accKeeper,
	}
	return ctx, executor, nil
}
