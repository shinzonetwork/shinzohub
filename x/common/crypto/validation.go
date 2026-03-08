package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// VerifyNodeIdentityKeySignature verifies a secp256k1 DER signature over SHA-256(message).
func VerifyNodeIdentityKeySignature(pubkey, message, signature []byte) error {
	pk, err := secp256k1.ParsePubKey(pubkey)
	if err != nil {
		return fmt.Errorf("invalid pubkey: %w", err)
	}

	sig, err := ecdsa.ParseDERSignature(signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	h := sha256.Sum256(message)
	if !sig.Verify(h[:], pk) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// VerifyPeerKeySignature verifies an Ed25519 signature.
func VerifyPeerKeySignature(pubkey, message, signature []byte) error {
	if len(pubkey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid peer key pubkey length: %d", len(pubkey))
	}
	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid peer key signature length: %d", len(signature))
	}
	if !ed25519.Verify(ed25519.PublicKey(pubkey), message, signature) {
		return fmt.Errorf("invalid peer key signature")
	}
	return nil
}

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
