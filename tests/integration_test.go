package tests

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"shinzohub/pkg/sourcehub"
	"shinzohub/pkg/validators"
)

const pathToTests = "../acp/tests.yaml"
const pathToRelationships = "../acp/relationships.yaml"

// TestUser represents a user in our test environment
type TestUser struct {
	DID              string
	Group            string
	IsBlockedIndexer bool
	IsBlockedHost    bool
	IsBanned         bool
	IsIndexer        bool
	IsHost           bool
	IsSubscriber     bool
}

// TestEnvironment holds all the components needed for testing
type TestEnvironment struct {
	Users        map[string]*TestUser
	DefraDBURL   string
	RegistrarURL string
	SourceHubURL string
	ACPClient    sourcehub.AcpClient
	Validator    validators.Validator
	// Add a map to store real DIDs for each test user
	RealDIDs map[string]string
	// Add a map to store signers for each test user
	Signers map[string]crypto.Signer
	// PolicyID is a placeholder for the actual policy ID used in permission checks
	PolicyID string
}

// TestCase represents a single access control test
type TestCase struct {
	Name           string
	UserDID        string
	Resource       string
	Action         string
	ExpectedResult bool // true = should succeed, false = should fail
}

func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Parse test files to get test users and cases
	testUsers, err := parseTestUsersFromFile(pathToRelationships)
	if err != nil {
		t.Fatalf("Failed to parse test users: %v", err)
	}

	// Generate real DIDs for each test user
	realDIDs, signers := generateRealDidsForTestUsers(t, testUsers)

	// Get the real policy ID from the .shinzohub/policy_id file (set during bootstrap)
	policyID := os.Getenv("POLICY_ID")
	if policyID == "" {
		// Try to read from the policy_id file that bootstrap.sh creates
		// Since tests run from tests/ directory, the .shinzohub is in the parent
		policyIDFile := "../.shinzohub/policy_id"
		if data, err := os.ReadFile(policyIDFile); err == nil {
			policyID = strings.TrimSpace(string(data))
			t.Logf("Read policy ID from file: %s", policyID)
		} else {
			t.Fatalf("Unable to run test suite: Could not read policy ID from %s: %v", policyIDFile, err)
		}
	} else {
		t.Logf("Using policy ID from environment: %s", policyID)
	}

	// Set the POLICY_ID environment variable so CreateAcpGoClient can use it
	os.Setenv("POLICY_ID", policyID)

	// Create test environment
	acpClient, err := sourcehub.CreateAcpGoClient("sourcehub-dev")
	if err != nil {
		t.Fatalf("Unable to create sourcehub acp client: %v", err)
	}
	env := &TestEnvironment{
		Users:        testUsers,
		DefraDBURL:   "http://localhost:9181",
		RegistrarURL: "http://localhost:8081",
		SourceHubURL: "http://localhost:26657",
		ACPClient:    acpClient, // Will be set below if SourceHub is available
		Validator:    &validators.RegistrarValidator{},
		RealDIDs:     realDIDs,
		Signers:      signers,
		PolicyID:     policyID,
	}

	// Fund the test client signer with tokens BEFORE testing functionality
	// This is necessary because the ACP client needs tokens to perform operations
	t.Logf("Funding test client signer with tokens...")
	if acpGoClient, ok := acpClient.(*sourcehub.AcpGoClient); ok {
		accountAddr := acpGoClient.GetSignerAccountAddress()
		t.Logf("Signer account address: %s", accountAddr)
		if err := fundTestClientSigner(accountAddr); err != nil {
			t.Fatalf("Failed to fund test client signer: %v", err)
		} else {
			t.Logf("✓ Successfully funded test client signer")
		}
	}

	return env
}

// TestAccessControl runs the comprehensive access control tests
func TestAccessControl(t *testing.T) {
	env := setupTestEnvironment(t)

	if err := waitForServices(env); err != nil {
		t.Fatalf("Services not ready: %v", err)
	}

	if err := setupInitialRelationships(env); err != nil {
		t.Fatalf("Failed to setup initial relationships: %v", err)
	}

	// Create blocks primitive, datafeed A, and datafeed B
	if err := createTestResources(env); err != nil {
		t.Fatalf("Failed to create test resources: %v", err)
	}

	// Run test cases
	testCases, err := generateTestCases()
	if err != nil {
		t.Fatalf("Encountered error generating test cases: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			runTestCase(t, env, tc)
		})
	}
}

func logRealDids(t *testing.T, env *TestEnvironment) {
	t.Logf("Generated DIDs for test users:")
	for username, realDID := range env.RealDIDs {
		t.Logf("  %s: %s", username, realDID)
	}
}

func printTestCases(t *testing.T, testCases []TestCase) {
	for _, tc := range testCases {
		b, err := json.Marshal(tc)
		if err != nil {
			t.Errorf("failed to marshal test case %q: %v", tc.Name, err)
			continue
		}
		t.Log(string(b))
	}
}

