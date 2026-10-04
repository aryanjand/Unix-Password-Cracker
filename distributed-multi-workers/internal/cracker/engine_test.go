package cracker

import (
	"testing"
)

func TestGenerateNextPasswordUnique(t *testing.T) {
	const n = 2000
	seen := make(map[string]uint64, n)

	for i := uint64(0); i < n; i++ {
		pw := generateNextPassword(i)
		if pw == "" {
			t.Fatalf("index %d produced empty password", i)
		}
		if prev, ok := seen[pw]; ok {
			t.Fatalf("password %q generated for both %d and %d", pw, prev, i)
		}
		seen[pw] = i
	}

	if len(seen) != n {
		t.Fatalf("got %d unique passwords, want %d", len(seen), n)
	}
}

func TestGenerateNextPasswordKnownValues(t *testing.T) {
	if got := generateNextPassword(0); got != string(charset[0]) {
		t.Fatalf("index 0: got %q, want %q", got, string(charset[0]))
	}

	// First wrap past a single-character password.
	base := uint64(len(charset))
	got := generateNextPassword(base)
	want := string(charset[0]) + string(charset[0])
	if got != want {
		t.Fatalf("index %d: got %q, want %q", base, got, want)
	}
}

func TestCrackChunkHitAndMiss(t *testing.T) {
	hash := hashFromShadowFixture(t, "shadow_ACE_sha256")

	index, ok := indexOfPassword("ACE", 200000)
	if !ok {
		t.Fatal("could not locate index for ACE in the first 200000 candidates")
	}

	found, err := CrackChunk(index, index+1, hash)
	if err != nil {
		t.Fatalf("hit range: %v", err)
	}
	if found != "ACE" {
		t.Fatalf("hit range: got %q, want ACE", found)
	}

	found, err = CrackChunk(index+1, index+8, hash)
	if err != nil {
		t.Fatalf("miss range: %v", err)
	}
	if found != "" {
		t.Fatalf("miss range: got %q, want empty", found)
	}
}

func indexOfPassword(want string, limit uint64) (uint64, bool) {
	for i := uint64(0); i < limit; i++ {
		if generateNextPassword(i) == want {
			return i, true
		}
	}
	return 0, false
}
