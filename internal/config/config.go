package config

import (
	"fmt"
	"os"

	"github.com/YOUCEFngg/mqtt-dashboard/internal/models"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig         `yaml:"server"`
	MQTT   MQTTConfig           `yaml:"mqtt"`
	Topics []models.TopicConfig `yaml:"topics"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type MQTTConfig struct {
	Broker   string `yaml:"broker"`
	ClientID string `yaml:"client_id"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Defaults
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.MQTT.Broker == "" {
		cfg.MQTT.Broker = "tcp://broker.hivemq.com:1883"
	}
	if cfg.MQTT.ClientID == "" {
		cfg.MQTT.ClientID = "mqtt-dashboard-client"
	}

	return &cfg, nil
}
