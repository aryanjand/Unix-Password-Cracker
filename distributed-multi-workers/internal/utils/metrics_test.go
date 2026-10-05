package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotUsesMilliseconds(t *testing.T) {
	m := NewMetrics()
	start := time.Unix(0, 0)
	m.ObserveControllerParsingTime(start, start.Add(2*time.Millisecond))
	m.ObserveJobDispatchRegistrationOverhead(start, start.Add(10*time.Millisecond))
	m.ObserveWorkAssignmentOverhead(start, start.Add(4*time.Millisecond), 1000)
	m.ObserveWorkerCrackingTime(start, start.Add(50*time.Millisecond))
	m.ObserveResultReturnLatency(start, start.Add(1*time.Millisecond))
	m.ObserveCheckpointOverhead(start, start.Add(5*time.Millisecond))
	m.ObserveEndToEndRuntime(start, start.Add(100*time.Millisecond))

	snap := m.Snapshot()
	if snap.ControllerSideParsingTimeMS != 2 {
		t.Fatalf("parse ms = %v, want 2", snap.ControllerSideParsingTimeMS)
	}
	if snap.TotalEndToEndRuntimeMS != 100 {
		t.Fatalf("end-to-end ms = %v, want 100", snap.TotalEndToEndRuntimeMS)
	}
	if snap.CheckpointImpactPct != 5 {
		t.Fatalf("checkpoint impact = %v, want 5", snap.CheckpointImpactPct)
	}
	if snap.ControllerOverheadMS != 6 {
		t.Fatalf("controller overhead = %v, want 6", snap.ControllerOverheadMS)
	}
	if snap.NetworkingOverheadMS != 11 {
		t.Fatalf("networking overhead = %v, want 11", snap.NetworkingOverheadMS)
	}
	if snap.WorkAssignmentOverheadPerUnitNS != 4000 {
		t.Fatalf("per-unit ns = %v, want 4000", snap.WorkAssignmentOverheadPerUnitNS)
	}
}

func TestWriteJSONFile(t *testing.T) {
	m := NewMetrics()
	start := time.Unix(0, 0)
	m.ObserveEndToEndRuntime(start, start.Add(25*time.Millisecond))

	path := filepath.Join(t.TempDir(), "nested", "metrics.json")
	if err := m.WriteJSONFile(path); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.TotalEndToEndRuntimeMS != 25 {
		t.Fatalf("decoded end-to-end ms = %v, want 25", snap.TotalEndToEndRuntimeMS)
	}
}

func TestWriteJSONFileEmptyPath(t *testing.T) {
	if err := NewMetrics().WriteJSONFile(""); err != nil {
		t.Fatal(err)
	}
}
