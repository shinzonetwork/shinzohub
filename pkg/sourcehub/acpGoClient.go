package sourcehub

import (
	"context"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/sourcenetwork/acp_core/pkg/did"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/sdk"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

type AcpGoClient struct {
	acp                *sdk.Client
	transactionBuilder *sdk.TxBuilder
	signer             sdk.TxSigner
	actor              *AcpActor
	policyId           string
	chainId            string
}

func CreateAcpGoClientFromEnvironmentVariable(chainId, privateKeyEnvironmentVariable string) (*AcpGoClient, error) {
	signer, err := NewApiSignerFromEnv(privateKeyEnvironmentVariable)
	if err != nil {
		return nil, fmt.Errorf("Failed to load API signer: %v", err)
	}

	return createAcpGoClient(chainId, signer)
}

func CreateAcpGoClientFromKeychain(chainId, keyringKey string) (*AcpGoClient, error) {
	keyRing, err := getKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	signer, err := sdk.NewTxSignerFromKeyringKey(keyRing, keyringKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get signer from keyring at key %s: %w", keyringKey, err)
	}

	return createAcpGoClient(chainId, signer)
}

func (client *AcpGoClient) RegisterObject(ctx context.Context, resourceName, objectID string) error {
	isRegistered, err := client.isObjectRegistered(ctx, resourceName, objectID)
	if err != nil {
		return fmt.Errorf("failed to check if object %s is registered: %w", objectID, err)
	}

	if isRegistered {
		// Object is already registered, exit early - attempting to register it again will throw an error
		return nil
	}

	cmd := acptypes.NewRegisterObjectCmd(coretypes.NewObject(resourceName, objectID))

	return client.ExecutePolicyCommands(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to register object %s: %w", objectID, e)
	})
}

func (client *AcpGoClient) isObjectRegistered(ctx context.Context, resourceName, objectID string) (bool, error) {
	queryRequest := &acptypes.QueryObjectOwnerRequest{
		PolicyId: client.policyId,
		Object:   coretypes.NewObject(resourceName, objectID),
	}

	result, err := client.acp.ACPQueryClient().ObjectOwner(ctx, queryRequest)
	if err != nil {
		return false, fmt.Errorf("failed to query object owner: %w", err)
	}

	return result.IsRegistered, nil
}

func (client *AcpGoClient) SetActorRelationship(ctx context.Context, resourceType, resourceName, relation, actorDid string) error {
	rel := coretypes.NewActorRelationship(resourceType, resourceName, relation, actorDid)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.ExecutePolicyCommands(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set relationship %s on %s:%s: %w", relation, resourceType, resourceName, e)
	})
}

func (client *AcpGoClient) SetRelationship(ctx context.Context, resourceType, resourceName, relation, subjectResourceType, subjectResourceName string) error {
	rel := coretypes.NewRelationship(resourceType, resourceName, relation, subjectResourceType, subjectResourceName)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.ExecutePolicyCommands(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set relationship %s on %s:%s: %w", relation, resourceType, resourceName, e)
	})
}

func (client *AcpGoClient) SetActorSetRelationship(ctx context.Context, resourceType, resourceName, relation, subjectResourceType, subjectResourceName, subjectRelation string) error {
	rel := coretypes.NewActorSetRelationship(resourceType, resourceName, relation, subjectResourceType, subjectResourceName, subjectRelation)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.ExecutePolicyCommands(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set relation %s on %s:%s for  %s:%s#%s: %w", relation, resourceType, resourceName, subjectResourceType, subjectResourceName, subjectRelation, e)
	})
}

func (client *AcpGoClient) DeleteActorRelationship(ctx context.Context, resourceType, resourceName, relation, actorDid string) error {
	rel := coretypes.NewActorRelationship(resourceType, resourceName, relation, actorDid)
	cmd := acptypes.NewDeleteRelationshipCmd(rel)

	return client.ExecutePolicyCommands(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set relationship %s on %s:%s: %w", relation, resourceType, resourceName, e)
	})
}

