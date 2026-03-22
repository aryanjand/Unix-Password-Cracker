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

const MAX_JOBS = 1

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

	for {

		select {

		case <-w.StopCh:
			os.Exit(0)
			return

		default:
			// 1. Sent a Job Request with last completed-job metrics (if any)
			w.Conn.Send <- protocol.Message{
				Command: protocol.MsgJobReq,
				JobRequest: &protocol.JobRequest{
					PreviousJobMetrics: w.TakeCompletedJobMetrics(),
				},
			}
			log.Printf("-> sent %s", protocol.MsgJobReq)

			// 2. Wait for the Job Response
			job := <-w.JobCh
			log.Printf("job started (chunk id=%d, password range=%d-%d, threads=%d)", job.Chunk.Id, job.Chunk.Start, job.Chunk.End, cfg.Threads)

			// 3. Get a Job Runner
			runner := worker.NewJobRunner(job, w.RecordTested)

			// 4. Run using multi threading
			computeStart := time.Now()
			w.MarkComputeStart(computeStart)
			result = runner.Run(cfg.Threads)
			computeEnd := time.Now()
			w.MarkComputeEnd(computeEnd)

		}

		if result != "" {
			break
		}

	}

	if result != "" {
		w.Conn.Send <- protocol.Message{
			Command: protocol.MsgFound,
			Result: &protocol.FoundResult{
				Password:         result,
				WorkerJobMetrics: w.TakeCompletedJobMetrics(),
				WorkerSentAt:     time.Now(),
			},
		}
	}

	w.Wg.Wait()
}
