package mqtt

import (
	"encoding/json"
	"log"
	"time"

	"github.com/YOUCEFngg/mqtt-dashboard/internal/models"
	"github.com/YOUCEFngg/mqtt-dashboard/internal/storage"
)

// Subscriber listens to MQTT topics and stores data
type Subscriber struct {
	client  *Client
	storage storage.Storage
	updates chan<- models.SensorData
}

// NewSubscriber creates a new subscriber
func NewSubscriber(client *Client, storage storage.Storage, updates chan<- models.SensorData) *Subscriber {
	return &Subscriber{
		client:  client,
		storage: storage,
		updates: updates,
	}
}

// SubscribeToTopics subscribes to all configured topics
func (s *Subscriber) SubscribeToTopics(topics []models.TopicConfig) error {
	for _, topic := range topics {
		if err := s.subscribe(topic.Topic); err != nil {
			return err
		}
	}
	return nil
}

func (s *Subscriber) subscribe(topic string) error {
	log.Printf("📡 Subscribing to: %s", topic)

	return s.client.Subscribe(topic, func(receivedTopic string, payload []byte) {
		// Parse JSON payload
		var data struct {
			Value float64 `json:"value"`
			Unit  string  `json:"unit"`
			Time  string  `json:"time"`
		}

		if err := json.Unmarshal(payload, &data); err != nil {
			log.Printf("Failed to parse message from %s: %v", receivedTopic, err)
			return
		}

		// Parse timestamp
		timestamp, _ := time.Parse(time.RFC3339, data.Time)
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		sensorData := models.SensorData{
			Topic:     receivedTopic,
			Value:     data.Value,
			Unit:      data.Unit,
			Timestamp: timestamp,
		}

		// Store and broadcast
		s.storage.Store(sensorData)

		select {
		case s.updates <- sensorData:
		default:
		}
	})
}
