package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

const (
	maxRetries        = 3
	deviceBusyBackoff = 3 * time.Second
	reconnectBackoff  = 2 * time.Second
	initialScanPasses = 3
)

// ReaderStats contains runtime statistics from the polling loop.
type ReaderStats struct {
	CycleCount    int64
	LastCycleTime time.Time
	LastCycleDur  time.Duration
}

// Reader continuously polls register groups from the inverter
// and updates the cache. Fast groups are read every cycle, slow
// groups only when their interval has elapsed.
type Reader struct {
	cfg        *Config
	client     *InverterClient
	cache      *RegisterCache
	fastGroups []RegisterGroup
	slowGroups []RegisterGroup

	cycleCount    atomic.Int64
	lastCycleTime atomic.Value // time.Time
	lastCycleDur  atomic.Int64 // nanoseconds
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

func (r *Reader) readGroup(ctx context.Context, unitID byte, g RegisterGroup) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		data, err := r.client.ReadRegisters(unitID, g.Address, g.Count)
		if err == nil {
			r.cache.SetFromBytes(unitID, g.Address, data)
			slog.Debug("read register group",
				"unit_id", unitID,
				"name", g.Name,
				"address", g.Address,
				"count", g.Count,
			)
			return nil
		}

		lastErr = err

		if IsDeviceBusy(err) || IsTimeout(err) {
			if attempt < maxRetries {
				slog.Warn("retryable error, retrying after backoff",
					"name", g.Name,
					"attempt", attempt,
					"backoff", deviceBusyBackoff,
					"error", err,
				)
				if err := sleepCtx(ctx, deviceBusyBackoff); err != nil {
					return err
				}
				continue
			}
		} else if IsTransactionMismatch(err) {
			if attempt < maxRetries {
				slog.Warn("transaction ID mismatch, reconnecting",
					"name", g.Name,
					"attempt", attempt,
				)
				r.client.Reconnect()
				if err := sleepCtx(ctx, reconnectBackoff); err != nil {
					return err
				}
				continue
			}
		} else {
			// Non-retryable error — return immediately
			break
		}
	}

	return fmt.Errorf("reading %s (addr=%d, count=%d): %w", g.Name, g.Address, g.Count, lastErr)
}

// sleepCtx sleeps for the given duration, returning early if the context
// is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// unitGroup pairs a unit ID with a register group for initial scan tracking.
type unitGroup struct {
	unitID byte
	group  RegisterGroup
}

// InitialScan reads all groups (fast and slow) for all unit IDs to populate
// the cache before the server starts accepting connections. Groups that fail
// are retried in additional passes, which handles inverter warm-up after TCP connect.
func (r *Reader) InitialScan(ctx context.Context) error {
	allGroups := make([]RegisterGroup, 0, len(r.fastGroups)+len(r.slowGroups))
	allGroups = append(allGroups, r.fastGroups...)
	allGroups = append(allGroups, r.slowGroups...)

	var pending []unitGroup
	for _, g := range allGroups {
		for _, uid := range r.cfg.Inverter.UnitIDs {
			pending = append(pending, unitGroup{unitID: uid, group: g})
		}
	}

	for pass := 1; pass <= initialScanPasses; pass++ {
		var failed []unitGroup
		for _, ug := range pending {
			if err := r.readGroup(ctx, ug.unitID, ug.group); err != nil {
				slog.Warn("initial scan: failed to read group",
					"unit_id", ug.unitID,
					"name", ug.group.Name,
					"address", ug.group.Address,
					"pass", pass,
					"error", err,
				)
				failed = append(failed, ug)
			}
		}
		if len(failed) == 0 {
			break
		}
		pending = failed
		if pass < initialScanPasses {
			slog.Info("initial scan: retrying failed groups",
				"count", len(failed),
				"pass", pass+1,
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

		// Read each fast group across all unit IDs before moving to the next group,
		// so that the same register group is temporally coherent across units.
		for _, g := range r.fastGroups {
			for _, uid := range r.cfg.Inverter.UnitIDs {
				if ctx.Err() != nil {
					return
				}
				if err := r.readGroup(ctx, uid, g); err != nil {
					slog.Warn("failed to read register group",
						"unit_id", uid,
						"name", g.Name,
						"error", err,
					)
				}
			}
		}

		// Check if slow groups are due
		if time.Since(lastSlowPoll) >= r.cfg.SlowInterval() {
			slog.Debug("reading slow groups")
			for _, g := range r.slowGroups {
				for _, uid := range r.cfg.Inverter.UnitIDs {
					if ctx.Err() != nil {
						return
					}
					if err := r.readGroup(ctx, uid, g); err != nil {
						slog.Warn("failed to read register group",
							"unit_id", uid,
							"name", g.Name,
							"error", err,
						)
					}
				}
			}
			lastSlowPoll = time.Now()
		}

		cycleDur := time.Since(cycleStart)
		r.cycleCount.Add(1)
		r.lastCycleTime.Store(time.Now())
		r.lastCycleDur.Store(int64(cycleDur))

		slog.Debug("fast cycle complete",
			"duration", cycleDur.Round(time.Millisecond).String(),
			"cached_registers", r.cache.Size(),
		)
	}
}

// Stats returns current runtime statistics.
func (r *Reader) Stats() ReaderStats {
	var lastTime time.Time
	if v := r.lastCycleTime.Load(); v != nil {
		lastTime = v.(time.Time)
	}
	return ReaderStats{
		CycleCount:    r.cycleCount.Load(),
		LastCycleTime: lastTime,
		LastCycleDur:  time.Duration(r.lastCycleDur.Load()),
	}
}

// FastGroupCount returns the number of fast polling groups.
func (r *Reader) FastGroupCount() int { return len(r.fastGroups) }

// SlowGroupCount returns the number of slow polling groups.
func (r *Reader) SlowGroupCount() int { return len(r.slowGroups) }
