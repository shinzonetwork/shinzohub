package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"shinzohub/tests"
)

func main() {
	var (
		sourceHubURL = flag.String("sourcehub", "localhost:9090", "SourceHub URL")
		list         = flag.Bool("list", false, "List cached DIDs")
		clear        = flag.Bool("clear", false, "Clear DID cache")
		generate     = flag.Bool("generate", false, "Generate DIDs for all test users")
	)
	flag.Parse()

	generator := tests.NewDIDGenerator(*sourceHubURL)

	switch {
	case *list:
		if err := generator.ListCachedDIDs(); err != nil {
			log.Fatalf("Failed to list DIDs: %v", err)
		}

	case *clear:
		if err := generator.ClearCache(); err != nil {
			log.Fatalf("Failed to clear cache: %v", err)
		}
		fmt.Println("DID cache cleared")

	case *generate:
		ctx := context.Background()
		userDIDs, err := generator.GenerateTestUsers(ctx)
		if err != nil {
			log.Fatalf("Failed to generate test users: %v", err)
		}

		fmt.Println("Generated DIDs for test users:")
		for username, did := range userDIDs {
			fmt.Printf("  %s: %s\n", username, did)
		}

	default:
		fmt.Println("Usage:")
		fmt.Println("  didgen -list                    # List cached DIDs")
		fmt.Println("  didgen -clear                   # Clear DID cache")
		fmt.Println("  didgen -generate                # Generate DIDs for all test users")
		fmt.Println("  didgen -sourcehub=localhost:9090 -generate  # Generate with custom SourceHub URL")
		os.Exit(1)
	}
}
