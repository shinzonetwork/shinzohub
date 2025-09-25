package sdk

import (
	"fmt"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

var secp256k1URL = cdctypes.MsgTypeURL(&secp256k1.PubKey{})

type TxSigner interface {
	GetAccAddress() string
	GetPrivateKey() cryptotypes.PrivKey
}

type keyringSigner struct {
	kr     keyring.Keyring
	name   string
	pubkey cryptotypes.PubKey
}

func NewTxSignerFromKeyringKey(kr keyring.Keyring, name string) (TxSigner, error) {
	rec, err := kr.Key(name)
	if err != nil || rec == nil {
		return nil, fmt.Errorf("key %s not found", name)
	}
	if rec.PubKey.TypeUrl != secp256k1URL {
		return nil, fmt.Errorf("key %s must be secp256k1", name)
	}
	pk := &secp256k1.PubKey{}
	if err := pk.Unmarshal(rec.PubKey.Value); err != nil {
		return nil, err
	}
	return &keyringSigner{kr: kr, name: name, pubkey: pk}, nil
}

func (s *keyringSigner) GetAccAddress() string {
	return sdk.AccAddress(s.pubkey.Address().Bytes()).String()
}
func (s *keyringSigner) GetPrivateKey() cryptotypes.PrivKey {
	return &keyringPK{kr: s.kr, name: s.name, pub: s.pubkey}
}

type keyringPK struct {
	kr   keyring.Keyring
	name string
	pub  cryptotypes.PubKey
}

func (k *keyringPK) Bytes() []byte                         { panic("not supported") }
func (k *keyringPK) PubKey() cryptotypes.PubKey            { return k.pub }
func (k *keyringPK) Type() string                          { return secp256k1.PrivKeyName }
func (k *keyringPK) Equals(cryptotypes.LedgerPrivKey) bool { return false }
func (k *keyringPK) Reset()                                {}
func (k *keyringPK) ProtoMessage()                         {}
func (k *keyringPK) String() string                        { return k.Type() }
func (k *keyringPK) Sign(msg []byte) ([]byte, error) {
	bytes, _, err := k.kr.Sign(k.name, msg, signing.SignMode_SIGN_MODE_DIRECT)
	return bytes, err
}
