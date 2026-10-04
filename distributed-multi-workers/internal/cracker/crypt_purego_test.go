//go:build !cgo_crypt

package cracker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hashFromShadowFixture reads a testdata/shadow/shadow_<user>_<algo> fixture
// and returns the hash field (second colon-separated field) of its single entry.
func hashFromShadowFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "shadow", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			t.Fatalf("fixture %s: malformed line %q", name, line)
		}
		return fields[1]
	}

	t.Fatalf("fixture %s: no entry found", name)
	return ""
}

func TestVerifyCandidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		password string
	}{
		{"bcrypt", "shadow_ACE_bcrypt", "ACE"},
		{"md5-crypt", "shadow_ACE_md5", "ACE"},
		{"sha256-crypt", "shadow_ACE_sha256", "ACE"},
		{"sha512-crypt", "shadow_ACE_sha512", "ACE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashFromShadowFixture(t, tt.fixture)

			ok, err := verifyCandidatePassword(tt.password, hash)
			if err != nil {
				t.Fatalf("verifyCandidatePassword(%q, hash) returned error: %v", tt.password, err)
			}
			if !ok {
				t.Fatalf("verifyCandidatePassword(%q, hash) = false, want true", tt.password)
			}

			ok, err = verifyCandidatePassword("definitely-wrong", hash)
			if err != nil {
				t.Fatalf("verifyCandidatePassword(wrong candidate) returned error: %v", err)
			}
			if ok {
				t.Fatalf("verifyCandidatePassword(wrong candidate) = true, want false")
			}
		})
	}
}

func TestVerifyCandidatePasswordYescryptUnsupported(t *testing.T) {
	hash := hashFromShadowFixture(t, "shadow_ACE_yescrypt")

	_, err := verifyCandidatePassword("ACE", hash)
	if err == nil {
		t.Fatal("verifyCandidatePassword on a yescrypt hash: expected an error under the pure-Go build, got nil")
	}
}

func TestVerifyCandidatePasswordUnrecognizedFormat(t *testing.T) {
	_, err := verifyCandidatePassword("anything", "not-a-crypt-hash")
	if err == nil {
		t.Fatal("verifyCandidatePassword on an unrecognized hash format: expected an error, got nil")
	}
}
