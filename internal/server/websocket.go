package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/YOUCEFngg/mqtt-dashboard/internal/models"
	"github.com/YOUCEFngg/mqtt-dashboard/internal/storage"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketManager struct {
	storage   storage.Storage
	clients   map[*websocket.Conn]bool
	mu        sync.Mutex
	broadcast chan models.SensorData
}

func NewWebSocketManager(storage storage.Storage, updates chan models.SensorData) *WebSocketManager {
	return &WebSocketManager{
		storage:   storage,
		clients:   make(map[*websocket.Conn]bool),
		broadcast: updates,
	}
}

func (wsm *WebSocketManager) Start() {
	for update := range wsm.broadcast {
		wsm.broadcastToAll(update)
	}
}

func (wsm *WebSocketManager) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	wsm.mu.Lock()
	wsm.clients[conn] = true
	wsm.mu.Unlock()

	log.Printf("Client connected. Total: %d", len(wsm.clients))

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	wsm.mu.Lock()
	delete(wsm.clients, conn)
	wsm.mu.Unlock()
	log.Printf("Client disconnected. Total: %d", len(wsm.clients))
}

func (wsm *WebSocketManager) broadcastToAll(data models.SensorData) {
	msg := models.WSMessage{
		Type:      "metric",
		Topic:     data.Topic,
		Value:     data.Value,
		Unit:      data.Unit,
		Timestamp: data.Timestamp,
	}

	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	for client := range wsm.clients {
		if err := client.WriteJSON(msg); err != nil {
			client.Close()
			delete(wsm.clients, client)
		}
	}
}
