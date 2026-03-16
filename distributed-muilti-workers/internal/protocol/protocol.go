package protocol

import (
	"time"
)

type Command string

const (
	MsgJobReq       Command = "jobReq"       // -> controller
	MsgJobRes       Command = "jobRes"       // -> worker
	MsgHeartbeatReq Command = "heartbeatReq" // -> worker
	MsgHeartbeatRes Command = "heartbeatRes" // -> controller
	MsgStop         Command = "stop"         // -> worker
	MsgStopAck      Command = "stopAck"      // -> controller
	MsgFound        Command = "Found"        // -> controller
	MsgError        Command = "error"        // -> controller

)

type Message struct {
	Command Command `json:"command"`

	JobRequest  *JobRequest  `json:"job_request,omitempty"`
	JobResponse *JobResponse `json:"job_response,omitempty"`

	HeartbeatRequest  *HeartbeatRequest  `json:"heartbeat_request,omitempty"`
	HeartbeatResponse *HeartbeatResponse `json:"heartbeat_response,omitempty"`

	ErrorMessage string       `json:"error,omitempty"`
	Result       *FoundResult `json:"result,omitempty"`
}

type Chunk struct {
	Id    uint64
	Start uint64
	End   uint64
}

type ShadowEntry struct {
	Username string
	Settings string
	FullHash string
}

type JobRequest struct{}
type JobResponse struct {
	Chunk       Chunk
	ShadowEntry ShadowEntry
}

type FoundResult struct {
	Password string `json:"password"`
}

type WorkerMetrics struct {
	TotalCrackingTimeNanos int64     `json:"total_cracking_time_ns"`
	WorkerReceiveJobNanos  time.Time `json:"worker_receive_job_ns"`
	WorkerSentResultsNanos time.Time `json:"worker_sent_results_ns"`
}

type HeartbeatRequest struct {
	Interval int `json:"interval"`
}

type HeartbeatResponse struct {
	DeltaTested   int64   `json:"delta_tested"`
	TotalTested   int64   `json:"total_tested"`
	ThreadsActive int64   `json:"threads_active"`
	CurrentRate   float64 `json:"current_rate"`
	CurrentChunk  string  `json:"current_chunk"`
}
