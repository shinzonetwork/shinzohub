package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ACPTestHelper provides utilities for testing access control
type ACPTestHelper struct {
	RegistrarURL string
	DefraDBURL   string
	HTTPClient   *http.Client
}

// NewACPTestHelper creates a new test helper
func NewACPTestHelper(registrarURL, defraDBURL string) *ACPTestHelper {
	return &ACPTestHelper{
		RegistrarURL: registrarURL,
		DefraDBURL:   defraDBURL,
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// RegistrarRequest represents a request to the registrar API
type RegistrarRequest struct {
	DID        string `json:"did"`
	DataFeedID string `json:"dataFeedId,omitempty"`
}

// RegistrarResponse represents a response from the registrar API
type RegistrarResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// AddUserToIndexerGroup adds a user to the indexer group via registrar
func (h *ACPTestHelper) AddUserToIndexerGroup(did string) error {
	req := RegistrarRequest{DID: did}
	return h.makeRegistrarRequest("/request-indexer-role", req)
}

// AddUserToHostGroup adds a user to the host group via registrar
func (h *ACPTestHelper) AddUserToHostGroup(did string) error {
	req := RegistrarRequest{DID: did}
	return h.makeRegistrarRequest("/request-host-role", req)
}

// SubscribeToDataFeed gives a user query access to a data feed
func (h *ACPTestHelper) SubscribeToDataFeed(did, dataFeedID string) error {
	req := RegistrarRequest{DID: did, DataFeedID: dataFeedID}
	return h.makeRegistrarRequest("/subscribe-to-data-feed", req)
}

// makeRegistrarRequest makes a request to the registrar API
func (h *ACPTestHelper) makeRegistrarRequest(endpoint string, req RegistrarRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := h.HTTPClient.Post(
		h.RegistrarURL+endpoint,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var response RegistrarResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("registrar request failed: %s", response.Error)
	}

	return nil
}

// DefraDBQuery represents a GraphQL query to DefraDB
type DefraDBQuery struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// DefraDBResponse represents a response from DefraDB
type DefraDBResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// QueryDefraDB makes a GraphQL query to DefraDB with user context
func (h *ACPTestHelper) QueryDefraDB(query DefraDBQuery, userDID string) (*DefraDBResponse, error) {
	jsonData, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequest("POST", h.DefraDBURL+"/graphql", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Add user DID to headers for ACP context
	req.Header.Set("X-User-DID", userDID)

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var response DefraDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// TestAccessControl tests if a user can perform an action on a resource
func (h *ACPTestHelper) TestAccessControl(userDID, resource, action string) (bool, error) {
	// Create a test query based on the resource and action
	query := h.createTestQuery(resource, action)

	response, err := h.QueryDefraDB(query, userDID)
	if err != nil {
		return false, fmt.Errorf("failed to query DefraDB: %w", err)
	}

	// Check if there are any errors (indicating access denied)
	if len(response.Errors) > 0 {
		// Check if the error is an access control error
		for _, err := range response.Errors {
			if h.isAccessControlError(err.Message) {
				return false, nil // Access denied
			}
		}
		return false, fmt.Errorf("unexpected errors: %v", response.Errors)
	}

	return true, nil // Access granted
}

// createTestQuery creates a GraphQL query for testing access control
func (h *ACPTestHelper) createTestQuery(resource, action string) DefraDBQuery {
	switch resource {
	case "primitive:blocks":
		switch action {
		case "read":
			return DefraDBQuery{
				Query: `query { Block { hash number timestamp } }`,
			}
		case "update":
			return DefraDBQuery{
				Query: `mutation { update_Block(where: {hash: {_eq: "test"}}, data: {timestamp: "test"}) { hash } }`,
			}
		case "query":
			return DefraDBQuery{
				Query: `query { Block(where: {number: {_gt: 0}}) { hash number } }`,
			}
		}
	case "view:datafeedA":
		switch action {
		case "read":
			return DefraDBQuery{
				Query: `query { DataFeedA { id name } }`,
			}
		case "update":
			return DefraDBQuery{
				Query: `mutation { update_DataFeedA(where: {id: {_eq: "test"}}, data: {name: "test"}) { id } }`,
			}
		case "query":
			return DefraDBQuery{
				Query: `query { DataFeedA(where: {id: {_eq: "test"}}) { id } }`,
			}
		}
	}

	// Default query
	return DefraDBQuery{
		Query: `query { __typename }`,
	}
}

// isAccessControlError checks if an error message indicates an access control denial
func (h *ACPTestHelper) isAccessControlError(message string) bool {
	accessControlKeywords := []string{
		"access denied",
		"permission denied",
		"unauthorized",
		"forbidden",
		"policy",
		"acp",
		"access control",
	}

	for _, keyword := range accessControlKeywords {
		if contains(message, keyword) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(s) == len(substr) ||
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SetupTestUsers sets up the test users with their appropriate group memberships
func (h *ACPTestHelper) SetupTestUsers() error {
	// Add indexers to indexer group
	indexers := []string{
		"did:user:anIndexer",
		"did:user:aBlockedIndexer",
		"did:user:aBannedIndexer",
	}

	for _, did := range indexers {
		if err := h.AddUserToIndexerGroup(did); err != nil {
			return fmt.Errorf("failed to add %s to indexer group: %w", did, err)
		}
	}

	// Add hosts to host group
	hosts := []string{
		"did:user:aHost",
		"did:user:aBlockedHost",
		"did:user:aBannedHost",
	}

	for _, did := range hosts {
		if err := h.AddUserToHostGroup(did); err != nil {
			return fmt.Errorf("failed to add %s to host group: %w", did, err)
		}
	}

	// Subscribe users to data feeds
	if err := h.SubscribeToDataFeed("did:user:subscriber", "datafeedA"); err != nil {
		return fmt.Errorf("failed to subscribe user to datafeedA: %w", err)
	}

	if err := h.SubscribeToDataFeed("did:user:subscriberToB", "datafeedB"); err != nil {
		return fmt.Errorf("failed to subscribe user to datafeedB: %w", err)
	}

	return nil
}
