package worker

import (
	"sync"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/cracker"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type JobRunner struct {
	job   *protocol.JobResponse
	alloc *chunk.ChunkAllocator
}

func NewJobRunner(job *protocol.JobResponse) JobRunner {
	start, end := job.Chunk.Start, job.Chunk.End
	alloc := chunk.NewChunkAllocator(1, start, end)

	return JobRunner{
		job:   job,
		alloc: alloc,
	}
}

func (j *JobRunner) Run(threads int) string {
	if threads <= 0 {
		threads = 1
	}

	found := make(chan string, 1)
	var wg sync.WaitGroup

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				ch, ok := j.alloc.GetNewWorkItem()
				if !ok {
					return
				}

				password, err := j.processChunk(ch)
				if err != nil {
					continue
				}

				if password != "" {
					select {
					case found <- password:
					default:
					}
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(found)
	}()

	password, ok := <-found
	if !ok {
		return ""
	}

	return password
}

func (j *JobRunner) processChunk(ch protocol.Chunk) (string, error) {
	return cracker.CrackChunk(ch.Start, ch.End, j.job.ShadowEntry.FullHash)
}
