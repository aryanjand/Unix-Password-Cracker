package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func shadowFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "shadow", name)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return path
}

func TestFindUserInShadow(t *testing.T) {
	entry, err := FindUserInShadow(shadowFixture(t, "shadow_ACE_bcrypt"), "aryan")
	if err != nil {
		t.Fatalf("FindUserInShadow: %v", err)
	}
	if entry.Username != "aryan" {
		t.Fatalf("username = %q, want aryan", entry.Username)
	}
	if !strings.HasPrefix(entry.FullHash, "$2") {
		t.Fatalf("full hash %q: want bcrypt prefix", entry.FullHash)
	}
	if !strings.HasPrefix(entry.Settings, "$2") {
		t.Fatalf("settings %q: want bcrypt prefix", entry.Settings)
	}
}

func TestFindUserInShadowMissingUser(t *testing.T) {
	_, err := FindUserInShadow(shadowFixture(t, "shadow_ACE_bcrypt"), "nobody")
	if err == nil {
		t.Fatal("expected user-not-found error")
	}
}

func TestFindUserInShadowInvalidFixture(t *testing.T) {
	_, err := FindUserInShadow(shadowFixture(t, "shadow_invalid"), "aryan")
	if err == nil {
		t.Fatal("locked/invalid hash should not parse as a valid user")
	}
}

func TestParseShadowLine(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		entry, err := parseShadowLine("aryan:$5$salt$hash:19000:0:99999:7:::")
		if err != nil {
			t.Fatalf("parseShadowLine: %v", err)
		}
		if entry.Username != "aryan" || entry.FullHash != "$5$salt$hash" || entry.Settings != "$5$salt" {
			t.Fatalf("got %+v", entry)
		}
	})

	t.Run("too few fields", func(t *testing.T) {
		if _, err := parseShadowLine("nocolon"); err == nil {
			t.Fatal("expected invalid format")
		}
	})

	t.Run("locked bang", func(t *testing.T) {
		if _, err := parseShadowLine("root:!:19000:0:99999:7:::"); err == nil {
			t.Fatal("expected locked-account error")
		}
	})

	t.Run("locked star", func(t *testing.T) {
		if _, err := parseShadowLine("bin:*:19000:0:99999:7:::"); err == nil {
			t.Fatal("expected locked-account error")
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		if _, err := parseShadowLine("aryan:!$2b$05$notvalid:19000:0:99999:7:::"); err == nil {
			t.Fatal("expected unsupported hash format")
		}
	})
}
