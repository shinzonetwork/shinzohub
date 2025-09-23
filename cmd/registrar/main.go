package main

import (
	"log"

	"shinzohub/pkg/registrar"
	"shinzohub/pkg/sourcehub"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	acpGoClient, err := sourcehub.CreateShinzoAcpGoClient("sourcehub-dev")
	if err != nil {
		log.Fatalf("Failed to create ACP Go client: %v", err)
	}

	service := registrar.NewRegistrarService(acpGoClient)

	log.Println("Server listening on :8081")
	if err := service.Start(":8081"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
