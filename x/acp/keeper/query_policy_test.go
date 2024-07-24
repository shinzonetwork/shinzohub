package keeper

import (
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func TestQueryPolicy_UnknownPolicyReturnsPolicyNotFoundErr(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	req := types.QueryPolicyRequest{
		Id: "not found",
	}

	resp, err := k.Policy(ctx, &req)

	require.Nil(t, resp)
	require.ErrorIs(t, err, errors.ErrorType_NOT_FOUND)
}
