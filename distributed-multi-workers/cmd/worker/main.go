package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/config"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
	"github.com/aryanjand/Unix-Password-Cracker/internal/worker"
)

func main() {
	log := utils.NewLogger("[Worker]")
	cfg, err := config.ParseWorker(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	address := fmt.Sprintf("%s:%d", cfg.ControllerHost, cfg.ControllerPort)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()
	log.Println("connected to controller")

	w := worker.NewWorker(conn, log)
	log.Print("created worker")

	var result string
	for result == "" {
		// 1. Pull work only when free; attach last-job timings for overhead.
		if err := w.Conn.SendMsg(protocol.Message{
			Command: protocol.MsgJobReq,
			JobRequest: &protocol.JobRequest{
				PreviousJobMetrics: w.TakeCompletedJobMetrics(),
			},
		}); err != nil {
			log.Printf("connection closed, exiting: %v", err)
			return
		}
		log.Printf("-> sent %s", protocol.MsgJobReq)

		// 2. Don't block only on JobCh — stop or a dead conn would hang.
		var job *protocol.JobResponse
		select {
		case job = <-w.JobCh:
		case <-w.StopCh:
			return
		case <-w.Conn.Stop.Done():
			log.Printf("connection closed while waiting for job, exiting")
			return
		}

		log.Printf("job started (chunk id=%d, password range=%d-%d, threads=%d)", job.Chunk.Id, job.Chunk.Start, job.Chunk.End, cfg.Threads)

		// 3. Per-job allocator so threads split this chunk, not the global space.
		runner := worker.NewJobRunner(job, w.RecordTested)

		// 4. One chunk at a time; wrap Run so the next message has compute timings.
		computeStart := time.Now()
		w.MarkComputeStart(computeStart)
		result = runner.Run(cfg.Threads)
		w.MarkComputeEnd(time.Now())
	}

	// 5. Only the finder sends this; everyone else already left via Stop.
	log.Printf("found password %q (local %s)", result, conn.LocalAddr())
	if err := w.Conn.SendMsg(protocol.Message{
		Command: protocol.MsgFound,
		Result: &protocol.FoundResult{
			Password:         result,
			WorkerJobMetrics: w.TakeCompletedJobMetrics(),
			WorkerSentAt:     time.Now(),
		},
	}); err != nil {
		log.Printf("send found result failed: %v", err)
	}

	// Let HandleWorker send StopAck before we exit.
	w.Wg.Wait()
}
