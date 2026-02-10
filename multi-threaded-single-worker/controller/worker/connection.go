package main

import (
	"encoding/json"
	"runtime"
	"sync/atomic"
	"time"

	"controller/protocol"
)

type WriteMsg any

type Metrics struct {
	JobDispatch  time.Duration
	WorkerCrack  time.Duration
	ResultReturn time.Duration
}

func writeRequests(encoder *json.Encoder, writeCh <-chan WriteMsg, log *Logger) {
	for msg := range writeCh {
		if err := encoder.Encode(msg); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}

func readRequests(decoder *json.Decoder, writeCh chan<- WriteMsg, jobCh chan<- *protocol.CrackingJob, delta_tested *int64, total_tested *int64, log *Logger) {
	var interval int
	for {
		var msg protocol.WorkerMessage
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("decode error: %v", err)
			close(writeCh)
			return
		}

		log.Printf("<- command %s received", msg)

		switch msg.Status {

		case protocol.ALIVE:

			delta := atomic.LoadInt64(delta_tested)
			total := atomic.LoadInt64(total_tested)
			hb := protocol.HeartbeatResponse{
				DeltaTested:   delta,
				TotalTested:   total,
				CurrentRate:   float64(delta) / float64(interval),
				ThreadsActive: int64(runtime.NumGoroutine()),
			}
			atomic.StoreInt64(delta_tested, 0)
			log.Println("sending heartbeat ->")
			writeCh <- protocol.WorkerMessage{Status: protocol.ALIVE}
			writeCh <- hb

		case protocol.SHUTDOWN:
			return

		case protocol.SENT_JOB:
			var job protocol.CrackingJob
			if err := decoder.Decode(&job); err != nil {
				log.Printf("decode job error: %v", err)
				close(writeCh)
				return
			}

			log.Printf("<- received job %d", job.Id)
			interval = job.Interval
			jobCh <- &job
		}
	}
}
