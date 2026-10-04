package utils

import (
	"fmt"
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

func (m *Metrics) Summary() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	controllerOverhead := m.controllerParsingTime.total + m.workAssignmentOverhead.total
	networkOverhead := m.jobDispatchRegistrationLatency.total + m.resultReturnLatency.total
	checkpointOverhead := m.checkpointOverhead.total
	mainOverhead := controllerOverhead + networkOverhead + checkpointOverhead
	runtimeTotal := m.endToEndRuntime.total

	checkpointImpactPct := 0.0
	if runtimeTotal > 0 {
		checkpointImpactPct = (float64(checkpointOverhead) / float64(runtimeTotal)) * 100
	}

	impact := "low impact"
	switch {
	case checkpointImpactPct >= 5:
		impact = "high impact"
	case checkpointImpactPct >= 1:
		impact = "moderate impact"
	}

	var b strings.Builder
	b.WriteString("\n===== Runtime Metrics Summary =====\n")
	b.WriteString(m.formatStatsLine("controller-side parsing time", m.controllerParsingTime, ""))
	b.WriteString(m.formatStatsLine("job dispatch/registration overhead", m.jobDispatchRegistrationLatency, ""))

	perUnit := ""
	if m.workAssignmentUnits > 0 && m.workAssignmentOverhead.total > 0 {
		nsPerUnit := float64(m.workAssignmentOverhead.total.Nanoseconds()) / float64(m.workAssignmentUnits)
		perUnit = fmt.Sprintf(" | per-unit=%.2fns (units=%d)", nsPerUnit, m.workAssignmentUnits)
	}
	b.WriteString(m.formatStatsLine("work assignment overhead", m.workAssignmentOverhead, perUnit))
	b.WriteString(m.formatStatsLine("worker cracking time (compute/search)", m.workerCrackingTime, ""))
	b.WriteString(m.formatStatsLine("result return latency (worker -> controller)", m.resultReturnLatency, ""))
	b.WriteString(m.formatStatsLine("checkpoint overhead observations", m.checkpointOverhead, ""))
	b.WriteString(m.formatStatsLine("total end-to-end runtime", m.endToEndRuntime, ""))
	b.WriteString("\nMain results (controller + networking + checkpoint overhead)\n")
	b.WriteString(fmt.Sprintf("  controller overhead: %s\n", formatDuration(controllerOverhead)))
	b.WriteString(fmt.Sprintf("  networking overhead: %s\n", formatDuration(networkOverhead)))
	b.WriteString(fmt.Sprintf("  checkpoint overhead: %s (%s, %.2f%% of end-to-end)\n", formatDuration(checkpointOverhead), impact, checkpointImpactPct))
	b.WriteString(fmt.Sprintf("  combined overhead:   %s\n", formatDuration(mainOverhead)))

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
