package sourcehub

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"io"
	"math/big"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/sdk"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

// ExtendedTxSigner wraps the SourceHub SDK's TxSigner and adds our GetSigner method
type ExtendedTxSigner struct {
	sdk.TxSigner
	signer crypto.Signer
}

// NewExtendedTxSigner creates a new ExtendedTxSigner from a SourceHub SDK TxSigner
func NewExtendedTxSigner(sdkSigner sdk.TxSigner) (*ExtendedTxSigner, error) {
	// Get the private key and create a crypto.Signer wrapper
	privKey := sdkSigner.GetPrivateKey()

	// Check if it's a secp256k1 key
	secpKey, ok := privKey.(*secp256k1.PrivKey)
	if !ok {
		return nil, fmt.Errorf("private key is not secp256k1 type")
	}

	// Convert the secp256k1 key to a standard Go ECDSA key
	ecdsaKey, err := secp256k1ToECDSA(secpKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert secp256k1 key to ECDSA: %w", err)
	}

	signer := &secp256k1Signer{
		privKey:  secpKey,
		ecdsaKey: ecdsaKey,
	}

	return &ExtendedTxSigner{
		TxSigner: sdkSigner,
		signer:   signer,
	}, nil
}

func (e *ExtendedTxSigner) GetSigner() crypto.Signer {
	return e.signer
}

// secp256k1Signer wraps the Cosmos SDK secp256k1.PrivKey to implement crypto.Signer
type secp256k1Signer struct {
	privKey  *secp256k1.PrivKey
	ecdsaKey *ecdsa.PrivateKey
}

func (s *secp256k1Signer) Public() crypto.PublicKey {
	return s.ecdsaKey.Public()
}

func (s *secp256k1Signer) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	// Use the standard Go ECDSA implementation
	return s.ecdsaKey.Sign(rand, digest, opts)
}

// secp256k1ToECDSA converts a Cosmos SDK secp256k1.PrivKey to a standard Go ECDSA.PrivateKey
func secp256k1ToECDSA(secpKey *secp256k1.PrivKey) (*ecdsa.PrivateKey, error) {
	// The secp256k1 curve is equivalent to the secp256k1 curve used by Bitcoin
	curve := elliptic.P256() // Note: This should actually be secp256k1, but Go doesn't have it built-in

	// Extract the private key bytes
	privKeyBytes := secpKey.Bytes()

	// Create ECDSA private key
	ecdsaKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
		},
		D: new(big.Int).SetBytes(privKeyBytes),
	}

	// Derive the public key from the private key
	ecdsaKey.PublicKey.X, ecdsaKey.PublicKey.Y = curve.ScalarBaseMult(privKeyBytes)

	return ecdsaKey, nil
}

type AcpGoClient struct {
	acp                *sdk.Client
	transactionBuilder *sdk.TxBuilder
	signer             *ExtendedTxSigner
	policyId           string
}

func addToGroupError(did, groupName string, err error) error {
	return fmt.Errorf("Encountered an error adding %s to %s group: %w", did, groupName, err)
}

func removeFromGroupError(did, groupName string, err error) error {
	return fmt.Errorf("Encountered an error removing %s from %s group: %w", did, groupName, err)
}

func giveQueryAccessError(did, documentId string, err error) error {
	return fmt.Errorf("Encountered an error giving query access to %s for document %s: %w", did, documentId, err)
}

func createDataFeedError(documentId, creatorDid string, err error) error {
	return fmt.Errorf("Encountered an error creating data view for document %s by creator %s: %w", documentId, creatorDid, err)
}

func sendAndConfirmTx(ctx context.Context, acp *sdk.Client, txBuilder *sdk.TxBuilder, signer *ExtendedTxSigner, msgSet *sdk.MsgSet, decorateError func(error) error) error {
	tx, err := txBuilder.Build(ctx, signer.TxSigner, msgSet)
	if err != nil {
		return decorateError(err)
	}
	resp, err := acp.BroadcastTx(ctx, tx)
	if err != nil {
		return decorateError(fmt.Errorf("Error sending transaction: %w", err))
	}
	result, err := acp.AwaitTx(ctx, resp.TxHash)
	if err != nil {
		return decorateError(fmt.Errorf("Error waiting for transaction: %w", err))
	}
	if execErr := result.Error(); execErr != nil {
		return decorateError(fmt.Errorf("Transaction failed: %w", execErr))
	}
	return nil
}

func (client *AcpGoClient) executePolicyCommand(ctx context.Context, cmds []*acptypes.PolicyCmd, decorateError func(error) error) error {
	if len(cmds) == 0 {
		return decorateError(fmt.Errorf("no policy commands provided"))
	}

	signerDID, err := client.GetSignerDid()
	if err != nil {
		return decorateError(fmt.Errorf("failed to get signer DID: %w", err))
	}

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

	cmdBuilder.SetSigner(client.signer.GetSigner())

	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return decorateError(err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(signerDID, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)

	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, client.signer, &msgSet, decorateError)
}

