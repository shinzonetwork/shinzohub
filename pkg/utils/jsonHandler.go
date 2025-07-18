package utils

import (
	"encoding/json"
	"net/http"
)

// HandlerFunc is a generic handler type for JSON APIs.
type HandlerFunc[Req any, Resp any] func(r *http.Request, req Req) (Resp, int, error)

// JSONHandler wraps a generic handler and handles JSON parsing and response writing.
func JSONHandler[Req any, Resp any](handler HandlerFunc[Req, Resp]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
		resp, status, err := handler(r, req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(resp)
	}
} 