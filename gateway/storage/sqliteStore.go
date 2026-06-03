package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS devices (
			device_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			platform TEXT,
			cpu_brand TEXT,
			cpu_cores INTEGER,
			total_memory INTEGER,
			total_disk INTEGER,
			os_name TEXT,
			os_version TEXT,
			architecture TEXT,
			boot_time INTEGER,
			uptime INTEGER,
			network_upload INTEGER,
			network_download INTEGER,
			last_seen INTEGER,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT,
			device_id TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			received_at INTEGER,
			event_type TEXT NOT NULL,
			category TEXT,
			process_name TEXT,
			window_title TEXT,
			pid INTEGER,
			metadata TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_device_time ON events(device_id, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_events_category ON events(category)`,
		`CREATE TABLE IF NOT EXISTS segments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			process_name TEXT NOT NULL,
			window_title TEXT,
			category TEXT,
			start_time INTEGER NOT NULL,
			end_time INTEGER NOT NULL,
			duration INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_segments_device_time ON segments(device_id, start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_segments_category ON segments(category)`,
		`CREATE TABLE IF NOT EXISTS custom_keywords (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category TEXT NOT NULL,
			keyword TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT 'process',
			created_at INTEGER NOT NULL,
			UNIQUE(category, keyword, scope)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// Best-effort column adds for older databases. Ignore the "duplicate column
	// name" error returned when the column already exists.
	_, _ = s.DB.Exec(`ALTER TABLE custom_keywords ADD COLUMN scope TEXT NOT NULL DEFAULT 'process'`)
	_, _ = s.DB.Exec(`ALTER TABLE events ADD COLUMN event_id TEXT`)
	_, _ = s.DB.Exec(`ALTER TABLE events ADD COLUMN received_at INTEGER`)
	// Idempotency: a partial unique index on event_id deduplicates re-sent
	// events while leaving legacy rows (NULL event_id) untouched. Created after
	// the ALTER so the column exists on upgraded databases.
	if _, err := s.DB.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_event_id ON events(event_id) WHERE event_id IS NOT NULL`,
	); err != nil {
		return fmt.Errorf("migrate event_id index: %w", err)
	}
	return nil
}
