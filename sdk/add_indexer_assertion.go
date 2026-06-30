package sdk

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	indexertypes "github.com/shinzonetwork/shinzohub/x/indexer/types"
)

// IndexerAssertionParams captures the fields for an indexer assertion mirrored
// from a source-chain outpost event.
type IndexerAssertionParams struct {
	Signer TxSigner

	SourceChain        string
	SourceChainID      uint64
	ValidatorPubkey    []byte
	AssertionAuthority []byte
	Nonce              uint64
	ChainSpecific      []byte

	OperatorAddress string
	PayoutAddress   string

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
		Signer:             p.Signer.GetAccAddress(),
		SourceChain:        p.SourceChain,
		SourceChainId:      p.SourceChainID,
		ValidatorPubkey:    p.ValidatorPubkey,
		AssertionAuthority: p.AssertionAuthority,
		Nonce:              p.Nonce,
		ChainSpecific:      p.ChainSpecific,
		OperatorAddress:    p.OperatorAddress,
		PayoutAddress:      p.PayoutAddress,
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
