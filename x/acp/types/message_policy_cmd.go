package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgSignedPolicyCmd{}

func NewMsgSignedPolicyCmd(creator string, payload string, contentType MsgSignedPolicyCmd_ContentType) *MsgSignedPolicyCmd {
	return &MsgSignedPolicyCmd{
		Creator: creator,
		Payload: payload,
		Type:    contentType,
	}
}

func NewMsgSignedPolicyCmdFromJWS(creator string, jws string) *MsgSignedPolicyCmd {
	return &MsgSignedPolicyCmd{
		Creator: creator,
		Type:    MsgSignedPolicyCmd_JWS,
		Payload: jws,
	}
}
