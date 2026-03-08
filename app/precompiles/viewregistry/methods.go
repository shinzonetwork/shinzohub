package viewregistry

import (
	"encoding/base64"
	"fmt"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/shinzonetwork/viewbundle-go"
)

const (
	MethodRegister            = "register"
	MethodRegisterWithPricing = "registerWithPricing"
	MethodGetView             = "getView"
)

var zero = uint256.NewInt(0)

// sdlTypeRe matches `type <Name>` in the SDL to extract the resource name.
var sdlTypeRe = regexp.MustCompile(`\btype\s+([A-Za-z0-9_]+)\b`)

func (p Precompile) Register(
	ctx sdk.Context,
	evm *vm.EVM,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	data, ok := args[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid data")
	}

	caller := contract.Caller()
	deployer := contract.Address()

	viewAddr, viewId, viewName, leftoverGas, err := p.prepareView(evm, deployer, caller, data, common.Address{}, contract.Gas)
	if err != nil {
		return nil, err
	}
	contract.Gas = leftoverGas

	creatorAddr := sdk.AccAddress(caller.Bytes()).String()
	if err := p.viewKeeper.RegisterView(ctx, viewId, viewName, creatorAddr, viewAddr.Hex(), data); err != nil {
		return nil, fmt.Errorf("failed to register view in keeper: %w", err)
	}

	emitViewCreated(ctx, stateDB, contract.Address(), viewAddr, caller, viewName, data)

	return method.Outputs.Pack(viewAddr)
}

func (p Precompile) RegisterWithPricing(
	ctx sdk.Context,
	evm *vm.EVM,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	data, ok := args[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid data")
	}

	pricingAddr, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid pricing address")
	}

	caller := contract.Caller()
	deployer := contract.Address()

	viewAddr, viewId, viewName, leftoverGas, err := p.prepareView(evm, deployer, caller, data, pricingAddr, contract.Gas)
	if err != nil {
		return nil, err
	}
	contract.Gas = leftoverGas

	creatorAddr := sdk.AccAddress(caller.Bytes()).String()
	if err := p.viewKeeper.RegisterView(ctx, viewId, viewName, creatorAddr, viewAddr.Hex(), data); err != nil {
		return nil, fmt.Errorf("failed to register view in keeper: %w", err)
	}

	emitViewCreated(ctx, stateDB, contract.Address(), viewAddr, caller, viewName, data)

	return method.Outputs.Pack(viewAddr)
}

func (p Precompile) prepareView(
	evm *vm.EVM,
	deployer, caller common.Address,
	data []byte,
	pricingAddr common.Address,
	gas uint64,
) (viewAddr common.Address, viewId string, resourceName string, leftoverGas uint64, err error) {
	// Decode viewbundle header to extract resource name from SDL.
	decoded, err := viewbundle.DecodeHeader(data)
	if err != nil {
		return common.Address{}, "", "", gas, fmt.Errorf("failed to decode viewbundle: %w", err)
	}

	matches := sdlTypeRe.FindStringSubmatch(decoded.Header.Sdl)
	if len(matches) < 2 {
		return common.Address{}, "", "", gas, fmt.Errorf("SDL missing type name")
	}
	resourceName = matches[1]

	// Deploy View.sol with constructor(resourceName, caller, pricingAddr).
	constructorArgs, err := ViewConstructorArgs.Pack(resourceName, caller, pricingAddr)
	if err != nil {
		return common.Address{}, "", "", gas, fmt.Errorf("failed to pack View constructor args: %w", err)
	}

	viewInitCode := append(ViewBytecode, constructorArgs...)
	_, viewAddr, leftoverGas, err = evm.Create(
		deployer,
		viewInitCode,
		gas,
		zero,
	)
	if err != nil {
		return common.Address{}, "", "", leftoverGas, fmt.Errorf("failed to deploy View contract: %w", err)
	}

	// Build viewId = resourceName_contractAddress using the actual deployed address.
	viewId = fmt.Sprintf("%s_%s", resourceName, viewAddr.Hex())

	return viewAddr, viewId, resourceName, leftoverGas, nil
}

func (p Precompile) GetView(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	viewAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid viewAddress")
	}

	view, found, err := p.viewKeeper.GetViewByAddress(ctx, viewAddress.Hex())
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack("")
	}
	return method.Outputs.Pack(view.Creator)
}

func emitViewCreated(
	ctx sdk.Context,
	stateDB vm.StateDB,
	precompileAddr common.Address,
	viewAddr, creator common.Address,
	name string,
	data []byte,
) {
	topic0 := crypto.Keccak256Hash([]byte("ViewCreated(address,address,string)"))
	nameData, _ := ViewConstructorArgs[:1].Pack(name)
	stateDB.AddLog(&gethtypes.Log{
		Address: precompileAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(viewAddr.Bytes()),
			common.BytesToHash(creator.Bytes()),
		},
		Data: nameData,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"ViewRegistered",
			sdk.NewAttribute("view_address", viewAddr.Hex()),
			sdk.NewAttribute("view_name", name),
			sdk.NewAttribute("creator", sdk.AccAddress(creator.Bytes()).String()),
			sdk.NewAttribute("data", base64.StdEncoding.EncodeToString(data)),
		),
	)
}
