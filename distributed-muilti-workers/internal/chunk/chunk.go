package chunk

import (
	"sync/atomic"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type ChunkAllocator struct {
	curIndex  atomic.Uint64
	maxIndex  uint64
	partition uint64
}

func NewChunkAllocator(partition uint64, start uint64, end uint64) *ChunkAllocator {
	ca := &ChunkAllocator{
		partition: partition,
		maxIndex:  end,
	}
	ca.curIndex.Store(start)
	return ca
}

func (ca *ChunkAllocator) GetNewGlobalChunk() (protocol.Chunk, bool) {
	start := ca.curIndex.Add(ca.partition) - ca.partition
	end := start + ca.partition
	id := end / ca.partition

	return protocol.Chunk{
		Id:    id,
		Start: start,
		End:   end,
	}, true
}

func (ca *ChunkAllocator) GetNewWorkItem() (protocol.Chunk, bool) {
	start := ca.curIndex.Add(ca.partition) - ca.partition

	if start >= ca.maxIndex {
		return protocol.Chunk{}, false
	}

	end := start + ca.partition
	id := end / ca.partition

	if end > ca.maxIndex {
		end = ca.maxIndex
	}

	return protocol.Chunk{
		Id:    id,
		Start: start,
		End:   end,
	}, true
}
