package crypto

import (
	"fmt"

	sdkcrypto "github.com/TBD54566975/ssi-sdk/crypto"
	didkey "github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// DeriveDID derives a DID from a secp256k1 node identity public key.
//
// The key is normalized to its compressed (33-byte) SEC1 encoding before the DID
// is formed. The secp256k1 did:key multicodec (0xe7) is defined over the compressed
// key, so this yields the canonical did:key regardless of whether the caller
// supplied a compressed or uncompressed pubkey.
func DeriveDID(nodeIdPubkey []byte) (string, error) {
	pk, err := secp256k1.ParsePubKey(nodeIdPubkey)
	if err != nil {
		return "", fmt.Errorf("invalid node identity pubkey: %w", err)
	}

	didDoc, err := didkey.CreateDIDKey(sdkcrypto.SECP256k1, pk.SerializeCompressed())
	if err != nil {
		return "", fmt.Errorf("error creating did")
	}

	return didDoc.String(), nil
}
