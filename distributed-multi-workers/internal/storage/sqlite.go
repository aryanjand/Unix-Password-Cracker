package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/aryanjand/Unix-Password-Cracker/internal/persistence"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type SQLiteStore struct {
	db *sql.DB
}

var _ persistence.Store = (*SQLiteStore)(nil)

// NewSQLiteStore opens (creating if necessary) a SQLite database at path and
// applies the tracking schema. The connection pool is capped at a single
// connection because the controller has multiple goroutines (one per
// connected worker) writing concurrently on every heartbeat/checkpoint;
// SQLite has no server-side connection arbiter like MySQL, so a single
// writer plus WAL mode is the standard way to avoid "database is locked"
// errors.
func NewSQLiteStore(path string, reset bool) (*SQLiteStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("sqlite database path is required")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(context.Background(), reset); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) UpsertWorkerState(ctx context.Context, workerID string, state string, lastError string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workers (worker_id, state, last_error, last_heartbeat_at)
		VALUES (?, ?, NULLIF(?, ''), CURRENT_TIMESTAMP)
		ON CONFLICT(worker_id) DO UPDATE SET
			state = excluded.state,
			last_error = excluded.last_error,
			last_heartbeat_at = excluded.last_heartbeat_at,
			updated_at = CURRENT_TIMESTAMP
	`, workerID, state, lastError)
	return err
}

func (s *SQLiteStore) RecordFailure(ctx context.Context, workerID string, reason string) error {
	if strings.TrimSpace(reason) == "" {
		reason = "worker failure"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO worker_failures (worker_id, reason) VALUES (?, ?)`,
		workerID, reason,
	)
	return err
}

func (s *SQLiteStore) AssignTask(ctx context.Context, workerID string, chunk protocol.Chunk) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (chunk_id, worker_id, chunk_start, chunk_end, status, assigned_at)
		VALUES (?, ?, ?, ?, 'assigned', CURRENT_TIMESTAMP)
		ON CONFLICT(chunk_id) DO UPDATE SET
			worker_id = excluded.worker_id,
			chunk_start = excluded.chunk_start,
			chunk_end = excluded.chunk_end,
			status = 'assigned',
			failure_reason = NULL,
			found_password = NULL,
			assigned_at = CURRENT_TIMESTAMP,
			completed_at = NULL,
			updated_at = CURRENT_TIMESTAMP
	`, chunk.Id, workerID, chunk.Start, chunk.End)
	return err
}

func (s *SQLiteStore) CompleteTask(ctx context.Context, chunkID uint64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET status = 'complete', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE chunk_id = ?
	`, chunkID)
	return err
}

func (s *SQLiteStore) CompleteTaskWithFound(ctx context.Context, chunkID uint64, foundPassword string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET status = 'found', found_password = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE chunk_id = ?
	`, foundPassword, chunkID)
	return err
}

func (s *SQLiteStore) FailTask(ctx context.Context, chunkID uint64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		reason = "task failed"
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET status = 'failed', failure_reason = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE chunk_id = ?
	`, reason, chunkID)
	return err
}

func (s *SQLiteStore) RecordCheckpoint(ctx context.Context, workerID string, chunk protocol.Chunk, report protocol.CheckpointReport) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO worker_checkpoints (worker_id, chunk_id, chunk_start, chunk_end, completed)
		VALUES (?, ?, ?, ?, ?)
	`, workerID, chunk.Id, chunk.Start, chunk.End, report.Completed)
	return err
}

func (s *SQLiteStore) GetLatestCheckpoint(ctx context.Context, workerID string, chunkID uint64) (uint64, error) {
	var completed uint64
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(completed), 0) FROM worker_checkpoints WHERE worker_id = ? AND chunk_id = ?`,
		workerID, chunkID,
	).Scan(&completed)
	if err != nil {
		return 0, err
	}
	return completed, nil
}

func (s *SQLiteStore) migrate(ctx context.Context, reset bool) error {
	if reset {
		for _, table := range []string{"worker_checkpoints", "tasks", "worker_failures", "workers"} {
			if _, err := s.db.ExecContext(ctx, `DROP TABLE IF EXISTS `+table); err != nil {
				return fmt.Errorf("drop table %s: %w", table, err)
			}
		}
	}

	for _, stmt := range schemaStatements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}

	return nil
}

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS workers (
		worker_id TEXT PRIMARY KEY,
		state TEXT NOT NULL,
		last_error TEXT,
		last_heartbeat_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS worker_failures (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		worker_id TEXT NOT NULL REFERENCES workers(worker_id) ON DELETE CASCADE,
		reason TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_worker_failures_worker_id_created ON worker_failures(worker_id, created_at)`,
	`CREATE TABLE IF NOT EXISTS tasks (
		chunk_id INTEGER PRIMARY KEY,
		worker_id TEXT NOT NULL REFERENCES workers(worker_id) ON DELETE CASCADE,
		chunk_start INTEGER NOT NULL,
		chunk_end INTEGER NOT NULL,
		status TEXT NOT NULL,
		failure_reason TEXT,
		found_password TEXT,
		assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_worker_id ON tasks(worker_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
	`CREATE TABLE IF NOT EXISTS worker_checkpoints (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		worker_id TEXT NOT NULL REFERENCES workers(worker_id) ON DELETE CASCADE,
		chunk_id INTEGER NOT NULL DEFAULT 0,
		chunk_start INTEGER NOT NULL DEFAULT 0,
		chunk_end INTEGER NOT NULL DEFAULT 0,
		completed INTEGER NOT NULL DEFAULT 0,
		reported_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_checkpoints_worker_reported ON worker_checkpoints(worker_id, reported_at)`,
	`CREATE INDEX IF NOT EXISTS idx_checkpoints_chunk_id ON worker_checkpoints(chunk_id)`,
}
