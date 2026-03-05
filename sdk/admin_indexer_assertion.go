package sdk

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	shinzohubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

type AdminIndexerAssertionParams struct {
	Signer          TxSigner
	ConsensusPubKey string
	DelegateAddress string
	SourceChain     string
	SourceChainID   uint64
	AssertionID     string
	MinGasPrice     string
}

func AdminIndexerAssertion(
	ctx context.Context,
	cli *Client,
	b *TxBuilder,
	p AdminIndexerAssertionParams,
) (*sdk.TxResponse, error) {
	msg := &shinzohubtypes.MsgAdminIndexerAssertion{
		Signer:          p.Signer.GetAccAddress(),
		ConsensusPubKey: p.ConsensusPubKey,
		DelegateAddress: p.DelegateAddress,
		SourceChain:     p.SourceChain,
		SourceChainId:   p.SourceChainID,
		AssertionId:     p.AssertionID,
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
