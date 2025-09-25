package sdk

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"

	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

var (
	ifaceReg = cdctypes.NewInterfaceRegistry()
	cdc      = codec.NewProtoCodec(ifaceReg)
)

func init() {
	cryptocdc.RegisterInterfaces(ifaceReg)
	sourcehubtypes.RegisterInterfaces(ifaceReg)
}

func Codec() *codec.ProtoCodec { return cdc }
