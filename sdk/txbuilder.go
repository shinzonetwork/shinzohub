package sdk

import (
	"context"
	"fmt"
	"math"

	sdkmath "cosmossdk.io/math"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdktx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type TxBuilder struct {
	chainID       string
	txCfg         sdkclient.TxConfig
	authClient    authtypes.QueryClient
	txClient      txtypes.ServiceClient
	feeDenom      string
	gasAdjustment float64
	gasLimit      uint64
	minGasPrice   string // e.g. "0.025stake"
}

type TxBuilderOpt func(*TxBuilder) error

func NewTxBuilder(opts ...TxBuilderOpt) (*TxBuilder, error) {
	reg := cdctypes.NewInterfaceRegistry()
	cfg := authtx.NewTxConfig(codec.NewProtoCodec(reg), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})

	b := &TxBuilder{
		chainID:       "shinzohub-dev",
		txCfg:         cfg,
		feeDenom:      "stake",
		gasAdjustment: 1.2,
		gasLimit:      200000,
		minGasPrice:   "0stake",
	}
	for _, o := range opts {
		if err := o(b); err != nil {
			return nil, err
		}
	}
	if b.authClient == nil || b.txClient == nil {
		return nil, fmt.Errorf("TxBuilder requires Auth and Tx clients")
	}
	return b, nil
}

func WithSDKClient(c *Client) TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.authClient = authtypes.NewQueryClient(c.conn)
		b.txClient = c.TxClient()
		return nil
	}
}
func WithChainID(id string) TxBuilderOpt {
	return func(b *TxBuilder) error { b.chainID = id; return nil }
}
func WithMinGasPrice(gp string) TxBuilderOpt {
	return func(b *TxBuilder) error { b.minGasPrice = gp; return nil }
}

func (b *TxBuilder) Build(ctx context.Context, signer TxSigner, msgs ...sdk.Msg) (xauthsigning.Tx, error) {
	txBuilder := b.txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}
	txBuilder.SetGasLimit(b.gasLimit)

	// get account
	acc, err := b.getAccount(ctx, signer.GetAccAddress())
	if err != nil {
		return nil, err
	}

	// set placeholder signature with real pubkey
	placeholder := signing.SignatureV2{
		PubKey:   signer.GetPrivateKey().PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: signing.SignMode_SIGN_MODE_DIRECT},
		Sequence: acc.GetSequence(),
	}
	if err := txBuilder.SetSignatures(placeholder); err != nil {
		return nil, err
	}

	// simulate
	enc := b.txCfg.TxEncoder()
	raw, err := enc(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}
	sim, err := b.txClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: raw})
	if err != nil {
		return nil, fmt.Errorf("simulate: %w", err)
	}

	used := sim.GetGasInfo().GetGasUsed()
	adj := uint64(math.Ceil(float64(used) * b.gasAdjustment))
	if adj == 0 {
		adj = b.gasLimit
	}
	txBuilder.SetGasLimit(adj)

	// fees = gas * min_gas_price
	fee, denom, err := parseGasPrice(b.minGasPrice)
	if err != nil {
		return nil, err
	}
	amount := fee.Mul(sdkmath.LegacyNewDecFromInt(sdkmath.NewIntFromUint64(adj))).Ceil().RoundInt()
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(denom, amount)))

	// final sign
	signerData := xauthsigning.SignerData{
		ChainID:       b.chainID,
		AccountNumber: acc.GetAccountNumber(),
		Sequence:      acc.GetSequence(),
		PubKey:        signer.GetPrivateKey().PubKey(),
	}
	sig, err := sdktx.SignWithPrivKey(ctx,
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData, txBuilder, signer.GetPrivateKey(), b.txCfg, acc.GetSequence())
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}

	if err := txBuilder.SetSignatures(sig); err != nil {
		return nil, err
	}
	return txBuilder.GetTx(), nil
}

func (b *TxBuilder) getAccount(ctx context.Context, bech string) (authtypes.BaseAccount, error) {
	resp, err := b.authClient.Account(ctx, &authtypes.QueryAccountRequest{Address: bech})
	if err != nil {
		return authtypes.BaseAccount{}, err
	}
	var base authtypes.BaseAccount
	if err := base.Unmarshal(resp.Account.Value); err != nil {
		return base, err
	}
	return base, nil
}

func parseGasPrice(gp string) (sdkmath.LegacyDec, string, error) {
	// "0.025stake"
	i := 0
	for ; i < len(gp); i++ {
		if gp[i] < '0' || gp[i] > '9' {
			if gp[i] != '.' {
				break
			}
		}
	}
	if i == 0 || i == len(gp) {
		return sdkmath.LegacyDec{}, "", fmt.Errorf("invalid gas price %q", gp)
	}
	dec, err := sdkmath.LegacyNewDecFromStr(gp[:i])
	if err != nil {
		return sdkmath.LegacyDec{}, "", err
	}
	return dec, gp[i:], nil
}
