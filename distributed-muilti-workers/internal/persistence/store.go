package persistence

import (
	"context"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

const (
	WorkerStateConnected    = "connected"
	WorkerStateRunning      = "running"
	WorkerStateCompleted    = "completed"
	WorkerStateFailed       = "failed"
	WorkerStateDisconnected = "disconnected"
)

type Store interface {
	Close() error
	UpsertWorkerState(ctx context.Context, workerID string, state string, lastError string) error
	RecordFailure(ctx context.Context, workerID string, reason string) error
	AssignTask(ctx context.Context, workerID string, chunk protocol.Chunk) error
	CompleteTask(ctx context.Context, chunkID uint64) error
	CompleteTaskWithFound(ctx context.Context, chunkID uint64, foundPassword string) error
	FailTask(ctx context.Context, chunkID uint64, reason string) error
	RecordCheckpoint(ctx context.Context, workerID string, chunk protocol.Chunk, report protocol.CheckpointReport) error
}
