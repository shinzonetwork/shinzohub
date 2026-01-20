package viewregistry

import (
	"fmt"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shinzonetwork/shinzohub/viewbundle"
)

const (
	ViewRegistryRegisterMethod = "register"
	ViewRegistryGetMethod      = "get"
)

func (p Precompile) ViewRegistryRegister(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	encodedValue, ok := args[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid type for value")
	}

	decodedValue, err := viewbundle.Decode(encodedValue)
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	re := regexp.MustCompile(`\btype\s+([A-Za-z0-9_]+)\b`)
	matches := re.FindStringSubmatch(decodedValue.Header.Sdl)
	if len(matches) < 2 {
		return nil, vm.ErrExecutionReverted
	}
	resourceName := matches[1]

	key := crypto.Keccak256Hash(contract.Caller().Bytes(), encodedValue)

	id := fmt.Sprintf("%s_%s", resourceName, key.Hex())

	loc := re.FindStringSubmatchIndex(decodedValue.Header.Sdl)
	if len(loc) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	decodedValue.Header.Sdl = decodedValue.Header.Sdl[:loc[2]] + id + decodedValue.Header.Sdl[loc[3]:]

	newEncodedValue, err := viewbundle.Encode(decodedValue)
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	err = p.sourcehubKeeper.RegisterObject(ctx, id)
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	creator := crypto.Keccak256Hash([]byte("view.creator"), key.Bytes())
	stateDB.SetState(
		contract.Address(),
		creator,
		common.BytesToHash(common.LeftPadBytes(contract.Caller().Bytes(), 32)),
	)

	eventSignature := []byte("Registered(bytes32,address)")
	topic0 := crypto.Keccak256Hash(eventSignature)
	topic1 := key
	topic2 := common.BytesToHash(contract.Caller().Bytes())

	evmLog := &types.Log{
		Address: contract.Address(),
		Topics:  []common.Hash{topic0, topic1, topic2},
		Data:    newEncodedValue,
	}
	stateDB.AddLog(evmLog)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"Registered",
			sdk.NewAttribute("key", key.Hex()),
			sdk.NewAttribute("creator", sdk.AccAddress(contract.Caller().Bytes()).String()),
			sdk.NewAttribute("view", string(newEncodedValue)),
		),
	)

	return nil, nil
}

func (p Precompile) ViewRegistryGet(ctx sdk.Context, contract *vm.Contract, stateDB vm.StateDB, method *abi.Method, args []interface{}) ([]byte, error) {
	key, ok := args[0].([32]byte) // bytes32 in Solidity maps to [32]byte
	if !ok {
		return nil, fmt.Errorf("invalid type for key")
	}

	// Fetch from storage under your precompile's address
	valueHash := stateDB.GetState(contract.Address(), common.BytesToHash(key[:]))

	return valueHash.Bytes(), nil
}
