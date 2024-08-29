# UdGo

This project implements an advanced UDP (User Datagram Protocol) framework in Go, featuring automated packet reassembly, customizable retry and timeout mechanisms, and packet prioritization with Quality of Service (QoS) support.

## Table of Contents

1. [Features](#features)
2. [Installation](#installation)
3. [Usage](#usage)
4. [API Reference](#api-reference)
5. [Configuration](#configuration)
6. [Examples](#examples)
7. [Contributing](#contributing)
8. [License](#license)

## Features

1. **Automated Packet Reassembly**: Ensures data integrity and order in a connectionless protocol.
2. **Customizable Retry and Timeout Mechanisms**: Improves reliability in high packet loss environments.
3. **Packet Prioritization and QoS**: Allows critical data to be transmitted first.

## Installation

To install the UDP framework, use the following command:

```bash
go get github.com/mrinalxdev/udpframework
```

## Usage

Here's a basic example of how to use the UDP framework:

```go
package main

import (
	"log"
	"net"

	"github.com/yourusername/udpframework"
)

func main() {
	retryConfig := udpframework.RetryConfig{
		MaxRetries:  3,
		BaseTimeout: time.Second,
		BackoffRate: 1.5,
	}

	qosConfig := udpframework.QoSConfig{
		PriorityLevels: 3,
		PriorityQueues: make([][]udpframework.Packet, 3),
	}

	framework, err := udpframework.NewUDPFramework(":8080", retryConfig, qosConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer framework.Close()

	// Send a packet
	packet := udpframework.Packet{
		SequenceNumber: 1,
		Priority:       2,
		Data:           []byte("Hello, UDP!"),
	}
	destAddr, _ := net.ResolveUDPAddr("udp", "localhost:9090")
	err = framework.SendPacket(packet, destAddr)
	if err != nil {
		log.Printf("Error sending packet: %v", err)
	}

	// Receive packets
	for {
		data, addr, err := framework.ReceivePacket()
		if err != nil {
			log.Printf("Error receiving packet: %v", err)
			continue
		}
		log.Printf("Received from %v: %s", addr, string(data))
	}
}
```

## API Reference

### Types

#### UDPFramework

```go
type UDPFramework struct {
	// Unexported fields
}
```

The main struct for the UDP framework.

#### Packet

```go
type Packet struct {
	SequenceNumber uint32
	Priority       int
	Data           []byte
	RetryCount     int
}
```

Represents a UDP packet with additional metadata.

#### RetryConfig

```go
type RetryConfig struct {
	MaxRetries  int
	BaseTimeout time.Duration
	BackoffRate float64
}
```

Holds the configuration for retry and timeout mechanisms.

#### QoSConfig

```go
type QoSConfig struct {
	PriorityLevels int
	PriorityQueues [][]Packet
}
```

Holds the configuration for Quality of Service.

### Functions

#### NewUDPFramework

```go
func NewUDPFramework(addr string, retryConfig RetryConfig, qosConfig QoSConfig) (*UDPFramework, error)
```

Creates a new instance of UDPFramework.

#### SendPacket

```go
func (uf *UDPFramework) SendPacket(packet Packet, destAddr *net.UDPAddr) error
```

Sends a packet with automatic retries and prioritization.

#### ReceivePacket

```go
func (uf *UDPFramework) ReceivePacket() ([]byte, *net.UDPAddr, error)
```

Receives and reassembles packets.

#### Close

```go
func (uf *UDPFramework) Close() error
```

Closes the UDP connection.

## Configuration

### Retry Configuration

- `MaxRetries`: Maximum number of retry attempts.
- `BaseTimeout`: Initial timeout duration.
- `BackoffRate`: Rate at which the timeout increases after each retry.

### QoS Configuration

- `PriorityLevels`: Number of priority levels.
- `PriorityQueues`: Slice of queues for each priority level.

## Examples

### Sending a High-Priority Packet

```go
packet := udpframework.Packet{
	SequenceNumber: 1,
	Priority:       2, // High priority
	Data:           []byte("Important message"),
}
destAddr, _ := net.ResolveUDPAddr("udp", "localhost:9090")
err := framework.SendPacket(packet, destAddr)
```

### Receiving and Reassembling Packets

```go
for {
	data, addr, err := framework.ReceivePacket()
	if err != nil {
		log.Printf("Error receiving packet: %v", err)
		continue
	}
	log.Printf("Reassembled message from %v: %s", addr, string(data))
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.