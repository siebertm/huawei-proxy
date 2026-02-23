# huawei-solar-proxy

A Modbus TCP caching proxy for Huawei SUN2000 solar inverters, designed to sit between [Home Assistant](https://www.home-assistant.io/) (using the [huawei_solar](https://github.com/wlcrs/huawei_solar) integration) and the inverter.

## The Problem

The Huawei SUN2000 inverter:

- Supports only **one Modbus TCP connection** at a time
- Needs **~500ms minimum gap** between read calls or it becomes unresponsive
- The HA integration sends many batch reads per 30s update cycle, each taking time

This means HA gets slow, sometimes unreliable responses, and no other tool can talk to the inverter while HA is connected.

## The Solution

```
┌──────────────┐         ┌──────────────────────┐         ┌───────────┐
│  HA instance │ ◄─────► │  huawei-solar-proxy   │ ◄─────► │  SUN2000  │
│  (huawei_    │  fast   │                      │  500ms   │  Inverter │
│   solar)     │  TCP    │  ┌──────────────┐    │  gaps    │  :502     │
│              │         │  │ register     │    │          │           │
│              │         │  │ cache        │    │          │           │
└──────────────┘         └──────────────────────┘         └───────────┘
```

The proxy:

1. **Continuously reads** all configured register groups from the inverter, respecting the 500ms inter-read pause
2. **Caches** all register values in memory
3. **Serves** Modbus TCP requests from HA instantly from cache
4. **Forwards writes** to the inverter (for config changes, forcible charge/discharge, etc.)
5. **Forwards unknown reads** directly to the inverter on cache miss (configurable)

HA can poll as aggressively as it wants — it always gets an instant response.

## Two-Tier Polling

Register groups are split into two tiers:

- **Fast** (every cycle, ~3.5s): Operational data — power, voltage, current, status, meter, battery SOC
- **Slow** (every 5 minutes): Device info, firmware versions, configuration, battery pack details

See [REGISTERS.md](REGISTERS.md) for the complete register map with all 511 known SUN2000 registers.

## Quick Start

### Docker (recommended)

```bash
# Create your config
cp config.example.yaml config.yaml
# Edit config.yaml — at minimum set inverter.host

# Build and run
docker build -t huawei-solar-proxy .
docker run -d \
  --name huawei-solar-proxy \
  -p 502:502 \
  -v $(pwd)/config.yaml:/etc/huawei-solar-proxy/config.yaml:ro \
  huawei-solar-proxy
```

### Docker Compose

```yaml
services:
  huawei-solar-proxy:
    build: .
    restart: unless-stopped
    ports:
      - "502:502"
    volumes:
      - ./config.yaml:/etc/huawei-solar-proxy/config.yaml:ro
```

### Native

```bash
go build -o huawei-solar-proxy .
./huawei-solar-proxy -config config.yaml
```

## Home Assistant Configuration

Point the huawei_solar integration at the proxy instead of the inverter:

| Setting | Value |
|---------|-------|
| Host | IP of the machine running the proxy |
| Port | 502 (or whatever you set in `server.listen`) |

Everything else stays the same — the proxy is fully transparent.

## Configuration

Copy `config.example.yaml` to `config.yaml` and adjust:

```yaml
inverter:
  host: "192.168.200.1"   # Your inverter's IP
  port: 502
  unit_id: 1

server:
  listen: ":502"           # Where HA connects

polling:
  read_pause_ms: 500       # Min gap between inverter reads
  slow_interval_s: 300     # Slow group interval (5 min)

forward_unknown_reads: true
log_level: "info"          # debug, info, warn, error

register_groups:
  - name: inverter_state
    address: 32000
    count: 11
    poll: fast
  # ... see config.example.yaml for all groups
```

### Tuning for your setup

- **No battery?** Remove `storage_unit_1`, `storage_aggregated`, `storage_unit_2`, `storage_versions`, `battery_soh`, `battery_packs_*`, `storage_config_*`, and `capacity_control` groups
- **No power meter?** Remove the `meter` group
- **Fewer PV strings?** Reduce `pv_strings` count (2 registers per string)
- **Want faster updates?** Lower `read_pause_ms` (not recommended below 300ms)
- **Want debug output?** Set `log_level: "debug"`

## How It Works

### Startup

1. Loads YAML config
2. Connects to inverter via Modbus TCP
3. Performs initial scan of all register groups (populates cache)
4. Starts the reader loop goroutine
5. Starts the Modbus TCP server

### Reader Loop

Runs continuously:
- Reads all **fast** groups in sequence, with `read_pause_ms` between each
- Every `slow_interval_s`, also reads **slow** groups
- Failed reads log a warning and continue (self-healing)

### Server

Standard Modbus TCP server handling:
- **0x03** (Read Holding Registers): Serve from cache
- **0x04** (Read Input Registers): Serve from cache
- **0x06** (Write Single Register): Forward to inverter, update cache
- **0x10** (Write Multiple Registers): Forward to inverter, update cache

### Cache Misses

When `forward_unknown_reads: true` (default), any register not in the cache is read directly from the inverter (respecting the pause). This handles HA's device detection probing at startup for registers not covered by your configured groups.

## License

MIT
