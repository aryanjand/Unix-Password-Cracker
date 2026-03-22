package chunk

import (
	"sync/atomic"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

type ChunkAllocator struct {
	chunkRequeueCh chan protocol.Chunk
	logger         utils.Logger
	curIndex       atomic.Uint64
	maxIndex       uint64
	partition      uint64
}

func NewChunkAllocator(partition uint64, start uint64, end uint64) *ChunkAllocator {
	ca := &ChunkAllocator{
		maxIndex:       end,
		partition:      partition,
		logger:         *utils.NewLogger("[GlobalChunkAllocator] "),
		chunkRequeueCh: make(chan protocol.Chunk, 32),
	}
	ca.curIndex.Store(start)
	return ca
}

func (ca *ChunkAllocator) GetNewGlobalChunk() protocol.Chunk {
	select {
	case chunk := <-ca.chunkRequeueCh:
		ca.logger.Printf("Dequeue chunk (id=%d, start=%d, end=%d)", chunk.Id, chunk.Start, chunk.End)
		return chunk

	default:
		start := ca.curIndex.Add(ca.partition) - ca.partition
		end := start + ca.partition
		id := end / ca.partition

		return protocol.Chunk{
			Id:    id,
			Start: start,
			End:   end,
		}
	}
}

func (ca *ChunkAllocator) GlobalRequeueChunk(chunk protocol.Chunk) {
	ca.logger.Printf("Enqueue chunk (id=%d, start=%d, end=%d)", chunk.Id, chunk.Start, chunk.End)
	ca.chunkRequeueCh <- chunk
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
