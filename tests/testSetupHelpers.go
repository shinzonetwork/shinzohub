package tests

import (
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/did"
	"github.com/sourcenetwork/sourcehub/sdk"

	// Import Cosmos SDK for bank transactions
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// fundTestClientSigner sends tokens from the validator account to the test client signer
// This function can be called during test setup to ensure the test client has tokens
func fundTestClientSigner(targetAddress string) error {
	fmt.Printf("Checking balance for address: %s\n", targetAddress)

	// Create a keyring to access the validator account
	reg := cdctypes.NewInterfaceRegistry()
	cryptocdc.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)

	// Use the test keyring backend and the .sourcehub directory
	kr, err := keyring.New("sourcehub", keyring.BackendTest, os.Getenv("HOME")+"/.sourcehub", nil, cdc)
	if err != nil {
		return fmt.Errorf("failed to create keyring: %w", err)
	}

	// Get the validator signer
	validatorSigner, err := sdk.NewTxSignerFromKeyringKey(kr, "validator")
	if err != nil {
		return fmt.Errorf("failed to get validator signer: %w", err)
	}

	// Create SourceHub SDK client using the default ports
	// Note: These should match the ports used by the local SourceHub instance
	client, err := sdk.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create SourceHub client: %w", err)
	}
	defer client.Close()

	// Convert string addresses to AccAddress
	validatorAddr, err := sdktypes.AccAddressFromBech32(validatorSigner.GetAccAddress())
	if err != nil {
		return fmt.Errorf("failed to convert validator address: %w", err)
	}

	targetAddr, err := sdktypes.AccAddressFromBech32(targetAddress)
	if err != nil {
		return fmt.Errorf("failed to convert target address: %w", err)
	}

	// First, check if the target address already has sufficient funds
	bankClient := client.BankQueryClient()
	balanceQuery := &banktypes.QueryBalanceRequest{
		Address: targetAddress,
		Denom:   "uopen",
	}

	balanceResp, err := bankClient.Balance(context.Background(), balanceQuery)
	if err != nil {
		fmt.Printf("Warning: Could not query balance: %v\n", err)
		fmt.Printf("Proceeding with funding transaction...\n")
	} else {
		currentBalance := balanceResp.Balance.Amount.Int64()
		requiredBalance := int64(100000000) // 100 million uopen minimum

		fmt.Printf("Current balance: %d uopen\n", currentBalance)

		if currentBalance >= requiredBalance {
			fmt.Printf("Address already has sufficient funds (%d uopen >= %d uopen). Skipping funding.\n",
				currentBalance, requiredBalance)
			return nil
		}

		fmt.Printf("Address has insufficient funds (%d uopen < %d uopen). Proceeding with funding...\n",
			currentBalance, requiredBalance)
	}

	// Create transaction builder
	txBuilder, err := sdk.NewTxBuilder(
		sdk.WithSDKClient(client),
		sdk.WithChainID("sourcehub-dev"),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction builder: %w", err)
	}

	// Create a bank send message
	amount := sdktypes.NewCoins(sdktypes.NewInt64Coin("uopen", 1000000000)) // 1 billion uopen
	msg := banktypes.NewMsgSend(validatorAddr, targetAddr, amount)

	// Build and send the transaction using the SourceHub SDK
	// We'll use the existing transaction builder directly with the bank message
	tx, err := txBuilder.BuildFromMsgs(context.Background(), validatorSigner, msg)
	if err != nil {
		return fmt.Errorf("failed to build transaction: %w", err)
	}

	resp, err := client.BroadcastTx(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Wait for the transaction to be processed
	result, err := client.AwaitTx(context.Background(), resp.TxHash)
	if err != nil {
		return fmt.Errorf("failed to await transaction: %w", err)
	}

	if result.Error() != nil {
		return fmt.Errorf("transaction failed: %w", result.Error())
	}

	fmt.Printf("Successfully funded test client signer with %s\n", amount.String())
	return nil
}

func generateRealDidsForTestUsers(t *testing.T, testUsers map[string]*TestUser) (map[string]string, map[string]crypto.Signer) {
	realDIDs := make(map[string]string)
	signers := make(map[string]crypto.Signer)

	testUsernames := mapKeys(testUsers)

	// Generate a DID for each test user
	for _, username := range testUsernames {
		// Use the ProduceDID function which generates a random key
		// This is more appropriate for testing since usernames aren't 32 bytes
		didStr, signer, err := did.ProduceDID()
		if err != nil {
			t.Fatalf("Failed to generate DID for user %s: %v", username, err)
		}
		realDIDs[username] = didStr
		signers[username] = signer
	}
	return realDIDs, signers
}

func mapKeys(m map[string]*TestUser) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func printTestUsers(users map[string]*TestUser) error {
	for did, user := range users {
		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n%s\n\n", did, string(data))
	}
	return nil
}
