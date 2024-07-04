package policy

import (
	"fmt"

	"cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"

	"github.com/sourcenetwork/sourcehub/x/acp/auth_engine"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// CreatePolicyCommand models an instruction to createa a new ACP Policy
type CreatePolicyCommand struct {
	// Cosmos Address of the Policy Creator
	Creator string

	// Policy Intermediary Representation
	Policy PolicyIR

	// Timestamp for Policy creation
	CreationTime *prototypes.Timestamp
}

// Execute consumes the data supplied in the command and creates a new ACP Policy and stores it in the given engine.
func (c *CreatePolicyCommand) Execute(ctx sdk.Context, kv store.KVStore, engine auth_engine.AuthEngine) (*types.Policy, error) {
	err := basicPolicyIRSpec(&c.Policy)
	if err != nil {
		return nil, fmt.Errorf("CreatePolicyCommand: %w", err)
	}

	counter := newPolicyCounter(kv)
	i, err := counter.GetNextAndIncrement(ctx)
	if err != nil {
		return nil, fmt.Errorf("CreatePolicyCommand: %v: %w", err, types.ErrAcpInternal)
	}

	factory := factory{}
	record, err := factory.Create(c.Policy, c.Creator, i, c.CreationTime)
	if err != nil {
		return nil, fmt.Errorf("CreatePolicyCommand: %w", err)
	}

	spec := validPolicySpec{}
	err = spec.Satisfies(record.Policy)
	if err != nil {
		return nil, fmt.Errorf("CreatePolicyCommand: %w", err)
	}

	err = engine.SetPolicy(ctx.Context(), record)
	if err != nil {
		return nil, fmt.Errorf("CreatePolicyCommand: %w", err)
	}

	return record.Policy, nil
}
