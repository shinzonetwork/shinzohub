package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgCheckAccess{}

func NewMsgCheckAccess(creator string, policyId string, accesReq *AccessRequest) *MsgCheckAccess {
	return &MsgCheckAccess{
		Creator:       creator,
		PolicyId:      policyId,
		AccessRequest: accesReq,
	}
}
