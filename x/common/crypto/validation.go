package crypto

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// VerifyDelegateSignature checks that sig (65-byte secp256k1 r||s||v) over digest
// (32 bytes) recovers to the address bytes encoded in delegateAddrBech32.
func VerifyDelegateSignature(delegateAddrBech32 string, digest, sig []byte) error {
	delegateBytes, err := sdk.AccAddressFromBech32(delegateAddrBech32)
	if err != nil {
		return fmt.Errorf("invalid delegate address: %w", err)
	}
	pub, err := ethcrypto.SigToPub(digest, sig)
	if err != nil {
		return fmt.Errorf("delegate signature recovery failed: %w", err)
	}
	recovered := ethcrypto.PubkeyToAddress(*pub)
	if !bytes.Equal(delegateBytes, recovered.Bytes()) {
		return fmt.Errorf("delegate signature mismatch: recovered %s, expected %s",
			recovered.Hex(), delegateAddrBech32)
	}
	return nil
}
