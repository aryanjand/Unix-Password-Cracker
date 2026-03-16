package worker

import (
	"net"
	"runtime"
	"sync/atomic"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/transport/tcp"
)

type Worker struct {
	Conn        *tcp.Conn
	DeltaTested int64
	TotalTested int64
	JobCh       chan *protocol.JobResponse
}

func NewWorker(conn net.Conn) *Worker {
	w := &Worker{
		Conn:  tcp.NewConn(conn),
		JobCh: make(chan *protocol.JobResponse, 1),
	}

	go w.HandleWorker()

	return w
}

func (w *Worker) HandleWorker() {
	for {
		select {
		case msg := <-w.Conn.Recv:
			switch msg.Command {
			case protocol.MsgJobRes:
				if msg.JobResponse != nil {
					w.JobCh <- msg.JobResponse
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
				}

				w.Conn.Send <- protocol.Message{
					Command:           protocol.MsgHeartbeatRes,
					HeartbeatResponse: &hb,
				}

			case protocol.MsgStop:
				w.Conn.Send <- protocol.Message{Command: protocol.MsgStopAck}
			}

		case <-w.Conn.Stop.Done():
			return
		}
	}
}
