package udpframework

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
	"sync"
	"time"
)

type Packet struct {
	SequenceNumber uint32
	Priority       int
	Data           []byte
	RetryCount     int
	Timestamp      time.Time
}


type UDPFramework struct {
	conn            *net.UDPConn
	reassemblyQueue map[uint32]*Packet
	retryConfig     RetryConfig
	qosConfig       QoSConfig
	bufferConfig    BufferConfig
	stats           Stats
	mu              sync.Mutex
}

type RetryConfig struct {
	MaxRetries  int
	BaseTimeout time.Duration
	BackoffRate float64
}

type QoSConfig struct {
	PriorityLevels int
	PriorityQueues [][]Packet
}


type BufferConfig struct {
	MaxBufferSize int
	FlushInterval time.Duration
}


type Stats struct {
	PacketsSent     uint64
	PacketsReceived uint64
	PacketsDropped  uint64
	RetryCount      uint64
}


func UdGo(addr string, retryConfig RetryConfig, qosConfig QoSConfig, bufferConfig BufferConfig) (*UDPFramework, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	uf := &UDPFramework{
		conn:            conn,
		reassemblyQueue: make(map[uint32]*Packet),
		retryConfig:     retryConfig,
		qosConfig:       qosConfig,
		bufferConfig:    bufferConfig,
		mu:              sync.Mutex{},
	}

	go uf.flushBufferPeriodically()

	return uf, nil
}


func (uf *UDPFramework) SendPacket(packet Packet, destAddr *net.UDPAddr) error {
	uf.mu.Lock()
	defer uf.mu.Unlock()


	uf.qosConfig.PriorityQueues[packet.Priority] = append(uf.qosConfig.PriorityQueues[packet.Priority], packet)


	for i := uf.qosConfig.PriorityLevels - 1; i >= 0; i-- {
		for len(uf.qosConfig.PriorityQueues[i]) > 0 {
			p := uf.qosConfig.PriorityQueues[i][0]
			uf.qosConfig.PriorityQueues[i] = uf.qosConfig.PriorityQueues[i][1:]

			err := uf.sendWithRetry(p, destAddr)
			if err != nil {
				uf.stats.PacketsDropped++
				return err
			}
			uf.stats.PacketsSent++
		}
	}

	return nil
}


func (uf *UDPFramework) sendWithRetry(packet Packet, destAddr *net.UDPAddr) error {
	timeout := uf.retryConfig.BaseTimeout
	for retry := 0; retry < uf.retryConfig.MaxRetries; retry++ {
		_, err := uf.conn.WriteToUDP(packet.Data, destAddr)
		if err == nil {
			return nil
		}

		uf.stats.RetryCount++
		time.Sleep(timeout)
		timeout = time.Duration(float64(timeout) * uf.retryConfig.BackoffRate)
	}

	return ErrMaxRetriesReached
}


func (uf *UDPFramework) ReceivePacket() ([]byte, *net.UDPAddr, error) {
	buffer := make([]byte, 1500) // Typical MTU size
	n, remoteAddr, err := uf.conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, nil, err
	}

	uf.stats.PacketsReceived++

	packet := Packet{
		SequenceNumber: binary.BigEndian.Uint32(buffer[:4]),
		Data:           buffer[4:n],
		Timestamp:      time.Now(),
	}

	uf.mu.Lock()
	defer uf.mu.Unlock()

	uf.reassemblyQueue[packet.SequenceNumber] = &packet


	assembledData := uf.tryReassemble()
	if assembledData != nil {
		return assembledData, remoteAddr, nil
	}


	return nil, remoteAddr, nil
}


func (uf *UDPFramework) tryReassemble() []byte {
	var assembledData []byte
	expectedSeq := uint32(0)

	for {
		packet, exists := uf.reassemblyQueue[expectedSeq]
		if !exists {
			break
		}

		assembledData = append(assembledData, packet.Data...)
		delete(uf.reassemblyQueue, expectedSeq)
		expectedSeq++
	}

	if len(assembledData) > 0 {
		return assembledData
	}

	return nil
}


func (uf *UDPFramework) flushBufferPeriodically() {
	ticker := time.NewTicker(uf.bufferConfig.FlushInterval)
	defer ticker.Stop()

	for range ticker.C {
		uf.flushBuffer()
	}
}


func (uf *UDPFramework) flushBuffer() {
	uf.mu.Lock()
	defer uf.mu.Unlock()

	now := time.Now()
	for seq, packet := range uf.reassemblyQueue {
		if now.Sub(packet.Timestamp) > uf.bufferConfig.FlushInterval {
			delete(uf.reassemblyQueue, seq)
			uf.stats.PacketsDropped++
		}
	}
}


func (uf *UDPFramework) GetStats() Stats {
	uf.mu.Lock()
	defer uf.mu.Unlock()
	return uf.stats
}


func (uf *UDPFramework) Close() error {
	return uf.conn.Close()
}


var ErrMaxRetriesReached = errors.New("maximum retries reached")


func (uf *UDPFramework) SimulatePacketLoss(lossPercentage int) {
	rand.Seed(time.Now().UnixNano())
	if rand.Intn(100) < lossPercentage {
		uf.stats.PacketsDropped++
		return
	}

}

func (uf *UDPFramework) Compress(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	var compressed []byte
	count := 1
	current := data[0]

	for i := 1; i < len(data); i++ {
		if data[i] == current && count < 255 {
			count++
		} else {
			compressed = append(compressed, byte(count), current)
			count = 1
			current = data[i]
		}
	}
	compressed = append(compressed, byte(count), current)

	return compressed
}

func (uf *UDPFramework) Decompress(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	var decompressed []byte

	for i := 0; i < len(data); i += 2 {
		count := int(data[i])
		value := data[i+1]
		for j := 0; j < count; j++ {
			decompressed = append(decompressed, value)
		}
	}

	return decompressed
}

func (uf *UDPFramework) EncryptData(data []byte, key []byte) []byte {
	encrypted := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		encrypted[i] = data[i] ^ key[i%len(key)]
	}
	return encrypted
}


func (uf *UDPFramework) DecryptData(data []byte, key []byte) []byte {
	return uf.EncryptData(data, key)
}


func (uf *UDPFramework) SendBulkData(data []byte, packetSize int, destAddr *net.UDPAddr) error {
	totalPackets := (len(data) + packetSize - 1) / packetSize
	for i := 0; i < totalPackets; i++ {
		start := i * packetSize
		end := start + packetSize
		if end > len(data) {
			end = len(data)
		}

		packet := Packet{
			SequenceNumber: uint32(i),
			Priority:       0,
			Data:           data[start:end],
			Timestamp:      time.Now(),
		}

		err := uf.SendPacket(packet, destAddr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (uf *UDPFramework) ReceiveBulkData(expectedPackets int) ([]byte, error) {
	var bulkData []byte
	receivedPackets := 0

	for receivedPackets < expectedPackets {
		data, _, err := uf.ReceivePacket()
		if err != nil {
			return nil, err
		}

		if data != nil {
			bulkData = append(bulkData, data...)
			receivedPackets++
		}
	}

	return bulkData, nil
}