func (client *AcpGoClient) DeleteRelationship(ctx context.Context, resourceType, resourceName, relation, subjectResourceType, subjectResourceName string) error {
	rel := coretypes.NewRelationship(resourceType, resourceName, relation, subjectResourceType, subjectResourceName)
	cmd := acptypes.NewDeleteRelationshipCmd(rel)

	return client.ExecutePolicyCommands(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set relationship %s on %s:%s: %w", relation, resourceType, resourceName, e)
	})
}

func (client *AcpGoClient) DeleteActorSetRelationship(ctx context.Context, resourceType, resourceName, relation, subjectResourceType, subjectResourceName, subjectRelation string) error {
	rel := coretypes.NewActorSetRelationship(resourceType, resourceName, relation, subjectResourceType, subjectResourceName, subjectRelation)
	cmd := acptypes.NewDeleteRelationshipCmd(rel)

	return client.ExecutePolicyCommands(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set relation %s on %s:%s for  %s:%s#%s: %w", relation, resourceType, resourceName, subjectResourceType, subjectResourceName, subjectRelation, e)
	})
}

func (client *AcpGoClient) GetActor() AcpActor {
	return *client.actor
}

func (client *AcpGoClient) SetActor(actor *AcpActor) {
	client.actor = actor
}

func (client *AcpGoClient) GetSigner() sdk.TxSigner {
	return client.signer
}

func (client *AcpGoClient) VerifyAccessRequest(ctx context.Context, resourceType, resourceName, permission, actorDid string) (bool, error) {
	accessRequest := &acptypes.QueryVerifyAccessRequestRequest{
		PolicyId: client.policyId,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject(resourceType, resourceName),
					Permission: permission,
				},
			},
			Actor: &coretypes.Actor{
				Id: actorDid,
			},
		},
	}

	result, err := client.acp.ACPQueryClient().VerifyAccessRequest(ctx, accessRequest)
	if err != nil {
		return false, fmt.Errorf("failed to verify access request: %w", err)
	}

	return result.Valid, nil
}

func (client *AcpGoClient) GetBalanceInUOpen(ctx context.Context) (*banktypes.QueryBalanceResponse, error) {
	bankClient := client.acp.BankQueryClient()
	balanceQuery := &banktypes.QueryBalanceRequest{
		Address: client.signer.GetAccAddress(),
		Denom:   "uopen",
	}

	balance, err := bankClient.Balance(ctx, balanceQuery)
	if err != nil {
		return balance, fmt.Errorf("Encountered error fetching balance: %v", err)
	}
	return balance, nil
}

func (client *AcpGoClient) FundAccount(ctx context.Context, fundingAccountAlias string, fundingAmount uint64) error {
	keyRing, err := getKeyring()

	fundingAccountSigner, err := sdk.NewTxSignerFromKeyringKey(keyRing, fundingAccountAlias)
	if err != nil {
		return fmt.Errorf("failed to get fundingAccountAlias %s signer: %w", fundingAccountAlias, err)
	}

	funderClient, err := sdk.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create SourceHub client: %w", err)
	}
	defer funderClient.Close()

	fundingAccountAddress, err := sdktypes.AccAddressFromBech32(fundingAccountSigner.GetAccAddress())
	if err != nil {
		return fmt.Errorf("failed to convert fundingAccountAlias %s address: %w", fundingAccountAlias, err)
	}

	targetAddress, err := sdktypes.AccAddressFromBech32(client.signer.GetAccAddress())
	if err != nil {
		return fmt.Errorf("failed to convert target address: %w", err)
	}

	transactionBuilder, err := sdk.NewTxBuilder(
		sdk.WithSDKClient(funderClient),
		sdk.WithChainID(client.chainId),
		sdk.WithFeeAmount(300),
		sdk.WithGasLimit(300000),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction builder: %w", err)
	}

	amount := sdktypes.NewCoins(sdktypes.NewInt64Coin("uopen", int64(fundingAmount)))
	msg := banktypes.NewMsgSend(fundingAccountAddress, targetAddress, amount)

	tx, err := transactionBuilder.BuildFromMsgs(context.Background(), fundingAccountSigner, msg)
	if err != nil {
		return fmt.Errorf("failed to build transaction: %w", err)
	}

	resp, err := funderClient.BroadcastTx(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	result, err := funderClient.AwaitTx(context.Background(), resp.TxHash)
	if err != nil {
		return fmt.Errorf("failed to await transaction: %w", err)
	}

	if result.Error() != nil {
		return fmt.Errorf("transaction failed: %w", result.Error())
	}

	return nil
}

