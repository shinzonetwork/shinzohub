package tests

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

func parseTestCasesFromFile(path string) ([]TestCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var testCases []TestCase
	scanner := bufio.NewScanner(file)

	section := ""
	authRegex := regexp.MustCompile(`^(!?)([a-zA-Z0-9:]+)#([a-zA-Z0-9_]+)@(.+)$`)
	delegRegex := regexp.MustCompile(`^(!?)(did:[^ ]+)\s*>\s*([a-zA-Z0-9:]+)#([a-zA-Z0-9_]+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		switch line {
		case "Authorizations {":
			section = "auth"
			continue
		case "Delegations {":
			section = "deleg"
			continue
		case "}":
			section = ""
			continue
		}

		if section == "auth" {
			// Example: primitive:blocks#read@did:user:randomUser
			// or: !primitive:blocks#update@did:user:randomUser
			m := authRegex.FindStringSubmatch(line)
			if m != nil {
				neg := m[1] == "!"
				resource := m[2]
				action := m[3]
				userDID := m[4]
				tc := TestCase{
					Name:           line,
					UserDID:        userDID,
					Resource:       resource,
					Action:         action,
					ExpectedResult: !neg,
				}
				testCases = append(testCases, tc)
			}
		} else if section == "deleg" {
			// Example: !did:user:shinzohub > view:datafeedA#admin
			// or: did:user:shinzohub > view:datafeedB#reader
			m := delegRegex.FindStringSubmatch(line)
			if m != nil {
				neg := m[1] == "!"
				userDID := m[2]
				resource := m[3]
				rawAction := m[4]
				action := "_can_manage_" + rawAction
				tc := TestCase{
					Name:           line,
					UserDID:        userDID,
					Resource:       resource,
					Action:         action,
					ExpectedResult: !neg,
				}
				testCases = append(testCases, tc)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return testCases, nil
}

var didUserRegex = regexp.MustCompile(`did:user:[a-zA-Z0-9_]+`)
var groupRegex = regexp.MustCompile(`group:([a-zA-Z0-9]+)`)

func parseTestUsersFromFile(path string) (map[string]*TestUser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	users := make(map[string]*TestUser)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Extract did:user:<username> if present
		did := didUserRegex.FindString(line)
		if did == "" {
			continue // skip lines not related to did:user
		}

		user, exists := users[did]
		if !exists {
			user = &TestUser{DID: did}
			users[did] = user
		}

		// Update flags based on content of line
		lower := strings.ToLower(line)

		// Roles & flags rules based on input sample and your TestUser fields
		user.IsIndexer = strings.Contains(lower, "group:indexer")
		user.IsHost = strings.Contains(lower, "group:host")

		if strings.Contains(lower, "subscriber") {
			user.IsSubscriber = true
		}
		if strings.Contains(lower, "banned") {
			user.IsBanned = true
		}
		if strings.Contains(lower, "blocked") {
			if strings.Contains(lower, "indexer") {
				user.IsBlockedIndexer = true
			} else if strings.Contains(lower, "host") {
				user.IsBlockedHost = true
			}
		}

		// Extract group name and set user.Group
		matches := groupRegex.FindStringSubmatch(lower)
		if len(matches) > 1 {
			user.Group = matches[1] // e.g. "indexer", "host", "shinzoteam"
		} else if user.Group == "" {
			user.Group = "" // default empty string if no group found
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
