package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Metrics struct {
	mu sync.Mutex

	controllerParsingTime          durationStats
	jobDispatchRegistrationLatency durationStats
	workAssignmentOverhead         durationStats
	workerCrackingTime             durationStats
	resultReturnLatency            durationStats
	checkpointOverhead             durationStats
	endToEndRuntime                durationStats

	workAssignmentUnits uint64

	checkpointNotes []string
	maxNotes        int
}

type durationStats struct {
	count int64
	total time.Duration
	min   time.Duration
	max   time.Duration
}

func NewMetrics() *Metrics {
	return &Metrics{
		maxNotes: 8,
	}
}

func (m *Metrics) ObserveControllerParsingTime(start, end time.Time) {
	m.observe(&m.controllerParsingTime, start, end)
}

func (m *Metrics) ObserveJobDispatchRegistrationOverhead(start, end time.Time) {
	m.observe(&m.jobDispatchRegistrationLatency, start, end)
}

func (m *Metrics) ObserveWorkAssignmentOverhead(start, end time.Time, units uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if d, ok := durationFrom(start, end); ok {
		m.workAssignmentOverhead.addDuration(d)
		m.workAssignmentUnits += units
	}
}

func (m *Metrics) ObserveWorkerCrackingTime(start, end time.Time) {
	m.observe(&m.workerCrackingTime, start, end)
}

func (m *Metrics) ObserveResultReturnLatency(start, end time.Time) {
	m.observe(&m.resultReturnLatency, start, end)
}

func (m *Metrics) ObserveCheckpointOverhead(start, end time.Time) {
	m.observe(&m.checkpointOverhead, start, end)
}

func (m *Metrics) ObserveEndToEndRuntime(start, end time.Time) {
	m.observe(&m.endToEndRuntime, start, end)
}

func (m *Metrics) AddCheckpointObservation(note string) {
	note = strings.TrimSpace(note)
	if note == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.checkpointNotes) >= m.maxNotes {
		return
	}
	m.checkpointNotes = append(m.checkpointNotes, note)
}

func (m *Metrics) PrintSummary() {
	fmt.Print(m.Summary())
}

// Snapshot is the machine-readable form of a run. Durations are milliseconds
// so graphing code does not have to parse Go duration strings.
type Snapshot struct {
	ControllerSideParsingTimeMS       float64 `json:"controller_side_parsing_time_ms"`
	JobDispatchRegistrationOverheadMS float64 `json:"job_dispatch_registration_overhead_ms"`
	WorkAssignmentOverheadTotalMS     float64 `json:"work_assignment_overhead_total_ms"`
	WorkAssignmentOverheadPerUnitNS   float64 `json:"work_assignment_overhead_per_unit_ns,omitempty"`
	WorkerCrackingTimeMS              float64 `json:"worker_cracking_time_ms"`
	ResultReturnLatencyMS             float64 `json:"result_return_latency_ms"`
	CheckpointOverheadMS              float64 `json:"checkpoint_overhead_ms"`
	CheckpointImpactPct               float64 `json:"checkpoint_impact_pct"`
	TotalEndToEndRuntimeMS            float64 `json:"total_end_to_end_runtime_ms"`
	ControllerOverheadMS              float64 `json:"controller_overhead_ms"`
	NetworkingOverheadMS              float64 `json:"networking_overhead_ms"`
	CombinedOverheadMS                float64 `json:"combined_overhead_ms"`
	CheckpointObservationCount        int64   `json:"checkpoint_observation_count,omitempty"`
	CheckpointAvgMS                   float64 `json:"checkpoint_avg_ms,omitempty"`
	CheckpointMinMS                   float64 `json:"checkpoint_min_ms,omitempty"`
	CheckpointMaxMS                   float64 `json:"checkpoint_max_ms,omitempty"`
	ControllerParsingCount            int64   `json:"controller_parsing_count,omitempty"`
	JobDispatchCount                  int64   `json:"job_dispatch_count,omitempty"`
	WorkAssignmentCount               int64   `json:"work_assignment_count,omitempty"`
	WorkerCrackingCount               int64   `json:"worker_cracking_count,omitempty"`
	ResultReturnCount                 int64   `json:"result_return_count,omitempty"`
	EndToEndCount                     int64   `json:"end_to_end_count,omitempty"`
}

func (m *Metrics) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked()
}

