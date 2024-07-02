package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
)

var _ sdk.Msg = &MsgDirectPolicyCmd{}

func NewMsgDirectPolicyCmd(creator string, policyId string, cmd *PolicyCmd, ts *prototypes.Timestamp) *MsgDirectPolicyCmd {
	return &MsgDirectPolicyCmd{
		Creator:      creator,
		PolicyId:     policyId,
		Cmd:          cmd,
		CreationTime: ts,
	}
}

func NewMsgDirectPolicyCmdNow(creator string, policyId string, cmd *PolicyCmd) *MsgDirectPolicyCmd {
	return &MsgDirectPolicyCmd{
		Creator:      creator,
		PolicyId:     policyId,
		Cmd:          cmd,
		CreationTime: prototypes.TimestampNow(),
	}
}
