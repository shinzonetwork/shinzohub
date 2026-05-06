package sourcehub

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func TestDecodeAck_Success(t *testing.T) {
	resp := &types.RegisterObjectMeta{ResourceName: "view", ObjectId: "v1"}
	anyResp, err := codectypes.NewAnyWithValue(resp)
	require.NoError(t, err)

	txMsgData := sdk.TxMsgData{MsgResponses: []*codectypes.Any{anyResp}}
	resultBz, err := gogoproto.Marshal(&txMsgData)
	require.NoError(t, err)

	ack := channeltypes.NewResultAcknowledgement(resultBz)
	ackBz := channeltypes.SubModuleCdc.MustMarshalJSON(&ack)

	status, errMsg, responses := decodeAck(ackBz)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_SUCCESS, status)
	require.Empty(t, errMsg)
	require.Len(t, responses, 1)
	require.NotEmpty(t, responses[0])
}

func TestDecodeAck_Error(t *testing.T) {
	ack := channeltypes.NewErrorAcknowledgement(errReason("policy not found"))
	ackBz := channeltypes.SubModuleCdc.MustMarshalJSON(&ack)

	status, errMsg, responses := decodeAck(ackBz)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_FAILURE, status)
	require.Contains(t, errMsg, "error handling packet")
	require.Nil(t, responses)
}

func TestDecodeAck_Malformed(t *testing.T) {
	status, errMsg, responses := decodeAck([]byte("{not a real ack"))
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_FAILURE, status)
	require.Contains(t, errMsg, "unparseable")
	require.Nil(t, responses)
}

func TestDecodeAck_SuccessNoMsgData(t *testing.T) {
	ack := channeltypes.NewResultAcknowledgement([]byte("opaque ok"))
	ackBz := channeltypes.SubModuleCdc.MustMarshalJSON(&ack)

	status, errMsg, _ := decodeAck(ackBz)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_SUCCESS, status)
	require.Empty(t, errMsg)
}

type errReason string

func (e errReason) Error() string { return string(e) }
