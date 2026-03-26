package controller

import (
	"context"
	"fmt"
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
	id                       string
	interval                 int
	checkpoint               uint64
	conn                     *tcp.Conn
	alloc                    *chunk.ChunkAllocator
	shadow                   protocol.ShadowEntry
	store                    persistence.Store
	activeChunk              *protocol.Chunk
	pendingDispatchStartedAt time.Time
	metrics                  *utils.Metrics
	logger                   *utils.Logger
	foundResultCh            chan<- string
}

func NewWorker(conn net.Conn, id string, shadow protocol.ShadowEntry, alloc *chunk.ChunkAllocator, interval int, checkpoint uint64, foundCh chan<- string, store persistence.Store, metrics *utils.Metrics, log *utils.Logger) *Worker {
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
		metrics:       metrics,
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
				dispatchStartedAt := time.Now()
				if msg.JobRequest != nil {
					w.observeWorkerJobMetrics(msg.JobRequest.PreviousJobMetrics)
				}

				if w.activeChunk != nil {
					w.persistTaskComplete("")
				}

				chunk := w.alloc.GetNewGlobalChunk()
				w.activeChunk = &chunk
				w.pendingDispatchStartedAt = dispatchStartedAt
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
				if w.metrics != nil {
					w.metrics.ObserveJobDispatchRegistrationOverhead(dispatchStartedAt, time.Now())
				}

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

				// Prefer chunk identity from the report itself to avoid attributing
				// late checkpoint messages to a newly assigned active chunk.
				chunk := report.Chunk
				if chunk.End <= chunk.Start && chunk.Id == 0 && w.activeChunk != nil {
					chunk = *w.activeChunk
				}

				w.logger.Printf(
					"checkpoint report received | chunk: [id=%d start=%d end=%d] | completed=%d",
					chunk.Id, chunk.Start, chunk.End, report.Completed,
				)

				w.persistCheckpoint(chunk, *report)
				if w.metrics != nil {
					w.metrics.ObserveCheckpointOverhead(report.ReportedAt, time.Now())
					w.metrics.AddCheckpointObservation(
						fmt.Sprintf("worker=%s chunk=%d completed=%d", w.id, chunk.Id, report.Completed),
					)
				}

			case protocol.MsgFound:
				resultReceivedAt := time.Now()
				result := msg.Result
				w.logger.Printf("<- received cracking result")
				if result != nil {
					if w.metrics != nil {
						w.observeWorkerJobMetrics(result.WorkerJobMetrics)
						w.metrics.ObserveResultReturnLatency(result.WorkerSentAt, resultReceivedAt)
					}
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
				requeueChunk, err := w.getFailedChunk()
				w.logger.Printf("%s", reason)
				w.persistTaskFailure(reason)
				w.persistFailure(reason)
				w.persistWorkerState(persistence.WorkerStateFailed, reason)
				if err != nil {
					w.conn.Close()
					return
				}
				w.alloc.GlobalRequeueChunk(requeueChunk)
				w.conn.Close()
				return

			case protocol.MsgStopAck:
				if msg.StopAck != nil {
					w.observeWorkerJobMetrics(msg.StopAck.WorkerJobMetrics)
				}
				w.persistTaskFailure("worker stopped")
				w.persistWorkerState(persistence.WorkerStateDisconnected, "")
				w.conn.Close()
				return
			}

		case <-heartbeat.C:
			if missedHeartbeats >= MAX_MISSED_HEARTBEATS {
				reason := "worker heartbeat timeout"
				requeueChunk, err := w.getFailedChunk()
				w.logger.Printf("%s, closing connection", reason)
				w.persistTaskFailure(reason)
				w.persistFailure(reason)
				w.persistWorkerState(persistence.WorkerStateFailed, reason)
				if err != nil {
					w.conn.Close()
					return
				}
				w.alloc.GlobalRequeueChunk(requeueChunk)
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
			requeueChunk, err := w.getFailedChunk()
			w.persistTaskFailure("connection closed")
			w.persistWorkerState(persistence.WorkerStateDisconnected, "")
			if err != nil {
				return
			}
			w.alloc.GlobalRequeueChunk(requeueChunk)
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

func (w *Worker) getFailedChunk() (protocol.Chunk, error) {
	if w.activeChunk == nil {
		return protocol.Chunk{}, fmt.Errorf("worker was not assigned chunk")
	}

	failed := *w.activeChunk
	completed, err := w.store.GetLatestCheckpoint(context.Background(), w.id, failed.Id)
	if err != nil {
		w.logger.Printf("db read latest checkpoint error: %v", err)
		return failed, nil
	}

	resumeStart := failed.Start + completed
	if resumeStart > failed.End {
		resumeStart = failed.End
	}

	failed.Start = resumeStart
	return failed, nil
}

func (w *Worker) observeWorkerJobMetrics(metrics *protocol.WorkerJobMetrics) {
	if w.metrics == nil || metrics == nil {
		return
	}

	if !w.pendingDispatchStartedAt.IsZero() {
		units := uint64(0)
		if w.activeChunk != nil && w.activeChunk.End > w.activeChunk.Start {
			units = w.activeChunk.End - w.activeChunk.Start
		}
		w.metrics.ObserveWorkAssignmentOverhead(w.pendingDispatchStartedAt, metrics.AssignmentReceivedAt, units)
		w.pendingDispatchStartedAt = time.Time{}
	}

	w.metrics.ObserveWorkerCrackingTime(metrics.ComputeStartedAt, metrics.ComputeFinishedAt)
}
