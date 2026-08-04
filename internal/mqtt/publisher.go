package mqtt

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/YOUCEFngg/mqtt-dashboard/internal/models"
)

// Publisher generates and publishes fake sensor data
type Publisher struct {
	client *Client
	topics []models.TopicConfig
}

// NewPublisher creates a new publisher
func NewPublisher(client *Client, topics []models.TopicConfig) *Publisher {
	return &Publisher{
		client: client,
		topics: topics,
	}
}

// Start begins publishing data for all topics
func (p *Publisher) Start() {
	for _, topic := range p.topics {
		go p.publishLoop(topic)
	}
}

func (p *Publisher) publishLoop(topic models.TopicConfig) {
	interval, err := time.ParseDuration(topic.Interval)
	if err != nil || interval <= 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		// Generate random value between min and max
		value := topic.MinValue + rand.Float64()*(topic.MaxValue-topic.MinValue)

		// Create payload
		payload := fmt.Sprintf(`{"value":%.2f,"unit":"%s","time":"%s"}`,
			value,
			topic.Unit,
			time.Now().Format(time.RFC3339),
		)

		// Publish
		if err := p.client.Publish(topic.Topic, []byte(payload)); err != nil {
			continue // Skip on error, will retry next tick
		}
	}
}
