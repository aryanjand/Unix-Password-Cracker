package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/aryanjand/Unix-Password-Cracker/internal/persistence"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

type checkpointStore struct {
	persistence.NoopStore
	completed uint64
	err       error
}

func (s checkpointStore) GetLatestCheckpoint(context.Context, string, uint64) (uint64, error) {
	return s.completed, s.err
}

func TestGetFailedChunkResume(t *testing.T) {
	log := utils.NewLogger("[test]")
	assigned := protocol.Chunk{Id: 3, Start: 100, End: 1000}

	t.Run("no active chunk", func(t *testing.T) {
		w := &Worker{store: checkpointStore{}, logger: log}
		if _, err := w.getFailedChunk(); err == nil {
			t.Fatal("expected error when no chunk is assigned")
		}
	})

	t.Run("no checkpoint keeps original start", func(t *testing.T) {
		w := &Worker{
			id:          "w1",
			store:       checkpointStore{completed: 0},
			activeChunk: &assigned,
			logger:      log,
		}
		got, err := w.getFailedChunk()
		if err != nil {
			t.Fatalf("getFailedChunk: %v", err)
		}
		if got != assigned {
			t.Fatalf("got %+v, want %+v", got, assigned)
		}
	})

	t.Run("checkpoint advances start", func(t *testing.T) {
		w := &Worker{
			id:          "w1",
			store:       checkpointStore{completed: 250},
			activeChunk: &assigned,
			logger:      log,
		}
		got, err := w.getFailedChunk()
		if err != nil {
			t.Fatalf("getFailedChunk: %v", err)
		}
		want := protocol.Chunk{Id: 3, Start: 350, End: 1000}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("checkpoint past end clamps to end", func(t *testing.T) {
		w := &Worker{
			id:          "w1",
			store:       checkpointStore{completed: 5000},
			activeChunk: &assigned,
			logger:      log,
		}
		got, err := w.getFailedChunk()
		if err != nil {
			t.Fatalf("getFailedChunk: %v", err)
		}
		want := protocol.Chunk{Id: 3, Start: 1000, End: 1000}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("store error returns original chunk", func(t *testing.T) {
		w := &Worker{
			id:          "w1",
			store:       checkpointStore{err: errors.New("db down")},
			activeChunk: &assigned,
			logger:      log,
		}
		got, err := w.getFailedChunk()
		if err != nil {
			t.Fatalf("store failure should not fail the requeue: %v", err)
		}
		if got != assigned {
			t.Fatalf("got %+v, want original %+v", got, assigned)
		}
	})
}
