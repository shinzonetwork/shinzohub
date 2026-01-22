package viewregistry

import (
	"encoding/base64"
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

	log := ctx.Logger().With(
		"module", "precompile.viewregistry",
		"method", "register",
		"caller", contract.Caller().Hex(),
		"contract", contract.Address().Hex(),
	)

	encodedValue, ok := args[0].([]byte)
	if !ok {
		log.Error("invalid arg type for value", "got", fmt.Sprintf("%T", args[0]))
		return nil, fmt.Errorf("invalid type for value")
	}

	// Don’t log bytes, log size + hash.
	log.Info("register received", "bytes", len(encodedValue), "hash", crypto.Keccak256Hash(encodedValue).Hex())

	decodedValue, err := viewbundle.DecodeHeader(encodedValue)
	if err != nil {
		log.Error("viewbundle decode failed", "err", err, "bytes", len(encodedValue))
		return nil, vm.ErrExecutionReverted
	}

	re := regexp.MustCompile(`\btype\s+([A-Za-z0-9_]+)\b`)
	matches := re.FindStringSubmatch(decodedValue.Header.Sdl)
	if len(matches) < 2 {
		log.Error("sdl parse failed: missing type name", "sdl_len", len(decodedValue.Header.Sdl))
		return nil, vm.ErrExecutionReverted
	}
	resourceName := matches[1]

	key := crypto.Keccak256Hash(contract.Caller().Bytes(), encodedValue)
	id := fmt.Sprintf("%s_%s", resourceName, key.Hex())

	loc := re.FindStringSubmatchIndex(decodedValue.Header.Sdl)
	if len(loc) < 4 {
		log.Error("sdl parse failed: could not locate type name", "loc", fmt.Sprintf("%v", loc))
		return nil, vm.ErrExecutionReverted
	}

	decodedValue.Header.Sdl = decodedValue.Header.Sdl[:loc[2]] + id + decodedValue.Header.Sdl[loc[3]:]

	newEncodedValue, err := viewbundle.EncodeHeader(decodedValue)
	if err != nil {
		log.Error("viewbundle encode failed", "err", err)
		return nil, vm.ErrExecutionReverted
	}

	// Keeper call
	if err := p.sourcehubKeeper.RegisterObject(ctx, id); err != nil {
		log.Error("RegisterObject failed", "err", err, "id", id)
		return nil, vm.ErrExecutionReverted
	}

	// Store creator mapping
	creator := crypto.Keccak256Hash([]byte("view.creator"), key.Bytes())
	stateDB.SetState(
		contract.Address(),
		creator,
		common.BytesToHash(common.LeftPadBytes(contract.Caller().Bytes(), 32)),
	)

	eventSignature := []byte("Registered(bytes32,address)")
	topic0 := crypto.Keccak256Hash(eventSignature)

	stateDB.AddLog(&types.Log{
		Address: contract.Address(),
		Topics:  []common.Hash{topic0, key, common.BytesToHash(contract.Caller().Bytes())},
		Data:    newEncodedValue,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"Registered",
			sdk.NewAttribute("key", key.Hex()),
			sdk.NewAttribute("creator", sdk.AccAddress(contract.Caller().Bytes()).String()),
			sdk.NewAttribute("view", base64.StdEncoding.EncodeToString(newEncodedValue)),
		),
	)

	log.Info("register success", "id", id, "key", key.Hex(), "payload_bytes", len(base64.StdEncoding.EncodeToString(newEncodedValue)))

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
