package config

import (
	"testing"
)

func TestParseControllerRequiredFlags(t *testing.T) {
	_, err := ParseController(nil)
	if err == nil {
		t.Fatal("expected error when required flags are missing")
	}

	_, err = ParseController([]string{
		"-p", "8080", "-f", "shadow", "-b", "1", "-c", "1000", "-k", "100",
	})
	if err == nil {
		t.Fatal("expected error when username is missing")
	}
}

func TestParseControllerRejectsBadPort(t *testing.T) {
	base := []string{"-f", "shadow", "-u", "aryan", "-b", "1", "-c", "1000", "-k", "100"}

	if _, err := ParseController(append([]string{"-p", "0"}, base...)); err == nil {
		t.Fatal("expected error for port 0")
	}
	if _, err := ParseController(append([]string{"-p", "70000"}, base...)); err == nil {
		t.Fatal("expected error for port 70000")
	}
}

func TestParseControllerPartitionAliases(t *testing.T) {
	common := []string{"-p", "8080", "-f", "shadow", "-u", "aryan", "-b", "1", "-k", "100"}

	withC, err := ParseController(append(append([]string{}, common...), "-c", "1000"))
	if err != nil {
		t.Fatalf("-c: %v", err)
	}
	if withC.PartitionSize != 1000 {
		t.Fatalf("-c: PartitionSize = %d, want 1000", withC.PartitionSize)
	}

	withS, err := ParseController(append(append([]string{}, common...), "-s", "250"))
	if err != nil {
		t.Fatalf("-s: %v", err)
	}
	if withS.PartitionSize != 250 {
		t.Fatalf("-s: PartitionSize = %d, want 250", withS.PartitionSize)
	}
}

func TestParseControllerDBPath(t *testing.T) {
	args := []string{"-p", "8080", "-f", "shadow", "-u", "aryan", "-b", "1", "-c", "1000", "-k", "100"}

	t.Run("default", func(t *testing.T) {
		t.Setenv("SQLITE_DB_PATH", "")
		cfg, err := ParseController(args)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DBPath != "cracker.db" {
			t.Fatalf("default DBPath = %q, want cracker.db", cfg.DBPath)
		}
	})

	t.Run("from env", func(t *testing.T) {
		t.Setenv("SQLITE_DB_PATH", "/tmp/from-env.db")
		cfg, err := ParseController(args)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DBPath != "/tmp/from-env.db" {
			t.Fatalf("env DBPath = %q, want /tmp/from-env.db", cfg.DBPath)
		}
	})

	t.Run("flag overrides env", func(t *testing.T) {
		t.Setenv("SQLITE_DB_PATH", "/tmp/from-env.db")
		cfg, err := ParseController(append(append([]string{}, args...), "-d", "explicit.db"))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DBPath != "explicit.db" {
			t.Fatalf("-d DBPath = %q, want explicit.db", cfg.DBPath)
		}
	})
}

func TestParseWorkerRequiredFlags(t *testing.T) {
	_, err := ParseWorker(nil)
	if err == nil {
		t.Fatal("expected error when required flags are missing")
	}

	got, err := ParseWorker([]string{"-c", "127.0.0.1", "-p", "8080", "-t", "4"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ControllerHost != "127.0.0.1" || got.ControllerPort != 8080 || got.Threads != 4 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseWorkerRejectsBadPortAndThreads(t *testing.T) {
	if _, err := ParseWorker([]string{"-c", "127.0.0.1", "-p", "0", "-t", "4"}); err == nil {
		t.Fatal("expected error for port 0")
	}
	if _, err := ParseWorker([]string{"-c", "127.0.0.1", "-p", "70000", "-t", "4"}); err == nil {
		t.Fatal("expected error for port 70000")
	}
	if _, err := ParseWorker([]string{"-c", "127.0.0.1", "-p", "8080", "-t", "0"}); err == nil {
		t.Fatal("expected error for threads 0")
	}
}
