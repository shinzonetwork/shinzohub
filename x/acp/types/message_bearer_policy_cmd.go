package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
)

var _ sdk.Msg = &MsgDirectPolicyCmd{}

func NewMsgBearerPolicyCmd(creator string, token string, policyId string, cmd *PolicyCmd, ts *prototypes.Timestamp) *MsgBearerPolicyCmd {
	return &MsgBearerPolicyCmd{
		Creator:      creator,
		BearerToken:  token,
		PolicyId:     policyId,
		Cmd:          cmd,
		CreationTime: ts,
	}
}

func NewMsgBearerPolicyCmdNow(creator string, token string, policyId string, cmd *PolicyCmd) *MsgBearerPolicyCmd {
	return &MsgBearerPolicyCmd{
		Creator:      creator,
		BearerToken:  token,
		PolicyId:     policyId,
		Cmd:          cmd,
		CreationTime: prototypes.TimestampNow(),
	}
}
