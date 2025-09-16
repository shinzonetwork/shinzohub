package viewregistry

import (
	"embed"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	sourcehubkeeper "github.com/shinzonetwork/shinzohub/x/sourcehub/keeper"
)

const (
	ViewregistryPrecompileAddress = "0x0000000000000000000000000000000000000210"
)

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

var _ vm.PrecompiledContract = &Precompile{}

type Precompile struct {
	cmn.Precompile
	baseGas         uint64
	sourcehubKeeper sourcehubkeeper.Keeper
}

func NewPrecompile(baseGas uint64, sourcehubKeeper sourcehubkeeper.Keeper) (*Precompile, error) {
	newABI, err := cmn.LoadABI(f, "abi.json")
	if err != nil {
		return nil, err
	}

	return &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newABI,
			KvGasConfig:          storetypes.GasConfig{},
			TransientKVGasConfig: storetypes.GasConfig{},
		},
		baseGas:         baseGas,
		sourcehubKeeper: sourcehubKeeper,
	}, nil
}

func (p Precompile) Address() common.Address {
	return common.HexToAddress(ViewregistryPrecompileAddress)
}

func (p Precompile) RequiredGas(_ []byte) uint64 {
	return p.baseGas
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	if value := contract.Value(); value.Sign() == 1 {
		return nil, fmt.Errorf("cannot receive funds, received: %s", contract.Value().String())
	}

	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	bz, err = p.HandleMethod(ctx, contract, stateDB, method, args)
	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(uint64(cost), nil, tracing.GasChangeUnspecified) {
		return nil, vm.ErrOutOfGas
	}

	// if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
	// 	return nil, err
	// }

	return bz, nil
}

func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case ViewRegistryRegisterMethod:
		return true
	case ViewRegistryGetMethod:
		return false
	default:
		return false
	}
}

// HandleMethod handles the execution of each of the ERC-20 methods.
func (p *Precompile) HandleMethod(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) (bz []byte, err error) {
	switch method.Name {
	case ViewRegistryRegisterMethod:
		bz, err = p.ViewRegistryRegister(ctx, contract, stateDB, method, args)
	case ViewRegistryGetMethod:
		bz, err = p.ViewRegistryGet(ctx, contract, stateDB, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	return bz, err
}
