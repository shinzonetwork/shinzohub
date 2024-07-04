package e2e

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/app"
	"github.com/sourcenetwork/sourcehub/sdk"
)

type TestNetwork struct {
	Network      *network.Network
	Config       network.Config
	Client       *sdk.Client
	ValidatorKey cryptotypes.PrivKey
}

func (n *TestNetwork) Setup(t *testing.T) {
	injectConfig := app.AppConfig()

	cfg, err := network.DefaultConfigWithAppConfig(injectConfig)
	require.NoError(t, err)
	cfg.NumValidators = 1

	network, err := network.New(t, t.TempDir(), cfg)
	require.NoError(t, err)

	n.Config = cfg
	n.Network = network

	client, err := sdk.NewClient(
		sdk.WithCometRPCAddr(n.GetCometRPCAddr()),
		sdk.WithGRPCAddr(n.GetGRPCAddr()),
	)
	require.NoError(t, err)
	n.Client = client

	keyring := n.Network.Validators[0].ClientCtx.Keyring
	record, err := keyring.Key("node0")
	require.NoError(t, err)

	any := record.GetLocal().PrivKey
	pkey := &secp256k1.PrivKey{}
	err = pkey.Unmarshal(any.Value)
	require.NoError(t, err)
	n.ValidatorKey = pkey
}

func (n *TestNetwork) TearDown() {
	n.Network.Cleanup()
}

func (n *TestNetwork) GetValidatorAddr() string {
	return n.Network.Validators[0].Address.String()
}

func (n *TestNetwork) GetGRPCAddr() string {
	return n.Network.Validators[0].AppConfig.GRPC.Address
}

func (n *TestNetwork) GetCometRPCAddr() string {
	return n.Network.Validators[0].RPCAddress
}

func (n *TestNetwork) GetSDKClient() *sdk.Client {
	return n.Client
}

func (n *TestNetwork) GetValidatorKey() cryptotypes.PrivKey {
	return n.ValidatorKey
}

func (n *TestNetwork) GetChainID() string {
	return n.Network.Config.ChainID
}
