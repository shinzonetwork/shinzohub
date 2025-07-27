package app

import (
	"io"

	"encoding/json"

	tmjson "github.com/cometbft/cometbft/libs/json"

	// Force-load the tracer engines to trigger registration due to Go-Ethereum v1.10.15 changes
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"

	_ "cosmossdk.io/api/cosmos/tx/config/v1" // import for side-effects
	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types" // cosmossdk.io/core/store // cosmossdk.io/store/types
	_ "cosmossdk.io/x/circuit"            // import for side-effects
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	_ "cosmossdk.io/x/evidence" // import for side-effects
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	_ "cosmossdk.io/x/feegrant/module" // import for side-effects
	nftkeeper "cosmossdk.io/x/nft/keeper"
	_ "cosmossdk.io/x/nft/module" // import for side-effects
	"cosmossdk.io/x/tx/signing"
	_ "cosmossdk.io/x/upgrade" // import for side-effects
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config" // import for side-effects
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	_ "github.com/cosmos/cosmos-sdk/x/auth/vesting" // import for side-effects
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	_ "github.com/cosmos/cosmos-sdk/x/authz/module" // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/bank"         // import for side-effects
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	_ "github.com/cosmos/cosmos-sdk/x/consensus" // import for side-effects
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	_ "github.com/cosmos/cosmos-sdk/x/crisis" // import for side-effects
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	_ "github.com/cosmos/cosmos-sdk/x/distribution" // import for side-effects
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	genutil "github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	_ "github.com/cosmos/cosmos-sdk/x/group/module" // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/mint"         // import for side-effects
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	_ "github.com/cosmos/cosmos-sdk/x/params" // import for side-effects
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	_ "github.com/cosmos/cosmos-sdk/x/slashing" // import for side-effects
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	_ "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts" // import for side-effects
	icacontrollerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"

	// Replace default transfer with EVM's transfer (if using IBC)
	shinzohubmodulekeeper "shinzohub/x/shinzohub/keeper"

	"github.com/cosmos/evm/ethereum/eip712"
	transferkeeper "github.com/cosmos/evm/x/ibc/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	// this line is used by starport scaffolding # stargate/app/moduleImport

	// EVM imports
	erc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"

	chainante "shinzohub/app/ante"

	evmante "github.com/cosmos/evm/ante"
	evmevmante "github.com/cosmos/evm/ante/evm"

	srvflags "github.com/cosmos/evm/server/flags"
	cosmosevmtypes "github.com/cosmos/evm/types"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"shinzohub/docs"

	evmutils "github.com/cosmos/evm/utils"
	erc20 "github.com/cosmos/evm/x/erc20"
	feemarket "github.com/cosmos/evm/x/feemarket"
	evm "github.com/cosmos/evm/x/vm"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/spf13/cast"

	enccodec "github.com/cosmos/evm/encoding/codec"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

const (
	// Name is the name of the application.
	Name = "shinzohub"

	// AccountAddressPrefix is the prefix for accounts addresses.
	AccountAddressPrefix = "shinzo"

	// ChainCoinType is the coin type of the chain.
	ChainCoinType = 60

	// Ethereum tools often expect chain IDs in a specific format (e.g., name_number-version)
	ChainID = "shinzohub_9000-1"

	// internal denom used in state and transactions
	// ashinzo = "atto-shinzo" (just like uatom = micro-atom). Using this convention keeps you interoperable with wallets, explorers, etc.
	BaseDenom = "ashinzo"

	// human-readable display denom
	DisplayDenom = "SHINZO"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string
)

var (
	_ runtime.AppI            = (*App)(nil)
	_ servertypes.Application = (*App)(nil)
)

// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// Add these fields
	FeeMarketKeeper feemarketkeeper.Keeper
	EVMKeeper       *evmkeeper.Keeper
	Erc20Keeper     erc20keeper.Keeper

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper

	SlashingKeeper       slashingkeeper.Keeper
	MintKeeper           mintkeeper.Keeper
	GovKeeper            *govkeeper.Keeper
	CrisisKeeper         *crisiskeeper.Keeper
	UpgradeKeeper        *upgradekeeper.Keeper
	ParamsKeeper         paramskeeper.Keeper
	AuthzKeeper          authzkeeper.Keeper
	EvidenceKeeper       evidencekeeper.Keeper
	FeeGrantKeeper       feegrantkeeper.Keeper
	GroupKeeper          groupkeeper.Keeper
	NFTKeeper            nftkeeper.Keeper
	CircuitBreakerKeeper circuitkeeper.Keeper

	// IBC
	IBCKeeper           *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	ICAControllerKeeper icacontrollerkeeper.Keeper
	ICAHostKeeper       icahostkeeper.Keeper
	TransferKeeper      transferkeeper.Keeper

	ShinzohubKeeper shinzohubmodulekeeper.Keeper
	// this line is used by starport scaffolding # stargate/app/keeperDeclaration

	// simulation manager
	sm *module.SimulationManager

	evmKvStoreKeys        map[string]*storetypes.KVStoreKey
	evmTransientStoreKeys map[string]*storetypes.TransientStoreKey
}

