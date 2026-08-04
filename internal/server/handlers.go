package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) handleTopics(w http.ResponseWriter, r *http.Request) {
	topics := []string{}
	// Get topics from storage - we'll implement this properly later
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topics)
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		http.Error(w, "topic parameter required", http.StatusBadRequest)
		return
	}

	data, exists := s.storage.GetLatest(topic)
	if !exists {
		http.Error(w, "no data for topic", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	topic := strings.Join(parts[3:], "/")
	limit := 100

	history := s.storage.GetHistory(topic, limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}
