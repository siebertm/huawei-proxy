# Huawei Solar Proxy

A Modbus TCP caching proxy for Huawei SUN2000 solar inverters. It sits between Home Assistant and your inverter, continuously polling registers and serving instant cached responses.

## Why use this?

The Huawei SUN2000 inverter requires a 500ms pause between Modbus reads and only supports a single TCP connection. This proxy:

- **Eliminates HA timeouts** by serving cached data instantly
- **Handles connection serialization** so HA never gets "device busy" errors
- **Polls registers continuously** with configurable fast/slow tiers

## Configuration

### Required

- **inverter_host**: IP address of your Huawei SUN2000 inverter (e.g., `192.168.200.1`)

### Optional

- **inverter_port**: Modbus TCP port on the inverter (default: `502`)
- **unit_ids**: Comma-separated Modbus unit/slave IDs to poll (default: `1`). Use `1,16` for two inverters on the same connection.
- **modbus_listen_port**: Port the proxy listens on for Modbus TCP connections from HA (default: `502`)
- **web_listen_port**: Port for the web status UI (default: `8080`)
- **read_pause_ms**: Minimum milliseconds between Modbus reads (default: `500`). The inverter needs this gap to stay responsive.
- **slow_interval_s**: Seconds between slow-tier polls for device info and configuration registers (default: `300`)
- **forward_unknown_reads**: Forward cache misses to the inverter (default: `true`). Required for HA's initial device detection.
- **log_level**: Logging verbosity: `debug`, `info`, `warn`, or `error` (default: `info`)
- **cache_ttl_h**: Hours before stale cached registers are purged (default: `2`)

### Hardware toggles

- **has_battery**: Enable battery/storage register groups (default: `false`). Turn on if you have a LUNA2000 battery.
- **has_power_meter**: Enable power meter register groups (default: `false`). Turn on if you have an external power meter (DTSU666-H or similar).

## Home Assistant setup

1. Install the add-on and configure `inverter_host`
2. The add-on exposes port `502` (Modbus TCP) and port `8080` (Web UI) by default. You can remap these in the add-on's **Network** configuration panel if they conflict with other services.
3. In the **huawei_solar** integration, point the Modbus connection to your HA host IP on the mapped Modbus port (default: `502`)
4. The web UI is available via the mapped web port (default: `8080`) — click "Open Web UI" in the add-on panel

## Troubleshooting

- **HA can't connect**: Check that the Modbus port mapping is correct in the add-on's Network settings. Remap to another host port (e.g., `1502`) if port `502` conflicts.
- **Stale data**: Reduce `cache_ttl_h` or check the web UI to see when registers were last updated.
- **Inverter unresponsive**: Increase `read_pause_ms` (try `750` or `1000`).
- **Missing battery/meter entities**: Enable `has_battery` or `has_power_meter` in the add-on configuration.
