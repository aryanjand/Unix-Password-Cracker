package chunk

import (
	"testing"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

func TestGetNewWorkItemBounds(t *testing.T) {
	alloc := NewChunkAllocator(3, 0, 10)

	want := []protocol.Chunk{
		{Id: 1, Start: 0, End: 3},
		{Id: 2, Start: 3, End: 6},
		{Id: 3, Start: 6, End: 9},
		{Id: 4, Start: 9, End: 10},
	}

	for i, exp := range want {
		got, ok := alloc.GetNewWorkItem()
		if !ok {
			t.Fatalf("item %d: got exhausted, want %+v", i, exp)
		}
		if got != exp {
			t.Fatalf("item %d: got %+v, want %+v", i, got, exp)
		}
	}

	if _, ok := alloc.GetNewWorkItem(); ok {
		t.Fatal("expected allocator to be exhausted")
	}
}

func TestGetNewWorkItemEmptyRange(t *testing.T) {
	alloc := NewChunkAllocator(1, 5, 5)
	if _, ok := alloc.GetNewWorkItem(); ok {
		t.Fatal("start == maxIndex should yield no work")
	}
}

func TestGlobalRequeuePreferredOverNewChunk(t *testing.T) {
	alloc := NewChunkAllocator(100, 0, 0)
	failed := protocol.Chunk{Id: 7, Start: 50, End: 100}

	alloc.GlobalRequeueChunk(failed)
	got := alloc.GetNewGlobalChunk()
	if got != failed {
		t.Fatalf("requeued chunk not returned first: got %+v, want %+v", got, failed)
	}

	next := alloc.GetNewGlobalChunk()
	want := protocol.Chunk{Id: 1, Start: 0, End: 100}
	if next != want {
		t.Fatalf("after requeue drain, got %+v, want %+v", next, want)
	}
}

func TestGetNewGlobalChunkSequence(t *testing.T) {
	alloc := NewChunkAllocator(1000, 0, 0)

	first := alloc.GetNewGlobalChunk()
	if first != (protocol.Chunk{Id: 1, Start: 0, End: 1000}) {
		t.Fatalf("first chunk: %+v", first)
	}

	second := alloc.GetNewGlobalChunk()
	if second != (protocol.Chunk{Id: 2, Start: 1000, End: 2000}) {
		t.Fatalf("second chunk: %+v", second)
	}
}
