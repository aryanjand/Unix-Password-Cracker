package main

import (
	"encoding/json"
	"runtime"
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

func readRequests(decoder *json.Decoder, writeCh chan<- WriteMsg, jobCh chan<- *protocol.CrackingJob, log *Logger) {
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
			// Controller heartbeat request → respond
			hb := protocol.HeartbeatResponse{
				DeltaTested:   0,
				TotalTested:   0,
				ThreadsActive: int64(runtime.NumGoroutine()),
				CurrentRate:   0,
			}
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
			jobCh <- &job
		}
	}
}
