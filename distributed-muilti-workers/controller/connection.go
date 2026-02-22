package main

import (
	"encoding/json"
	"fmt"
	"time"

	"controller/protocol"
)

type ResultMsg struct {
	Password string
	Metrics  *Metrics
	Err      error
}

type Metrics struct {
	JobDispatch  time.Duration
	WorkerCrack  time.Duration
	ResultReturn time.Duration
}

func writeRequests(encoder *json.Encoder, interval int, writeCh <-chan protocol.Message, log *Logger) {
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
			heartbeat := protocol.Message{
				Command: protocol.MsgHeartbeat,
			}
			if err := encoder.Encode(heartbeat); err != nil {
				log.Printf("heartbeat send failed: %v", err)
				return
			}
			log.Println("heartbeat sent")
		}
	}
}

func readRequests(decoder *json.Decoder, job protocol.CrackingJob, writeCh chan<- protocol.Message, resultCh chan<- ResultMsg, log *Logger) {
	var jobSentTime time.Time

	for {
		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			resultCh <- ResultMsg{
				Err: fmt.Errorf("decode worker message: %w", err),
			}
			return
		}

		log.Printf("<- command %s received", msg.Command)

		switch msg.Command {

		case protocol.MsgReady:
			jobSentTime = time.Now()
			log.Printf("-> enqueue cracking job")
			jobMsg := protocol.Message{
				Command: protocol.MsgJob,
				Job:     &job,
			}
			writeCh <- jobMsg

		case protocol.MsgResult:
			result := msg.Result

			log.Printf("<- received cracking result")
			fmt.Println("Check the results")
			jobReceiveTime := time.Now()

			metrics := Metrics{
				WorkerCrack:  time.Duration(result.Metrics.TotalCrackingTimeNanos),
				JobDispatch:  result.Metrics.WorkerReceiveJobNanos.Sub(jobSentTime),
				ResultReturn: jobReceiveTime.Sub(result.Metrics.WorkerSentResultsNanos),
			}

			resultCh <- ResultMsg{
				Metrics:  &metrics,
				Password: result.Password,
			}
			return

		case protocol.MsgError:
			resultCh <- ResultMsg{
				Err: fmt.Errorf("worker reported failure"),
			}
			return

		case protocol.MsgHeartbeat:
			hb := msg.Heartbeat
			log.Printf(
				"heartbeat | delta: %-10d | total: %-12d | threads: %-3d | rate: %.2f/sec",
				hb.DeltaTested,
				hb.TotalTested,
				hb.ThreadsActive,
				hb.CurrentRate,
			)

		default:
			log.Printf("⚠ unknown worker status: %q", msg.Command)

		}
	}
}
