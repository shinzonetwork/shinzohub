package main

import (
	"log"
	"net/http"
	"os"

	"shinzohub/api"
	"shinzohub/pkg/sourcehub"
	"shinzohub/pkg/utils"
	"shinzohub/pkg/validators"

	"github.com/sourcenetwork/sourcehub/sdk"
)

func main() {
	registrar := buildRegistrarHandler()

	type RegistrarRequest struct {
		DID        string `json:"did"`
		DataFeedID string `json:"dataFeedId,omitempty"`
	}
	type RegistrarResponse struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	registrarMux := http.NewServeMux()

	registrarMux.HandleFunc("/request-indexer-role", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := registrar.RequestIndexerRole(r.Context(), req.DID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/request-host-role", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := registrar.RequestHostRole(r.Context(), req.DID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/block-indexer", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := registrar.BlockIndexer(r.Context(), req.DID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/block-host", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := registrar.BlockHost(r.Context(), req.DID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/subscribe-to-data-feed", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := registrar.SubscribeToDataFeed(r.Context(), req.DID, req.DataFeedID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/ban-user-from-resource", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := registrar.BanUserFromResource(r.Context(), req.DID, req.DataFeedID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/create-data-feed", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := registrar.CreateDataFeed(r.Context(), req.DID, req.DataFeedID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	mainMux := http.NewServeMux()
	mainMux.Handle("/registrar/", http.StripPrefix("/registrar", registrarMux))

	log.Println("Server listening on :8081")
	if err := http.ListenAndServe(":8081", mainMux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func buildRegistrarHandler() api.ShinzoRegistrar {
	signer, err := sourcehub.NewApiSignerFromEnv()
	if err != nil {
		log.Fatalf("Failed to load API signer: %v", err)
	}

	acpClient, err := sdk.NewClient()
	if err != nil {
		log.Fatalf("Failed to create ACP SDK client: %v", err)
	}
	txBuilder, err := sdk.NewTxBuilder(sdk.WithSDKClient(acpClient))
	if err != nil {
		log.Fatalf("Failed to create TxBuilder: %v", err)
	}

	policyId := os.Getenv("POLICY_ID")
	if policyId == "" {
		log.Fatalf("POLICY_ID environment variable is required")
	}

	acpGoClient := sourcehub.NewAcpGoClient(acpClient, &txBuilder, signer, policyId)

	registrar := api.ShinzoRegistrar{
		Validator: &validators.RegistrarValidator{},
		Acp:       acpGoClient,
	}
	return registrar
}
