package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"controller/protocol"
)

type Metrics struct {
	JobDispatch  time.Duration
	WorkerCrack  time.Duration
	ResultReturn time.Duration
}

func handleWorkerConnection(conn net.Conn, job *protocol.CrackingJob, log *Logger) (*protocol.CrackResult, Metrics, error) {

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	var result *protocol.CrackResult
	var jobSentTime, resultReceivedTime time.Time

	for {
		var msg protocol.WorkerMessage
		if err := decoder.Decode(&msg); err != nil {
			return nil, Metrics{}, fmt.Errorf("decode worker msg: %w", err)
		}
		conn.SetReadDeadline(time.Time{})
		switch msg.Status {

		case protocol.IDLE:
			continue

		case protocol.READY:
			jobSentTime = time.Now()
			log.Printf("→ Sending cracking job")
			if err := encoder.Encode(job); err != nil {
				return nil, Metrics{}, err
			}

		case protocol.SUCCESS:
			log.Printf("← Receiving cracking result")
			if err := decoder.Decode(&result); err != nil {
				return nil, Metrics{}, err
			}
			resultReceivedTime = time.Now()
			metrics := Metrics{
				JobDispatch:  resultReceivedTime.Sub(jobSentTime),
				WorkerCrack:  time.Duration(result.Metrics.TotalCrackingTimeNanos),
				ResultReturn: resultReceivedTime.Sub(time.Unix(0, result.Metrics.WorkerSentResultsNanos)),
			}
			log.Printf("✓ Result received: password=%q", result.Password)
			return result, metrics, nil

		case protocol.FAILED:
			return nil, Metrics{}, fmt.Errorf("worker reported failure")

		default:
			log.Printf("⚠ Unknown worker command: %q", msg.Status)
		}
	}
}
