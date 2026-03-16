package controller

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/transport/tcp"
)

type Worker struct {
	interval      int
	conn          *tcp.Conn
	alloc         *chunk.ChunkAllocator
	shadow        protocol.ShadowEntry
	foundResultCh chan<- string
}

func NewWorker(conn net.Conn, shadow protocol.ShadowEntry, alloc *chunk.ChunkAllocator, interval int, foundResultCh chan<- string) *Worker {
	cc := tcp.NewConn(conn)

	worker := &Worker{
		conn:          cc,
		alloc:         alloc,
		shadow:        shadow,
		interval:      interval,
		foundResultCh: foundResultCh,
	}

	go worker.HandleWorker()

	return worker
}

func (w *Worker) HandleWorker() {
	heartbeat := time.NewTicker(time.Duration(w.interval) * time.Second)
	defer heartbeat.Stop()

	for {

		fmt.Println("About to enter select")
		select {
		case msg := <-w.conn.Recv:

			switch msg.Command {

			case protocol.MsgJobReq:

				chunk, _ := w.alloc.GetNewGlobalChunk()
				w.conn.Send <- protocol.Message{
					Command: protocol.MsgJobRes,
					JobResponse: &protocol.JobResponse{
						Chunk:       chunk,
						ShadowEntry: w.shadow,
					},
				}

				fmt.Println("Check the Job Response ", protocol.Message{
					Command: protocol.MsgJobRes,
					JobResponse: &protocol.JobResponse{
						Chunk:       chunk,
						ShadowEntry: w.shadow,
					},
				})

			case protocol.MsgHeartbeatRes:
				hb := msg.HeartbeatResponse
				log.Printf(
					"heartbeat | delta: %-10d | total: %-12d | threads: %-3d | rate: %.2f/sec",
					hb.DeltaTested,
					hb.TotalTested,
					hb.ThreadsActive,
					hb.CurrentRate,
				)

			case protocol.MsgFound:
				result := msg.Result
				log.Printf("<- received cracking result")
				w.foundResultCh <- result.Password
				return

			case protocol.MsgError:
				log.Printf("worker reported failure")
				w.conn.Close()
				return
			}

		case <-heartbeat.C:
			w.conn.Send <- protocol.Message{Command: protocol.MsgHeartbeatReq}

		case <-w.conn.Stop.Done():
			return
		}

	}
}