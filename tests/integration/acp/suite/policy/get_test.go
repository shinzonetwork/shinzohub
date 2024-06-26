package policy

/*
import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/acp_core/test"
)

func TestQueryPolicy_UnknownPolicyReturnsPolicyNotFoundErr(t *testing.T) {
	ctx := test.NewTestCtx(t)

	req := types.GetPolicyRequest{
		Id: "blahblabh",
	}

	resp, err := ctx.Engine.GetPolicy(ctx, &req)

	require.Nil(t, resp)
	require.ErrorIs(t, err, types.ErrPolicyNotFound)
}

func TestGetPolicy_ReturnsAnExistingPolicy(t *testing.T) {
	ctx := test.NewTestCtx(t)

	bob := ctx.Actors.DID("bob")
	ctx.SetPrincipal("bob")

	pol := `
name: policy
`
	action := test.CreatePolicyAction{
		Policy: pol,
	}
	action.Run(ctx)

	req := types.GetPolicyRequest{
		Id: "a969e15fbc568e85a4fadf4758b0fc69ae59248e7ffc983b6caa63bcff19c3cc",
	}
	resp, err := ctx.Engine.GetPolicy(ctx, &req)

	want := &types.Policy{
		Id:           "a969e15fbc568e85a4fadf4758b0fc69ae59248e7ffc983b6caa63bcff19c3cc",
		Name:         "policy",
		Creator:      bob,
		CreationTime: test.DefaultTs,
		ActorResource: &types.ActorResource{
			Name: "actor",
		},
	}
	require.Equal(t, resp.Policy, want)
	require.NoError(t, err)
}

*/
