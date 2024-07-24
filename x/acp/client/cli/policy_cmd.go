package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/spf13/cobra"
)

type dispatcher = func(*cobra.Command, string, *types.PolicyCmd) error

func CmdSetRelationship(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-relationship policy-id resource objectId relation [subject resource] subjectId [subjRel]",
		Short: "Issue a SetRelationship PolicyCmd",
		Long:  ``,
		Args:  cobra.MinimumNArgs(5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId, polCmd, err := parseSetRelationshipArgs(args)
			if err != nil {
				return err
			}

			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdDeleteRelationship(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-relationship policy-id resource objectId relation [subject resource] subjectId [subjRel]",
		Short: "Issues a DeleteRelationship PolicyCmd",
		Long:  ``,
		Args:  cobra.MinimumNArgs(5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId, polCmd, err := parseDeleteRelationshipArgs(args)
			if err != nil {
				return err
			}

			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRegisterObject(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-object policy-id resource objectId",
		Short: "Issue RegisterObject PolicyCmd",
		Long:  ``,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId, polCmd, err := parseRegisterObjectArgs(args)
			if err != nil {
				return err
			}

			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdUnregisterObject(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unregister-object policy-id resource objectId",
		Short: "Issue UnregisterObject PolicyCmd",
		Long:  ``,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId, polCmd, err := parseUnregisterObjectArgs(args)
			if err != nil {
				return err
			}
			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// parseSetRelationshipArgs parses a SetRelationship from cli args
// format: policy-id resource objectId relation [subject resource] subjectId [subjRel]
func parseSetRelationshipArgs(args []string) (string, *types.PolicyCmd, error) {
	polId := args[0]
	resource := args[1]
	objId := args[2]
	relation := args[3]
	var relationship *coretypes.Relationship
	if len(args) == 5 {
		subjectId := args[4]
		relationship = coretypes.NewActorRelationship(resource, objId, relation, subjectId)
	} else if len(args) == 6 {
		subjResource := args[4]
		subjectId := args[5]
		relationship = coretypes.NewRelationship(resource, objId, relation, subjResource, subjectId)
	} else if len(args) == 7 {
		subjResource := args[4]
		subjectId := args[5]
		subjRel := args[6]
		relationship = coretypes.NewActorSetRelationship(resource, objId, relation, subjResource, subjectId, subjRel)
	} else {
		return "", nil, fmt.Errorf("invalid number of arguments: Set Relationship expects [5,7] cli args")
	}
	cmd := types.NewSetRelationshipCmd(relationship)
	return polId, cmd, nil
}

// parseDeleteRelationshipArgs parses a DeleteRelationship from cli args
// format: policy-id resource objectId relation [subject resource] subjectId [subjRel]
func parseDeleteRelationshipArgs(args []string) (string, *types.PolicyCmd, error) {
	polId := args[0]
	resource := args[1]
	objId := args[2]
	relation := args[3]
	var relationship *coretypes.Relationship
	if len(args) == 5 {
		subjectId := args[4]
		relationship = coretypes.NewActorRelationship(resource, objId, relation, subjectId)
	} else if len(args) == 6 {
		subjResource := args[4]
		subjectId := args[5]
		relationship = coretypes.NewRelationship(resource, objId, relation, subjResource, subjectId)
	} else if len(args) == 7 {
		subjResource := args[4]
		subjectId := args[5]
		subjRel := args[6]
		relationship = coretypes.NewActorSetRelationship(resource, objId, relation, subjResource, subjectId, subjRel)
	} else {
		return "", nil, fmt.Errorf("invalid number of arguments: Delete Relationship expects [5,7] cli args")
	}
	cmd := types.NewDeleteRelationshipCmd(relationship)
	return polId, cmd, nil
}

// parseRegisterObjectArgs parses a RegisterObject from cli args
// format: policy-id resource objectId
func parseRegisterObjectArgs(args []string) (string, *types.PolicyCmd, error) {
	if len(args) != 3 {
		return "", nil, fmt.Errorf("RegisterObject: invalid number of arguments: policy-id resource objectId")
	}
	polId := args[0]
	resource := args[1]
	objId := args[2]
	return polId, types.NewRegisterObjectCmd(coretypes.NewObject(resource, objId)), nil
}

func parseUnregisterObjectArgs(args []string) (string, *types.PolicyCmd, error) {
	if len(args) != 3 {
		return "", nil, fmt.Errorf("UnregisterObject: invalid number of arguments: policy-id resource objectId")
	}
	polId := args[0]
	resource := args[1]
	objId := args[2]
	return polId, types.NewUnregisterObjectCmd(coretypes.NewObject(resource, objId)), nil
}
