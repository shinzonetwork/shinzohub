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

	if err := p.emitClaimed(stateDB, contract.Address(), claimerEVM, amountBig, remaining.BigInt()); err != nil {
		return nil, err
	}

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

// emitClaimed appends the Claimed EVM log, deriving the topic hash and
// non-indexed data layout from the ABI. A pack failure is surfaced rather than
// silently emitting a zero-amount log.
func (p Precompile) emitClaimed(
	stateDB vm.StateDB,
	precompileAddr common.Address,
	claimer common.Address,
	amount *big.Int,
	remaining *big.Int,
) error {
	event := p.ABI.Events["Claimed"]
	data, err := event.Inputs.NonIndexed().Pack(amount, remaining)
	if err != nil {
		return fmt.Errorf("pack Claimed event: %w", err)
	}
	stateDB.AddLog(&gethtypes.Log{
		Address: precompileAddr,
		Topics: []common.Hash{
			event.ID,
			common.BytesToHash(claimer.Bytes()),
		},
		Data: data,
	})
	return nil
}
