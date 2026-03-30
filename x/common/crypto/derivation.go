package crypto

import (
	"fmt"

	sdkcrypto "github.com/TBD54566975/ssi-sdk/crypto"
	didkey "github.com/TBD54566975/ssi-sdk/did/key"
)

// DeriveDID derives a DID from a secp256k1 node identity public key.
func DeriveDID(nodeIdPubkey []byte) (string, error) {
	didDoc, err := didkey.CreateDIDKey(sdkcrypto.SECP256k1, nodeIdPubkey)
	if err != nil {
		return "", fmt.Errorf("error creating did")
	}

	return didDoc.String(), nil
}
