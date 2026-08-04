package mqtt

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
)

type SimpleBroker struct {
	port     int
	clients  map[net.Conn]*ClientInfo
	topics   map[string][]net.Conn
	mu       sync.RWMutex
	listener net.Listener
	running  bool
}

type ClientInfo struct {
	conn   net.Conn
	id     string
	topics map[string]bool
}

func NewBroker(port int) *SimpleBroker {
	return &SimpleBroker{
		port:    port,
		clients: make(map[net.Conn]*ClientInfo),
		topics:  make(map[string][]net.Conn),
	}
}

func (b *SimpleBroker) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", b.port))
	if err != nil {
		return fmt.Errorf("failed to start broker: %w", err)
	}
	b.listener = listener
	b.running = true

	log.Printf("Built-in MQTT broker starting on port %d", b.port)

	go b.acceptLoop()
	return nil
}

func (b *SimpleBroker) acceptLoop() {
	for b.running {
		conn, err := b.listener.Accept()
		if err != nil {
			if b.running {
				log.Printf("Accept error: %v", err)
			}
			continue
		}
		go b.handleClient(conn)
	}
}

func (b *SimpleBroker) handleClient(conn net.Conn) {
	defer conn.Close()

	client := &ClientInfo{
		conn:   conn,
		id:     conn.RemoteAddr().String(),
		topics: make(map[string]bool),
	}

	b.mu.Lock()
	b.clients[conn] = client
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.clients, conn)
		for topic := range client.topics {
			b.removeSubscriber(topic, conn)
		}
		b.mu.Unlock()
	}()

	reader := bufio.NewReader(conn)

	for b.running {
		header, err := reader.ReadByte()
		if err != nil {
			return
		}

		packetType := (header >> 4) & 0x0F
		remainingLength, err := b.readRemainingLength(reader)
		if err != nil {
			return
		}

		payload := make([]byte, remainingLength)
		_, err = reader.Read(payload)
		if err != nil {
			return
		}

		switch packetType {
		case 1:
			b.handleConnect(client, payload)
		case 8:
			b.handleSubscribe(client, payload)
		case 3:
			b.handlePublish(client, payload)
		case 12:
			b.handlePing(client)
		case 14:
			return
		}
	}
}

func (b *SimpleBroker) readRemainingLength(reader *bufio.Reader) (int, error) {
	var length int
	var multiplier int = 1

	for {
		byteVal, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		length += int(byteVal&0x7F) * multiplier
		if (byteVal & 0x80) == 0 {
			break
		}
		multiplier *= 128
	}

	return length, nil
}

func (b *SimpleBroker) handleConnect(client *ClientInfo, payload []byte) {
	ack := []byte{0x20, 0x02, 0x00, 0x00}
	client.conn.Write(ack)
}

func (b *SimpleBroker) handleSubscribe(client *ClientInfo, payload []byte) {
	if len(payload) < 3 {
		return
	}

	topicLen := int(payload[2])<<8 | int(payload[3])
	if len(payload) < 4+topicLen {
		return
	}

	topic := string(payload[4 : 4+topicLen])

	b.mu.Lock()
	b.topics[topic] = append(b.topics[topic], client.conn)
	client.topics[topic] = true
	b.mu.Unlock()

	ack := []byte{0x90, 0x03, payload[0], payload[1], 0x00}
	client.conn.Write(ack)
}

func (b *SimpleBroker) handlePublish(client *ClientInfo, payload []byte) {
	if len(payload) < 2 {
		return
	}

	topicLen := int(payload[0])<<8 | int(payload[1])
	if len(payload) < 2+topicLen {
		return
	}

	topic := string(payload[2 : 2+topicLen])
	message := payload[2+topicLen:]

	b.mu.RLock()
	subscribers := b.topics[topic]
	b.mu.RUnlock()

	packet := b.buildPublishPacket(topic, message)

	for _, sub := range subscribers {
		if sub != client.conn {
			sub.Write(packet)
		}
	}
}

func (b *SimpleBroker) buildPublishPacket(topic string, message []byte) []byte {
	topicBytes := []byte(topic)
	topicLen := len(topicBytes)
	msgLen := len(message)
	remainingLen := 2 + topicLen + msgLen

	packet := make([]byte, 0, 2+remainingLen)
	packet = append(packet, 0x30)

	if remainingLen < 128 {
		packet = append(packet, byte(remainingLen))
	} else {
		packet = append(packet, byte(remainingLen&0x7F|0x80))
		packet = append(packet, byte(remainingLen>>7))
	}

	packet = append(packet, byte(topicLen>>8), byte(topicLen))
	packet = append(packet, topicBytes...)
	packet = append(packet, message...)

	return packet
}

func (b *SimpleBroker) handlePing(client *ClientInfo) {
	client.conn.Write([]byte{0xD0, 0x00})
}

func (b *SimpleBroker) removeSubscriber(topic string, conn net.Conn) {
	subs := b.topics[topic]
	for i, sub := range subs {
		if sub == conn {
			b.topics[topic] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

func (b *SimpleBroker) Stop() error {
	b.running = false
	if b.listener != nil {
		return b.listener.Close()
	}
	return nil
}
