package persistence

import (
	"context"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type NoopStore struct{}

func (NoopStore) Close() error { return nil }

func (NoopStore) UpsertWorkerState(context.Context, string, string, string) error { return nil }

func (NoopStore) RecordFailure(context.Context, string, string) error { return nil }

func (NoopStore) AssignTask(context.Context, string, protocol.Chunk) error { return nil }

func (NoopStore) CompleteTask(context.Context, uint64) error { return nil }

func (NoopStore) CompleteTaskWithFound(context.Context, uint64, string) error { return nil }

func (NoopStore) FailTask(context.Context, uint64, string) error { return nil }

func (NoopStore) RecordCheckpoint(context.Context, string, protocol.Chunk, protocol.CheckpointReport) error {
	return nil
}