func (m *Metrics) snapshotLocked() Snapshot {
	controllerOverhead := m.controllerParsingTime.total + m.workAssignmentOverhead.total
	networkOverhead := m.jobDispatchRegistrationLatency.total + m.resultReturnLatency.total
	checkpointOverhead := m.checkpointOverhead.total
	runtimeTotal := m.endToEndRuntime.total

	checkpointImpactPct := 0.0
	if runtimeTotal > 0 {
		checkpointImpactPct = (float64(checkpointOverhead) / float64(runtimeTotal)) * 100
	}

	snap := Snapshot{
		ControllerSideParsingTimeMS:       durationMS(m.controllerParsingTime.total),
		JobDispatchRegistrationOverheadMS: durationMS(m.jobDispatchRegistrationLatency.total),
		WorkAssignmentOverheadTotalMS:     durationMS(m.workAssignmentOverhead.total),
		WorkerCrackingTimeMS:              durationMS(m.workerCrackingTime.total),
		ResultReturnLatencyMS:             durationMS(m.resultReturnLatency.total),
		CheckpointOverheadMS:              durationMS(checkpointOverhead),
		CheckpointImpactPct:               checkpointImpactPct,
		TotalEndToEndRuntimeMS:            durationMS(runtimeTotal),
		ControllerOverheadMS:              durationMS(controllerOverhead),
		NetworkingOverheadMS:              durationMS(networkOverhead),
		CombinedOverheadMS:                durationMS(controllerOverhead + networkOverhead + checkpointOverhead),
		CheckpointObservationCount:        m.checkpointOverhead.count,
		ControllerParsingCount:            m.controllerParsingTime.count,
		JobDispatchCount:                  m.jobDispatchRegistrationLatency.count,
		WorkAssignmentCount:               m.workAssignmentOverhead.count,
		WorkerCrackingCount:               m.workerCrackingTime.count,
		ResultReturnCount:                 m.resultReturnLatency.count,
		EndToEndCount:                     m.endToEndRuntime.count,
	}

	if m.workAssignmentUnits > 0 && m.workAssignmentOverhead.total > 0 {
		snap.WorkAssignmentOverheadPerUnitNS = float64(m.workAssignmentOverhead.total.Nanoseconds()) / float64(m.workAssignmentUnits)
	}
	if m.checkpointOverhead.count > 0 {
		snap.CheckpointAvgMS = durationMS(m.checkpointOverhead.avg())
		snap.CheckpointMinMS = durationMS(m.checkpointOverhead.min)
		snap.CheckpointMaxMS = durationMS(m.checkpointOverhead.max)
	}

	return snap
}

func (m *Metrics) WriteJSONFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	data, err := json.MarshalIndent(m.Snapshot(), "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func (m *Metrics) Summary() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	snap := m.snapshotLocked()

	impact := "low impact"
	switch {
	case snap.CheckpointImpactPct >= 5:
		impact = "high impact"
	case snap.CheckpointImpactPct >= 1:
		impact = "moderate impact"
	}

	var b strings.Builder
	b.WriteString("\n===== Runtime Metrics Summary =====\n")
	b.WriteString(m.formatStatsLine("controller-side parsing time", m.controllerParsingTime, ""))
	b.WriteString(m.formatStatsLine("job dispatch/registration overhead", m.jobDispatchRegistrationLatency, ""))

	perUnit := ""
	if m.workAssignmentUnits > 0 && m.workAssignmentOverhead.total > 0 {
		perUnit = fmt.Sprintf(" | per-unit=%.2fns (units=%d)", snap.WorkAssignmentOverheadPerUnitNS, m.workAssignmentUnits)
	}
	b.WriteString(m.formatStatsLine("work assignment overhead", m.workAssignmentOverhead, perUnit))
	b.WriteString(m.formatStatsLine("worker cracking time (compute/search)", m.workerCrackingTime, ""))
	b.WriteString(m.formatStatsLine("result return latency (worker -> controller)", m.resultReturnLatency, ""))
	b.WriteString(m.formatStatsLine("checkpoint overhead observations", m.checkpointOverhead, ""))
	b.WriteString(m.formatStatsLine("total end-to-end runtime", m.endToEndRuntime, ""))
	b.WriteString("\nMain results (controller + networking + checkpoint overhead)\n")
	b.WriteString(fmt.Sprintf("  controller overhead: %s\n", formatDuration(m.controllerParsingTime.total+m.workAssignmentOverhead.total)))
	b.WriteString(fmt.Sprintf("  networking overhead: %s\n", formatDuration(m.jobDispatchRegistrationLatency.total+m.resultReturnLatency.total)))
	b.WriteString(fmt.Sprintf("  checkpoint overhead: %s (%s, %.2f%% of end-to-end)\n", formatDuration(m.checkpointOverhead.total), impact, snap.CheckpointImpactPct))
	b.WriteString(fmt.Sprintf("  combined overhead:   %s\n", formatDuration(m.controllerParsingTime.total+m.workAssignmentOverhead.total+m.jobDispatchRegistrationLatency.total+m.resultReturnLatency.total+m.checkpointOverhead.total)))

	if len(m.checkpointNotes) > 0 {
		b.WriteString("\nCheckpoint observations:\n")
		for _, note := range m.checkpointNotes {
			b.WriteString("  - ")
			b.WriteString(note)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func durationMS(d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(d) / float64(time.Millisecond)
}

func (m *Metrics) observe(stat *durationStats, start, end time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if d, ok := durationFrom(start, end); ok {
		stat.addDuration(d)
	}
}

func (m *Metrics) formatStatsLine(name string, s durationStats, tail string) string {
	if s.count == 0 {
		return fmt.Sprintf("- %s: no samples\n", name)
	}

	return fmt.Sprintf(
		"- %s: count=%d total=%s avg=%s min=%s max=%s%s\n",
		name,
		s.count,
		formatDuration(s.total),
		formatDuration(s.avg()),
		formatDuration(s.min),
		formatDuration(s.max),
		tail,
	)
}

func durationFrom(start, end time.Time) (time.Duration, bool) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0, false
	}
	return end.Sub(start), true
}

func (s *durationStats) addDuration(d time.Duration) {
	if d < 0 {
		return
	}
	if s.count == 0 {
		s.min = d
		s.max = d
	} else {
		if d < s.min {
			s.min = d
		}
		if d > s.max {
			s.max = d
		}
	}
	s.count++
	s.total += d
}

func (s durationStats) avg() time.Duration {
	if s.count == 0 {
		return 0
	}
	return s.total / time.Duration(s.count)
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	return d.String()
}
