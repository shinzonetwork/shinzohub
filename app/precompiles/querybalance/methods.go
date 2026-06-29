package querybalance

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	MethodFund      = "fund"
	MethodFundFor   = "fundFor"
	MethodBalanceOf = "balanceOf"
)

func (p Precompile) Fund(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	amountBig, ok := args[0].(*big.Int)
	if !ok || amountBig == nil {
		return nil, fmt.Errorf("invalid amount")
	}
	caller := contract.Caller()
	return p.fundCore(ctx, contract, stateDB, method, caller, caller, amountBig)
}

func (p Precompile) FundFor(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	recipient, ok := args[0].(common.Address)
	if !ok || recipient == (common.Address{}) {
		return nil, fmt.Errorf("invalid recipient")
	}
	amountBig, ok := args[1].(*big.Int)
	if !ok || amountBig == nil {
		return nil, fmt.Errorf("invalid amount")
	}
	return p.fundCore(ctx, contract, stateDB, method, contract.Caller(), recipient, amountBig)
}

func (p Precompile) fundCore(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	funderEVM common.Address,
	recipientEVM common.Address,
	amountBig *big.Int,
) ([]byte, error) {
	if amountBig.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	amount := math.NewIntFromBigInt(amountBig)
	funder := sdk.AccAddress(funderEVM.Bytes())
	recipient := sdk.AccAddress(recipientEVM.Bytes())

	if err := p.qbKeeper.Fund(ctx, funder, recipient, amount); err != nil {
		return nil, fmt.Errorf("fund: %w", err)
	}

	emitFunded(stateDB, contract.Address(), funderEVM, recipientEVM, amountBig)

	return method.Outputs.Pack()
}

func (p Precompile) BalanceOf(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	holder, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid holder address")
	}

	balance := p.qbKeeper.GetBalance(ctx, sdk.AccAddress(holder.Bytes()))
	return method.Outputs.Pack(balance.BigInt())
}

func emitFunded(
	stateDB vm.StateDB,
	precompileAddr common.Address,
	funder common.Address,
	recipient common.Address,
	amount *big.Int,
) {
	topic0 := crypto.Keccak256Hash([]byte("Funded(address,address,uint256)"))
	dataArgs := abi.Arguments{
		{Type: mustABIType("uint256")},
	}
	data, _ := dataArgs.Pack(amount)
	stateDB.AddLog(&gethtypes.Log{
		Address: precompileAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(funder.Bytes()),
			common.BytesToHash(recipient.Bytes()),
		},
		Data: data,
	})
}

func mustABIType(t string) abi.Type {
	at, err := abi.NewType(t, "", nil)
	if err != nil {
		panic(err)
	}
	return at
}