func init() {
	var err error
	clienthelpers.EnvPrefix = Name
	DefaultNodeHome, err = clienthelpers.GetNodeHomeDirectory("." + Name)
	if err != nil {
		panic(err)
	}
}

const BaseDenomUnit int64 = 18 // add this if not already defined

func init() {
	sdk.DefaultPowerReduction = cosmosevmtypes.AttoPowerReduction
}

// getGovProposalHandlers return the chain proposal handlers.
func getGovProposalHandlers() []govclient.ProposalHandler {
	var govProposalHandlers []govclient.ProposalHandler
	// this line is used by starport scaffolding # stargate/app/govProposalHandlers

	govProposalHandlers = append(govProposalHandlers,
		paramsclient.ProposalHandler,
		// this line is used by starport scaffolding # stargate/app/govProposalHandler
	)

	return govProposalHandlers
}

func ProvideCustomSigners() []signing.CustomGetSigner {
	return []signing.CustomGetSigner{
		evmtypes.MsgEthereumTxCustomGetSigner,
	}
}

func UpdateInterfaceRegistry(interfaceRegistry codectypes.InterfaceRegistry) {
	enccodec.RegisterInterfaces(interfaceRegistry)
}

func UpdateLegacyAmino(interfaceRegistry codectypes.InterfaceRegistry, cdc *codec.LegacyAmino) {
	legacytx.RegressionTestingAminoCodec = cdc
	eip712.SetEncodingConfig(cdc, interfaceRegistry)
}

// AppConfig returns the default app config.
func AppConfig() depinject.Config {
	return depinject.Configs(
		appConfig,
		// Alternatively, load the app config from a YAML file.
		depinject.Invoke(UpdateInterfaceRegistry, UpdateLegacyAmino),
		depinject.Provide(ProvideCustomSigners),
		depinject.Supply(
			map[string]module.AppModuleBasic{
				genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
				govtypes.ModuleName:     gov.NewAppModuleBasic(getGovProposalHandlers()),
				// this line is used by starport scaffolding # stargate/appConfig/moduleBasic
			},
		),
	)
}

// New returns a reference to an initialized App.
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	evmAppOptions EVMOptionsFn,
	baseAppOptions ...func(*baseapp.BaseApp),
) (*App, error) {
	var (
		app = &App{
			evmKvStoreKeys:        make(map[string]*storetypes.KVStoreKey),
			evmTransientStoreKeys: make(map[string]*storetypes.TransientStoreKey),
		}

		appBuilder *runtime.AppBuilder

		// merge the AppConfig and other configuration in one config
		appConfig = depinject.Configs(
			AppConfig(),
			depinject.Supply(
				appOpts, // supply app options
				logger,  // supply logger
				// Supply with IBC keeper getter for the IBC modules with App Wiring.
				// The IBC Keeper cannot be passed because it has not been initiated yet.
				// Passing the getter, the app IBC Keeper will always be accessible.
				// This needs to be removed after IBC supports App Wiring.
				app.GetIBCKeeper,

				// here alternative options can be supplied to the DI container.
				// those options can be used f.e to override the default behavior of some modules.
				// for instance supplying a custom address codec for not using bech32 addresses.
				// read the depinject documentation and depinject module wiring for more information
				// on available options and how to use them.
			),
		)
	)

	if err := depinject.Inject(appConfig,
		&appBuilder,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.AccountKeeper,
		&app.BankKeeper,
		&app.StakingKeeper,
		&app.DistrKeeper,
		&app.ConsensusParamsKeeper,
		&app.SlashingKeeper,
		&app.MintKeeper,
		&app.GovKeeper,
		&app.CrisisKeeper,
		&app.UpgradeKeeper,
		&app.ParamsKeeper,
		&app.AuthzKeeper,
		&app.EvidenceKeeper,
		&app.FeeGrantKeeper,
		&app.NFTKeeper,
		&app.GroupKeeper,
		&app.CircuitBreakerKeeper,
		&app.ShinzohubKeeper,
		// this line is used by starport scaffolding # stargate/app/keeperDefinition
	); err != nil {
		panic(err)
	}

	// add to default baseapp options
	// enable optimistic execution
	baseAppOptions = append(baseAppOptions, baseapp.SetOptimisticExecution())

	// build app
	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	if err := evmAppOptions(app.ChainID()); err != nil {
		return nil, err
	}

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

	// register legacy modules
	if err := app.registerIBCModules(appOpts); err != nil {
		return nil, err
	}

	// register streaming services
	if err := app.RegisterStreamingServices(appOpts, app.kvStoreKeys()); err != nil {
		return nil, err
	}

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

	// Set AnteHandler with EVM options
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

	anteHandler := chainante.NewAnteHandler(anteOpts)

	app.SetAnteHandler(anteHandler)

	/****  Module Options ****/

	app.ModuleManager.RegisterInvariants(app.CrisisKeeper)

	app.RegisterModules(
		evm.NewAppModule(app.EVMKeeper, app.AccountKeeper),
		feemarket.NewAppModule(app.FeeMarketKeeper),
		erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
	)

	// create the simulation manager and define the order of the modules for deterministic simulations
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)
	app.sm.RegisterStoreDecoders()

	// A custom InitChainer sets if extra pre-init-genesis logic is required.
	// This is necessary for manually registered modules that do not support app wiring.
	// Manually set the module version map as shown below.
	// The upgrade module will automatically handle de-duplication of the module version map.
	app.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
		// Unmarshal the existing app state
		var genesisState cosmosevmtypes.GenesisState

		if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
			return nil, err
		}

		// Inject or override EVM modules

		// Add/override FeeMarket genesis
		feemarketGenesis := feemarkettypes.DefaultGenesisState()
		genesisState[feemarkettypes.ModuleName] = app.appCodec.MustMarshalJSON(feemarketGenesis)

		// Add/override EVM genesis
		evmGenesis := evmtypes.DefaultGenesisState()

		evmGenesis.Params.EvmDenom = BaseDenom
		evmGenesis.Params.ChainConfig = *evmtypes.DefaultChainConfig(app.ChainID())
		evmGenesis.Params.ActiveStaticPrecompiles = evmtypes.AvailableStaticPrecompiles
		genesisState[evmtypes.ModuleName] = app.appCodec.MustMarshalJSON(evmGenesis)

		// Add/override ERC20 genesis
		erc20Genesis := erc20types.DefaultGenesisState()
		erc20Genesis.TokenPairs = ExampleTokenPairs // you should define this
		erc20Genesis.Params.NativePrecompiles = append(erc20Genesis.Params.NativePrecompiles, WTokenContractMainnet)
		genesisState[erc20types.ModuleName] = app.appCodec.MustMarshalJSON(erc20Genesis)

		transferGenesis := ibctransfertypes.DefaultGenesisState()
		genesisState[ibctransfertypes.ModuleName] = app.appCodec.MustMarshalJSON(transferGenesis)

		// Update module version map
		if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap()); err != nil {
			return nil, err
		}

		appStateBytes, err := json.Marshal(genesisState)
		if err != nil {
			return nil, err
		}

		// Call the default InitChainer with modified state
		return app.InitChainer(ctx, &abci.RequestInitChain{
			Time:            req.Time,
			ChainId:         req.ChainId,
			ConsensusParams: req.ConsensusParams,
			Validators:      req.Validators,
			AppStateBytes:   appStateBytes,
			InitialHeight:   req.InitialHeight,
		})
	})

	if err := app.Load(loadLatest); err != nil {
		return nil, err
	}

	return app, nil
}

