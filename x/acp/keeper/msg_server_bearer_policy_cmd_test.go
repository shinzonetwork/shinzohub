package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/bearer_token"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func setupTestBearerPolicyCmdSetRelationship(t *testing.T) (ctx sdk.Context, srv types.MsgServer, pol *types.Policy, creator string, alice string, bearer string) {
	policy := `
    name: policy
    resources:
      file:
        relations:
          owner:
            types:
              - actor
          admin:
            manages:
              - reader
            types:
              - actor
          reader:
            types:
              - actor
    `
	ctx, srv, accKeep := setupMsgServer(t)
	creator = accKeep.FirstAcc().GetAddress().String()

	resp, err := srv.CreatePolicy(ctx, &types.MsgCreatePolicy{
		Creator:      creator,
		CreationTime: timestamp,
		Policy:       policy,
		MarshalType:  types.PolicyMarshalingType_SHORT_YAML,
	})
	require.Nil(t, err)
	pol = resp.Policy

	alice, signer := mustGenerateActor()

	ctx = ctx.WithBlockTime(time.Date(2024, 06, 18, 10, 0, 0, 0, time.UTC))
	token := bearer_token.NewBearerTokenFromTime(
		alice,
		creator,
		time.Date(2024, 06, 18, 9, 0, 0, 0, time.UTC),
		time.Date(2024, 06, 18, 11, 0, 0, 0, time.UTC),
	)
	bearer, err = token.ToJWS(signer)
	require.NoError(t, err)

	return
}

func TestBearerPolicyCmd_SetRelationship_OwnerCanSetRelationshipForObjectTheyOwn(t *testing.T) {
	ctx, srv, pol, creator, alice, token := setupTestBearerPolicyCmdSetRelationship(t)

	msg := types.MsgBearerPolicyCmd{
		Creator:      creator,
		BearerToken:  token,
		PolicyId:     pol.Id,
		Cmd:          types.NewSetRelationshipCmd(types.NewActorRelationship("file", "foo", "reader", alice)),
		CreationTime: timestamp,
	}
	got, err := srv.BearerPolicyCmd(ctx, &msg)

	want := &types.MsgBearerPolicyCmdResponse{
		Result: &types.PolicyCmdResult{
			Result: &types.PolicyCmdResult_SetRelationshipResult{
				SetRelationshipResult: &types.SetRelationshipCmdResult{
					RecordExisted: false,
					Record: &types.RelationshipRecord{
						Actor:        alice,
						CreationTime: timestamp,
						Creator:      creator,
						PolicyId:     pol.Id,
						Relationship: types.NewActorRelationship("file", "foo", "reader", alice),
						Archived:     false,
					},
				},
			},
		},
	}
	require.Nil(t, err)
	require.Equal(t, want, got)
}
