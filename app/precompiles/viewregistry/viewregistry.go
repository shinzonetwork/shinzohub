package viewregistry

import (
	"embed"

	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
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
	baseGas uint64
	abi.ABI
}

func NewPrecompile(baseGas uint64) (*Precompile, error) {
	newABI, err := cmn.LoadABI(f, "abi.json")
	if err != nil {
		return nil, err
	}
	return &Precompile{
		baseGas: baseGas,
		ABI:     newABI,
	}, nil
}

func (p Precompile) Address() common.Address {
	return common.HexToAddress(ViewregistryPrecompileAddress)
}

func (p Precompile) RequiredGas(_ []byte) uint64 {
	return p.baseGas
}

func (p Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (bz []byte, err error) {
	if len(contract.Input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	methodId := contract.Input[:4]

	method, err := p.MethodById(methodId)
	if err != nil {
		return nil, err
	}

	argsBz := contract.Input[4:]
	args, err := method.Inputs.Unpack(argsBz)
	if err != nil {
		return nil, err
	}

	switch method.Name {
	case ViewRegistryRegisterMethod:
		bz, err = p.ViewRegistryRegister(method, args)
	case ViewRegistryGetMethod:
		bz, err = p.ViewRegistryGet(method, args)
	}

	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	return bz, nil
}
