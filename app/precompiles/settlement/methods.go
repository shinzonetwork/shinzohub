package settlement

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
	MethodClaim     = "claim"
	MethodBalanceOf = "balanceOf"
)

func (p Precompile) Claim(
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
	if amountBig.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	claimerEVM := contract.Caller()
	claimer := sdk.AccAddress(claimerEVM.Bytes())

	pending := p.settlementKeeper.GetBalance(ctx, claimer)
	amount := math.NewIntFromBigInt(amountBig)
	if pending.LT(amount) {
		return nil, fmt.Errorf("insufficient settlement balance: have %s, want %s",
			pending.String(), amount.String())
	}

	if err := p.settlementKeeper.Claim(ctx, claimer, amount); err != nil {
		return nil, fmt.Errorf("claim: %w", err)
	}

	remaining := p.settlementKeeper.GetBalance(ctx, claimer)

	emitClaimed(stateDB, contract.Address(), claimerEVM, amountBig, remaining.BigInt())

	return method.Outputs.Pack(remaining.BigInt())
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

	balance := p.settlementKeeper.GetBalance(ctx, sdk.AccAddress(holder.Bytes()))
	return method.Outputs.Pack(balance.BigInt())
}

func emitClaimed(
	stateDB vm.StateDB,
	precompileAddr common.Address,
	claimer common.Address,
	amount *big.Int,
	remaining *big.Int,
) {
	topic0 := crypto.Keccak256Hash([]byte("Claimed(address,uint256,uint256)"))
	dataArgs := abi.Arguments{
		{Type: mustABIType("uint256")},
		{Type: mustABIType("uint256")},
	}
	data, _ := dataArgs.Pack(amount, remaining)
	stateDB.AddLog(&gethtypes.Log{
		Address: precompileAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(claimer.Bytes()),
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
