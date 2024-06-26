package test

import "github.com/sourcenetwork/sourcehub/x/acp/types"

func ExecMsg(executor MsgExecutor, msg any) (any, error) {
	return nil, nil
}

type MsgExecutor interface {
	CreatePolicy(ctx *TestCtx, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error)
	BearerPolicyCmd(ctx *TestCtx, msg *types.MsgBearerPolicyCmd) (*types.MsgBearerPolicyCmdResponse, error)
	PolicyCmd(ctx *TestCtx, msg *types.MsgPolicyCmd) (*types.MsgPolicyCmdResponse, error)
}

type KeeperExecutor struct {
	k types.MsgServer
}

func (e *KeeperExecutor) BearerPolicyCmd(ctx *TestCtx, msg *types.MsgBearerPolicyCmd) (*types.MsgBearerPolicyCmdResponse, error) {
	return e.k.BearerPolicyCmd(ctx, msg)
}

func (e *KeeperExecutor) PolicyCmd(ctx *TestCtx, msg *types.MsgPolicyCmd) (*types.MsgPolicyCmdResponse, error) {
	return e.k.PolicyCmd(ctx, msg)
}

func (e *KeeperExecutor) CreatePolicy(ctx *TestCtx, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error) {
	return e.k.CreatePolicy(ctx, msg)
}
