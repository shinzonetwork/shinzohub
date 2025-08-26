package registrar

import (
	"log"
	"net/http"

	"shinzohub/api"
	"shinzohub/pkg/sourcehub"
	"shinzohub/pkg/utils"
	"shinzohub/pkg/validators"
)

type RegistrarService struct {
	registrar api.ShinzoRegistrar
	mux       *http.ServeMux
	server    *http.Server
}

type RegistrarRequest struct {
	DID               string   `json:"did"`
	DataFeedID        string   `json:"dataFeedId,omitempty"`
	ParentResourceIDs []string `json:"parentResourceIds,omitempty"`
}

type RegistrarResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func NewRegistrarService(acpClient sourcehub.ShinzoAcpClient) *RegistrarService {
	registrar := api.ShinzoRegistrar{
		Validator: &validators.RegistrarValidator{},
		Acp:       acpClient,
	}

	service := &RegistrarService{
		registrar: registrar,
		mux:       http.NewServeMux(),
	}

	service.setupRoutes()
	return service
}

func (s *RegistrarService) setupRoutes() {
	registrarMux := http.NewServeMux()

	registrarMux.HandleFunc("/request-indexer-role", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := s.registrar.RequestIndexerRole(r.Context(), req.DID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/request-host-role", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := s.registrar.RequestHostRole(r.Context(), req.DID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/block-indexer", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := s.registrar.BlockIndexer(r.Context(), req.DID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/block-host", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := s.registrar.BlockHost(r.Context(), req.DID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/subscribe-to-data-feed", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := s.registrar.SubscribeToDataFeed(r.Context(), req.DID, req.DataFeedID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/ban-user-from-resource", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := s.registrar.BanUserFromView(r.Context(), req.DID, req.DataFeedID)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	registrarMux.HandleFunc("/create-data-feed", utils.JSONHandler(func(r *http.Request, req RegistrarRequest) (RegistrarResponse, int, error) {
		err := s.registrar.CreateDataFeed(r.Context(), req.DID, req.DataFeedID, req.ParentResourceIDs)
		if err != nil {
			return RegistrarResponse{Success: false, Error: err.Error()}, http.StatusBadRequest, nil
		}
		return RegistrarResponse{Success: true}, http.StatusOK, nil
	}))

	s.mux.Handle("/registrar/", http.StripPrefix("/registrar", registrarMux))
}

func (s *RegistrarService) Start(addr string) error {
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	log.Printf("Registrar service starting on %s", addr)
	return s.server.ListenAndServe()
}

func (s *RegistrarService) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
