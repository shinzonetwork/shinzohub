package tests

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"shinzohub/pkg/sourcehub"
	"shinzohub/pkg/validators"

	"github.com/joho/godotenv"
	"github.com/sourcenetwork/acp_core/pkg/did"
)

const pathToTests = "../acp/tests.yaml"
const pathToRelationships = "../acp/relationships.yaml"

type MembershipLevel int

const (
	None MembershipLevel = iota
	Guest
	Admin
	Owner
)

var groupMembershipLevels map[MembershipLevel]string = map[MembershipLevel]string{
	None:  "none",
	Guest: "guest",
	Admin: "admin",
	Owner: "owner",
}

// TestUser represents a user in our test environment
type TestUser struct {
	DID                  string
	Group                string
	IsBlockedIndexer     bool
	IsBlockedHost        bool
	IsBanned             bool
	IsSubscriber         bool
	IndexerMembership    MembershipLevel
	HostMembership       MembershipLevel
	ShinzoteamMembership MembershipLevel
}

// TestEnvironment holds all the components needed for testing
type TestEnvironment struct {
	Users              map[string]*TestUser
	DefraDBURL         string
	RegistrarURL       string
	SourceHubURL       string
	ShinzohubACPClient sourcehub.ShinzoAcpClient
	ValidatorACPClient sourcehub.ShinzoAcpClient
	Validator          validators.Validator
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

func init() {
	fmt.Println("Loading .env file...")
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Printf("Error loading .env: %v\n", err)
	} else {
		fmt.Println("Successfully loaded .env file")
	}
}

func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Parse test files to get test users and cases
	testUsers, err := parseTestUsersFromFile(pathToRelationships)
	if err != nil {
		t.Fatalf("Failed to parse test users: %v", err)
	}

	// Generate real DIDs for each test user
	realDIDs, signers := generateRealDidsForTestUsers(t, testUsers)

	t.Log("@@@ Printing all parsed test users:\n")
	printTestUsers(testUsers)
	t.Log("@@@ Done printing all parsed test users\n")

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
	acpClient, err := sourcehub.CreateShinzoAcpGoClient("sourcehub-dev")
	if err != nil {
		t.Fatalf("Unable to create sourcehub acp client: %v", err)
	}
	validatorAcpClient, err := sourcehub.CreateShinzoAcpGoClientWithValidatorSender("sourcehub-dev")
	if err != nil {
		t.Fatalf("Unable to create sourcehub acp client: %v", err)
	}
	t.Logf("ShinzoHub client address: %s | %s\n", acpClient.GetSignerAddress(), acpClient.GetSignerAccountAddress())
	t.Logf("Validator client address: %s | %s\n", validatorAcpClient.GetSignerAddress(), validatorAcpClient.GetSignerAccountAddress())

	env := &TestEnvironment{
		Users:              testUsers,
		DefraDBURL:         "http://localhost:9181",
		RegistrarURL:       "http://localhost:8081/registrar",
		SourceHubURL:       "http://localhost:26657",
		ShinzohubACPClient: acpClient,
		ValidatorACPClient: validatorAcpClient,
		Validator:          &validators.RegistrarValidator{},
		RealDIDs:           realDIDs,
		Signers:            signers,
		PolicyID:           policyID,
	}

	return env
}

