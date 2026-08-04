package server

import (
	"log"
	"net/http"

	"github.com/YOUCEFngg/mqtt-dashboard/internal/models"
	"github.com/YOUCEFngg/mqtt-dashboard/internal/storage"
)

type Server struct {
	storage   storage.Storage
	wsManager *WebSocketManager
}

func NewServer(storage storage.Storage, updates chan models.SensorData) *Server {
	wsManager := NewWebSocketManager(storage, updates)
	go wsManager.Start()
	
	return &Server{
		storage:   storage,
		wsManager: wsManager,
	}
}

func (s *Server) Start(addr string) error {
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/api/topics", s.handleTopics)
	http.HandleFunc("/api/latest", s.handleLatest)
	http.HandleFunc("/api/history/", s.handleHistory)
	http.HandleFunc("/ws", s.wsManager.HandleConnection)
	
	log.Printf("HTTP server starting on %s", addr)
	return http.ListenAndServe(addr, nil)
}
