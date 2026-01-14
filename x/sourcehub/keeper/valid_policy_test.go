package keeper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/acp_core/pkg/auth"
	"github.com/sourcenetwork/acp_core/pkg/runtime"
	"github.com/sourcenetwork/acp_core/pkg/services"
	"github.com/sourcenetwork/acp_core/pkg/types"
)

// Test ensure the YAML ACP Policy does not become stale,
// by trying to create it with against the acp engine directly
func Test_ShinzoYamlPolicyIsValid(t *testing.T) {
	manager, err := runtime.NewRuntimeManager(
		runtime.WithMemKV(),
	)
	require.NoError(t, err)

	engine := services.NewACPEngine(manager)

	ctx := context.Background()
	ctx = auth.InjectPrincipal(ctx, types.RootPrincipal())
	_, err = engine.CreatePolicy(ctx, &types.CreatePolicyRequest{
		Policy:      policy,
		MarshalType: types.PolicyMarshalingType_YAML,
	})
	require.NoError(t, err)
}
