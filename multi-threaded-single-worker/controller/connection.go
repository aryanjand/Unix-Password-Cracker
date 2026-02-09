package main

import (
	"encoding/json"
	"fmt"
	"time"

	"controller/protocol"
)

type WriteMsg any

type ResultMsg struct {
	Result  *protocol.CrackResult
	Metrics *Metrics
	Err     error
}

type Metrics struct {
	JobDispatch  time.Duration
	WorkerCrack  time.Duration
	ResultReturn time.Duration
}

func writeRequests(encoder *json.Encoder, interval int, writeCh <-chan WriteMsg, log *Logger) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-writeCh:
			if !ok {
				log.Println("write channel closed, writer exiting")
				return
			}
			if err := encoder.Encode(msg); err != nil {
				log.Printf("write error: %v", err)
				return
			}

		case <-ticker.C:
			hb := protocol.WorkerMessage{Status: protocol.ALIVE}
			if err := encoder.Encode(hb); err != nil {
				log.Printf("heartbeat send failed: %v", err)
				return
			}
			log.Println("heartbeat sent")
		}
	}
}

func readRequests(decoder *json.Decoder, job protocol.CrackingJob, writeCh chan<- WriteMsg, resultCh chan<- ResultMsg, log *Logger) {
	var jobSentTime time.Time

	for {
		var msg protocol.WorkerMessage
		if err := decoder.Decode(&msg); err != nil {
			resultCh <- ResultMsg{
				Err: fmt.Errorf("decode worker message: %w", err),
			}
			return
		}

		switch msg.Status {

		case protocol.IDLE:
			continue

		case protocol.READY:
			jobSentTime = time.Now()
			log.Printf("-> enqueue cracking job")
			writeCh <- protocol.WorkerMessage{Status: protocol.SENT_JOB}
			writeCh <- job

		case protocol.SUCCESS:
			log.Printf("<- receiving cracking result")

			var result protocol.CrackResult
			if err := decoder.Decode(&result); err != nil {
				resultCh <- ResultMsg{
					Err: fmt.Errorf("decode result: %w", err),
				}
				return
			}

			metrics := Metrics{
				JobDispatch:  time.Since(jobSentTime),
				WorkerCrack:  time.Duration(result.Metrics.TotalCrackingTimeNanos),
				ResultReturn: time.Since(time.Unix(0, result.Metrics.WorkerSentResultsNanos)),
			}

			resultCh <- ResultMsg{
				Result:  &result,
				Metrics: &metrics,
			}
			return

		case protocol.FAILED:
			resultCh <- ResultMsg{
				Err: fmt.Errorf("worker reported failure"),
			}
			return

		case protocol.ALIVE:
			var hb protocol.HeartbeatResponse
			if err := decoder.Decode(&hb); err != nil {
				resultCh <- ResultMsg{
					Err: fmt.Errorf("decode heartbeat: %w", err),
				}
				return
			}

			log.Printf(
				"heartbeat | delta: %-10d | total: %-12d | threads: %-3d | rate: %d/sec",
				hb.DeltaTested,
				hb.TotalTested,
				hb.ThreadsActive,
				hb.CurrentRate,
			)

		default:
			log.Printf("⚠ unknown worker status: %q", msg.Status)

		}
	}
}
