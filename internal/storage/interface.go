package storage

import "github.com/YOUCEFngg/mqtt-dashboard/internal/models"

// Storage defines the contract for all storage backends
type Storage interface {
	// Store saves a new sensor reading
	Store(data models.SensorData)

	// GetLatest returns the most recent reading for a topic
	GetLatest(topic string) (models.SensorData, bool)

	// GetHistory returns the last N data points for a topic
	GetHistory(topic string, limit int) []models.SensorData

	// Subscribe returns a channel that receives all new updates
	Subscribe() chan models.SensorData

	// Unsubscribe removes a subscriber channel
	Unsubscribe(ch chan models.SensorData)
}
