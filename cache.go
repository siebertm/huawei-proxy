package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// CachedRegister represents a single register value with its update timestamp.
type CachedRegister struct {
	UnitID    byte
	Address   uint16
	Value     uint16
	UpdatedAt time.Time
}

// RegisterCache is a SQLite-backed cache of Modbus register values.
// Keys are register addresses, values are raw uint16 register values.
type RegisterCache struct {
	db         *sql.DB
	stmtUpsert *sql.Stmt
	stmtGet    *sql.Stmt
	stmtCount  *sql.Stmt
	stmtAll    *sql.Stmt
}

func NewRegisterCache(dbPath string) (*RegisterCache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// Single connection — SQLite doesn't benefit from a pool and this
	// avoids "database is locked" under concurrent access.
	db.SetMaxOpenConns(1)

	// Performance pragmas — this is a cache, so durability is not critical.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec %s: %w", pragma, err)
		}
	}

	// Migrate from old schema (no unit_id column) by dropping and recreating.
	// This is safe — it's a cache, InitialScan will repopulate.
	var hasUnitID bool
	rows, err := db.Query(`PRAGMA table_info(registers)`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull int
			var dfltValue *string
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err == nil {
				if name == "unit_id" {
					hasUnitID = true
				}
			}
		}
		rows.Close()
	}

	if !hasUnitID {
		// Drop old table if it exists (no unit_id column)
		db.Exec(`DROP TABLE IF EXISTS registers`)
		slog.Info("cache: migrated schema (dropped old table without unit_id)")
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS registers (
			unit_id    INTEGER NOT NULL,
			address    INTEGER NOT NULL,
			value      INTEGER NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (unit_id, address)
		)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	stmtUpsert, err := db.Prepare(`INSERT INTO registers (unit_id, address, value, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(unit_id, address) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare upsert: %w", err)
	}

	stmtGet, err := db.Prepare(`SELECT address, value FROM registers WHERE unit_id = ? AND address >= ? AND address < ? ORDER BY address`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare get: %w", err)
	}

	stmtCount, err := db.Prepare(`SELECT COUNT(*) FROM registers`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare count: %w", err)
	}

	stmtAll, err := db.Prepare(`SELECT unit_id, address, value, updated_at FROM registers ORDER BY unit_id, address`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare all: %w", err)
	}

	return &RegisterCache{
		db:         db,
		stmtUpsert: stmtUpsert,
		stmtGet:    stmtGet,
		stmtCount:  stmtCount,
		stmtAll:    stmtAll,
	}, nil
}

// Close releases database resources.
func (c *RegisterCache) Close() error {
	c.stmtUpsert.Close()
	c.stmtGet.Close()
	c.stmtCount.Close()
	c.stmtAll.Close()
	return c.db.Close()
}

// All returns all cached registers ordered by unit_id then address.
func (c *RegisterCache) All() []CachedRegister {
	rows, err := c.stmtAll.Query()
	if err != nil {
		slog.Warn("cache all: query", "error", err)
		return nil
	}
	defer rows.Close()

	var result []CachedRegister
	for rows.Next() {
		var unitID, addr, val int
		var updatedAt string
		if err := rows.Scan(&unitID, &addr, &val, &updatedAt); err != nil {
			slog.Warn("cache all: scan", "error", err)
			return nil
		}
		t, _ := time.Parse(time.RFC3339Nano, updatedAt)
		result = append(result, CachedRegister{
			UnitID:    byte(unitID),
			Address:   uint16(addr),
			Value:     uint16(val),
			UpdatedAt: t,
		})
	}
	return result
}

// Set stores register values starting at the given address for a unit ID.
func (c *RegisterCache) Set(unitID byte, address uint16, values []uint16) {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := c.db.Begin()
	if err != nil {
		slog.Warn("cache set: begin tx", "error", err)
		return
	}

	stmt := tx.Stmt(c.stmtUpsert)
	for i, v := range values {
		addr := int(address) + i
		if _, err := stmt.Exec(int(unitID), addr, int(v), now); err != nil {
			tx.Rollback()
			slog.Warn("cache set: exec", "address", addr, "error", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Warn("cache set: commit", "error", err)
	}
}

// SetFromBytes stores register values from raw Modbus response bytes (big-endian).
func (c *RegisterCache) SetFromBytes(unitID byte, address uint16, data []byte) {
	values := make([]uint16, len(data)/2)
	for i := range values {
		values[i] = uint16(data[i*2])<<8 | uint16(data[i*2+1])
	}
	c.Set(unitID, address, values)
}

// Get retrieves register values for the given unit ID and address range.
// Returns nil if any register in the range is not cached.
func (c *RegisterCache) Get(unitID byte, address, count uint16) []uint16 {
	rows, err := c.stmtGet.Query(int(unitID), int(address), int(address)+int(count))
	if err != nil {
		slog.Warn("cache get: query", "address", address, "count", count, "error", err)
		return nil
	}
	defer rows.Close()

	result := make([]uint16, count)
	found := 0
	for rows.Next() {
		var addr, val int
		if err := rows.Scan(&addr, &val); err != nil {
			slog.Warn("cache get: scan", "error", err)
			return nil
		}
		idx := addr - int(address)
		if idx < 0 || idx >= int(count) {
			continue
		}
		result[idx] = uint16(val)
		found++
	}

	if found != int(count) {
		return nil // all-or-nothing: every register in the range must be cached
	}
	return result
}

// GetBytes retrieves register values as raw bytes for a Modbus response.
// Returns nil if any register in the range is not cached.
func (c *RegisterCache) GetBytes(unitID byte, address, count uint16) []byte {
	values := c.Get(unitID, address, count)
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
	var count int
	if err := c.stmtCount.QueryRow().Scan(&count); err != nil {
		slog.Warn("cache size: query", "error", err)
		return 0
	}
	return count
}
