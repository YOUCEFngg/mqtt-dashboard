package models

import "time"

// SensorData represents a single sensor reading
type SensorData struct {
	Topic     string    `json:"topic"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

// MQTTMessage is the internal message format
type MQTTMessage struct {
	Topic   string
	Payload []byte
}

// WSMessage is sent to WebSocket clients
type WSMessage struct {
	Type      string    `json:"type"`
	Topic     string    `json:"topic"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

// TopicConfig defines a topic to publish/subscribe
type TopicConfig struct {
	Name     string  `yaml:"name"`
	Topic    string  `yaml:"topic"`
	Interval string  `yaml:"interval"`
	MinValue float64 `yaml:"min_value"`
	MaxValue float64 `yaml:"max_value"`
	Unit     string  `yaml:"unit"`
}