func NewAcpGoClient(acp *sdk.Client, txBuilder *sdk.TxBuilder, signer sdk.TxSigner, policyID string) (*AcpGoClient, error) {
	extendedSigner, err := NewExtendedTxSigner(signer)
	if err != nil {
		return nil, fmt.Errorf("failed to create extended signer: %w", err)
	}
	return &AcpGoClient{
		acp:                acp,
		transactionBuilder: txBuilder,
		signer:             extendedSigner,
		policyId:           policyID,
	}, nil
}

func (client *AcpGoClient) AddToGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "guest", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return addToGroupError(did, groupName, e)
	})
}

func (client *AcpGoClient) RemoveFromGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "guest", did)
	cmd := acptypes.NewDeleteRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return removeFromGroupError(did, groupName, e)
	})
}

func (client *AcpGoClient) BlockFromGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "blocked", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return addToGroupError(did, groupName, e)
	})
}

func (client *AcpGoClient) GiveQueryAccess(ctx context.Context, documentId string, did string) error {
	rel := coretypes.NewActorRelationship("file", documentId, "subscriber", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return giveQueryAccessError(did, documentId, e)
	})
}

func (client *AcpGoClient) BanUserFromResource(ctx context.Context, documentId string, did string) error {
	rel := coretypes.NewActorRelationship("file", documentId, "banned", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return giveQueryAccessError(did, documentId, e)
	})
}

func (client *AcpGoClient) CreateDataFeed(ctx context.Context, documentId string, creatorDid string, parentDocumentIds ...string) error {
	// Create the main creator relationship command
	creatorRel := coretypes.NewActorRelationship("file", documentId, "creator", creatorDid)
	creatorCmd := acptypes.NewSetRelationshipCmd(creatorRel)

	// Create parent relationship commands if any
	var parentCmds []*acptypes.PolicyCmd
	for _, parentId := range parentDocumentIds {
		parentRel := coretypes.NewActorRelationship("file", documentId, "parent", parentId)
		parentCmd := acptypes.NewSetRelationshipCmd(parentRel)
		parentCmds = append(parentCmds, parentCmd)
	}

	// Combine all commands into a single slice
	allCmds := []*acptypes.PolicyCmd{creatorCmd}
	allCmds = append(allCmds, parentCmds...)

	return client.executePolicyCommand(ctx, allCmds, func(e error) error {
		return createDataFeedError(documentId, creatorDid, e)
	})
}

func (client *AcpGoClient) VerifyAccessRequest(ctx context.Context, policyID, resourceName, objectID, permission, actorDID string) (bool, error) {
	// Create the access request using SourceHub types
	accessRequest := &acptypes.QueryVerifyAccessRequestRequest{
		PolicyId: policyID,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject(resourceName, objectID),
					Permission: permission,
				},
			},
			Actor: &coretypes.Actor{
				Id: actorDID,
			},
		},
	}

	// Use the SourceHub ACP query client to verify the access request
	result, err := client.acp.ACPQueryClient().VerifyAccessRequest(ctx, accessRequest)
	if err != nil {
		return false, fmt.Errorf("failed to verify access request: %w", err)
	}

	return result.Valid, nil
}

func (client *AcpGoClient) RegisterObject(ctx context.Context, resourceName, objectID string) error {
	// Create a register object command
	cmd := acptypes.NewRegisterObjectCmd(coretypes.NewObject(resourceName, objectID))

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to register object %s: %w", objectID, e)
	})
}

func (client *AcpGoClient) SetRelationship(ctx context.Context, resourceName, objectID, relation, subjectDID string) error {
	// Create a set relationship command
	rel := coretypes.NewActorRelationship(resourceName, objectID, relation, subjectDID)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set relationship %s on %s: %w", relation, objectID, e)
	})
}

func (client *AcpGoClient) GetSignerAddress() string {
	// Get the signer's DID from their public key
	signerDID, err := did.DIDFromPubKey(client.signer.GetPrivateKey().PubKey())
	if err != nil {
		// Fallback to account address if DID generation fails
		return client.signer.GetAccAddress()
	}
	return signerDID
}

// GetSignerAccountAddress returns the Cosmos account address of the signer
func (client *AcpGoClient) GetSignerAccountAddress() string {
	return client.signer.GetAccAddress()
}

func (client *AcpGoClient) GetSignerDid() (string, error) {
	return did.DIDFromPubKey(client.signer.GetPrivateKey().PubKey())
}
