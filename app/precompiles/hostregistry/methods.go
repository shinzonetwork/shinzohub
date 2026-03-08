package hostregistry

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
	MethodRegister     = "register"
	MethodIsRegistered = "isRegistered"
	MethodGetDid       = "getDid"
	MethodGetPid       = "getPid"
)

func (p *Precompile) Register(
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

	caller := contract.Caller().Bytes()

	did, pid, err := p.hostKeeper.RegisterHost(
		ctx,
		peerKeyPubkey,
		peerKeySignature,
		nodeIdentityKeyPubkey,
		nodeIdentityKeySignature,
		message,
		caller,
	)
	if err != nil {
		return nil, err
	}

	precompAddr := contract.Address()
	topic0 := crypto.Keccak256Hash([]byte("Registered(address,bytes,bytes)"))
	event := p.ABI.Events["Registered"]
	data, err := event.Inputs.NonIndexed().Pack(did, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to pack Registered event: %w", err)
	}
	stateDB.AddLog(&gethtypes.Log{
		Address: precompAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(caller),
		},
		Data: data,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"HostRegistered",
			sdk.NewAttribute("owner", sdk.AccAddress(caller).String()),
			sdk.NewAttribute("did", string(did)),
			sdk.NewAttribute("pid", string(pid)),
		),
	)

	return nil, nil
}

func (p *Precompile) IsRegistered(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid type for addr")
	}

	registered := p.hostKeeper.IsRegisteredHost(ctx, addr.Bytes())
	return method.Outputs.Pack(registered)
}

func (p *Precompile) GetDid(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid type for addr")
	}

	bech32Addr := sdk.AccAddress(addr.Bytes()).String()
	host, found, err := p.hostKeeper.GetHost(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack([]byte{})
	}
	return method.Outputs.Pack([]byte(host.Did))
}

func (p *Precompile) GetPid(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid type for addr")
	}

	bech32Addr := sdk.AccAddress(addr.Bytes()).String()
	host, found, err := p.hostKeeper.GetHost(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack([]byte{})
	}
	return method.Outputs.Pack([]byte(host.Pid))
}
