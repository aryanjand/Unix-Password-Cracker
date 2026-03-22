package controller

import (
	"context"
	"net"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/persistence"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/transport/tcp"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

const MAX_MISSED_HEARTBEATS = 2

type Worker struct {
	id            string
	interval      int
	checkpoint    uint64
	conn          *tcp.Conn
	alloc         *chunk.ChunkAllocator
	shadow        protocol.ShadowEntry
	store         persistence.Store
	activeChunk   *protocol.Chunk
	logger        *utils.Logger
	foundResultCh chan<- string
}

func NewWorker(conn net.Conn, id string, shadow protocol.ShadowEntry, alloc *chunk.ChunkAllocator, interval int, checkpoint uint64, foundCh chan<- string, store persistence.Store, log *utils.Logger) *Worker {
	cc := tcp.NewConn(conn)
	if store == nil {
		store = persistence.NoopStore{}
	}

	worker := &Worker{
		id:            id,
		conn:          cc,
		alloc:         alloc,
		shadow:        shadow,
		interval:      interval,
		checkpoint:    checkpoint,
		store:         store,
		logger:        log,
		foundResultCh: foundCh,
	}
	worker.persistWorkerState(persistence.WorkerStateConnected, "")

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
				if w.activeChunk != nil {
					w.persistTaskComplete("")
				}

				chunk, _ := w.alloc.GetNewGlobalChunk()
				w.activeChunk = &chunk
				w.conn.Send <- protocol.Message{
					Command: protocol.MsgJobRes,
					JobResponse: &protocol.JobResponse{
						Chunk:       chunk,
						Checkpoint:  w.checkpoint,
						ShadowEntry: w.shadow,
					},
				}
				w.persistTaskAssignment(chunk)
				w.persistWorkerState(persistence.WorkerStateRunning, "")

			case protocol.MsgHeartbeatRes:
				missedHeartbeats = 0
				hb := msg.HeartbeatResponse
				if hb == nil {
					w.logger.Printf("heartbeat response payload missing")
					continue
				}
				w.logger.Printf(
					"heartbeat | delta: %-10d | total: %-12d | threads: %-3d | rate: %.2f/sec | chunk: %s",
					hb.DeltaTested,
					hb.TotalTested,
					hb.ThreadsActive,
					hb.CurrentRate,
					hb.CurrentChunk,
				)
				w.persistWorkerState(persistence.WorkerStateRunning, "")

			case protocol.MsgCheckpointReport:
				report := msg.CheckpointReport
				if report == nil {
					w.logger.Printf("checkpoint report payload missing")
					continue
				}

				chunk := protocol.Chunk{}
				if w.activeChunk != nil {
					chunk = *w.activeChunk
				}

				w.logger.Printf(
					"checkpoint report received | chunk: [id=%d start=%d end=%d] | completed=%d",
					chunk.Id, chunk.Start, chunk.End, report.Completed,
				)

				w.persistCheckpoint(chunk, *report)

			case protocol.MsgFound:
				result := msg.Result
				w.logger.Printf("<- received cracking result")
				if result != nil {
					w.foundResultCh <- result.Password
					w.persistTaskComplete(result.Password)
				} else {
					w.persistTaskComplete("")
				}
				w.persistWorkerState(persistence.WorkerStateCompleted, "")
				return

			case protocol.MsgError:
				reason := msg.ErrorMessage
				if reason == "" {
					reason = "worker reported failure"
				}
				w.logger.Printf("%s", reason)
				w.persistTaskFailure(reason)
				w.persistFailure(reason)
				w.persistWorkerState(persistence.WorkerStateFailed, reason)
				w.conn.Close()
				return

			case protocol.MsgStopAck:
				w.persistWorkerState(persistence.WorkerStateDisconnected, "")
				w.conn.Close()
				return
			}

		case <-heartbeat.C:
			if missedHeartbeats >= MAX_MISSED_HEARTBEATS {
				reason := "worker heartbeat timeout"
				w.logger.Printf("%s, closing connection", reason)
				w.persistTaskFailure(reason)
				w.persistFailure(reason)
				w.persistWorkerState(persistence.WorkerStateFailed, reason)
				w.conn.Close()
				return
			}
			missedHeartbeats++
			w.conn.Send <- protocol.Message{
				Command: protocol.MsgHeartbeatReq,
				HeartbeatRequest: &protocol.HeartbeatRequest{
					Interval: w.interval,
				},
			}

		case <-w.conn.Stop.Done():
			w.persistTaskFailure("connection closed")
			w.persistWorkerState(persistence.WorkerStateDisconnected, "")
			// Todo: when we detect failure here, signal resign chunk
			return
		}

	}
}

func (w *Worker) persistTaskAssignment(chunk protocol.Chunk) {
	if err := w.store.AssignTask(context.Background(), w.id, chunk); err != nil {
		w.logger.Printf("db assign task error: %v", err)
	}
}

func (w *Worker) persistTaskComplete(foundPassword string) {
	if w.activeChunk == nil {
		return
	}

	chunkID := w.activeChunk.Id
	w.activeChunk = nil

	var err error
	if foundPassword != "" {
		err = w.store.CompleteTaskWithFound(context.Background(), chunkID, foundPassword)
	} else {
		err = w.store.CompleteTask(context.Background(), chunkID)
	}
	if err != nil {
		w.logger.Printf("db complete task error: %v", err)
	}
}

func (w *Worker) persistTaskFailure(reason string) {
	if w.activeChunk == nil {
		return
	}

	chunkID := w.activeChunk.Id
	w.activeChunk = nil
	if err := w.store.FailTask(context.Background(), chunkID, reason); err != nil {
		w.logger.Printf("db fail task error: %v", err)
	}
}

func (w *Worker) persistWorkerState(state string, lastError string) {
	if err := w.store.UpsertWorkerState(context.Background(), w.id, state, lastError); err != nil {
		w.logger.Printf("db update worker state error: %v", err)
	}
}

func (w *Worker) persistFailure(reason string) {
	if err := w.store.RecordFailure(context.Background(), w.id, reason); err != nil {
		w.logger.Printf("db insert worker failure error: %v", err)
	}
}

func (w *Worker) persistCheckpoint(chunk protocol.Chunk, report protocol.CheckpointReport) {
	if err := w.store.RecordCheckpoint(context.Background(), w.id, chunk, report); err != nil {
		w.logger.Printf("db insert checkpoint error: %v", err)
	}
}
