package querybalance

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
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
	_ []interface{},
) ([]byte, error) {
	caller := contract.Caller()
	return p.fundCore(ctx, contract, stateDB, method, caller, caller)
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
	return p.fundCore(ctx, contract, stateDB, method, contract.Caller(), recipient)
}

func (p Precompile) fundCore(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	funderEVM common.Address,
	recipientEVM common.Address,
) ([]byte, error) {
	value := contract.Value()
	if value == nil || value.Sign() == 0 {
		return nil, fmt.Errorf("must send a non-zero amount")
	}

	// msg.value is denominated in the EVM coin's extended denom; use it directly
	// rather than the staking bond denom so the escrow always matches the coin
	// the EVM actually parked at this precompile.
	amount := sdk.NewCoin(
		evmtypes.GetEVMCoinExtendedDenom(),
		math.NewIntFromBigInt(value.ToBig()),
	)

	funder := sdk.AccAddress(funderEVM.Bytes())
	recipient := sdk.AccAddress(recipientEVM.Bytes())
	// A payable call parks msg.value in this precompile's own account. Escrow it
	// from there into the module account — pulling from the funder again would
	// double-charge them and strand the value here.
	escrowAcc := sdk.AccAddress(contract.Address().Bytes())

	if err := p.qbKeeper.EscrowAndCredit(ctx, escrowAcc, funder, recipient, amount); err != nil {
		return nil, fmt.Errorf("fund: %w", err)
	}

	// The bank transfer above bypasses the EVM journal, so reconcile the StateDB
	// to keep the precompile's EVM-visible balance consistent with the move.
	stateDB.SubBalance(contract.Address(), value, tracing.BalanceChangeUnspecified)

	emitFunded(stateDB, contract.Address(), funderEVM, recipientEVM, value.ToBig())

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
