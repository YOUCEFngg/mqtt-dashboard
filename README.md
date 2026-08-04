# MQTT Live Dashboard

A real-time IoT dashboard built in Go that connects to MQTT brokers, publishes sensor data, and displays it on an interactive web interface. Includes a built-in MQTT broker for local testing.

## 🚀 Quick Start

### Download Pre-built Binaries

| Platform | Download |
|----------|----------|
| Linux | [mqtt-dashboard-linux](https://github.com/YOUCEFngg/mqtt-dashboard/releases/latest/download/mqtt-dashboard-linux) |
| Windows | [mqtt-dashboard-windows.exe](https://github.com/YOUCEFngg/mqtt-dashboard/releases/latest/download/mqtt-dashboard-windows.exe) |
| Mac | [mqtt-dashboard-mac](https://github.com/YOUCEFngg/mqtt-dashboard/releases/latest/download/mqtt-dashboard-mac) |

> **Note:** After downloading, make executable: `chmod +x mqtt-dashboard-linux`

### Run

```bash
# Linux/Mac
./mqtt-dashboard-linux

# Windows
mqtt-dashboard-windows.exe
```

Open browser: `http://localhost:8081`

---

## Features

- **MQTT Client**: Connects to HiveMQ public broker
- **Data Publisher**: Generates simulated sensor data
- **Real-time Dashboard**: WebSocket-powered live charts
- **Built-in MQTT Broker**: Custom lightweight broker on port 1883
- **Interactive UI**: Toggle topics, view statistics (min/max/avg)
- **Persistent Topics**: Creates topics under `rmbtech/interview/rug/NegadYoucef/`

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Publisher  │────▶│  HiveMQ     │────▶│  Subscriber │
│  (Go app)   │     │  (Cloud)    │     │  (Go app)   │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                                │
┌─────────────┐     ┌─────────────┐     ┌─────┴─────┐
│ Built-in    │     │   Storage   │────▶│ WebSocket │
│ Broker      │     │  (Memory)   │     │  Server   │
│ (Port 1883) │     └─────────────┘     └─────┬─────┘
└─────────────┘                               │
                                              ▼
                                        ┌─────────────┐
                                        │   Browser   │
                                        │  (Chart.js) │
                                        └─────────────┘
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22 |
| MQTT Client | eclipse/paho.mqtt.golang |
| WebSocket | gorilla/websocket |
| Frontend | Chart.js, vanilla JS |
| Config | YAML |

## Project Structure

```
mqtt-dashboard/
├── cmd/dashboard/
│   └── main.go              # Application entry point
├── internal/
│   ├── mqtt/
│   │   ├── client.go        # MQTT client (HiveMQ connection)
│   │   ├── publisher.go     # Simulated sensor data generator
│   │   ├── subscriber.go    # Message receiver
│   │   └── broker.go        # Built-in MQTT broker
│   ├── server/
│   │   ├── http.go          # HTTP server
│   │   ├── handlers.go      # REST API endpoints
│   │   └── websocket.go     # WebSocket manager
│   ├── storage/
│   │   ├── interface.go     # Storage contract
│   │   └── memory.go        # In-memory ring buffer
│   ├── config/
│   │   └── config.go        # YAML configuration loader
│   └── models/
│       └── models.go        # Data structures
├── web/
│   ├── index.html           # Dashboard UI
│   ├── app.js               # Frontend logic
│   └── style.css            # Styling
└── config.yaml              # Application configuration
```

## How It Works

### 1. Data Flow

1. **Publisher** generates fake sensor data (temperature, humidity, pressure)
2. **Publishes** to HiveMQ broker under `rmbtech/interview/rug/NegadYoucef/sensors/`
3. **Subscriber** receives data from same topics
4. **Storage** keeps last 1000 data points per topic
5. **WebSocket** broadcasts updates to all connected browsers
6. **Dashboard** displays real-time charts and statistics

### 2. Built-in MQTT Broker

The application includes a custom MQTT broker implementation:

- Listens on TCP port 1883
- Handles CONNECT, SUBSCRIBE, PUBLISH, PING, DISCONNECT packets
- Routes messages to subscribers
- No external dependencies

### 3. Topic Structure

```
rmbtech/interview/rug/NegadYoucef/sensors/temperature
rmbtech/interview/rug/NegadYoucef/sensors/humidity
rmbtech/interview/rug/NegadYoucef/sensors/pressure
```

## How to Build from Source

### Prerequisites

- Go 1.22+
- mosquitto clients (optional, for testing)

### Installation

```bash
git clone https://github.com/YOUCEFngg/mqtt-dashboard.git
cd mqtt-dashboard
go mod tidy
```

### Build

```bash
# Linux
go build -o mqtt-dashboard-linux cmd/dashboard/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o mqtt-dashboard-windows.exe cmd/dashboard/main.go

# Mac
GOOS=darwin GOARCH=amd64 go build -o mqtt-dashboard-mac cmd/dashboard/main.go
```

### Run

```bash
./mqtt-dashboard-linux
```

Open browser: `http://localhost:8081`

### Test Built-in Broker

Terminal 1 - Subscribe:
```bash
mosquitto_sub -h localhost -p 1883 -t "test/topic"
```

Terminal 2 - Publish:
```bash
mosquitto_pub -h localhost -p 1883 -t "test/topic" -m "hello"
```

### Verify HiveMQ Topics

```bash
mosquitto_sub -h broker.hivemq.com -p 1883 -t "rmbtech/interview/rug/NegadYoucef/sensors/#"
```

## Configuration

Edit `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8081

mqtt:
  broker: "tcp://broker.hivemq.com:1883"
  client_id: "youcefng-dashboard"

topics:
  - name: "temperature"
    topic: "rmbtech/interview/rug/NegadYoucef/sensors/temperature"
    interval: "3s"
    min_value: 20.0
    max_value: 35.0
    unit: "°C"
```

## Key Go Concepts Demonstrated

| Concept | Implementation |
|---------|---------------|
| Goroutines | Concurrent publisher, subscriber, broker |
| Channels | Safe data passing between components |
| Interfaces | Swappable storage backends |
| Mutex | Thread-safe storage access |
| Context | Graceful shutdown handling |
| Struct tags | YAML/JSON serialization |
| TCP sockets | Custom MQTT broker implementation |
| Binary protocols | MQTT packet parsing |

## License

MIT
