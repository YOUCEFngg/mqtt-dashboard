package storage

import (
	"sync"

	"github.com/YOUCEFngg/mqtt-dashboard/internal/models"
)

// MemoryStorage implements Storage with in-memory ring buffer
type MemoryStorage struct {
	mu          sync.RWMutex
	data        map[string][]models.SensorData
	subscribers map[chan models.SensorData]bool
	maxPoints   int
}

// NewMemoryStorage creates new memory storage
func NewMemoryStorage(maxPoints int) *MemoryStorage {
	if maxPoints <= 0 {
		maxPoints = 1000
	}
	return &MemoryStorage{
		data:        make(map[string][]models.SensorData),
		subscribers: make(map[chan models.SensorData]bool),
		maxPoints:   maxPoints,
	}
}

// Store saves data and broadcasts to subscribers
func (s *MemoryStorage) Store(data models.SensorData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[data.Topic] = append(s.data[data.Topic], data)
	if len(s.data[data.Topic]) > s.maxPoints {
		s.data[data.Topic] = s.data[data.Topic][1:]
	}

	for ch := range s.subscribers {
		select {
		case ch <- data:
		default:
		}
	}
}

// GetLatest returns most recent reading for a topic
func (s *MemoryStorage) GetLatest(topic string) (models.SensorData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	points, exists := s.data[topic]
	if !exists || len(points) == 0 {
		return models.SensorData{}, false
	}
	return points[len(points)-1], true
}

// GetHistory returns last N data points for a topic
func (s *MemoryStorage) GetHistory(topic string, limit int) []models.SensorData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	points, exists := s.data[topic]
	if !exists {
		return nil
	}

	if limit >= len(points) {
		result := make([]models.SensorData, len(points))
		copy(result, points)
		return result
	}

	result := make([]models.SensorData, limit)
	copy(result, points[len(points)-limit:])
	return result
}

// Subscribe creates a new subscriber channel
func (s *MemoryStorage) Subscribe() chan models.SensorData {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan models.SensorData, 100)
	s.subscribers[ch] = true
	return ch
}

// Unsubscribe removes a subscriber channel
func (s *MemoryStorage) Unsubscribe(ch chan models.SensorData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscribers, ch)
	close(ch)
}
