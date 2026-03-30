package indexerregistry

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

	indexerkeeper "github.com/shinzonetwork/shinzohub/x/indexer/keeper"
)

const (
	PrecompileAddress = "0x0000000000000000000000000000000000000212"
)

//go:embed abi.json
var f embed.FS

var _ vm.PrecompiledContract = &Precompile{}

type Precompile struct {
	cmn.Precompile
	baseGas       uint64
	indexerKeeper indexerkeeper.Keeper
}

func NewPrecompile(baseGas uint64, indexerKeeper indexerkeeper.Keeper) (*Precompile, error) {
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
		baseGas:       baseGas,
		indexerKeeper: indexerKeeper,
	}, nil
}

func (p Precompile) Address() common.Address {
	return common.HexToAddress(PrecompileAddress)
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

	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	bz, err = p.HandleMethod(ctx, contract, stateDB, method, args)
	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas
	if !contract.UseGas(uint64(cost), nil, tracing.GasChangeUnspecified) {
		return nil, vm.ErrOutOfGas
	}

	return bz, nil
}

func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case MethodRegister:
		return true
	default:
		return false
	}
}

func (p *Precompile) HandleMethod(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) (bz []byte, err error) {
	switch method.Name {
	case MethodRegister:
		bz, err = p.Register(ctx, contract, stateDB, method, args)
	case MethodIsRegistered:
		bz, err = p.IsRegistered(ctx, method, args)
	case MethodGetDid:
		bz, err = p.GetDid(ctx, method, args)
	case MethodGetConnectionString:
		bz, err = p.GetConnectionString(ctx, method, args)
	case MethodGetSourceChain:
		bz, err = p.GetSourceChain(ctx, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	return bz, err
}
