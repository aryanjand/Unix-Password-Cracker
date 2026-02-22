package main

import (
	"encoding/json"
	"runtime"
	"sync/atomic"
	"time"

	"controller/protocol"
)

type Metrics struct {
	JobDispatch  time.Duration
	WorkerCrack  time.Duration
	ResultReturn time.Duration
}

type ResultMsg struct {
	Found string
	Err   error
}

func writeRequests(encoder *json.Encoder, writeCh <-chan protocol.Message, log *Logger) {
	for msg := range writeCh {
		if err := encoder.Encode(msg); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}

func readRequests(decoder *json.Decoder, writeCh chan<- protocol.Message, jobCh chan<- *protocol.CrackingJob, delta_tested *int64, total_tested *int64, log *Logger) {
	var interval int
	for {
		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("decode error: %v", err)
			close(writeCh)
			return
		}

		log.Printf("<- command %s received", msg.Command)

		switch msg.Command {

		case protocol.MsgHeartbeat:
			delta := atomic.LoadInt64(delta_tested)
			total := atomic.LoadInt64(total_tested)
			hb := protocol.HeartbeatResponse{
				DeltaTested:   delta,
				TotalTested:   total,
				ThreadsActive: int64(runtime.NumGoroutine()),
				CurrentRate:   float64(delta) / float64(interval),
			}
			atomic.StoreInt64(delta_tested, 0)
			log.Println("sending heartbeat ->")
			hbResponse := protocol.Message{
				Command:   protocol.MsgHeartbeat,
				Heartbeat: &hb,
			}
			writeCh <- hbResponse

		case protocol.MsgShutdown:
			return

		case protocol.MsgJob:
			job := msg.Job
			log.Printf("<- received job %d", job.Id)
			interval = job.Interval
			jobCh <- job
		}
	}
}
