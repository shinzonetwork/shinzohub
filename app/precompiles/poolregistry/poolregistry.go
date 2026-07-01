package poolregistry

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

	poolkeeper "github.com/shinzonetwork/shinzohub/x/pool/keeper"
)

const PrecompileAddress = "0x0000000000000000000000000000000000000213"

//go:embed abi.json
var f embed.FS

var _ vm.PrecompiledContract = &Precompile{}

type Precompile struct {
	cmn.Precompile
	baseGas    uint64
	poolKeeper poolkeeper.Keeper
}

func NewPrecompile(baseGas uint64, poolKeeper poolkeeper.Keeper) (*Precompile, error) {
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
		poolKeeper: poolKeeper,
	}, nil
}

func (p Precompile) Address() common.Address {
	return common.HexToAddress(PrecompileAddress)
}

func (p Precompile) RequiredGas(_ []byte) uint64 {
	return p.baseGas
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	bz, err = p.handle(ctx, evm, contract, stateDB, method, args)
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
	switch method.Name {
	case MethodRegisterDemandForView, MethodJoinPool, MethodLeavePool:
		return true
	}
	return false
}

func (p *Precompile) handle(
	ctx sdk.Context,
	evm *vm.EVM,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	switch method.Name {
	case MethodRegisterDemandForView:
		return p.RegisterDemandForView(ctx, evm, contract, stateDB, method, args)
	case MethodPoolsOf:
		return p.PoolsOf(ctx, method, args)
	case MethodViewOfPool:
		return p.ViewOfPool(ctx, method, args)
	case MethodGetPool:
		return p.GetPool(ctx, method, args)
	case MethodGetPoolFor:
		return p.GetPoolFor(ctx, method, args)
	case MethodGetPoolDetail:
		return p.GetPoolDetail(ctx, method, args)
	case MethodJoinPool:
		return p.JoinPool(ctx, contract, method, args)
	case MethodLeavePool:
		return p.LeavePool(ctx, contract, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}
}
