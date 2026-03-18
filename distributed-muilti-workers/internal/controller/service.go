package controller

import (
	"net"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/transport/tcp"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

const MAX_MISSED_HEARTBEATS = 2

type Worker struct {
	interval      int
	checkpoint    int
	conn          *tcp.Conn
	alloc         *chunk.ChunkAllocator
	shadow        protocol.ShadowEntry
	logger        *utils.Logger
	foundResultCh chan<- string
}

func NewWorker(conn net.Conn, shadow protocol.ShadowEntry, alloc *chunk.ChunkAllocator, interval int, checkpoint int, foundCh chan<- string, log *utils.Logger) *Worker {
	cc := tcp.NewConn(conn)

	worker := &Worker{
		conn:          cc,
		alloc:         alloc,
		shadow:        shadow,
		interval:      interval,
		checkpoint:    checkpoint,
		logger:        log,
		foundResultCh: foundCh,
	}

	go worker.HandleWorker()

	return worker
}

func (w *Worker) HandleWorker() {
	heartbeat := time.NewTicker(time.Duration(w.interval) * time.Second)
	defer heartbeat.Stop()
	missedHeartbeats := 0

	for {

		select {
		case msg := <-w.conn.Recv:
			w.logger.Printf("-> command received %s", msg.Command)

			switch msg.Command {

			case protocol.MsgJobReq:
				chunk, _ := w.alloc.GetNewGlobalChunk()
				w.conn.Send <- protocol.Message{
					Command: protocol.MsgJobRes,
					JobResponse: &protocol.JobResponse{
						Chunk:       chunk,
						Checkpoint:  w.checkpoint,
						ShadowEntry: w.shadow,
					},
				}

			case protocol.MsgHeartbeatRes:
				missedHeartbeats = 0
				hb := msg.HeartbeatResponse
				w.logger.Printf(
					"heartbeat | delta: %-10d | total: %-12d | threads: %-3d | rate: %.2f/sec | chunk: %s",
					hb.DeltaTested,
					hb.TotalTested,
					hb.ThreadsActive,
					hb.CurrentRate,
					hb.CurrentChunk,
				)

			case protocol.MsgFound:
				result := msg.Result
				w.logger.Printf("<- received cracking result")
				w.foundResultCh <- result.Password
				return

			case protocol.MsgError:
				w.logger.Printf("worker reported failure")
				w.conn.Stop.Done()
				return
			}

		case <-heartbeat.C:
			if missedHeartbeats >= MAX_MISSED_HEARTBEATS {
				w.logger.Printf("worker heartbeat timeout, closing connection")
				w.conn.Stop.Done()
				return
			}
			missedHeartbeats++
			w.conn.Send <- protocol.Message{Command: protocol.MsgHeartbeatReq}

		case <-w.conn.Stop.Done():
			return
		}

	}
}
