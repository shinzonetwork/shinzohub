package tests

import (
	"bytes"
	"context"
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

	// Import the SourceHub DID package
	did "github.com/sourcenetwork/acp_core/pkg/did"

	// Import SourceHub SDK for ACP client creation
	"github.com/sourcenetwork/sourcehub/sdk"
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
	realDIDs := generateRealDidsForTestUsers(t, testUsers)

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

	// Create test environment
	env := &TestEnvironment{
		Users:        testUsers,
		DefraDBURL:   "http://localhost:9181",
		RegistrarURL: "http://localhost:8081",
		SourceHubURL: "http://localhost:26657",
		ACPClient:    nil, // Will be set below if SourceHub is available
		Validator:    &validators.RegistrarValidator{},
		RealDIDs:     realDIDs,
		PolicyID:     policyID,
	}

	// Try to create a real SourceHub ACP client
	// This will connect to the local SourceHub instance
	acpClient, err := createSourceHubACPClient(env.SourceHubURL, policyID)
	if err != nil {
		t.Logf("Warning: Could not create SourceHub ACP client: %v", err)
		t.Logf("Permission checking will be limited - tests may not reflect actual ACP behavior")
		env.ACPClient = nil
	} else {
		env.ACPClient = acpClient
		t.Logf("Successfully created SourceHub ACP client")
	}

	return env
}

// createSourceHubACPClient creates a real SourceHub ACP client for testing
func createSourceHubACPClient(sourceHubURL, policyID string) (sourcehub.AcpClient, error) {
	// Create the SourceHub SDK client
	client, err := sdk.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create SourceHub client: %w", err)
	}

	// Create the transaction builder
	txBuilder, err := sdk.NewTxBuilder(sdk.WithSDKClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction builder: %w", err)
	}

	// Create the API signer from environment (same as registrar)
	signer, err := sourcehub.NewApiSignerFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create API signer: %w", err)
	}

	// Create and return the ACP client
	acpGoClient := sourcehub.NewAcpGoClient(client, &txBuilder, signer, policyID)
	return acpGoClient, nil
}

// mapKeys returns the keys of a map as a slice
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
		fmt.Printf("%s -> %s\n", did, string(data))
	}
	return nil
}

func generateRealDidsForTestUsers(t *testing.T, testUsers map[string]*TestUser) map[string]string {
	realDIDs := make(map[string]string)

	testUsernames := mapKeys(testUsers)

	// Generate a DID for each test user
	for _, username := range testUsernames {
		// Use the ProduceDID function which generates a random key
		// This is more appropriate for testing since usernames aren't 32 bytes
		didStr, _, err := did.ProduceDID()
		if err != nil {
			t.Fatalf("Failed to generate DID for user %s: %v", username, err)
		}
		realDIDs[username] = didStr
	}
	return realDIDs
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

// Todo fix
// createTestResources creates the necessary test resources:
// For now, we're using direct permission checking instead of creating actual documents
func createTestResources(env *TestEnvironment) error {
	fmt.Println("Using direct permission checking - no documents need to be created")
	fmt.Println("✓ Test resources ready for permission testing!")
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
		fmt.Printf("This means the createSourceHubACPClient function needs to be implemented\n")
		fmt.Printf("to create a real connection to the SourceHub instance\n")
		return false
	}

	fmt.Printf("Checking permission: user %s wants to %s on %s (resource: %s, type: %s)\n",
		userDID, action, resource, resourceName, resourceType)

	// Use the ACP client to verify the access request
	// For now, we'll use a test object ID since we're not creating actual documents
	testObjectID := "test-object-" + resourceName

	ctx := context.Background()
	hasPermission, err := env.ACPClient.VerifyAccessRequest(ctx, policyID, resourceName, testObjectID, action, userDID)
	if err != nil {
		fmt.Printf("Error checking permission: %v\n", err)
		return false
	}

	fmt.Printf("Permission check result: %t\n", hasPermission)
	return hasPermission
}
