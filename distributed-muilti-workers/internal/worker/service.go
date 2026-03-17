package worker

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/transport/tcp"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

type Worker struct {
	Conn        *tcp.Conn
	DeltaTested int64
	TotalTested int64

	JobCh  chan *protocol.JobResponse
	Logger *utils.Logger
	Wg     sync.WaitGroup
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
				}

			case protocol.MsgHeartbeatReq:
				req := msg.HeartbeatRequest
				interval := 1
				if req != nil && req.Interval > 0 {
					interval = req.Interval
				}

				delta := atomic.LoadInt64(&w.DeltaTested)
				atomic.StoreInt64(&w.DeltaTested, 0)

				hb := protocol.HeartbeatResponse{
					DeltaTested:   delta,
					TotalTested:   atomic.LoadInt64(&w.TotalTested),
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
				w.Conn.Stop.Done()
			}

		case <-w.Conn.Stop.Done():
			return
		}
	}
}

func (w *Worker) Wait() {
	w.Wg.Wait()
}
