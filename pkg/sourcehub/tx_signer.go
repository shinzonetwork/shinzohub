package sourcehub

import (
	"os"
	"encoding/hex"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TxSigner interface {
	GetAccAddress() string
	GetPrivateKey() cryptotypes.PrivKey
}

type ApiSigner struct {
	privKey cryptotypes.PrivKey
	address string
}

func NewApiSignerFromEnv() (*ApiSigner, error) {
	hexKey := os.Getenv("SHINZOHUB_PRIVATE_KEY")
	if hexKey == "" {
		return nil, ErrMissingApiSignerEnv
	}
	privKeyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	privKey := secp256k1.PrivKey{Key: privKeyBytes}
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address()).String()
	return &ApiSigner{
		privKey: &privKey,
		address: addr,
	}, nil
}

func (a *ApiSigner) GetAccAddress() string {
	return a.address
}

func (a *ApiSigner) GetPrivateKey() cryptotypes.PrivKey {
	return a.privKey
}

var ErrMissingApiSignerEnv = &MissingApiSignerEnvError{}

type MissingApiSignerEnvError struct{}

func (e *MissingApiSignerEnvError) Error() string {
	return "SHINZOHUB_PRIVATE_KEY environment variable not set"
} 