package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/goburrow/modbus"
)

// InverterClient wraps a Modbus TCP client with mutex-based serialization
// and enforced inter-read pause to avoid overwhelming the inverter.
type InverterClient struct {
	mu       sync.Mutex
	client   modbus.Client
	handler  *modbus.TCPClientHandler
	pauseMs  int
	lastRead time.Time
}

func NewInverterClient(cfg *Config) (*InverterClient, error) {
	handler := modbus.NewTCPClientHandler(cfg.InverterAddr())
	handler.Timeout = cfg.InverterTimeout()
	handler.SlaveId = cfg.Inverter.UnitID
	handler.IdleTimeout = 0 // keep connection alive

	if err := handler.Connect(); err != nil {
		return nil, fmt.Errorf("connecting to inverter at %s: %w", cfg.InverterAddr(), err)
	}

	client := modbus.NewClient(handler)

	return &InverterClient{
		client:  client,
		handler: handler,
		pauseMs: cfg.Polling.ReadPauseMs,
	}, nil
}

func (ic *InverterClient) Close() {
	ic.handler.Close()
}

// enforcePause sleeps if needed to maintain the minimum gap between reads.
func (ic *InverterClient) enforcePause() {
	if ic.lastRead.IsZero() {
		return
	}
	elapsed := time.Since(ic.lastRead)
	gap := time.Duration(ic.pauseMs)*time.Millisecond - elapsed
	if gap > 0 {
		time.Sleep(gap)
	}
}

// ReadRegisters reads holding registers from the inverter.
// Thread-safe; enforces the configured inter-read pause.
func (ic *InverterClient) ReadRegisters(address, count uint16) ([]byte, error) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	ic.enforcePause()
	result, err := ic.client.ReadHoldingRegisters(address, count)
	ic.lastRead = time.Now()
	return result, err
}

// WriteSingleRegister writes a single holding register on the inverter.
func (ic *InverterClient) WriteSingleRegister(address, value uint16) ([]byte, error) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	ic.enforcePause()
	result, err := ic.client.WriteSingleRegister(address, value)
	ic.lastRead = time.Now()
	return result, err
}

// WriteMultipleRegisters writes multiple holding registers on the inverter.
func (ic *InverterClient) WriteMultipleRegisters(address, count uint16, data []byte) ([]byte, error) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	ic.enforcePause()
	result, err := ic.client.WriteMultipleRegisters(address, count, data)
	ic.lastRead = time.Now()
	return result, err
}
