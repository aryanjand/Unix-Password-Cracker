package protocol

import (
	"time"
)

type Command string

const (
	MsgJobReq           Command = "jobReq"           // -> controller
	MsgJobRes           Command = "jobRes"           // -> worker
	MsgHeartbeatReq     Command = "heartbeatReq"     // -> worker
	MsgHeartbeatRes     Command = "heartbeatRes"     // -> controller
	MsgCheckpointReport Command = "checkpointReport" // -> controller
	MsgStop             Command = "stop"             // -> worker
	MsgStopAck          Command = "stopAck"          // -> controller
	MsgFound            Command = "Found"            // -> controller
	MsgError            Command = "error"            // -> controller

)

type Message struct {
	Command Command `json:"command"`

	JobRequest  *JobRequest  `json:"job_request,omitempty"`
	JobResponse *JobResponse `json:"job_response,omitempty"`

	HeartbeatRequest  *HeartbeatRequest  `json:"heartbeat_request,omitempty"`
	HeartbeatResponse *HeartbeatResponse `json:"heartbeat_response,omitempty"`

	CheckpointReport *CheckpointReport `json:"checkpoint_report,omitempty"`

	ErrorMessage string       `json:"error,omitempty"`
	Result       *FoundResult `json:"result,omitempty"`
	StopAck      *StopAck     `json:"stop_ack,omitempty"`
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

type JobRequest struct {
	PreviousJobMetrics *WorkerJobMetrics `json:"previous_job_metrics,omitempty"`
}
type JobResponse struct {
	Chunk       Chunk
	Checkpoint  uint64
	ShadowEntry ShadowEntry
}

type FoundResult struct {
	Password         string            `json:"password"`
	WorkerJobMetrics *WorkerJobMetrics `json:"worker_job_metrics,omitempty"`
	WorkerSentAt     time.Time         `json:"worker_sent_at"`
}

type WorkerJobMetrics struct {
	AssignmentReceivedAt time.Time `json:"assignment_received_at"`
	ComputeStartedAt     time.Time `json:"compute_started_at"`
	ComputeFinishedAt    time.Time `json:"compute_finished_at"`
}

type StopAck struct {
	WorkerJobMetrics *WorkerJobMetrics `json:"worker_job_metrics,omitempty"`
	WorkerSentAt     time.Time         `json:"worker_sent_at"`
}

type HeartbeatRequest struct {
	Interval int `json:"interval"`
}

type HeartbeatResponse struct {
	DeltaTested   uint64  `json:"delta_tested"`
	TotalTested   uint64  `json:"total_tested"`
	ThreadsActive int64   `json:"threads_active"`
	CurrentRate   float64 `json:"current_rate"`
	CurrentChunk  string  `json:"current_chunk"`
}

type CheckpointReport struct {
	Chunk      Chunk     `json:"chunk"`
	Completed  uint64    `json:"completed"`
	ReportedAt time.Time `json:"reported_at"`
}
