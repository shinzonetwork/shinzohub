package poolregistry

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// The compiled Pool.sol bytecode must be present, otherwise RegisterDemandForView
// silently skips the CREATE2 deploy and join/leave can never work.
func TestPoolBytecode_IsPopulated(t *testing.T) {
	require.NotEmpty(t, PoolBytecode, "Pool.sol bytecode must be compiled in")
}

// buildPoolInitCode must be the bytecode followed by the 32-byte ABI-encoded
// constructor arg (viewAddress), with no error.
func TestBuildPoolInitCode_BytecodePlusPackedArg(t *testing.T) {
	view := common.HexToAddress("0x1111111111111111111111111111111111111111")

	initCode, err := buildPoolInitCode(view, poolConfigInput{WindowSize: 100})
	require.NoError(t, err)
	require.Len(t, initCode, len(PoolBytecode)+32, "init code = bytecode + 32-byte packed address")
	require.True(t, bytes.HasPrefix(initCode, PoolBytecode), "init code must start with the contract bytecode")
}

// derivePoolAddress must match geth's CREATE2 derivation exactly (the precompile
// asserts the deployed address equals this), be deterministic, and change when
// either the view or the config changes — otherwise distinct pools would collide.
func TestDerivePoolAddress(t *testing.T) {
	viewA := common.HexToAddress("0x1111111111111111111111111111111111111111")
	viewB := common.HexToAddress("0x2222222222222222222222222222222222222222")
	cfg100 := poolConfigInput{WindowSize: 100}
	cfg200 := poolConfigInput{WindowSize: 200}

	got := derivePoolAddress(viewA, cfg100)

	// Matches an independent CreateAddress2 over the same deployer/salt/initcode.
	initCode, err := buildPoolInitCode(viewA, cfg100)
	require.NoError(t, err)
	salt := poolSalt(viewA, cfg100)
	want := crypto.CreateAddress2(
		common.HexToAddress(PrecompileAddress),
		salt,
		crypto.Keccak256(initCode),
	)
	require.Equal(t, want, got)

	// Deterministic for identical inputs.
	require.Equal(t, got, derivePoolAddress(viewA, cfg100))

	// Sensitive to both the view and the config.
	require.NotEqual(t, got, derivePoolAddress(viewB, cfg100), "different view must yield a different pool")
	require.NotEqual(t, got, derivePoolAddress(viewA, cfg200), "different config must yield a different pool")
}
