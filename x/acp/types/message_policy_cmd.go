package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSignedPolicyCmd{}

func NewMsgPolicyCmd(creator string, payload string, contentType MsgSignedPolicyCmd_ContentType) *MsgSignedPolicyCmd {
	return &MsgSignedPolicyCmd{
		Creator: creator,
		Payload: payload,
		Type:    contentType,
	}
}

func NewMsgPolicyCmdFromJWS(creator string, jws string) *MsgSignedPolicyCmd {
	return &MsgSignedPolicyCmd{
		Creator: creator,
		Type:    MsgSignedPolicyCmd_JWS,
		Payload: jws,
	}
}

func (msg *MsgSignedPolicyCmd) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
