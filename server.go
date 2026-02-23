package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
)

// Server is a Modbus TCP server that serves register values from cache
// and forwards writes to the inverter.
type Server struct {
	cfg            *Config
	cache          *RegisterCache
	inverterClient *InverterClient
}

func NewServer(cfg *Config, cache *RegisterCache, inverterClient *InverterClient) *Server {
	return &Server{
		cfg:            cfg,
		cache:          cache,
		inverterClient: inverterClient,
	}
}

// ListenAndServe starts accepting Modbus TCP connections.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.cfg.Server.Listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.cfg.Server.Listen, err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	slog.Info("modbus TCP server listening", "address", s.cfg.Server.Listen)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("accept error", "error", err)
			continue
		}

		slog.Info("client connected", "remote", conn.RemoteAddr())
		go s.handleConnection(ctx, conn)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	defer slog.Info("client disconnected", "remote", conn.RemoteAddr())

	for {
		if ctx.Err() != nil {
			return
		}

		// Read MBAP header: 7 bytes
		// [0:2] Transaction ID
		// [2:4] Protocol ID (must be 0)
		// [4:6] Length (remaining bytes including unit ID)
		// [6]   Unit ID
		header := make([]byte, 7)
		if _, err := io.ReadFull(conn, header); err != nil {
			if err != io.EOF {
				slog.Debug("read header error", "error", err)
			}
			return
		}

		txID := binary.BigEndian.Uint16(header[0:2])
		protocolID := binary.BigEndian.Uint16(header[2:4])
		length := binary.BigEndian.Uint16(header[4:6])
		unitID := header[6]

		if protocolID != 0 {
			slog.Warn("invalid protocol ID, closing connection", "protocol_id", protocolID)
			return
		}

		if length < 2 || length > 256 {
			slog.Warn("invalid MBAP length, closing connection", "length", length)
			return
		}

		// Read PDU: length-1 bytes (length includes the unit ID we already read)
		pdu := make([]byte, length-1)
		if _, err := io.ReadFull(conn, pdu); err != nil {
			slog.Debug("read PDU error", "error", err)
			return
		}

		fc := pdu[0]

		switch fc {
		case 0x03, 0x04: // Read Holding Registers / Read Input Registers
			s.handleReadRegisters(conn, txID, unitID, fc, pdu)
		case 0x06: // Write Single Register
			s.handleWriteSingleRegister(conn, txID, unitID, pdu)
		case 0x10: // Write Multiple Registers
			s.handleWriteMultipleRegisters(conn, txID, unitID, pdu)
		default:
			slog.Warn("unsupported function code", "fc", fc)
			sendException(conn, txID, unitID, fc, 0x01) // Illegal Function
		}
	}
}

func (s *Server) handleReadRegisters(conn net.Conn, txID uint16, unitID byte, fc byte, pdu []byte) {
	if len(pdu) < 5 {
		sendException(conn, txID, unitID, fc, 0x03)
		return
	}

	address := binary.BigEndian.Uint16(pdu[1:3])
	count := binary.BigEndian.Uint16(pdu[3:5])

	if count == 0 || count > 125 {
		sendException(conn, txID, unitID, fc, 0x03) // Illegal Data Value
		return
	}

	// Try cache first
	data := s.cache.GetBytes(unitID, address, count)
	if data != nil {
		slog.Debug("cache hit", "unit_id", unitID, "address", address, "count", count)
		sendReadResponse(conn, txID, unitID, fc, data)
		return
	}

	// Cache miss
	if s.cfg.ForwardUnknownReads {
		slog.Info("cache miss, forwarding to inverter", "unit_id", unitID, "address", address, "count", count)
		result, err := s.inverterClient.ReadRegisters(unitID, address, count)
		if err != nil {
			slog.Warn("forward read failed", "unit_id", unitID, "address", address, "count", count, "error", err)
			sendException(conn, txID, unitID, fc, 0x04) // Server Device Failure
			return
		}
		// Cache the forwarded result
		s.cache.SetFromBytes(unitID, address, result)
		sendReadResponse(conn, txID, unitID, fc, result)
		return
	}

	slog.Debug("cache miss, forwarding disabled", "address", address, "count", count)
	sendException(conn, txID, unitID, fc, 0x02) // Illegal Data Address
}

