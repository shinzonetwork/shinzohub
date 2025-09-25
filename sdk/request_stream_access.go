// sdk/request_stream_access.go
package sdk

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	shinzohubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

type RequestStreamAccessParams struct {
	Signer      TxSigner
	StreamId    string
	Resource    shinzohubtypes.Resource
	Identity    string
	Expiration  uint64
	MinGasPrice string
}

func RequestStreamAccess(ctx context.Context, cli *Client, b *TxBuilder, p RequestStreamAccessParams) (*sdk.TxResponse, error) {
	msg := &shinzohubtypes.MsgRequestStreamAccess{
		Signer:     p.Signer.GetAccAddress(),
		StreamId:   p.StreamId,
		Did:        p.Identity,
		Resource:   p.Resource,
		Expiration: p.Expiration,
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	// allow per-call min gas price override
	if p.MinGasPrice != "" {
		if err := WithMinGasPrice(p.MinGasPrice)(b); err != nil {
			return nil, err
		}
	}

	tx, err := b.Build(ctx, p.Signer, msg)
	if err != nil {
		return nil, err
	}

	mode := txtypes.BroadcastMode_BROADCAST_MODE_SYNC

	encode := authtx.DefaultTxEncoder()
	bz, err := encode(tx)
	if err != nil {
		return nil, err
	}

	res, err := cli.TxClient().BroadcastTx(ctx, &txtypes.BroadcastTxRequest{TxBytes: bz, Mode: mode})
	if err != nil {
		return nil, err
	}
	if res.TxResponse.Code != 0 {
		return res.TxResponse, fmt.Errorf("rejected: %s", res.TxResponse.RawLog)
	}
	return res.TxResponse, nil
}
