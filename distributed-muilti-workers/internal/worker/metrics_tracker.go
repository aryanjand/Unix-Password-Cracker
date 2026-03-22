package worker

import (
	"sync"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type JobMetricsTracker struct {
	mu sync.Mutex

	currentAssignmentRecvAt  time.Time
	currentComputeStartedAt  time.Time
	pendingCompletedJobStats *protocol.WorkerJobMetrics
}

func (t *JobMetricsTracker) markComputeStart(at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentComputeStartedAt = at
}

func (t *JobMetricsTracker) markComputeEnd(at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	startedAt := t.currentComputeStartedAt
	if startedAt.IsZero() {
		startedAt = at
	}

	t.pendingCompletedJobStats = &protocol.WorkerJobMetrics{
		AssignmentReceivedAt: t.currentAssignmentRecvAt,
		ComputeStartedAt:     startedAt,
		ComputeFinishedAt:    at,
	}
	t.currentAssignmentRecvAt = time.Time{}
	t.currentComputeStartedAt = time.Time{}
}

func (t *JobMetricsTracker) takeCompletedJobMetrics() *protocol.WorkerJobMetrics {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.pendingCompletedJobStats == nil {
		return nil
	}

	metrics := *t.pendingCompletedJobStats
	t.pendingCompletedJobStats = nil
	return &metrics
}

func (t *JobMetricsTracker) markAssignmentReceived(at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentAssignmentRecvAt = at
}

func (t *JobMetricsTracker) takeMetricsForStop(stopAt time.Time) *protocol.WorkerJobMetrics {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.pendingCompletedJobStats != nil {
		metrics := *t.pendingCompletedJobStats
		t.pendingCompletedJobStats = nil
		return &metrics
	}

	if t.currentAssignmentRecvAt.IsZero() && t.currentComputeStartedAt.IsZero() {
		return nil
	}

	startedAt := t.currentComputeStartedAt
	if startedAt.IsZero() {
		startedAt = stopAt
	}

	metrics := &protocol.WorkerJobMetrics{
		AssignmentReceivedAt: t.currentAssignmentRecvAt,
		ComputeStartedAt:     startedAt,
		ComputeFinishedAt:    stopAt,
	}
	t.currentAssignmentRecvAt = time.Time{}
	t.currentComputeStartedAt = time.Time{}
	return metrics
}
