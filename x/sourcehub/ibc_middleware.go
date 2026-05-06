package sourcehub

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/keeper"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

type IBCMiddleware struct {
	keeper keeper.Keeper
	next   porttypes.IBCModule
}

func NewIBCMiddleware(k keeper.Keeper, next porttypes.IBCModule) IBCMiddleware {
	return IBCMiddleware{keeper: k, next: next}
}

var _ porttypes.IBCModule = IBCMiddleware{}

func (m IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID, channelID string,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return m.next.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

func (m IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID, channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return m.next.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

func (m IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID, channelID, counterpartyChannelID, counterpartyVersion string,
) error {
	return m.next.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (m IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return m.next.OnChanOpenConfirm(ctx, portID, channelID)
}

func (m IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return m.next.OnChanCloseInit(ctx, portID, channelID)
}

func (m IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return m.next.OnChanCloseConfirm(ctx, portID, channelID)
}

func (m IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	return m.next.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

func (m IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	status, errMsg, msgResponses := decodeAck(acknowledgement)
	m.resolveAndDispatch(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence, status, errMsg, msgResponses)
	return m.next.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

func (m IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	m.resolveAndDispatch(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence, types.RequestStatus_REQUEST_STATUS_TIMEOUT, "packet timed out", nil)
	return m.next.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

func (m IBCMiddleware) resolveAndDispatch(
	ctx sdk.Context,
	portID, channelID string,
	sequence uint64,
	status types.RequestStatus,
	errMsg string,
	msgResponses [][]byte,
) {
	req, found, err := m.keeper.ResolvePendingRequest(ctx, portID, channelID, sequence, status, errMsg, msgResponses)
	if err != nil || !found {
		return
	}

	emitResolvedEvent(ctx, req)

	for _, cb := range m.keeper.GetAckCallbacks(req.Kind) {
		_ = cb.OnPacketAck(ctx, req)
	}
}

func decodeAck(bz []byte) (types.RequestStatus, string, [][]byte) {
	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(bz, &ack); err != nil {
		return types.RequestStatus_REQUEST_STATUS_FAILURE, "unparseable acknowledgement: " + err.Error(), nil
	}

	if !ack.Success() {
		return types.RequestStatus_REQUEST_STATUS_FAILURE, ack.GetError(), nil
	}

	var txMsgData sdk.TxMsgData
	if err := gogoproto.Unmarshal(ack.GetResult(), &txMsgData); err != nil {
		return types.RequestStatus_REQUEST_STATUS_SUCCESS, "", nil
	}
	responses := make([][]byte, 0, len(txMsgData.MsgResponses))
	for _, r := range txMsgData.MsgResponses {
		if r == nil {
			continue
		}
		responses = append(responses, r.Value)
	}
	return types.RequestStatus_REQUEST_STATUS_SUCCESS, "", responses
}

func emitResolvedEvent(ctx sdk.Context, req types.PendingICARequest) {
	eventType := types.EventTypeRequestAcknowledged
	switch req.Status {
	case types.RequestStatus_REQUEST_STATUS_FAILURE:
		eventType = types.EventTypeRequestFailed
	case types.RequestStatus_REQUEST_STATUS_TIMEOUT:
		eventType = types.EventTypeRequestTimedOut
	}
	attrs := []sdk.Attribute{
		sdk.NewAttribute(types.AttrKeySequence, itoa(req.Sequence)),
		sdk.NewAttribute(types.AttrKeyPortID, req.PortId),
		sdk.NewAttribute(types.AttrKeyChannelID, req.ChannelId),
		sdk.NewAttribute(types.AttrKeyRequestKind, req.Kind.String()),
		sdk.NewAttribute(types.AttrKeyRequestor, req.Requestor),
	}
	if req.Error != "" {
		attrs = append(attrs, sdk.NewAttribute(types.AttrKeyError, req.Error))
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(eventType, attrs...))
}

func itoa(v uint64) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
}