func (s *Server) handleWriteSingleRegister(conn net.Conn, txID uint16, unitID byte, pdu []byte) {
	if len(pdu) < 5 {
		sendException(conn, txID, unitID, 0x06, 0x03)
		return
	}

	address := binary.BigEndian.Uint16(pdu[1:3])
	value := binary.BigEndian.Uint16(pdu[3:5])

	slog.Info("forwarding write single register", "unit_id", unitID, "address", address, "value", value)

	_, err := s.inverterClient.WriteSingleRegister(unitID, address, value)
	if err != nil {
		slog.Warn("write single register failed", "unit_id", unitID, "address", address, "error", err)
		sendException(conn, txID, unitID, 0x06, 0x04) // Server Device Failure
		return
	}

	// Update cache
	s.cache.Set(unitID, address, []uint16{value})

	// Response echoes back address + value
	sendWriteSingleResponse(conn, txID, unitID, address, value)
}

func (s *Server) handleWriteMultipleRegisters(conn net.Conn, txID uint16, unitID byte, pdu []byte) {
	if len(pdu) < 6 {
		sendException(conn, txID, unitID, 0x10, 0x03)
		return
	}

	address := binary.BigEndian.Uint16(pdu[1:3])
	count := binary.BigEndian.Uint16(pdu[3:5])
	byteCount := pdu[5]

	if int(byteCount) != int(count)*2 || len(pdu) < 6+int(byteCount) {
		sendException(conn, txID, unitID, 0x10, 0x03)
		return
	}

	data := pdu[6 : 6+byteCount]

	slog.Info("forwarding write multiple registers", "unit_id", unitID, "address", address, "count", count)

	_, err := s.inverterClient.WriteMultipleRegisters(unitID, address, count, data)
	if err != nil {
		slog.Warn("write multiple registers failed", "unit_id", unitID, "address", address, "error", err)
		sendException(conn, txID, unitID, 0x10, 0x04) // Server Device Failure
		return
	}

	// Update cache
	s.cache.SetFromBytes(unitID, address, data)

	sendWriteMultipleResponse(conn, txID, unitID, address, count)
}

// --- Modbus TCP response helpers ---

func sendReadResponse(conn net.Conn, txID uint16, unitID byte, fc byte, data []byte) {
	byteCount := byte(len(data))
	mbapLen := uint16(3 + len(data)) // unitID + fc + byteCount + data

	resp := make([]byte, 0, 9+len(data))
	resp = binary.BigEndian.AppendUint16(resp, txID)
	resp = binary.BigEndian.AppendUint16(resp, 0) // protocol ID
	resp = binary.BigEndian.AppendUint16(resp, mbapLen)
	resp = append(resp, unitID, fc, byteCount)
	resp = append(resp, data...)

	conn.Write(resp)
}

func sendException(conn net.Conn, txID uint16, unitID byte, fc byte, exCode byte) {
	resp := make([]byte, 0, 9)
	resp = binary.BigEndian.AppendUint16(resp, txID)
	resp = binary.BigEndian.AppendUint16(resp, 0) // protocol ID
	resp = binary.BigEndian.AppendUint16(resp, 3) // length: unitID + errorFC + exCode
	resp = append(resp, unitID, fc|0x80, exCode)

	conn.Write(resp)
}

func sendWriteSingleResponse(conn net.Conn, txID uint16, unitID byte, address, value uint16) {
	resp := make([]byte, 0, 12)
	resp = binary.BigEndian.AppendUint16(resp, txID)
	resp = binary.BigEndian.AppendUint16(resp, 0) // protocol ID
	resp = binary.BigEndian.AppendUint16(resp, 6) // length: unitID + fc + addr(2) + value(2)
	resp = append(resp, unitID, 0x06)
	resp = binary.BigEndian.AppendUint16(resp, address)
	resp = binary.BigEndian.AppendUint16(resp, value)

	conn.Write(resp)
}

func sendWriteMultipleResponse(conn net.Conn, txID uint16, unitID byte, address, count uint16) {
	resp := make([]byte, 0, 12)
	resp = binary.BigEndian.AppendUint16(resp, txID)
	resp = binary.BigEndian.AppendUint16(resp, 0) // protocol ID
	resp = binary.BigEndian.AppendUint16(resp, 6) // length: unitID + fc + addr(2) + count(2)
	resp = append(resp, unitID, 0x10)
	resp = binary.BigEndian.AppendUint16(resp, address)
	resp = binary.BigEndian.AppendUint16(resp, count)

	conn.Write(resp)
}
