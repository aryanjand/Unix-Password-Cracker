package worker

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

func TestJobRunnerHitAndMiss(t *testing.T) {
	hash := aceSHA256Hash(t)
	index := indexOfPassword(t, "ACE")

	job := aceJob(hash, index, index+1)
	hit := NewJobRunner(job, nil)
	if got := hit.Run(4); got != "ACE" {
		t.Fatalf("hit: got %q, want ACE", got)
	}

	miss := NewJobRunner(aceJob(hash, index+1, index+8), nil)
	if got := miss.Run(4); got != "" {
		t.Fatalf("miss: got %q, want empty", got)
	}
}

func TestJobRunnerDefaultThreadCount(t *testing.T) {
	hash := aceSHA256Hash(t)
	index := indexOfPassword(t, "ACE")
	zero := NewJobRunner(aceJob(hash, index, index+1), nil)
	if got := zero.Run(0); got != "ACE" {
		t.Fatalf("Run(0): got %q, want ACE", got)
	}
	neg := NewJobRunner(aceJob(hash, index, index+1), nil)
	if got := neg.Run(-1); got != "ACE" {
		t.Fatalf("Run(-1): got %q, want ACE", got)
	}
}

func TestJobRunnerProgressSumsRange(t *testing.T) {
	hash := aceSHA256Hash(t)
	index := indexOfPassword(t, "ACE")
	const n = uint64(20)
	start := index + 1
	job := aceJob(hash, start, start+n)

	var tested atomic.Uint64
	runner := NewJobRunner(job, func(count uint64) {
		tested.Add(count)
	})
	got := runner.Run(4)

	if got != "" {
		t.Fatalf("progress range should miss, got %q", got)
	}
	if tested.Load() != n {
		t.Fatalf("onProgress summed to %d, want %d", tested.Load(), n)
	}
}

func aceJob(hash string, start, end uint64) *protocol.JobResponse {
	return &protocol.JobResponse{
		Chunk:       protocol.Chunk{Start: start, End: end},
		ShadowEntry: protocol.ShadowEntry{FullHash: hash},
	}
}

func aceSHA256Hash(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "shadow", "shadow_ACE_sha256"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			t.Fatalf("malformed fixture line %q", line)
		}
		return fields[1]
	}
	t.Fatal("fixture has no entry")
	return ""
}

// Same mapping as cracker.generateNextPassword — used only to place ACE
// in the keyspace so JobRunner ranges stay tiny.
func indexOfPassword(t *testing.T, want string) uint64 {
	t.Helper()
	const limit = 200000
	for i := uint64(0); i < limit; i++ {
		if candidatePassword(i) == want {
			return i
		}
	}
	t.Fatalf("could not locate %q in the first %d candidates", want, limit)
	return 0
}

func candidatePassword(value uint64) string {
	charset := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789" +
		"@#%^&*()_+-=.,:;?")
	base := uint64(len(charset))

	result := []rune{}
	for {
		result = append([]rune{charset[value%base]}, result...)
		if value < base {
			break
		}
		value = value/base - 1
	}
	return string(result)
}