// TestAccessControl runs the comprehensive access control tests
func TestAccessControl(t *testing.T) {
	env := setupTestEnvironment(t)

	if err := waitForServices(env); err != nil {
		t.Fatalf("Services not ready: %v", err)
	}

	// Fund the test client signer with tokens BEFORE testing functionality
	// This is necessary because the ACP client needs tokens to perform operations
	t.Logf("Funding test client signer with tokens...")
	if acpGoClient, ok := env.ShinzohubACPClient.(*sourcehub.ShinzoAcpGoClient); ok {
		accountAddr := acpGoClient.GetSignerAccountAddress()
		t.Logf("Signer account address: %s", accountAddr)
		if err := fundTestClientSigner(accountAddr); err != nil {
			t.Fatalf("Failed to fund test client signer: %v", err)
		} else {
			t.Logf("✓ Successfully funded test client signer")
		}
	}

	// Create blocks primitive, datafeed A, and datafeed B + groups
	if err := createTestResources(env); err != nil {
		t.Fatalf("Failed to create test resources: %v", err)
	}

	if err := makeShinzohubAdminOfEverything(env, env.ShinzohubACPClient.GetSignerAddress()); err != nil {
		t.Fatalf("Failed to make shinzohub admin of everything: %v", err)
	}

	if err := setupInitialGroupRelationships(env); err != nil {
		t.Fatalf("Failed to setup initial group relationships: %v", err)
	}

	if err := setupInitialCollectionRelationships(env); err != nil { // Todo implement - this function should be giving group members or dids read/write/sync/ban/creator access to collections using the shinzohub client - note that creator may not work with current policy (I think there is no manages relation for it right now; that may need to be done by sourcehub)
		t.Fatalf("Failed to setup initial collection relationships: %v", err)
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
		resp, err := http.Get(url + "/")
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

func setupInitialGroupRelationships(env *TestEnvironment) error {
	client := &http.Client{}
	for username, user := range env.Users {
		err := setupGroupGuestRelationships(client, env, username, user)
		if err != nil {
			return fmt.Errorf("Encountered issue setting up group guest relations for %s: %w", username, err)
		}

		err = setupGroupAdminRelationships(env, username, user)
		if err != nil {
			return fmt.Errorf("Encountered issue setting up group admin relations for %s: %w", username, err)
		}

		err = blockFromGroupsAsAppropriate(client, env, username, user)
		if err != nil {
			return fmt.Errorf("Encountered an issue blocking %s from groups: %w", username, err)
		}
	}

	return nil
}

func setupGroupGuestRelationships(client *http.Client, env *TestEnvironment, username string, user *TestUser) error {
	if user.IndexerMembership == Guest {
		err := setGuestRelation(client, env, username, user, "indexer")
		if err != nil {
			return fmt.Errorf("Encountered error adding %s to indexer group as guest: %w", username, err)
		}
	}

	if user.HostMembership == Guest {
		err := setGuestRelation(client, env, username, user, "host")
		if err != nil {
			return fmt.Errorf("Encountered error adding %s to host group as guest: %w", username, err)
		}
	}

	if user.ShinzoteamMembership == Guest {
		client, ok := env.ShinzohubACPClient.(*sourcehub.ShinzoAcpGoClient)
		if !ok {
			return fmt.Errorf("Encountered error adding %s to shinzoteam group as guest: no AcpGoClient in test environment", username)
		}
		err := client.AddToGroup(context.Background(), "shinzoteam", user.DID)
		if err != nil {
			return fmt.Errorf("Encountered error adding %s to shinzoteam group as guest: %w", username, err)
		}
	}
	return nil
}

func setGuestRelation(client *http.Client, env *TestEnvironment, username string, user *TestUser, group string) error {
	realDID := user.DID
	fmt.Printf("Setting up %s guest relationship for %s with DID: %s\n",
		group, username, realDID)

	jsonBytes, err := json.Marshal(addToGroupRequest{
		DID: realDID,
	})
	if err != nil {
		return err
	}

	requestURL := env.RegistrarURL + fmt.Sprintf("/request-%s-role", group)

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func setupGroupAdminRelationships(env *TestEnvironment, username string, user *TestUser) error {
	if user.HostMembership == Admin {
		err := setGroupAdminRelationship(env, username, user, "host")
		if err != nil {
			return err
		}
	}

	if user.IndexerMembership == Admin {
		err := setGroupAdminRelationship(env, username, user, "indexer")
		if err != nil {
			return err
		}
	}

	if user.ShinzoteamMembership == Admin {
		err := setGroupAdminRelationship(env, username, user, "shinzoteam")
		if err != nil {
			return err
		}
	}

	return nil
}

func setGroupAdminRelationship(env *TestEnvironment, username string, user *TestUser, group string) error {
	client, ok := env.ValidatorACPClient.(*sourcehub.ShinzoAcpGoClient)
	if !ok {
		return fmt.Errorf("Encountered error adding %s to %s group as admin: no AcpGoClient in test environment", username, group)
	}
	err := client.MakeGroupAdmin(context.Background(), group, user.DID)
	if err != nil {
		return fmt.Errorf("Encountered error adding %s to %s group as admin: %w", username, group, err)
	}
	return nil
}

func blockFromGroupsAsAppropriate(client *http.Client, env *TestEnvironment, username string, user *TestUser) error {
	if user.IsBlockedIndexer {
		err := blockFromGroup(client, env, username, user, "indexer")
		if err != nil {
			return fmt.Errorf("Encountered error blocking %s from indexer group: %w", username, err)
		}
	}
	if user.IsBlockedHost {
		err := blockFromGroup(client, env, username, user, "host")
		if err != nil {
			return fmt.Errorf("Encountered error blocking %s from host group: %w", username, err)
		}
	}
	return nil
}

func blockFromGroup(client *http.Client, env *TestEnvironment, username string, user *TestUser, group string) error {
	realDID := user.DID
	fmt.Printf("Blocking %s user %s with did %s\n", group, username, realDID)
	jsonBytes, err := json.Marshal(addToGroupRequest{
		DID: realDID,
	})
	if err != nil {
		return err
	}

	requestURL := env.RegistrarURL + fmt.Sprintf("/block-%s", group)

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func generateTestCases() ([]TestCase, error) {
	return parseTestCasesFromFile(pathToTests)
}

// resolveDID resolves an alias DID (e.g., "did:user:randomUser") to the real DID
func (env *TestEnvironment) resolveDID(aliasDID string) string {
	if realDID, exists := env.RealDIDs[aliasDID]; exists {
		return realDID
	}
	return aliasDID
}

func (env *TestEnvironment) getSigner(aliasDID string) crypto.Signer {
	if signer, exists := env.Signers[aliasDID]; exists {
		return signer
	}
	return nil
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

type resource struct {
	resourceName string
	objectName   string
}

func Resource(resourceName, objectName string) resource {
	return resource{
		resourceName: resourceName,
		objectName:   objectName,
	}
}

var testResources = []resource{
	Resource("primitive", "blocks"),
	Resource("view", "datafeedA"),
	Resource("view", "datafeedB"),
	Resource("group", "shinzoteam"),
	Resource("group", "host"),
	Resource("group", "indexer"),
}

// createTestResources creates the necessary test resources:
// 1. Creates minimal test objects that represent the ACP resources
// 2. Registers these objects with the ACP system
// 3. Sets up the ownership relationships defined in relationships.yaml
func createTestResources(env *TestEnvironment) error {
	fmt.Println("Creating and registering test resources with ACP system...")

	ACPClient := env.ValidatorACPClient

	fmt.Printf("Validator did %s\n", ACPClient.GetSignerAddress())

	ctx := context.Background()

	for _, testResource := range testResources {
		fmt.Printf("Registering %s object on %s resource\n", testResource.objectName, testResource.resourceName)
		if err := ACPClient.RegisterObject(ctx, testResource.resourceName, testResource.objectName); err != nil {
			return fmt.Errorf("Failed to register %s object on %s resource: %w", testResource.objectName, testResource.resourceName, err)
		}
	}

	fmt.Println("✓ Test resources created and registered with ACP successfully!")
	return nil
}

func makeShinzohubAdminOfEverything(env *TestEnvironment, shinzoHubDid string) error {
	fmt.Println("Making Shinzohub admin of all test resources...")

	ACPClient := env.ValidatorACPClient

	fmt.Printf("Validator did %s\n", ACPClient.GetSignerAddress())
	fmt.Printf("Shinzohub did %s\n", shinzoHubDid)

	ctx := context.Background()

	for _, testResource := range testResources {
		fmt.Printf("Making shinzohub admin of %s:%s\n", testResource.resourceName, testResource.objectName)
		if err := ACPClient.SetRelationship(ctx, testResource.resourceName, testResource.objectName, "admin", shinzoHubDid); err != nil {
			return fmt.Errorf("Failed to make shinzohub admin of %s:%s: %w", testResource.resourceName, testResource.objectName, err)
		}
	}

	fmt.Println("✓ Shinzohub is now admin of all test resources!")
	return nil
}

func setupInitialCollectionRelationships(env *TestEnvironment) error {
	parsedRelations, err := parseAcpRelationsFromFile(pathToRelationships)
	if err != nil {
		return fmt.Errorf("Encountered error parsing relations file: %v", err)
	}

	for _, parsedRelation := range parsedRelations {
		if strings.Contains(parsedRelation.SourceLine, "sourcehub") || strings.Contains(parsedRelation.SourceLine, "shinzohub") || parsedRelation.Relation.Relation == "owner" {
			continue // Relations set previously
		}

		err := setRelationship(env, parsedRelation.Relation)
		if err != nil {
			return fmt.Errorf("Encountered error setting relation defined by %s : %v", parsedRelation.SourceLine, err)
		}
	}

	return nil
}

func setRelationship(env *TestEnvironment, relation AcpRelations) error {
	client := env.ShinzohubACPClient
	if relation.Relation == "admin" {
		client = env.ValidatorACPClient // Validator would be setting these relationships during deployment
	}

	if relation.IsParentRelation {
		fmt.Printf("Setting parent relation: %s:%s -> %s:%s\n", relation.ResourceName, relation.ObjectName, relation.ParentResourceType, relation.ParentResourceName)
		return client.SetParentRelationship(context.Background(), relation.ResourceName, relation.ObjectName, relation.ParentResourceType, relation.ParentResourceName)
	}

	if relation.IsDidActor {
		fmt.Printf("Giving %s -> %s %s relation on %s:%s\n", relation.Did, env.resolveDID(relation.Did), relation.Relation, relation.ResourceName, relation.ObjectName)
		return client.SetRelationship(context.Background(), relation.ResourceName, relation.ObjectName, relation.Relation, env.resolveDID(relation.Did))
	}

	fmt.Printf("Giving group:%s#%s %s relation on %s:%s\n", relation.GroupName, relation.GroupRelation, relation.Relation, relation.ResourceName, relation.ObjectName)
	return client.SetGroupRelationship(context.Background(), relation.ResourceName, relation.ObjectName, relation.Relation, relation.GroupName, relation.GroupRelation)
}

func runTestCase(t *testing.T, env *TestEnvironment, tc TestCase) {
	// Resolve the alias DID to the real DID
	realUserDID := env.resolveDID(tc.UserDID)

	// Attempt the action with the real DID
	var result bool
	if strings.HasPrefix(tc.Action, "_can_manage_") {
		userSigner := env.getSigner(tc.UserDID)
		if userSigner == nil {
			t.Errorf("No crypto.Signer found for %s", tc.UserDID)
		}
		result = attemptManage(env, realUserDID, tc.Resource, tc.Action, userSigner)
	} else {
		result = attemptAction(env, realUserDID, tc.Resource, tc.Action)
	}

	// Verify the result
	if result != tc.ExpectedResult {
		t.Errorf("Expected %v for %s %s on %s, got %v",
			tc.ExpectedResult, tc.Action, tc.Resource, tc.UserDID, result)
	}
}

func attemptAction(env *TestEnvironment, userDID, resource, action string) bool {
	resourceType, resourceName, err := parseResource(resource)
	if err != nil {
		fmt.Printf("Encountered error attempting action %s by %s on %s: %v", action, userDID, resource, err)
		return false
	}

	// Get the policy ID from environment
	policyID := env.PolicyID
	if policyID == "" {
		fmt.Printf("No policy ID available for permission checking\n")
		return false
	}

	// Check if we have an ACP client available
	if env.ShinzohubACPClient == nil {
		fmt.Printf("No ACP client available for permission checking - SourceHub ACP client not created\n")
		return false
	}

	fmt.Printf("Checking permission: user %s wants to %s on %s (resource: %s, type: %s)\n",
		userDID, action, resource, resourceName, resourceType)

	ctx := context.Background()
	hasPermission, err := env.ShinzohubACPClient.VerifyAccessRequest(ctx, policyID, resourceType, resourceName, action, userDID)
	if err != nil {
		fmt.Printf("Error checking permission: %v\n", err)
		return false
	}

	fmt.Printf("Permission check result: %t\n", hasPermission)
	return hasPermission
}

func parseResource(resource string) (resourceType, resourceName string, err error) {
	parts := strings.Split(resource, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid resource format: %s, expected format like 'primitive:blocks'\n", resource)
	}

	resourceType = parts[0] // "primitive" or "view"
	resourceName = parts[1] // "blocks", "datafeedA", etc.

	return resourceType, resourceName, nil
}

func attemptManage(env *TestEnvironment, userDID, resource, action string, userSigner crypto.Signer) bool {
	resourceType, resourceName, err := parseResource(resource)
	if err != nil {
		fmt.Printf("Encountered error attempting action %s by %s on %s: %v", action, userDID, resource, err)
		return false
	}

	// Get the policy ID from environment
	policyID := env.PolicyID
	if policyID == "" {
		fmt.Printf("No policy ID available for permission checking\n")
		return false
	}

	// Check if we have an ACP client available
	if env.ValidatorACPClient == nil {
		fmt.Printf("No ACP client available for permission checking - SourceHub ACP client not created\n")
		return false
	}

	fmt.Printf("Checking permission: user %s wants to %s on %s (resource: %s, type: %s)\n",
		userDID, action, resource, resourceName, resourceType)

	action = strings.TrimPrefix(action, "_can_manage_")

	validatorDid := env.ValidatorACPClient.GetSignerAddress()
	validatorSigner := env.ValidatorACPClient.GetSigner()
	env.ValidatorACPClient.SetActor(userDID, userSigner) // Temporarily set the actor to be the user so we see if they have permissions to manage the given relation
	defer env.ValidatorACPClient.SetActor(validatorDid, validatorSigner)
	randomDid, _, err := did.ProduceDID()
	if err != nil {
		fmt.Printf("Encountered error attempting action %s by %s on %s: Unable to generate random did: %v", action, userDID, resource, err)
		return false
	}
	err = env.ValidatorACPClient.SetRelationship(context.Background(), resourceType, resourceName, action, randomDid)
	if err != nil {
		return false
	}
	return true
}
