package main

import (
	"log"
	"net/http"

	"shinzohub/api"
	"shinzohub/pkg/sourcehub"
	"shinzohub/pkg/utils"
	"shinzohub/pkg/validators"
)

func main() {
	registrar := api.ShinzoRegistrar{
		Validator: &validators.RegistrarValidator{},
		Acp:       &sourcehub.AcpGoClient{},
	}

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

	registrarMux.HandleFunc("/subscribe-to-data-feed", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := registrar.SubscribeToDataFeed(r.Context(), req.DID, req.DataFeedID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	mainMux := http.NewServeMux()
	mainMux.Handle("/registrar/", http.StripPrefix("/registrar", registrarMux))

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", mainMux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
