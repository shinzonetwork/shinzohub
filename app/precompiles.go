package app

import (
	"fmt"
	"maps"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	bankprecompile "github.com/cosmos/evm/precompiles/bank"
	"github.com/cosmos/evm/precompiles/bech32"
	distprecompile "github.com/cosmos/evm/precompiles/distribution"

	// evidenceprecompile "github.com/cosmos/evm/precompiles/evidence"
	govprecompile "github.com/cosmos/evm/precompiles/gov"
	ics20precompile "github.com/cosmos/evm/precompiles/ics20"
	"github.com/cosmos/evm/precompiles/p256"
	slashingprecompile "github.com/cosmos/evm/precompiles/slashing"
	stakingprecompile "github.com/cosmos/evm/precompiles/staking"
	erc20Keeper "github.com/cosmos/evm/x/erc20/keeper"
	transferkeeper "github.com/cosmos/evm/x/ibc/transfer/keeper"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	sourcehubkeeper "github.com/shinzonetwork/shinzohub/x/sourcehub/keeper"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/shinzonetwork/shinzohub/app/precompiles/viewregistry"
)

const bech32PrecompileBaseGas = 6_000
const viewRegsitryPrecompileBaseGas = 0

// Optionals contains optional codecs for precompiles
type Optionals struct {
	AddressCodec       address.Codec // used by gov/staking
	ValidatorAddrCodec address.Codec // used by slashing
	ConsensusAddrCodec address.Codec // used by slashing
}

func defaultOptionals() Optionals {
	return Optionals{
		AddressCodec:       addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddrCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddrCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

type Option func(*Optionals)

func WithAddressCodec(c address.Codec) Option { return func(o *Optionals) { o.AddressCodec = c } }
func WithValidatorAddrCodec(c address.Codec) Option {
	return func(o *Optionals) { o.ValidatorAddrCodec = c }
}
func WithConsensusAddrCodec(c address.Codec) Option {
	return func(o *Optionals) { o.ConsensusAddrCodec = c }
}

func GetAvailableStaticPrecompiles() []string {
	customAvailableStaticPrecompiles := []string{
		viewregistry.ViewregistryPrecompileAddress,
		// register custom address here
	}

	return append(evmtypes.AvailableStaticPrecompiles, customAvailableStaticPrecompiles...)
}

// NewAvailableStaticPrecompiles returns the list of all available static precompiled contracts from EVM.
//
// NOTE: this should only be used during initialization of the Keeper.
func NewAvailableStaticPrecompiles(
	stakingKeeper stakingkeeper.Keeper,
	distributionKeeper distributionkeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	erc20Keeper erc20Keeper.Keeper,
	authzKeeper authzkeeper.Keeper,
	transferKeeper transferkeeper.Keeper,
	channelKeeper channelkeeper.Keeper,
	evmKeeper *evmkeeper.Keeper,
	govKeeper govkeeper.Keeper,
	slashingKeeper slashingkeeper.Keeper,
	sourcehubKeeper sourcehubkeeper.Keeper,
	codec codec.Codec,
	opts ...Option,
) map[common.Address]vm.PrecompiledContract {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	// Clone the mapping from the latest EVM fork.
	precompiles := maps.Clone(vm.PrecompiledContractsBerlin)

	// secp256r1 precompile as per EIP-7212
	p256Precompile := &p256.Precompile{}

	bech32Precompile, err := bech32.NewPrecompile(bech32PrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate bech32 precompile: %w", err))
	}

	stakingPrecompile, err := stakingprecompile.NewPrecompile(stakingKeeper, options.AddressCodec)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate staking precompile: %w", err))
	}

	distributionPrecompile, err := distprecompile.NewPrecompile(
		distributionKeeper,
		stakingKeeper,
		evmKeeper,
		options.AddressCodec,
	)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate distribution precompile: %w", err))
	}

	ibcTransferPrecompile, err := ics20precompile.NewPrecompile(
		bankKeeper,
		stakingKeeper,
		transferKeeper,
		&channelKeeper,
		evmKeeper,
	)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate ICS20 precompile: %w", err))
	}

	bankPrecompile, err := bankprecompile.NewPrecompile(bankKeeper, erc20Keeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate bank precompile: %w", err))
	}

	govPrecompile, err := govprecompile.NewPrecompile(govKeeper, codec, options.AddressCodec)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate gov precompile: %w", err))
	}

	slashingPrecompile, err := slashingprecompile.NewPrecompile(slashingKeeper, options.ValidatorAddrCodec, options.ConsensusAddrCodec)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate slashing precompile: %w", err))
	}

	// register custom precompiles
	viewRegistryPrecompile, err := viewregistry.NewPrecompile(viewRegsitryPrecompileBaseGas, sourcehubKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate view registry precompile: %w", err))
	}

	// Stateless precompiles
	precompiles[bech32Precompile.Address()] = bech32Precompile
	precompiles[p256Precompile.Address()] = p256Precompile

	// Stateful precompiles
	precompiles[stakingPrecompile.Address()] = stakingPrecompile
	precompiles[distributionPrecompile.Address()] = distributionPrecompile
	precompiles[ibcTransferPrecompile.Address()] = ibcTransferPrecompile
	precompiles[bankPrecompile.Address()] = bankPrecompile
	precompiles[govPrecompile.Address()] = govPrecompile
	precompiles[slashingPrecompile.Address()] = slashingPrecompile

	precompiles[viewRegistryPrecompile.Address()] = viewRegistryPrecompile

	return precompiles
}
