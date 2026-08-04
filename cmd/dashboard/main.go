package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YOUCEFngg/mqtt-dashboard/internal/config"
	"github.com/YOUCEFngg/mqtt-dashboard/internal/models"
	"github.com/YOUCEFngg/mqtt-dashboard/internal/mqtt"
	"github.com/YOUCEFngg/mqtt-dashboard/internal/server"
	"github.com/YOUCEFngg/mqtt-dashboard/internal/storage"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	store := storage.NewMemoryStorage(1000)
	updates := make(chan models.SensorData, 100)

	// Start built-in MQTT broker
	broker := mqtt.NewBroker(1883)
	go func() {
		log.Println("Starting built-in MQTT broker on port 1883")
		if err := broker.Start(); err != nil {
			log.Printf("Broker error: %v", err)
		}
	}()
	defer broker.Stop()

	// Connect to HiveMQ
	mqttClient, err := mqtt.NewClient(
		cfg.MQTT.Broker,
		cfg.MQTT.ClientID,
		cfg.MQTT.Username,
		cfg.MQTT.Password,
	)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}

	if err := mqttClient.Connect(); err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mqttClient.Disconnect()

	// Start publisher and subscriber
	publisher := mqtt.NewPublisher(mqttClient, cfg.Topics)
	publisher.Start()

	subscriber := mqtt.NewSubscriber(mqttClient, store, updates)
	if err := subscriber.SubscribeToTopics(cfg.Topics); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Serve web files
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", fs)

	// Start HTTP server
	srv := server.NewServer(store, updates)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Dashboard starting on http://%s", addr)
		if err := srv.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = shutdownCtx
}
