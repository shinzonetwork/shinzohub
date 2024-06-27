package did

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cyware/ssi-sdk/crypto"
	"github.com/cyware/ssi-sdk/did"
	"github.com/cyware/ssi-sdk/did/key"
	"github.com/cyware/ssi-sdk/did/resolution"
)

func ExtractVerificationKey(doc DIDDocument) (cryptotypes.PubKey, error) {
	return nil, fmt.Errorf("ExtractVerificationKey not implemented")
}

type DIDDocument struct {
	*did.Document
}

type Resolver interface {
	Resolve(ctx context.Context, did string) (DIDDocument, error)
}

func IsValidDID(didStr string) error {
	_, err := resolution.GetMethodForDID(didStr)
	if err != nil {
		return err
	}
	return nil
}

// IssueDID produces a DID for a SourceHub account
func IssueDID(acc sdk.AccountI) (string, error) {
	return DIDFromPubKey(acc.GetPubKey())
}

func DIDFromPubKey(pk cryptotypes.PubKey) (string, error) {
	var keyType crypto.KeyType
	switch t := pk.(type) {
	case *secp256k1.PubKey:
		keyType = crypto.SECP256k1
	case *ed25519.PubKey:
		keyType = crypto.Ed25519
	default:
		return "", fmt.Errorf("failed to issue did for key %v: account key type must be secp256k1 or ed25519, got %v", pk.Bytes(), t)
	}

	did, err := key.CreateDIDKey(keyType, pk.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to generated did for %v: %v", pk.Bytes(), err)
	}

	return did.String(), nil
}
