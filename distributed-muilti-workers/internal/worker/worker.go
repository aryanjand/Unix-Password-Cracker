package worker

/*
#cgo LDFLAGS: -lcrypt
#include <stdlib.h>
#include <crypt.h>
#include <string.h>
*/
import "C"

import (
	"fmt"
	"net"
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type Worker struct {
	Conn         *Conn
	Delta_tested int64
	Total_tested int64

	JobCh         chan *protocol.JobResponse
	FoundResultCh chan<- string
}

func NewWorker(conn net.Conn) *Worker {
	cc := NewConn(conn)
	// Todo: determine when will alloc is initialized. Note job from the worker will have (start, end] value which has to be used to initialized alloc.
	// Todo: Job can only initialized in side HandleWorker, when we receive the job.
	worker := &Worker{
		Conn:          cc,
		FoundResultCh: make(chan<- string),
	}

	go worker.HandleWorker()

	return worker
}

// probably need to extended it to induced worker class
func (w *Worker) HandleWorker() {
	// This is where the business logic will live
	for {
		select {

		case msg := <-w.Conn.Recv:

			switch msg.Command {
			// 1. Receive a Job
			case protocol.MsgJobRes:
				w.JobCh <- msg.JobResponse

			// 2. Respond to heartbeat
			case protocol.MsgHeartbeatReq:
				req := msg.HeartbeatRequest
				delta := atomic.LoadInt64(&w.Delta_tested)
				atomic.StoreInt64(&w.Delta_tested, 0)
				hb := protocol.HeartbeatResponse{
					DeltaTested:   delta,
					TotalTested:   atomic.LoadInt64(&w.Total_tested),
					ThreadsActive: int64(runtime.NumGoroutine()),
					CurrentRate:   float64(delta) / float64(req.Interval),
				}
				w.Conn.Send <- protocol.Message{
					Command:           protocol.MsgHeartbeatRes,
					HeartbeatResponse: &hb,
				}

			// 3. Stop message
			case protocol.MsgStop:
				w.Conn.Send <- protocol.Message{
					Command: protocol.MsgStopAck,
				}
			}

		case <-w.Conn.Stop.Done():
			continue
		}

	}
}

var charset = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "abcdefghijklmnopqrstuvwxyz" + "0123456789" + "@#%^&*()_+-=.,:;?")

type JobRunner struct {
	job   *protocol.JobResponse
	alloc *chunk.ChunkAllocator
}

func NewJobRunner(job *protocol.JobResponse) JobRunner {
	start, end := job.Chunk.Start, job.Chunk.End
	alloc := chunk.NewChunkAllocator(1, start, end)

	runner := JobRunner{
		job:   job,
		alloc: alloc,
	}

	return runner
}

func (j *JobRunner) Run(threads int) string {
	result := make(chan string, 1)

	for i := 0; i < threads; i++ {
		go func() {
			for {
				chunk, ok := j.alloc.GetNewWorkItem()
				if !ok {
					return
				}

				password, err := j.processChunk(chunk)
				if err != nil {
				}

				result <- password
			}
		}()
	}

	return <-result
}

func (j *JobRunner) processChunk(chunk protocol.Chunk) (string, error) {

	for i := chunk.Start; i < chunk.End; i++ {
		candidate := generateNextPassword(i)
		fmt.Println("Check the next candidate ", candidate)
		matched, err := j.verifyCandidatePassword(candidate)
		if err != nil {
			return "", fmt.Errorf("failed to test password with crypt")
		}

		if matched {
			return candidate, nil
		}
	}

	return "", nil

}

func (j *JobRunner) verifyCandidatePassword(candidate string) (bool, error) {
	data := C.struct_crypt_data{}
	hash := j.job.ShadowEntry.FullHash
	C.memset(unsafe.Pointer(&data), 0, C.size_t(unsafe.Sizeof(data)))

	cHash := C.CString(hash)
	cPass := C.CString(candidate)
	defer C.free(unsafe.Pointer(cHash))
	defer C.free(unsafe.Pointer(cPass))

	res := C.crypt_r(cPass, cHash, &data)
	if res == nil {
		return false, fmt.Errorf("crypt_r failed")
	}
	return C.GoString(res) == hash, nil
}

func generateNextPassword(value uint64) string {
	base := uint64(len(charset))

	result := []rune{}
	for {
		result = append([]rune{charset[value%base]}, result...)
		if value < base {
			break
		}
		value = value/base - 1
	}
	return string(result)
}
