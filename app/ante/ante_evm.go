package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmante "github.com/cosmos/evm/ante/evm"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/shinzonetwork/shinzohub/app/decorators"
)

// newMonoEVMAnteHandler creates the sdk.AnteHandler implementation for the EVM transactions.
func newMonoEVMAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.NewEVMMonoDecorator(
			options.AccountKeeper,
			options.FeeMarketKeeper,
			options.EvmKeeper,
			options.MaxTxGasWanted,
		),
		// After the mono decorator: it runs signature verification, which is what
		// populates MsgEthereumTx.From, so the sender is available here.
		evmParticipantDecorator{},
	)
}

// evmParticipantDecorator emits the shinzo participant events for an EVM tx: the
// recovered sender (From) and, when present, the recipient (To; nil for contract
// creation). Both are the bech32 of the 20-byte EVM address, the form the bank
// module already uses for an EVM account, so an account keeps one address key
// across its EVM and Cosmos activity.
type evmParticipantDecorator struct{}

func (evmParticipantDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	var senders, recipients []string
	for _, msg := range tx.GetMsgs() {
		ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			continue
		}
		senders = append(senders, ethMsg.GetFrom().String())
		// AsTransaction is non-nil here in practice (the mono decorator already
		// unpacked and verified the inner tx), but guard it so a reordering of the
		// chain can't turn this into an ante-path panic.
		if ethTx := ethMsg.AsTransaction(); ethTx != nil {
			if to := ethTx.To(); to != nil {
				recipients = append(recipients, sdk.AccAddress(to.Bytes()).String())
			}
		}
	}
	decorators.EmitParticipants(ctx, senders, recipients)

	return next(ctx, tx, simulate)
}
