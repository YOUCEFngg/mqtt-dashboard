package mqtt

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps the Paho MQTT client
type Client struct {
	client mqtt.Client
	broker string
}

// NewClient creates a new MQTT client
func NewClient(broker, clientID, username, password string) (*Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second)

	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}

	// Connection handlers
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("✅ Connected to MQTT broker")
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("❌ Connection lost: %v", err)
	}

	client := mqtt.NewClient(opts)

	return &Client{
		client: client,
		broker: broker,
	}, nil
}

// Connect establishes connection to broker
func (c *Client) Connect() error {
	token := c.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	return nil
}

// Disconnect cleanly closes connection
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
}

// Publish sends a message to a topic
func (c *Client) Publish(topic string, payload []byte) error {
	token := c.client.Publish(topic, 0, false, payload)
	token.Wait()
	return token.Error()
}

// Subscribe registers a handler for a topic
func (c *Client) Subscribe(topic string, handler func(topic string, payload []byte)) error {
	token := c.client.Subscribe(topic, 0, func(c mqtt.Client, m mqtt.Message) {
		handler(m.Topic(), m.Payload())
	})
	token.Wait()
	return token.Error()
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}
