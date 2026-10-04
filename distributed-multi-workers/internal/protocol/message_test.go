package protocol

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestMessageJSONRoundTrip(t *testing.T) {
	sentAt := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	metrics := &WorkerJobMetrics{
		AssignmentReceivedAt: sentAt.Add(-2 * time.Second),
		ComputeStartedAt:     sentAt.Add(-time.Second),
		ComputeFinishedAt:    sentAt,
	}
	chunk := Chunk{Id: 3, Start: 100, End: 200}
	shadow := ShadowEntry{Username: "aryan", Settings: "$5$salt", FullHash: "$5$salt$hash"}

	cases := []Message{
		{
			Command: MsgJobReq,
			JobRequest: &JobRequest{
				PreviousJobMetrics: metrics,
			},
		},
		{
			Command: MsgJobRes,
			JobResponse: &JobResponse{
				Chunk:       chunk,
				Checkpoint:  50,
				ShadowEntry: shadow,
			},
		},
		{
			Command:          MsgHeartbeatReq,
			HeartbeatRequest: &HeartbeatRequest{Interval: 1},
		},
		{
			Command: MsgHeartbeatRes,
			HeartbeatResponse: &HeartbeatResponse{
				DeltaTested:   10,
				TotalTested:   100,
				ThreadsActive: 4,
				CurrentRate:   12.5,
				CurrentChunk:  "100-200",
			},
		},
		{
			Command: MsgCheckpointReport,
			CheckpointReport: &CheckpointReport{
				Chunk:      chunk,
				Completed:  50,
				ReportedAt: sentAt,
			},
		},
		{Command: MsgStop},
		{
			Command: MsgStopAck,
			StopAck: &StopAck{
				WorkerJobMetrics: metrics,
				WorkerSentAt:     sentAt,
			},
		},
		{
			Command: MsgFound,
			Result: &FoundResult{
				Password:         "ACE",
				WorkerJobMetrics: metrics,
				WorkerSentAt:     sentAt,
			},
		},
		{Command: MsgError, ErrorMessage: "worker reported failure"},
	}

	for _, want := range cases {
		t.Run(string(want.Command), func(t *testing.T) {
			raw, err := json.Marshal(want)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var got Message
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if !messagesEqual(got, want) {
				t.Fatalf("round trip mismatch\n got %#v\nwant %#v", got, want)
			}
		})
	}
}

func messagesEqual(a, b Message) bool {
	normalizeTimes(&a)
	normalizeTimes(&b)
	return reflect.DeepEqual(a, b)
}

func normalizeTimes(m *Message) {
	if m.JobRequest != nil && m.JobRequest.PreviousJobMetrics != nil {
		m.JobRequest.PreviousJobMetrics = utcMetrics(m.JobRequest.PreviousJobMetrics)
	}
	if m.Result != nil {
		m.Result.WorkerSentAt = m.Result.WorkerSentAt.UTC()
		m.Result.WorkerJobMetrics = utcMetrics(m.Result.WorkerJobMetrics)
	}
	if m.StopAck != nil {
		m.StopAck.WorkerSentAt = m.StopAck.WorkerSentAt.UTC()
		m.StopAck.WorkerJobMetrics = utcMetrics(m.StopAck.WorkerJobMetrics)
	}
	if m.CheckpointReport != nil {
		m.CheckpointReport.ReportedAt = m.CheckpointReport.ReportedAt.UTC()
	}
}

func utcMetrics(in *WorkerJobMetrics) *WorkerJobMetrics {
	if in == nil {
		return nil
	}
	out := *in
	out.AssignmentReceivedAt = in.AssignmentReceivedAt.UTC()
	out.ComputeStartedAt = in.ComputeStartedAt.UTC()
	out.ComputeFinishedAt = in.ComputeFinishedAt.UTC()
	return &out
}
