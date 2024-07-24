package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/acp_core/pkg/engine"
	acpruntime "github.com/sourcenetwork/acp_core/pkg/runtime"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/access_decision"
	"github.com/sourcenetwork/sourcehub/x/acp/stores"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		accountKeeper types.AccountKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,

) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		logger:        logger,
		accountKeeper: accountKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) GetAccessDecisionRepository(ctx sdk.Context) access_decision.Repository {
	kv := k.storeService.OpenKVStore(ctx)
	prefixKey := []byte(types.AccessDecisionRepositoryKey)
	adapted := runtime.KVStoreAdapter(kv)
	adapted = prefix.NewStore(adapted, prefixKey)
	return access_decision.NewAccessDecisionRepository(adapted)
}

func (k *Keeper) GetACPEngine(ctx sdk.Context) (coretypes.ACPEngineServer, error) {
	kv := k.storeService.OpenKVStore(ctx)
	adapted := runtime.KVStoreAdapter(kv)
	raccoonAdapted := stores.RaccoonKVFromCosmos(adapted)
	runtime, err := acpruntime.NewRuntimeManager(
		acpruntime.WithKVStore(raccoonAdapted),
	)
	if err != nil {
		return nil, err
	}

	return engine.NewACPEngine(runtime), nil
}
