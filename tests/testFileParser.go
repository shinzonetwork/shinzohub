package tests

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

func parseFile(path string) ([]TestCase, error) {
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
