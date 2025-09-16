package viewregistry

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	ViewRegistryRegisterMethod = "register"
	ViewRegistryGetMethod      = "get"
)

func (p Precompile) ViewRegistryRegister(ctx sdk.Context, contract *vm.Contract, stateDB vm.StateDB, method *abi.Method, args []interface{}) ([]byte, error) {
	value, ok := args[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid type for value")
	}

	err := p.sourcehubKeeper.SendPolicy(ctx, "name: ica test policy") // TODO: add actual policy
	if err != nil {
		// convert into an EVM revert instead of panic
		return nil, vm.ErrExecutionReverted
	}

	// Key = keccak256(msg.sender, value)
	key := crypto.Keccak256Hash(contract.Caller().Bytes(), value)

	// Store in StateDB using the precompile's own address as the account
	stateDB.SetState(contract.Address(), key, common.BytesToHash(value))

	// store the view creator also
	creator := crypto.Keccak256Hash([]byte("creator"), key.Bytes())
	stateDB.SetState(contract.Address(), creator, common.BytesToHash(contract.Caller().Bytes()))

	// -----------------------
	// Emit EVM Log (Event)
	// -----------------------
	eventSignature := []byte("Registered(bytes32,address)")
	topic0 := crypto.Keccak256Hash(eventSignature)          // keccak256("Registered(bytes32,address)")
	topic1 := key                                           // indexed key
	topic2 := common.BytesToHash(contract.Caller().Bytes()) // indexed sender

	evmLog := &types.Log{
		Address: contract.Address(),
		Topics:  []common.Hash{topic0, topic1, topic2},
		Data:    value, // non-indexed payload (raw bytes stored)
	}

	stateDB.AddLog(evmLog)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"Registered",
			sdk.NewAttribute("key", key.Hex()),
			// sdk.NewAttribute("creator", contract.CallerAddress.Hex()), // can be hex or cosmos
			sdk.NewAttribute("creator", sdk.AccAddress(contract.Caller().Bytes()).String()),
			sdk.NewAttribute("view", string(value)),
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
