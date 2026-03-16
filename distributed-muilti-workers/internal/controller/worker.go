package controller

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type Worker struct {
	interval      int
	conn          *Conn
	alloc         *chunk.ChunkAllocator
	shadow        protocol.ShadowEntry
	foundResultCh chan<- string
}

func NewWorker(conn net.Conn, shadow protocol.ShadowEntry, alloc *chunk.ChunkAllocator, interval int, foundResultCh chan<- string) *Worker {
	cc := NewConn(conn)

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
		case msg := <-w.conn.recv:

			switch msg.Command {

			case protocol.MsgJobReq:

				chunk, _ := w.alloc.GetNewGlobalChunk()
				w.conn.send <- protocol.Message{
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
			w.conn.send <- protocol.Message{Command: protocol.MsgHeartbeatReq}

		case <-w.conn.stop.Done():
			return
		}

	}
}

type WorkerManager struct {
	sync.Mutex

	workers map[string]Worker
}

func NewWorkerManger() *WorkerManager {
	return &WorkerManager{
		workers: make(map[string]Worker),
	}
}

func (cm *WorkerManager) Count() int {
	cm.Lock()
	defer cm.Unlock()

	return len(cm.workers)
}

func (cm *WorkerManager) AddWorker(id string, worker Worker) {
	cm.Lock()
	defer cm.Unlock()

	cm.workers[id] = worker
}

func (cm *WorkerManager) RemoveWorker(id string) {
	cm.Lock()
	defer cm.Unlock()

	delete(cm.workers, id)

}

func (cm *WorkerManager) GetWorker(id string) (Worker, bool) {
	cm.Lock()
	defer cm.Unlock()

	worker, ok := cm.workers[id]
	return worker, ok
}

func (cm *WorkerManager) BroadcastMessage(msg protocol.Command) {
	cm.Lock()
	defer cm.Unlock()

	for id, worker := range cm.workers {
		worker.conn.send <- protocol.Message{Command: msg}
		if msg == protocol.MsgStop {
			delete(cm.workers, id)
		}
	}
}
