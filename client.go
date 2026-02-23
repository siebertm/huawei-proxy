package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
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

// Reconnect closes the TCP connection and resets state so the next
// operation triggers an automatic reconnect by the modbus library.
func (ic *InverterClient) Reconnect() {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	ic.handler.Close()
	ic.lastRead = time.Time{}
}

// IsDeviceBusy returns true if the error is a Modbus exception 0x06
// (server device busy).
func IsDeviceBusy(err error) bool {
	var mbErr *modbus.ModbusError
	if errors.As(err, &mbErr) {
		return mbErr.ExceptionCode == modbus.ExceptionCodeServerDeviceBusy
	}
	return false
}

// IsTransactionMismatch returns true if the error indicates a Modbus TCP
// transaction ID mismatch. The goburrow library reports this as a plain
// fmt.Errorf, so we match on the error string.
func IsTransactionMismatch(err error) bool {
	return err != nil && strings.Contains(err.Error(), "transaction id")
}

// IsTimeout returns true if the error is a network timeout (i/o timeout).
func IsTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return errors.Is(err, os.ErrDeadlineExceeded)
}
