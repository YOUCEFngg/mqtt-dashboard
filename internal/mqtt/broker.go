package mqtt

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// MQTT Packet Types
const (
	CONNECT     = 1
	CONNACK     = 2
	PUBLISH     = 3
	PUBACK      = 4
	SUBSCRIBE   = 8
	SUBACK      = 9
	UNSUBSCRIBE = 10
	UNSUBACK    = 11
	PINGREQ     = 12
	PINGRESP    = 13
	DISCONNECT  = 14
)

// Broker implements a basic MQTT broker
type Broker struct {
	port      int
	listener  net.Listener
	clients   map[string]*BrokerClient
	topics    map[string][]*BrokerClient
	mu        sync.RWMutex
	running   bool
}

// BrokerClient represents a connected MQTT client
type BrokerClient struct {
	id       string
	conn     net.Conn
	topics   map[string]bool
	broker   *Broker
}

func NewBroker(port int) *Broker {
	return &Broker{
		port:    port,
		clients: make(map[string]*BrokerClient),
		topics:  make(map[string][]*BrokerClient),
	}
}

// Start begins listening for MQTT connections
func (b *Broker) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", b.port))
	if err != nil {
		return fmt.Errorf("failed to start broker on port %d: %w", b.port, err)
	}
	b.listener = listener
	b.running = true

	log.Printf("Built-in MQTT broker starting on port %d", b.port)

	go b.acceptLoop()
	return nil
}

func (b *Broker) acceptLoop() {
	for b.running {
		conn, err := b.listener.Accept()
		if err != nil {
			if b.running {
				log.Printf("Broker accept error: %v", err)
			}
			continue
		}
		go b.handleConnection(conn)
	}
}

func (b *Broker) handleConnection(conn net.Conn) {
	defer conn.Close()

	client := &BrokerClient{
		conn:   conn,
		topics: make(map[string]bool),
		broker: b,
	}

	reader := bufio.NewReader(conn)

	for b.running {
		// Read fixed header
		header, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				log.Printf("Broker read header error: %v", err)
			}
			return
		}

		packetType := (header >> 4) & 0x0F

		// Read remaining length
		remainingLength, err := readRemainingLength(reader)
		if err != nil {
			log.Printf("Broker read length error: %v", err)
			return
		}

		// Read payload
		payload := make([]byte, remainingLength)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			log.Printf("Broker read payload error: %v", err)
			return
		}

		// Handle packet
		switch packetType {
		case CONNECT:
			if err := b.handleConnect(client, payload); err != nil {
				log.Printf("Broker connect error: %v", err)
				return
			}
		case PUBLISH:
			b.handlePublish(client, header, payload)
		case SUBSCRIBE:
			b.handleSubscribe(client, payload)
		case PINGREQ:
			b.handlePing(client)
		case DISCONNECT:
			b.removeClient(client)
			return
		}
	}
}

func readRemainingLength(reader *bufio.Reader) (int, error) {
	var length int
	multiplier := 1

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		length += int(b&0x7F) * multiplier
		multiplier *= 128
		if (b & 0x80) == 0 {
			break
		}
		if multiplier > 128*128*128 {
			return 0, fmt.Errorf("malformed remaining length")
		}
	}
	return length, nil
}

func (b *Broker) handleConnect(client *BrokerClient, payload []byte) error {
	// Skip protocol name length and name
	if len(payload) < 2 {
		return fmt.Errorf("payload too short for connect")
	}
	idx := 2 + int(binary.BigEndian.Uint16(payload[0:2]))
	if len(payload) < idx+1 {
		return fmt.Errorf("payload too short for protocol level")
	}
	// Skip protocol level
	idx++
	if len(payload) < idx+1 {
		return fmt.Errorf("payload too short for connect flags")
	}
	// Skip connect flags
	idx++
	if len(payload) < idx+2 {
		return fmt.Errorf("payload too short for keep alive")
	}
	// Skip keep alive
	idx += 2

	// Client ID length and ID
	if len(payload) < idx+2 {
		return fmt.Errorf("payload too short for client ID length")
	}
	idLen := int(binary.BigEndian.Uint16(payload[idx : idx+2]))
	idx += 2
	if len(payload) < idx+idLen {
		return fmt.Errorf("payload too short for client ID")
	}
	client.id = string(payload[idx : idx+idLen])

	// Send CONNACK (session present = 0, return code = 0)
	connack := []byte{0x20, 0x02, 0x00, 0x00}
	_, err := client.conn.Write(connack)
	if err != nil {
		return err
	}

	b.mu.Lock()
	b.clients[client.id] = client
	b.mu.Unlock()

	log.Printf("Broker client connected: %s", client.id)
	return nil
}

