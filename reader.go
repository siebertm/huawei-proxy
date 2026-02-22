package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Reader continuously polls register groups from the inverter
// and updates the cache. Fast groups are read every cycle, slow
// groups only when their interval has elapsed.
type Reader struct {
	cfg        *Config
	client     *InverterClient
	cache      *RegisterCache
	fastGroups []RegisterGroup
	slowGroups []RegisterGroup
}

func NewReader(cfg *Config, client *InverterClient, cache *RegisterCache) *Reader {
	r := &Reader{
		cfg:    cfg,
		client: client,
		cache:  cache,
	}

	for _, g := range cfg.RegisterGroups {
		switch g.Poll {
		case "fast":
			r.fastGroups = append(r.fastGroups, g)
		case "slow":
			r.slowGroups = append(r.slowGroups, g)
		}
	}

	return r
}

func (r *Reader) readGroup(ctx context.Context, g RegisterGroup) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	data, err := r.client.ReadRegisters(g.Address, g.Count)
	if err != nil {
		return fmt.Errorf("reading %s (addr=%d, count=%d): %w", g.Name, g.Address, g.Count, err)
	}

	r.cache.SetFromBytes(g.Address, data)

	slog.Debug("read register group",
		"name", g.Name,
		"address", g.Address,
		"count", g.Count,
	)

	return nil
}

// InitialScan reads all groups (fast and slow) once to populate the cache
// before the server starts accepting connections.
func (r *Reader) InitialScan(ctx context.Context) error {
	allGroups := make([]RegisterGroup, 0, len(r.fastGroups)+len(r.slowGroups))
	allGroups = append(allGroups, r.fastGroups...)
	allGroups = append(allGroups, r.slowGroups...)

	for _, g := range allGroups {
		if err := r.readGroup(ctx, g); err != nil {
			// Log warning but continue — some groups may fail (e.g., battery
			// registers when no battery is connected).
			slog.Warn("initial scan: failed to read group",
				"name", g.Name,
				"address", g.Address,
				"error", err,
			)
		}
	}

	slog.Info("initial scan complete", "cached_registers", r.cache.Size())
	return nil
}

// Run starts the continuous reader loop. It reads fast groups every cycle
// and slow groups when their interval has elapsed.
func (r *Reader) Run(ctx context.Context) {
	slog.Info("reader loop started",
		"fast_groups", len(r.fastGroups),
		"slow_groups", len(r.slowGroups),
		"slow_interval", r.cfg.SlowInterval(),
	)

	lastSlowPoll := time.Now() // slow groups were just read in InitialScan

	for {
		if ctx.Err() != nil {
			slog.Info("reader loop stopped")
			return
		}

		cycleStart := time.Now()

		// Read fast groups
		for _, g := range r.fastGroups {
			if ctx.Err() != nil {
				return
			}
			if err := r.readGroup(ctx, g); err != nil {
				slog.Warn("failed to read register group",
					"name", g.Name,
					"error", err,
				)
			}
		}

		// Check if slow groups are due
		if time.Since(lastSlowPoll) >= r.cfg.SlowInterval() {
			slog.Debug("reading slow groups")
			for _, g := range r.slowGroups {
				if ctx.Err() != nil {
					return
				}
				if err := r.readGroup(ctx, g); err != nil {
					slog.Warn("failed to read register group",
						"name", g.Name,
						"error", err,
					)
				}
			}
			lastSlowPoll = time.Now()
		}

		slog.Debug("fast cycle complete",
			"duration", time.Since(cycleStart).Round(time.Millisecond),
			"cached_registers", r.cache.Size(),
		)
	}
}
