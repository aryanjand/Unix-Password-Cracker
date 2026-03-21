package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	dbsqlc "github.com/aryanjand/Unix-Password-Cracker/internal/db/sqlc"
	"github.com/aryanjand/Unix-Password-Cracker/internal/persistence"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type MySQLStore struct {
	db      *sql.DB
	queries *dbsqlc.Queries
}

var _ persistence.Store = (*MySQLStore)(nil)

func NewMySQLStore(dsn string, reset bool) (*MySQLStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("mysql DSN is required")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	store := &MySQLStore{
		db:      db,
		queries: dbsqlc.New(db),
	}

	if err := store.migrate(context.Background(), reset); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) UpsertWorkerState(ctx context.Context, workerID string, state string, lastError string) error {
	return s.queries.UpsertWorkerState(ctx, dbsqlc.UpsertWorkerStateParams{
		WorkerID:  workerID,
		State:     state,
		LastError: lastError,
	})
}

func (s *MySQLStore) RecordFailure(ctx context.Context, workerID string, reason string) error {
	if strings.TrimSpace(reason) == "" {
		reason = "worker failure"
	}

	if err := s.queries.InsertWorkerFailure(ctx, dbsqlc.InsertWorkerFailureParams{
		WorkerID: workerID,
		Reason:   reason,
	}); err != nil {
		return err
	}

	return nil
}

func (s *MySQLStore) AssignTask(ctx context.Context, workerID string, chunk protocol.Chunk) error {
	return s.queries.AssignTask(ctx, dbsqlc.AssignTaskParams{
		ChunkID:    chunk.Id,
		WorkerID:   workerID,
		ChunkStart: chunk.Start,
		ChunkEnd:   chunk.End,
	})
}

func (s *MySQLStore) CompleteTask(ctx context.Context, chunkID uint64) error {
	return s.queries.CompleteTask(ctx, chunkID)
}

func (s *MySQLStore) CompleteTaskWithFound(ctx context.Context, chunkID uint64, foundPassword string) error {
	return s.queries.CompleteTaskWithFound(ctx, dbsqlc.CompleteTaskWithFoundParams{
		FoundPassword: foundPassword,
		ChunkID:       chunkID,
	})
}

func (s *MySQLStore) FailTask(ctx context.Context, chunkID uint64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		reason = "task failed"
	}

	return s.queries.FailTask(ctx, dbsqlc.FailTaskParams{
		FailureReason: reason,
		ChunkID:       chunkID,
	})
}

func (s *MySQLStore) RecordCheckpoint(ctx context.Context, workerID string, chunk protocol.Chunk, hb protocol.HeartbeatResponse) error {
	currentChunk := hb.CurrentChunk
	if strings.TrimSpace(currentChunk) == "" {
		currentChunk = fmt.Sprintf("%d-%d", chunk.Start, chunk.End)
	}

	return s.queries.InsertCheckpoint(ctx, dbsqlc.InsertCheckpointParams{
		WorkerID:      workerID,
		ChunkID:       chunk.Id,
		ChunkStart:    chunk.Start,
		ChunkEnd:      chunk.End,
		DeltaTested:   hb.DeltaTested,
		TotalTested:   hb.TotalTested,
		ThreadsActive: hb.ThreadsActive,
		CurrentRate:   hb.CurrentRate,
		CurrentChunk:  currentChunk,
	})
}

func (s *MySQLStore) migrate(ctx context.Context, reset bool) error {
	start := 1
	if reset {
		start = 0
	}

	for idx := start; idx < len(migrations); idx++ {
		if _, err := s.db.ExecContext(ctx, migrations[idx]); err != nil {
			return fmt.Errorf("apply migration %d: %w", idx+1, err)
		}
	}

	return nil
}

var migrations = []string{
	`DROP TABLE IF EXISTS worker_checkpoints, tasks, worker_failures, workers`,
	`CREATE TABLE IF NOT EXISTS workers (
		worker_id VARCHAR(255) PRIMARY KEY,
		state VARCHAR(32) NOT NULL,
		last_error TEXT NULL,
		last_heartbeat_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS worker_failures (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		worker_id VARCHAR(255) NOT NULL,
		reason TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_worker_failures_worker_id_created (worker_id, created_at),
		CONSTRAINT fk_worker_failures_worker
			FOREIGN KEY (worker_id) REFERENCES workers(worker_id)
			ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS tasks (
		chunk_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
		worker_id VARCHAR(255) NOT NULL,
		chunk_start BIGINT UNSIGNED NOT NULL,
		chunk_end BIGINT UNSIGNED NOT NULL,
		status VARCHAR(32) NOT NULL,
		failure_reason TEXT NULL,
		found_password VARCHAR(255) NULL,
		assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_tasks_worker_id (worker_id),
		INDEX idx_tasks_status (status),
		CONSTRAINT fk_tasks_worker
			FOREIGN KEY (worker_id) REFERENCES workers(worker_id)
			ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS worker_checkpoints (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		worker_id VARCHAR(255) NOT NULL,
		chunk_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		chunk_start BIGINT UNSIGNED NOT NULL DEFAULT 0,
		chunk_end BIGINT UNSIGNED NOT NULL DEFAULT 0,
		delta_tested BIGINT NOT NULL,
		total_tested BIGINT NOT NULL,
		threads_active INT NOT NULL,
		current_rate DOUBLE NOT NULL,
		current_chunk VARCHAR(64) NOT NULL,
		reported_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_checkpoints_worker_reported (worker_id, reported_at),
		INDEX idx_checkpoints_chunk_id (chunk_id),
		CONSTRAINT fk_checkpoints_worker
			FOREIGN KEY (worker_id) REFERENCES workers(worker_id)
			ON DELETE CASCADE
	)`,
}
