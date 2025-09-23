package tests

import (
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"shinzohub/pkg/sourcehub"
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/did"
)

func fundTestClientSigner(t *testing.T, shinzoClient sourcehub.ShinzoAcpGoClient) error {
	client := shinzoClient.Acp

	balanceResp, err := client.GetBalanceInUOpen(context.Background())
	if err != nil {
		t.Logf("Warning: Could not query balance: %v\n", err)
		t.Logf("Proceeding with funding transaction...\n")
	} else {
		currentBalance := balanceResp.Balance.Amount.Int64()
		requiredBalance := int64(100000000) // 100 million uopen minimum

		t.Logf("Current balance: %d uopen\n", currentBalance)

		if currentBalance >= requiredBalance {
			t.Logf("Address already has sufficient funds (%d uopen >= %d uopen). Skipping funding.\n",
				currentBalance, requiredBalance)
			return nil
		}

		t.Logf("Address has insufficient funds (%d uopen < %d uopen). Proceeding with funding...\n",
			currentBalance, requiredBalance)
	}
	amount := 100000000

	err = client.FundAccount(context.Background(), "validator", 100000000)
	if err != nil {
		return fmt.Errorf("Encountered error funding account: %v", err)
	}

	t.Logf("Successfully funded test client signer with %d uopen\n", amount)
	return nil
}

func generateRealDidsForTestUsers(t *testing.T, testUsers map[string]*TestUser) (map[string]string, map[string]crypto.Signer) {
	realDIDs := make(map[string]string)
	signers := make(map[string]crypto.Signer)

	// Generate a DID for each test user
	for username, user := range testUsers {
		// Use the ProduceDID function which generates a random key
		// This is more appropriate for testing since usernames aren't 32 bytes
		didStr, signer, err := did.ProduceDID()
		user.DID = didStr
		if err != nil {
			t.Fatalf("Failed to generate DID for user %s: %v", username, err)
		}
		realDIDs[username] = didStr
		signers[username] = signer
	}
	return realDIDs, signers
}

func printTestUsers(t *testing.T, users map[string]*TestUser) error {
	for did, user := range users {
		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		t.Logf("%s\n%s\n\n", did, string(data))
	}
	return nil
}
