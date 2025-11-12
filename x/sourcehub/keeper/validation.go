package keeper

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

func verifynodeIdentityKeySignature(pubkey, message, signature []byte) error {
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

func verifyPeerKeySignature(pubkey, message, signature []byte) error {
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
