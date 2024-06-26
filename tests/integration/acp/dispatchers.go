package test

import (
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/bearer_token"
	"github.com/sourcenetwork/sourcehub/x/acp/policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type AuthenticationStrategy int

const (
	Direct AuthenticationStrategy = iota
	BearerToken
	SignedPayload
)

func setRelationshipDispatcher(ctx *TestCtx, action *SetRelationshipAction) (result *types.SetRelationshipCmdResult, err error) {
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, action.Actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:      ctx.TxSigner.SourceHubAddr,
			BearerToken:  jws,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewSetRelationshipCmd(action.Relationship),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetSetRelationshipResult()
		}
		err = respErr
	case SignedPayload:
		builder := policy_cmd.CmdBuilder{}
		builder.SetRelationship(action.Relationship)
		builder.Actor(action.Actor.DID)
		builder.CreationTimestamp(TimeToProto(ctx.TokenIssueTs))
		builder.PolicyID(action.PolicyId)
		builder.SetSigner(action.Actor.Signer)
		jws, err := builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgPolicyCmd_JWS,
		}
		//container, err = ctx.Executor.PolicyCmd(ctx, msg)
		_ = msg
	}
	return
}

func deleteRelationshipDispatcher(ctx *TestCtx, action *DeleteRelationshipAction) (*types.DeleteRelationshipCmdResult, error) {
	var result *types.DeleteRelationshipCmdResult
	var resultErr error
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, action.Actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:      ctx.TxSigner.SourceHubAddr,
			BearerToken:  jws,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewDeleteRelationshipCmd(action.Relationship),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, err := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetDeleteRelationshipResult()
		}
		resultErr = err
	case SignedPayload:
		builder := policy_cmd.CmdBuilder{}
		builder.DeleteRelationship(action.Relationship)
		builder.Actor(action.Actor.DID)
		builder.CreationTimestamp(TimeToProto(ctx.TokenIssueTs))
		builder.PolicyID(action.PolicyId)
		builder.SetSigner(action.Actor.Signer)
		jws, err := builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgPolicyCmd_JWS,
		}
		//container, err = ctx.Executor.PolicyCmd(ctx, msg)
		_ = msg
	}
	return result, resultErr
}

func registerObjectDispatcher(ctx *TestCtx, action *RegisterObjectAction) (result *types.RegisterObjectCmdResult, err error) {
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, action.Actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:      ctx.TxSigner.SourceHubAddr,
			BearerToken:  jws,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewRegisterObjectCmd(action.Object),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetRegisterObjectResult()
		}
		err = respErr
	case SignedPayload:
		builder := policy_cmd.CmdBuilder{}
		builder.RegisterObject(action.Object)
		builder.Actor(action.Actor.DID)
		builder.CreationTimestamp(TimeToProto(ctx.TokenIssueTs))
		builder.PolicyID(action.PolicyId)
		builder.SetSigner(action.Actor.Signer)
		jws, err := builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgPolicyCmd_JWS,
		}
		//container, err = ctx.Executor.PolicyCmd(ctx, msg)
		_ = msg
	}
	return result, err
}

func unregisterObjectDispatcher(ctx *TestCtx, action *UnregisterObjectAction) (result *types.UnregisterObjectCmdResult, err error) {
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, action.Actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:      ctx.TxSigner.SourceHubAddr,
			BearerToken:  jws,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewUnregisterObjectCmd(action.Object),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetUnregisterObjectResult()
		}
		err = respErr
	case SignedPayload:
		builder := policy_cmd.CmdBuilder{}
		builder.UnregisterObject(action.Object)
		builder.Actor(action.Actor.DID)
		builder.CreationTimestamp(TimeToProto(ctx.TokenIssueTs))
		builder.PolicyID(action.PolicyId)
		builder.SetSigner(action.Actor.Signer)
		jws, err := builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgPolicyCmd_JWS,
		}
		//container, err = ctx.Executor.PolicyCmd(ctx, msg)
		_ = msg
	}
	return result, err
}

func genToken(ctx *TestCtx, actor *TestActor) string {
	token := bearer_token.BearerToken{
		IssuerID:          actor.DID,
		AuthorizedAccount: ctx.TxSigner.SourceHubAddr,
		IssuedTime:        ctx.TokenIssueTs.Unix(),
		ExpirationTime:    ctx.TokenIssueTs.Add(bearer_token.DefaultExpirationTime).Unix(),
	}
	jws, err := token.ToJWS(actor.Signer)
	require.NoError(ctx.T, err)
	return jws
}
