package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Server struct {
	Port int
}

func NewServer(port int) *Server {
	return &Server{Port: port}
}

func (s *Server) Start() error {
	http.HandleFunc("/health", s.handleHealth)
	// Additional endpoints for /draft, /campaigns will be added in sub-tasks
	
	fmt.Printf("🌐 musu-marketer API Server starting on port %d...\n", s.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}
