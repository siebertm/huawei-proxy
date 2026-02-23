package main

import (
	"sync"
	"time"
)

// RegisterCache is a thread-safe cache of Modbus register values.
// Keys are register addresses, values are raw uint16 register values.
type RegisterCache struct {
	mu         sync.RWMutex
	registers  map[uint16]uint16
	timestamps map[uint16]time.Time
}

func NewRegisterCache() *RegisterCache {
	return &RegisterCache{
		registers:  make(map[uint16]uint16),
		timestamps: make(map[uint16]time.Time),
	}
}

// Set stores register values starting at the given address.
func (c *RegisterCache) Set(address uint16, values []uint16) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for i, v := range values {
		addr := address + uint16(i)
		c.registers[addr] = v
		c.timestamps[addr] = now
	}
}

// SetFromBytes stores register values from raw Modbus response bytes (big-endian).
func (c *RegisterCache) SetFromBytes(address uint16, data []byte) {
	values := make([]uint16, len(data)/2)
	for i := range values {
		values[i] = uint16(data[i*2])<<8 | uint16(data[i*2+1])
	}
	c.Set(address, values)
}

// Get retrieves register values for the given address range.
// Returns nil if any register in the range is not cached.
func (c *RegisterCache) Get(address, count uint16) []uint16 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]uint16, count)
	for i := uint16(0); i < count; i++ {
		v, ok := c.registers[address+i]
		if !ok {
			return nil
		}
		result[i] = v
	}
	return result
}

// GetBytes retrieves register values as raw bytes for a Modbus response.
// Returns nil if any register in the range is not cached.
func (c *RegisterCache) GetBytes(address, count uint16) []byte {
	values := c.Get(address, count)
	if values == nil {
		return nil
	}

	data := make([]byte, len(values)*2)
	for i, v := range values {
		data[i*2] = byte(v >> 8)
		data[i*2+1] = byte(v)
	}
	return data
}

// Size returns the number of cached registers.
func (c *RegisterCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.registers)
}