func (b *Broker) handlePublish(client *BrokerClient, header byte, payload []byte) {
	if len(payload) < 2 {
		log.Printf("Broker publish payload too short")
		return
	}

	qos := (header >> 1) & 0x03

	// Read topic
	topicLen := int(binary.BigEndian.Uint16(payload[0:2]))
	if len(payload) < 2+topicLen {
		log.Printf("Broker publish payload too short for topic")
		return
	}
	topic := string(payload[2 : 2+topicLen])

	idx := 2 + topicLen

	// Skip packet ID if QoS > 0
	if qos > 0 {
		if len(payload) < idx+2 {
			log.Printf("Broker publish payload too short for packet ID")
			return
		}
		idx += 2
	}

	if len(payload) < idx {
		log.Printf("Broker publish payload too short for message")
		return
	}
	message := payload[idx:]

	log.Printf("Broker received publish on %s: %s", topic, string(message))

	// Forward to subscribers
	b.mu.RLock()
	subscribers := b.topics[topic]
	b.mu.RUnlock()

	for _, sub := range subscribers {
		if sub != client {
			b.sendPublish(sub, topic, message, qos)
		}
	}

	// Send PUBACK if QoS 1
	if qos == 1 && len(payload) >= idx-2+2 {
		puback := []byte{0x40, 0x02, payload[idx-2], payload[idx-1]}
		client.conn.Write(puback)
	}
}

func (b *Broker) sendPublish(client *BrokerClient, topic string, message []byte, qos byte) {
	topicBytes := []byte(topic)
	topicLen := len(topicBytes)
	msgLen := len(message)

	var buf bytes.Buffer
	buf.WriteByte(0x30 | (qos << 1)) // PUBLISH header

	remainingLen := 2 + topicLen + msgLen
	if qos > 0 {
		remainingLen += 2 // packet ID
	}

	// Write remaining length
	for {
		b := byte(remainingLen % 128)
		remainingLen /= 128
		if remainingLen > 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if remainingLen == 0 {
			break
		}
	}

	// Write topic
	buf.WriteByte(byte(topicLen >> 8))
	buf.WriteByte(byte(topicLen))
	buf.Write(topicBytes)

	// Write packet ID if QoS > 0
	if qos > 0 {
		buf.WriteByte(0x00)
		buf.WriteByte(0x01)
	}

	// Write message
	buf.Write(message)

	client.conn.Write(buf.Bytes())
}

func (b *Broker) handleSubscribe(client *BrokerClient, payload []byte) {
	if len(payload) < 2 {
		log.Printf("Broker subscribe payload too short")
		return
	}
	// Skip packet ID
	idx := 2

	for idx < len(payload) {
		if len(payload) < idx+2 {
			break
		}
		topicLen := int(binary.BigEndian.Uint16(payload[idx : idx+2]))
		idx += 2
		if len(payload) < idx+topicLen+1 {
			break
		}
		topic := string(payload[idx : idx+topicLen])
		idx += topicLen
		qos := payload[idx]
		idx++

		b.mu.Lock()
		b.topics[topic] = append(b.topics[topic], client)
		client.topics[topic] = true
		b.mu.Unlock()

		log.Printf("Broker client %s subscribed to: %s (QoS %d)", client.id, topic, qos)
	}

	// Send SUBACK
	suback := make([]byte, 5)
	suback[0] = 0x90
	suback[1] = 0x03
	suback[2] = payload[0]
	suback[3] = payload[1]
	suback[4] = 0x00 // QoS 0 granted
	client.conn.Write(suback)
}

func (b *Broker) handlePing(client *BrokerClient) {
	// Send PINGRESP
	client.conn.Write([]byte{0xD0, 0x00})
}

func (b *Broker) removeClient(client *BrokerClient) {
	b.mu.Lock()
	delete(b.clients, client.id)
	for topic := range client.topics {
		subs := b.topics[topic]
		for i, sub := range subs {
			if sub == client {
				b.topics[topic] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
	b.mu.Unlock()

	log.Printf("Broker client disconnected: %s", client.id)
}

// Stop shuts down the broker
func (b *Broker) Stop() error {
	b.running = false
	if b.listener != nil {
		return b.listener.Close()
	}
	return nil
}
