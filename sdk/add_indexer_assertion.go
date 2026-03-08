package sdk

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	indexertypes "github.com/shinzonetwork/shinzohub/x/indexer/types"
)

// IndexerAssertionParams captures the minimal fields for indexer assertion.
type IndexerAssertionParams struct {
	Signer          TxSigner
	ConsensusPubKey string
	DelegateAddress string
	SourceChain     string
	SourceChainID   uint64
	AssertionID     string

	// DelegateDigest is the 32-byte hash that the delegate signed on the source chain.
	DelegateDigest []byte
	// DelegateSignature is the 65-byte secp256k1 r‖s‖v signature over DelegateDigest.
	DelegateSignature []byte

	// Optional per-call override.
	MinGasPrice string
}

// AddIndexerAssertion builds, signs, and broadcasts a MsgIndexerAssertion.
func AddIndexerAssertion(
	ctx context.Context,
	cli *Client,
	b *TxBuilder,
	p IndexerAssertionParams,
) (*sdk.TxResponse, error) {
	msg := &indexertypes.MsgIndexerAssertion{
		Signer:            p.Signer.GetAccAddress(),
		ConsensusPubKey:   p.ConsensusPubKey,
		DelegateAddress:   p.DelegateAddress,
		SourceChain:       p.SourceChain,
		SourceChainId:     p.SourceChainID,
		AssertionId:       p.AssertionID,
		DelegateDigest:    p.DelegateDigest,
		DelegateSignature: p.DelegateSignature,
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if p.MinGasPrice != "" {
		if err := WithMinGasPrice(p.MinGasPrice)(b); err != nil {
			return nil, err
		}
	}

	tx, err := b.Build(ctx, p.Signer, msg)
	if err != nil {
		return nil, err
	}

	return cli.BroadcastTx(ctx, tx)
}
