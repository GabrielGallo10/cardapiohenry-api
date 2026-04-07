package http

import (
	"encoding/json"
	"net/http"
	"time"

	"cardapio-henry-api/internal/config"
)

type Server struct {
	config config.Config
}

func NewServer(cfg config.Config) *Server {
	return &Server{config: cfg}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.healthHandler)

	return mux
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	response := map[string]string{
		"status":      "ok",
		"service":     s.config.AppName,
		"environment": s.config.Environment,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
