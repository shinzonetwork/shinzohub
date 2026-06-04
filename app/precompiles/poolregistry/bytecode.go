package poolregistry

import "github.com/ethereum/go-ethereum/accounts/abi"

// PoolBytecode is the compiled deployment bytecode of Pool.sol.
//
// Replace with the bytecode produced by your Solidity build pipeline
// (e.g. the `bytecode.object` field from artifacts/Pool.json after `forge build`).
//
// IMPORTANT: changing this byte sequence changes every pool's deterministic
// address, so any keeper state written under the old bytecode becomes
// orphaned. Set the real value once and leave it alone.
var PoolBytecode = []byte{} // TODO: paste compiled Pool.sol bytecode

// PoolConstructorArgs packs (viewAddress) for Pool's constructor.
// The pool's configuration is not stored on the contract — callers fetch it
// from PoolRegistry.getPoolDetail via Pool.snapshot().
var PoolConstructorArgs = abi.Arguments{
	{Type: mustABIType("address")},
}