func waitForServices(env *TestEnvironment) error {
	if err := waitForRegistrar(env.RegistrarURL); err != nil {
		return err
	}
	if err := waitForDefraDB(env.DefraDBURL); err != nil {
		return err
	}
	return nil
}

func waitForRegistrar(url string) error {
	for i := 0; i < 30; i++ {
		resp, err := http.Get(url + "/registrar/")
		if err == nil && resp.StatusCode == 404 {
			// 404 is expected for root path
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("registrar did not become ready within 30 seconds")
}

func waitForDefraDB(url string) error {
	query := `{"query":"{ Block { __typename } }"}`
	for i := 0; i < 30; i++ {
		resp, err := http.Post(url+"/api/v0/graphql", "application/json", bytes.NewBufferString(query))
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if strings.Contains(string(body), `"Block"`) {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("DefraDB did not become ready within 30 seconds")
}

type addToGroupRequest struct {
	DID string `json:"did"`
}

func setupInitialRelationships(env *TestEnvironment) error {
	// Use the registrar API to add users to groups
	client := &http.Client{}
	for username, user := range env.Users {
		path := ""
		if user.IsIndexer {
			path = "/request-indexer-role"
		} else if user.IsHost {
			path = "/request-host-role"
		} else {
			continue
		}

		realDID := user.DID
		fmt.Printf("Setting up %s relationship for %s with DID: %s\n", user.Group, username, realDID)
		jsonBytes, err := json.Marshal(addToGroupRequest{
			DID: realDID,
		})
		if err != nil {
			return err
		}
		req, err := http.NewRequest("POST",
			env.RegistrarURL+path,
			bytes.NewBuffer(jsonBytes))
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if user.IsBlockedIndexer || user.IsBlockedHost {
			if user.IsBlockedIndexer {
				path = "/block-indexer"
			} else if user.IsBlockedHost {
				path = "/block-host"
			} else {
				return errors.New("Encountered fatal error setting up initial ACP relationships with SourceHub: unable to parse test configuration: encountered a user who is banned and is neither an indexer or host")
			}

			jsonBytes, err = json.Marshal(addToGroupRequest{
				DID: realDID,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Blocking user %s with did %s from group %s", username, realDID, user.Group)
			req, err = http.NewRequest("POST",
				env.RegistrarURL+path,
				bytes.NewBuffer(jsonBytes))
			if err != nil {
				return err
			}
			resp, err = client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
		}
	}

	return nil
}

func generateTestCases() ([]TestCase, error) {
	return parseTestCasesFromFile(pathToTests)
}

// resolveDID resolves an alias DID (e.g., "did:user:randomUser") to the real DID
func (env *TestEnvironment) resolveDID(aliasDID string) string {
	// Check if it's an alias format: "did:user:<username>"
	if len(aliasDID) > 10 && aliasDID[:10] == "did:user:" {
		username := aliasDID[10:]
		if realDID, exists := env.RealDIDs[username]; exists {
			return realDID
		}
	}
	// If it's not an alias or not found, return as-is
	return aliasDID
}

// getRealDID returns the real DID for a given username
func (env *TestEnvironment) getRealDID(username string) (string, bool) {
	realDID, exists := env.RealDIDs[username]
	return realDID, exists
}

// getUsernameFromAlias extracts the username from an alias DID
func (env *TestEnvironment) getUsernameFromAlias(aliasDID string) (string, bool) {
	if len(aliasDID) > 10 && aliasDID[:10] == "did:user:" {
		username := aliasDID[10:]
		_, exists := env.RealDIDs[username]
		return username, exists
	}
	return "", false
}

// createTestResources creates the necessary test resources:
// 1. Creates minimal test objects that represent the ACP resources
// 2. Registers these objects with the ACP system
// 3. Sets up the key relationships defined in relationships.yaml
func createTestResources(env *TestEnvironment) error {
	fmt.Println("Creating and registering test resources with ACP system...")

	// Check if we have an ACP client available
	if env.ACPClient == nil {
		return fmt.Errorf("no ACP client available - cannot create test resources")
	}

	ctx := context.Background()

	// 1. Create and register the blocks primitive resource
	// This represents the "primitive:blocks" resource from relationships.yaml
	blocksObjectID := "testblocks"
	fmt.Printf("Registering blocks primitive resource: %s\n", blocksObjectID)

	if err := env.ACPClient.RegisterObject(ctx, "primitive", blocksObjectID); err != nil {
		return fmt.Errorf("failed to register blocks primitive: %w", err)
	}

	// 2. Create and register the datafeedA view resource
	datafeedAObjectID := "datafeedA"
	fmt.Printf("Registering datafeedA view resource: %s\n", datafeedAObjectID)

	if err := env.ACPClient.RegisterObject(ctx, "view", datafeedAObjectID); err != nil {
		return fmt.Errorf("failed to register datafeedA view: %w", err)
	}

	// 3. Create and register the datafeedB view resource
	datafeedBObjectID := "datafeedB"
	fmt.Printf("Registering datafeedB view resource: %s\n", datafeedBObjectID)

	if err := env.ACPClient.RegisterObject(ctx, "view", datafeedBObjectID); err != nil {
		return fmt.Errorf("failed to register datafeedB view: %w", err)
	}

	// 4. Set up key relationships that match relationships.yaml
	fmt.Println("Setting up ACP relationships...")

	// For blocks primitive - set up key relationships
	// Note: We'll use the signer's address as the owner/admin for testing
	// In a real scenario, these would be the actual DIDs from relationships.yaml

	// Set owner relationship on blocks
	if err := env.ACPClient.SetRelationship(ctx, "primitive", blocksObjectID, "owner", env.ACPClient.(*sourcehub.AcpGoClient).GetSignerAddress()); err != nil {
		return fmt.Errorf("failed to set owner relationship on blocks: %w", err)
	}

	// Set admin relationship on blocks
	if err := env.ACPClient.SetRelationship(ctx, "primitive", blocksObjectID, "admin", env.ACPClient.(*sourcehub.AcpGoClient).GetSignerAddress()); err != nil {
		return fmt.Errorf("failed to set admin relationship on blocks: %w", err)
	}

	// For datafeedA - set up key relationships
	// Set owner relationship on datafeedA
	if err := env.ACPClient.SetRelationship(ctx, "view", datafeedAObjectID, "owner", env.ACPClient.(*sourcehub.AcpGoClient).GetSignerAddress()); err != nil {
		return fmt.Errorf("failed to set owner relationship on datafeedA: %w", err)
	}

	// Set parent relationship (datafeedA -> blocks)
	if err := env.ACPClient.SetRelationship(ctx, "view", datafeedAObjectID, "parent", blocksObjectID); err != nil {
		return fmt.Errorf("failed to set parent relationship on datafeedA: %w", err)
	}

	// For datafeedB - set up key relationships
	// Set owner relationship on datafeedB
	if err := env.ACPClient.SetRelationship(ctx, "view", datafeedBObjectID, "owner", env.ACPClient.(*sourcehub.AcpGoClient).GetSignerAddress()); err != nil {
		return fmt.Errorf("failed to set owner relationship on datafeedB: %w", err)
	}

	// Set parent relationship (datafeedB -> datafeedA)
	if err := env.ACPClient.SetRelationship(ctx, "view", datafeedBObjectID, "parent", datafeedAObjectID); err != nil {
		return fmt.Errorf("failed to set parent relationship on datafeedB: %w", err)
	}

	// Todo add subscribers and banned relationships

	fmt.Println("✓ Test resources created and registered with ACP successfully!")
	return nil
}

func runTestCase(t *testing.T, env *TestEnvironment, tc TestCase) {
	// Resolve the alias DID to the real DID
	realUserDID := env.resolveDID(tc.UserDID)
	t.Logf("Resolved %s to %s", tc.UserDID, realUserDID)

	// Attempt the action with the real DID
	result := attemptAction(env, realUserDID, tc.Resource, tc.Action)

	// Verify the result
	if result != tc.ExpectedResult {
		t.Errorf("Expected %v for %s %s on %s, got %v",
			tc.ExpectedResult, tc.Action, tc.Resource, tc.UserDID, result)
	}
}

func attemptAction(env *TestEnvironment, userDID, resource, action string) bool {
	// Use the SourceHub SDK to check if the user has permission to perform the action on the resource
	// This is the proper way to test ACP enforcement by actually querying the ACP system

	// Parse the resource to extract the resource type and ID
	// Format: "primitive:blocks", "view:datafeedA", etc.
	parts := strings.Split(resource, ":")
	if len(parts) != 2 {
		fmt.Printf("Invalid resource format: %s, expected format like 'primitive:blocks'\n", resource)
		return false
	}

	resourceType := parts[0] // "primitive" or "view"
	resourceName := parts[1] // "blocks", "datafeedA", etc.

	// Get the policy ID from environment
	policyID := env.PolicyID
	if policyID == "" {
		fmt.Printf("No policy ID available for permission checking\n")
		return false
	}

	// Check if we have an ACP client available
	if env.ACPClient == nil {
		fmt.Printf("No ACP client available for permission checking - SourceHub ACP client not created\n")
		return false
	}

	fmt.Printf("Checking permission: user %s wants to %s on %s (resource: %s, type: %s)\n",
		userDID, action, resource, resourceName, resourceType)

	// Use the ACP client to verify the access request
	// For now, we'll use a test object ID since we're not creating actual documents
	testObjectID := "testobject" + resourceName

	ctx := context.Background()
	hasPermission, err := env.ACPClient.VerifyAccessRequest(ctx, policyID, resourceName, testObjectID, action, userDID)
	if err != nil {
		fmt.Printf("Error checking permission: %v\n", err)
		return false
	}

	fmt.Printf("Permission check result: %t\n", hasPermission)
	return hasPermission
}
