# CLAUDE.md

Project context and instructions for AI assistants working on this codebase.

## Project Overview

**huawei-solar-proxy** is a Modbus TCP caching proxy for Huawei SUN2000 solar inverters, written in Go. It sits between Home Assistant (using the `huawei_solar` integration) and the inverter to provide instant cached responses.

## Architecture

- **Single Go package** (`package main`) with focused files:
  - `main.go` — entry point, wiring, signal handling
  - `config.go` — YAML config parsing and validation
  - `cache.go` — thread-safe `map[uint16]uint16` register cache
  - `client.go` — Modbus TCP client with mutex serialization and inter-read pause
  - `reader.go` — continuous polling loop (fast/slow tiers)
  - `server.go` — Modbus TCP server (serves from cache, forwards writes)

- **Key dependencies:**
  - `github.com/goburrow/modbus` — Modbus TCP client
  - `gopkg.in/yaml.v3` — YAML config
  - `log/slog` — structured logging (stdlib)

## Building

```bash
go build -o huawei-solar-proxy .
```

## Testing

No inverter needed for compilation checks:
```bash
go vet ./...
go build ./...
```

Integration testing requires a real inverter or a Modbus TCP simulator.

## Key Design Decisions

- **500ms inter-read pause**: The inverter becomes unresponsive without gaps between reads. This is enforced in `client.go` via a mutex + sleep.
- **Two-tier polling**: Fast groups (operational data) every cycle (~3.5s), slow groups (config/device info) every 5 minutes.
- **Forward unknown reads**: Cache misses are forwarded to the inverter by default, so HA's device detection works without pre-configuring every register.
- **Write forwarding**: Writes go through the same mutex as reads, ensuring the pause is respected.
- **All registers are holding registers** (function code 0x03). The HA integration never uses input registers (0x04), but the proxy handles both identically for robustness.

## Register Map

See `REGISTERS.md` for the complete map of all 511 known SUN2000 registers with addresses, lengths, and groupings. This was extracted from the `huawei_solar` Python library v2.5.0.

## Config

- `config.example.yaml` has all known register groups with comments
- Users copy to `config.yaml` and set their inverter IP
- Groups for unused hardware (battery, meter) should be removed

## Code Style

- Standard Go formatting (`gofmt`)
- `slog` for all logging (JSON to stdout)
- No frameworks — stdlib + minimal dependencies
- Keep the single-package flat structure
