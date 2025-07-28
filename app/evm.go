package app

import (
	chainante "shinzohub/app/ante"

	storetypes "cosmossdk.io/store/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	evmante "github.com/cosmos/evm/ante"
	evmevmante "github.com/cosmos/evm/ante/evm"
	srvflags "github.com/cosmos/evm/server/flags"
	cosmosevmtypes "github.com/cosmos/evm/types"
	erc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/spf13/cast"
)

func SetupEVM(app *App, appOpts servertypes.AppOptions) error {
	app.evmKvStoreKeys[evmtypes.StoreKey] = storetypes.NewKVStoreKey(evmtypes.StoreKey)
	app.evmKvStoreKeys[feemarkettypes.StoreKey] = storetypes.NewKVStoreKey(feemarkettypes.StoreKey)
	app.evmKvStoreKeys[erc20types.StoreKey] = storetypes.NewKVStoreKey(erc20types.StoreKey)

	app.evmTransientStoreKeys[evmtypes.TransientKey] = storetypes.NewTransientStoreKey(evmtypes.TransientKey)
	app.evmTransientStoreKeys[feemarkettypes.TransientKey] = storetypes.NewTransientStoreKey(feemarkettypes.TransientKey)

	app.App.MountKVStores(app.evmKvStoreKeys)
	app.App.MountTransientStores(app.evmTransientStoreKeys)

	// Register subspaces (after Build)
	app.ParamsKeeper.Subspace(evmtypes.ModuleName)
	app.ParamsKeeper.Subspace(feemarkettypes.ModuleName)
	app.ParamsKeeper.Subspace(erc20types.ModuleName)

	// Tracer string
	tracer := cast.ToString(appOpts.Get(srvflags.EVMTracer))

	// FeeMarketKeeper
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		app.appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.evmKvStoreKeys[feemarkettypes.StoreKey],
		app.evmTransientStoreKeys[feemarkettypes.TransientKey],
	)

	// EVMKeeper (must come before Erc20Keeper)
	app.EVMKeeper = evmkeeper.NewKeeper(
		app.appCodec,
		app.evmKvStoreKeys[evmtypes.StoreKey],
		app.evmTransientStoreKeys[evmtypes.TransientKey],
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.FeeMarketKeeper,
		&app.Erc20Keeper, // must be passed by reference even though it's not initialized yet
		tracer,
	)

	// Erc20Keeper
	app.Erc20Keeper = erc20keeper.NewKeeper(
		app.evmKvStoreKeys[erc20types.StoreKey],
		app.appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.EVMKeeper,
		app.StakingKeeper,
		&app.TransferKeeper,
	)

	return nil
}

func SetupEVMPrecompiles(app *App, appOpts servertypes.AppOptions) error {
	// Configure EVM precompiles
	corePrecompiles := NewAvailableStaticPrecompiles(
		*app.StakingKeeper,
		app.DistrKeeper,
		app.BankKeeper,
		app.Erc20Keeper,
		app.AuthzKeeper,
		app.TransferKeeper,
		*app.IBCKeeper.ChannelKeeper,
		app.EVMKeeper,
		*app.GovKeeper,
		app.SlashingKeeper,
		app.EvidenceKeeper,
	)

	app.EVMKeeper.WithStaticPrecompiles(
		corePrecompiles,
	)

	return nil
}

func SetupEVMAnteHandler(app *App, appOpts servertypes.AppOptions) error {
	anteOpts := chainante.HandlerOptions{
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		SignModeHandler:        app.txConfig.SignModeHandler(),
		FeegrantKeeper:         app.FeeGrantKeeper,
		SigGasConsumer:         evmante.SigVerificationGasConsumer,
		ExtensionOptionChecker: cosmosevmtypes.HasDynamicFeeExtensionOption,
		TxFeeChecker:           evmevmante.NewDynamicFeeChecker(app.FeeMarketKeeper),
		EvmKeeper:              app.EVMKeeper,
		FeeMarketKeeper:        app.FeeMarketKeeper,
		MaxTxGasWanted:         cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted)),
		Cdc:                    app.appCodec,
		IBCKeeper:              app.IBCKeeper,
	}

	app.SetAnteHandler(chainante.NewAnteHandler(anteOpts))

	return nil
}

func CustomizeEVMGenesis(app *App, genesisState cosmosevmtypes.GenesisState) cosmosevmtypes.GenesisState {
	feemarketGenesis := feemarkettypes.DefaultGenesisState()
	genesisState[feemarkettypes.ModuleName] = app.appCodec.MustMarshalJSON(feemarketGenesis)

	evmGenesis := evmtypes.DefaultGenesisState()
	evmGenesis.Params.EvmDenom = BaseDenom
	evmGenesis.Params.ChainConfig = *evmtypes.DefaultChainConfig(app.ChainID())
	evmGenesis.Params.ActiveStaticPrecompiles = evmtypes.AvailableStaticPrecompiles
	genesisState[evmtypes.ModuleName] = app.appCodec.MustMarshalJSON(evmGenesis)

	erc20Genesis := erc20types.DefaultGenesisState()
	erc20Genesis.TokenPairs = ExampleTokenPairs
	erc20Genesis.Params.NativePrecompiles = append(erc20Genesis.Params.NativePrecompiles, WTokenContractMainnet)
	genesisState[erc20types.ModuleName] = app.appCodec.MustMarshalJSON(erc20Genesis)

	transferGenesis := ibctransfertypes.DefaultGenesisState()
	genesisState[ibctransfertypes.ModuleName] = app.appCodec.MustMarshalJSON(transferGenesis)

	return genesisState
}
