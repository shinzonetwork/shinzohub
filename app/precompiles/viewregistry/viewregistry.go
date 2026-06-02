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

	viewkeeper "github.com/shinzonetwork/shinzohub/x/view/keeper"
)

const PrecompileAddress = "0x0000000000000000000000000000000000000210"

//go:embed abi.json
var f embed.FS

var _ vm.PrecompiledContract = &Precompile{}

type Precompile struct {
	cmn.Precompile
	baseGas    uint64
	viewKeeper viewkeeper.Keeper
}

func NewPrecompile(baseGas uint64, viewKeeper viewkeeper.Keeper) (*Precompile, error) {
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
		baseGas:    baseGas,
		viewKeeper: viewKeeper,
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

	bz, err = p.handle(ctx, contract, stateDB, method, args)
	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas
	if !contract.UseGas(cost, nil, tracing.GasChangeUnspecified) {
		return nil, vm.ErrOutOfGas
	}
	return bz, nil
}

func (Precompile) IsTransaction(method *abi.Method) bool {
	return method.Name == MethodRegister
}

func (p *Precompile) handle(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	switch method.Name {
	case MethodRegister:
		return p.Register(ctx, contract, stateDB, method, args)
	case MethodGetView:
		return p.GetView(ctx, method, args)
	case MethodListViews:
		return p.ListViews(ctx, method, args)
	case MethodViewCount:
		return p.ViewCount(ctx, method)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}
}