func createAcpGoClient(chainId string, signer sdk.TxSigner) (*AcpGoClient, error) {
	acpClient, err := sdk.NewClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create ACP SDK client: %v", err)
	}
	txBuilder, err := sdk.NewTxBuilder(
		sdk.WithSDKClient(acpClient),
		sdk.WithChainID(chainId),
		sdk.WithFeeAmount(300),
		sdk.WithGasLimit(300000))
	if err != nil {
		return nil, fmt.Errorf("Failed to create TxBuilder: %v", err)
	}

	policyId := os.Getenv("POLICY_ID")
	if policyId == "" {
		return nil, fmt.Errorf("POLICY_ID environment variable is required")
	}

	acpDID, acpSigner, err := did.ProduceDID() // Todo figure out some way to fix this - it gives me a random did each time instead of one derived from our signer TxSigner
	if err != nil {
		return nil, fmt.Errorf("Failed to create ACP DID and signer: %v", err)
	}
	actor := AcpActor{
		Did:    acpDID,
		Signer: acpSigner,
	}

	acpGoClient, err := newAcpGoClient(acpClient, &txBuilder, signer, actor, policyId, chainId)
	if err != nil {
		return nil, fmt.Errorf("Failed to create ACP Go client: %v", err)
	}
	return acpGoClient, nil
}

func newAcpGoClient(acp *sdk.Client, txBuilder *sdk.TxBuilder, signer sdk.TxSigner, actor AcpActor, policyID string, chainId string) (*AcpGoClient, error) {
	return &AcpGoClient{
		acp:                acp,
		transactionBuilder: txBuilder,
		signer:             signer,
		actor:              &actor,
		policyId:           policyID,
		chainId:            chainId,
	}, nil
}

func getKeyring() (keyring.Keyring, error) {
	reg := cdctypes.NewInterfaceRegistry()
	cryptocdc.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)
	kr, err := keyring.New("sourcehub", keyring.BackendTest, os.Getenv("HOME")+"/.sourcehub", nil, cdc)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	return kr, nil
}

func (client *AcpGoClient) ExecutePolicyCommands(ctx context.Context, cmds []*acptypes.PolicyCmd, decorateError func(error) error) error {
	if len(cmds) == 0 {
		return decorateError(fmt.Errorf("no policy commands provided"))
	}

	signerDID := client.actor.Did

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return decorateError(err)
	}

	cmdBuilder.Actor(signerDID)
	cmdBuilder.PolicyID(client.policyId)

	// Add all commands
	for _, cmd := range cmds {
		cmdBuilder.PolicyCmd(cmd)
	}

	cmdBuilder.SetSigner(client.actor.Signer)

	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return decorateError(err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(client.signer.GetAccAddress(), jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)

	return client.sendAndConfirmTx(ctx, &msgSet, decorateError)
}

func (client *AcpGoClient) sendAndConfirmTx(ctx context.Context, msgSet *sdk.MsgSet, decorateError func(error) error) error {
	tx, err := client.transactionBuilder.Build(ctx, client.signer, msgSet)
	if err != nil {
		return decorateError(err)
	}
	resp, err := client.acp.BroadcastTx(ctx, tx)
	if err != nil {
		return decorateError(fmt.Errorf("Error sending transaction: %w", err))
	}
	result, err := client.acp.AwaitTx(ctx, resp.TxHash)
	if err != nil {
		return decorateError(fmt.Errorf("Error waiting for transaction: %w", err))
	}
	if execErr := result.Error(); execErr != nil {
		return decorateError(fmt.Errorf("Transaction failed: %w", execErr))
	}
	return nil
}
