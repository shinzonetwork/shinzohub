package tests

import (
	"bufio"
	"fmt"
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

		// Parse group memberships with proper roles
		if strings.HasPrefix(lower, "group:indexer#") {
			if strings.Contains(lower, "#owner") {
				user.IndexerMembership = Owner
			} else if strings.Contains(lower, "#admin") {
				user.IndexerMembership = Admin
			} else if strings.Contains(lower, "#guest") {
				user.IndexerMembership = Guest
			}
		}
		if strings.HasPrefix(lower, "group:host#") {
			if strings.Contains(lower, "#owner") {
				user.HostMembership = Owner
			} else if strings.Contains(lower, "#admin") {
				user.HostMembership = Admin
			} else if strings.Contains(lower, "#guest") {
				user.HostMembership = Guest
			}
		}
		if strings.HasPrefix(lower, "group:shinzoteam#") {
			if strings.Contains(lower, "#owner") {
				user.ShinzoteamMembership = Owner
			} else if strings.Contains(lower, "#admin") {
				user.ShinzoteamMembership = Admin
			} else if strings.Contains(lower, "#guest") {
				user.ShinzoteamMembership = Guest
			}
		}

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

/*
* AcpRelations defines in Go code the relations between one object and a did or relation
* You can think of them as a codified representation of the relationships that are defined in the ACP playground during the design stage
* Our tests will parse the relationships.yaml file into this representation and then apply those relations on our test (dummy) objects
* Example 1: primitive:blocks#syncer@did:user:randomUser -> AcpRelations{ResourceName: "primitive", ObjectName: "blocks", Relation: "syncer", IsDidActor: true, Did: "did:user:randomUser"}
* Example 2: primitive:blocks#writer@group:indexer#member -> AcpRelations{ResourceName: "primitive", ObjectName: "blocks", Relation: "writer", IsDidActor: false, GroupName: "indexer", GroupRelation: "member"}
* Example 3: group:host#blocked@did:user:aBlockedHost -> AcpRelations{ResourceName: "group", ObjectName: "host", Relation: "blocked", IsDidActor: true, Did: "did:user:aBlockedHost"}
* Example 4: view:datafeedA#parent@primitive:blocks -> AcpRelations{ResourceName: "view", ObjectName: "datafeedA", Relation: "parent", IsParentRelation: true, ParentResourceType: "primitive", ParentResourceName: "blocks"}
 */
type AcpRelations struct {
	ResourceName       string
	ObjectName         string
	Relation           string
	IsDidActor         bool
	Did                string
	GroupName          string
	GroupRelation      string
	IsParentRelation   bool
	ParentResourceType string
	ParentResourceName string
}

// ParsedRelation represents a parsed relationship with its source line from the file
type ParsedRelation struct {
	SourceLine string
	Relation   AcpRelations
}

// parseAcpRelationsFromFile parses the relationships.yaml file and returns a slice of LineRelation
func parseAcpRelationsFromFile(path string) ([]ParsedRelation, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var relations []ParsedRelation
	scanner := bufio.NewScanner(file)

	// Regex to match the pattern: resource:object#relation@actor
	// This will capture: resource, object, relation, and the full actor part
	relationRegex := regexp.MustCompile(`^([a-zA-Z0-9]+):([a-zA-Z0-9]+)#([a-zA-Z0-9_]+)@(.+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Skip section markers
		if line == "Authorizations {" || line == "Delegations {" || line == "}" {
			continue
		}

		// First, check if this is a parent relation
		if strings.Contains(line, "#parent@") {
			// This is a parent relation: resource:object#parent@parentResource:parentObject
			// Parse the parts manually
			parts := strings.Split(line, "#parent@")
			if len(parts) == 2 {
				// Parse resource:object from the first part
				resourceParts := strings.Split(parts[0], ":")
				// Parse parentResource:parentObject from the second part
				parentParts := strings.Split(parts[1], ":")

				if len(resourceParts) == 2 && len(parentParts) == 2 {
					rel := AcpRelations{
						ResourceName:       resourceParts[0],
						ObjectName:         resourceParts[1],
						Relation:           "parent",
						IsParentRelation:   true,
						ParentResourceType: parentParts[0],
						ParentResourceName: parentParts[1],
					}
					relations = append(relations, ParsedRelation{
						SourceLine: line,
						Relation:   rel,
					})
					continue
				}
			}
		}

		// Parse the regular relationship line
		matches := relationRegex.FindStringSubmatch(line)
		if matches == nil {
			continue // Skip lines that don't match the pattern
		}

		resourceName := matches[1]
		objectName := matches[2]
		relation := matches[3]
		actorPart := matches[4]

		// Determine if the actor is a DID or a group
		if strings.HasPrefix(actorPart, "did:") {
			// DID actor: did:user:randomUser
			rel := AcpRelations{
				ResourceName: resourceName,
				ObjectName:   objectName,
				Relation:     relation,
				IsDidActor:   true,
				Did:          actorPart,
			}
			relations = append(relations, ParsedRelation{
				SourceLine: line,
				Relation:   rel,
			})
		} else if strings.HasPrefix(actorPart, "group:") {
			// Group actor: group:indexer#member
			// Parse the group part: group:groupName#groupRelation
			groupParts := strings.Split(actorPart, "#")
			if len(groupParts) == 2 {
				groupName := strings.TrimPrefix(groupParts[0], "group:")
				groupRelation := groupParts[1]

				rel := AcpRelations{
					ResourceName:  resourceName,
					ObjectName:    objectName,
					Relation:      relation,
					IsDidActor:    false,
					GroupName:     groupName,
					GroupRelation: groupRelation,
				}
				relations = append(relations, ParsedRelation{
					SourceLine: line,
					Relation:   rel,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return relations, nil
}

// printParsedRelations prints all parsed relations in the format: sourceLine -> Relation
func printParsedRelations(relations []ParsedRelation) {
	for _, parsedRel := range relations {
		fmt.Printf("%s -> %+v\n", parsedRel.SourceLine, parsedRel.Relation)
	}
}