// LegacyAmino returns App's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns App's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns App's interfaceRegistry.
func (app *App) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns App's tx config.
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	kvStoreKey, ok := app.UnsafeFindStoreKey(storeKey).(*storetypes.KVStoreKey)
	if !ok {
		return nil
	}
	return kvStoreKey
}

// GetMemKey returns the MemoryStoreKey for the provided store key.
func (app *App) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	key, ok := app.UnsafeFindStoreKey(storeKey).(*storetypes.MemoryStoreKey)
	if !ok {
		return nil
	}

	return key
}

// kvStoreKeys returns all the kv store keys registered inside App.
func (app *App) kvStoreKeys() map[string]*storetypes.KVStoreKey {
	keys := make(map[string]*storetypes.KVStoreKey)
	for _, k := range app.GetStoreKeys() {
		if kv, ok := k.(*storetypes.KVStoreKey); ok {
			keys[kv.Name()] = kv
		}
	}

	return keys
}

// GetSubspace returns a param subspace for a given module name.
func (app *App) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// GetIBCKeeper returns the IBC keeper.
func (app *App) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// SimulationManager implements the SimulationApp interface.
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	app.App.RegisterAPIRoutes(apiSvr, apiConfig)
	// register swagger API in app.go so that other applications can override easily
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}

	// register app's OpenAPI routes.
	docs.RegisterOpenAPIService(Name, apiSvr.Router)
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dup := make(map[string][]string)
	for _, perms := range moduleAccPerms {
		dup[perms.Account] = perms.Permissions
	}
	return dup
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	blockedAddrs := make(map[string]bool)

	// Step 1: Add from existing blocked addresses or default module accounts
	if len(blockAccAddrs) > 0 {
		for _, addr := range blockAccAddrs {
			blockedAddrs[addr] = true
		}
	} else {
		for addr := range GetMaccPerms() {
			blockedAddrs[addr] = true
		}
	}

	// Step 2: Append precompiled contract addresses to blocked list
	blockedPrecompilesHex := evmtypes.AvailableStaticPrecompiles
	for _, addr := range vm.PrecompiledAddressesBerlin {
		blockedPrecompilesHex = append(blockedPrecompilesHex, addr.Hex())
	}

	for _, precompile := range blockedPrecompilesHex {
		blockedAddrs[evmutils.EthHexToCosmosAddr(precompile).String()] = true
	}

	return blockedAddrs
}
