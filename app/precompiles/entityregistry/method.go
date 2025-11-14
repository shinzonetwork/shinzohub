package entityregistry

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	EntityRegistryRegisterMethod = "register"
)

func (p Precompile) EntityRegistryRegister(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	_ *abi.Method,
	args []interface{},
) ([]byte, error) {
	peerKeyPubkey, ok := args[0].([]byte)
	if !ok || len(peerKeyPubkey) == 0 {
		return nil, fmt.Errorf("invalid peerKeyPubkey")
	}

	peerKeySignature, ok := args[1].([]byte)
	if !ok || len(peerKeySignature) == 0 {
		return nil, fmt.Errorf("invalid peerKeySignature")
	}

	nodeIdentityKeyPubkey, ok := args[2].([]byte)
	if !ok || len(nodeIdentityKeyPubkey) == 0 {
		return nil, fmt.Errorf("invalid nodeIdentityKeyPubkey")
	}

	nodeIdentityKeySignature, ok := args[3].([]byte)
	if !ok || len(nodeIdentityKeySignature) == 0 {
		return nil, fmt.Errorf("invalid nodeIdentityKeySignature")
	}

	message, ok := args[4].([]byte)
	if !ok || len(message) == 0 {
		return nil, fmt.Errorf("invalid message")
	}

	entity, ok := args[5].(uint8)
	if !ok {
		return nil, fmt.Errorf("invalid entity")
	}

	caller := contract.Caller().Bytes()

	did, pid, err := p.sourcehubKeeper.RegisterEntity(
		ctx,
		peerKeyPubkey,
		peerKeySignature,
		nodeIdentityKeyPubkey,
		nodeIdentityKeySignature,
		message,
		entity,
		caller,
	)
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	key := crypto.Keccak256Hash(caller, did)

	// topic0 = keccak256("EntityRegistered(bytes32,address,bytes,bytes,uint8)")
	topic0 := crypto.Keccak256Hash([]byte("EntityRegistered(bytes32,address,bytes,bytes,uint8)"))

	// Encode (did, pid) as event data
	argsDef := abi.Arguments{
		{Type: abi.Type{T: abi.BytesTy}},
		{Type: abi.Type{T: abi.BytesTy}},
		{Type: abi.Type{T: abi.UintTy}},
	}

	data, packErr := argsDef.Pack(did, pid, entity)
	if packErr != nil {
		return nil, vm.ErrExecutionReverted
	}

	stateDB.AddLog(&gethtypes.Log{
		Address: contract.Address(),
		Topics: []common.Hash{
			topic0,
			key,
			common.BytesToHash(caller),
		},
		Data: data,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"EntityRegistered",
			sdk.NewAttribute("key", key.Hex()),
			sdk.NewAttribute("owner", sdk.AccAddress(caller).String()),
			sdk.NewAttribute("did", string(did)),
			sdk.NewAttribute("pid", string(pid)),
			sdk.NewAttribute("entity", string(entity)),
		),
	)

	return nil, nil
}
