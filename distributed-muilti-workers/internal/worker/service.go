package worker

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/transport/tcp"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

type Worker struct {
	Conn        *tcp.Conn
	DeltaTested uint64
	TotalTested uint64

	JobCh  chan *protocol.JobResponse
	Wg     sync.WaitGroup
	Logger *utils.Logger
}

func NewWorker(conn net.Conn, log *utils.Logger) *Worker {
	w := &Worker{
		Conn:   tcp.NewConn(conn),
		Logger: log,
		JobCh:  make(chan *protocol.JobResponse, 1),
	}

	w.Wg.Add(1)
	go w.HandleWorker()

	return w
}

func (w *Worker) HandleWorker() {
	defer w.Wg.Done()
	var chunk protocol.Chunk
	for {
		select {
		case msg := <-w.Conn.Recv:
			w.Logger.Printf("-> command received %s", msg.Command)

			switch msg.Command {
			case protocol.MsgJobRes:
				if msg.JobResponse != nil {
					w.JobCh <- msg.JobResponse
					chunk = msg.JobResponse.Chunk
					checkpoint := msg.JobResponse.Checkpoint
					go w.monitorCheckpoint(chunk.Start, chunk.End, checkpoint)
				}

			case protocol.MsgHeartbeatReq:
				req := msg.HeartbeatRequest
				interval := 1
				if req != nil && req.Interval > 0 {
					interval = req.Interval
				}

				delta := atomic.LoadUint64(&w.DeltaTested)
				atomic.StoreUint64(&w.DeltaTested, 0)

				hb := protocol.HeartbeatResponse{
					DeltaTested:   delta,
					TotalTested:   atomic.LoadUint64(&w.TotalTested),
					ThreadsActive: int64(runtime.NumGoroutine()),
					CurrentRate:   float64(delta) / float64(interval),
					CurrentChunk:  fmt.Sprintf("%d-%d", chunk.Start, chunk.End),
				}

				w.Conn.Send <- protocol.Message{
					Command:           protocol.MsgHeartbeatRes,
					HeartbeatResponse: &hb,
				}

			case protocol.MsgStop:
				w.Conn.Send <- protocol.Message{Command: protocol.MsgStopAck}
				w.Conn.Close()
				return
			}

		case <-w.Conn.Stop.Done():
			return
		}
	}
}

func (w *Worker) Wait() {
	w.Wg.Wait()
}

func (w *Worker) RecordTested(tested uint64) {
	if tested == 0 {
		return
	}
	atomic.AddUint64(&w.TotalTested, tested)
	atomic.AddUint64(&w.DeltaTested, tested)
}

func (w *Worker) monitorCheckpoint(start uint64, end uint64, checkpoint uint64) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	startTotal := atomic.LoadUint64(&w.TotalTested)
	jobSize := end - start
	next := checkpoint

	for {
		select {
		case <-w.Conn.Stop.Done():
			return
		case <-ticker.C:
			completed := atomic.LoadUint64(&w.TotalTested) - startTotal
			for completed >= next {
				w.Conn.Send <- protocol.Message{
					Command:          protocol.MsgCheckpointReport,
					CheckpointReport: &protocol.CheckpointReport{Completed: next},
				}
				next += checkpoint
			}
			if completed >= jobSize {
				return
			}
		}
	}
}
