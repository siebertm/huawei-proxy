package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Inverter            InverterConfig  `yaml:"inverter"`
	Server              ServerConfig    `yaml:"server"`
	Web                 WebConfig       `yaml:"web"`
	Polling             PollingConfig   `yaml:"polling"`
	ForwardUnknownReads bool            `yaml:"forward_unknown_reads"`
	RegisterGroups      []RegisterGroup `yaml:"register_groups"`
	LogLevel            string          `yaml:"log_level"`
	CachePath           string          `yaml:"cache_path"`
	CacheTTLH           int             `yaml:"cache_ttl_h"`
}

type InverterConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	UnitIDs   []byte `yaml:"unit_ids"`
	TimeoutMs int    `yaml:"timeout_ms"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type WebConfig struct {
	Listen string `yaml:"listen"`
}

type PollingConfig struct {
	ReadPauseMs   int `yaml:"read_pause_ms"`
	SlowIntervalS int `yaml:"slow_interval_s"`
}

type RegisterGroup struct {
	Name    string `yaml:"name"`
	Address uint16 `yaml:"address"`
	Count   uint16 `yaml:"count"`
	Poll    string `yaml:"poll"` // "fast" or "slow"
}

func (c *Config) ReadPause() time.Duration {
	return time.Duration(c.Polling.ReadPauseMs) * time.Millisecond
}

func (c *Config) SlowInterval() time.Duration {
	return time.Duration(c.Polling.SlowIntervalS) * time.Second
}

func (c *Config) InverterTimeout() time.Duration {
	return time.Duration(c.Inverter.TimeoutMs) * time.Millisecond
}

func (c *Config) CacheTTL() time.Duration {
	return time.Duration(c.CacheTTLH) * time.Hour
}

func (c *Config) InverterAddr() string {
	return fmt.Sprintf("%s:%d", c.Inverter.Host, c.Inverter.Port)
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := &Config{
		Inverter: InverterConfig{
			Port:      502,
			UnitIDs:   []byte{1},
			TimeoutMs: 5000,
		},
		Server: ServerConfig{
			Listen: ":502",
		},
		Web: WebConfig{
			Listen: ":8080",
		},
		Polling: PollingConfig{
			ReadPauseMs:   500,
			SlowIntervalS: 300,
		},
		ForwardUnknownReads: true,
		LogLevel:            "info",
		CachePath:           "cache.db",
		CacheTTLH:           2,
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Inverter.Host == "" {
		return nil, fmt.Errorf("inverter.host is required")
	}

	if len(cfg.RegisterGroups) == 0 {
		return nil, fmt.Errorf("at least one register_group is required")
	}

	if len(cfg.Inverter.UnitIDs) == 0 {
		return nil, fmt.Errorf("inverter.unit_ids is required (e.g. [1])")
	}

	// Validate no duplicate unit IDs
	seen := make(map[byte]bool, len(cfg.Inverter.UnitIDs))
	for _, uid := range cfg.Inverter.UnitIDs {
		if seen[uid] {
			return nil, fmt.Errorf("inverter.unit_ids: duplicate unit ID %d", uid)
		}
		seen[uid] = true
	}

	for i, g := range cfg.RegisterGroups {
		if g.Count == 0 {
			return nil, fmt.Errorf("register_groups[%d] (%s): count must be > 0", i, g.Name)
		}
		if g.Count > 125 {
			return nil, fmt.Errorf("register_groups[%d] (%s): count must be <= 125 (modbus limit)", i, g.Name)
		}
		if g.Poll != "fast" && g.Poll != "slow" {
			return nil, fmt.Errorf("register_groups[%d] (%s): poll must be 'fast' or 'slow'", i, g.Name)
		}
	}

	return cfg, nil
}
