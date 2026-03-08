package view

import (
	"context"
	"encoding/json"

	"cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/shinzonetwork/shinzohub/x/view/keeper"
	"github.com/shinzonetwork/shinzohub/x/view/types"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic = (*AppModule)(nil)
	_ module.HasGenesis     = (*AppModule)(nil)
	_ appmodule.AppModule   = (*AppModule)(nil)
)

type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
	ss     storetypes.KVStoreService
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	ss storetypes.KVStoreService,
) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: keeper,
		ss:     ss,
	}
}

func (AppModule) IsAppModule()        {}
func (AppModule) IsOnePerModuleType() {}

func (AppModule) Name() string { return types.ModuleName }

func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return err
	}
	return gs.Validate()
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)
	am.keeper.InitGenesis(ctx, genState)
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))
}

func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

func (AppModule) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}
