package crypto

import (
	"fmt"

	sdkcrypto "github.com/TBD54566975/ssi-sdk/crypto"
	didkey "github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// DeriveDID derives a DID from a secp256k1 node identity public key.
//
// The key is normalized to its uncompressed (65-byte) SEC1 encoding before the
// DID is formed, so the resulting did:key is canonical regardless of whether the
// caller supplied a compressed or uncompressed pubkey.
func DeriveDID(nodeIdPubkey []byte) (string, error) {
	pk, err := secp256k1.ParsePubKey(nodeIdPubkey)
	if err != nil {
		return "", fmt.Errorf("invalid node identity pubkey: %w", err)
	}

	didDoc, err := didkey.CreateDIDKey(sdkcrypto.SECP256k1, pk.SerializeUncompressed())
	if err != nil {
		return "", fmt.Errorf("error creating did")
	}

	return didDoc.String(), nil
}
