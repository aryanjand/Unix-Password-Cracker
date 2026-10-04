package config

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type Controller struct {
	Port              int
	Username          string
	ShadowFilePath    string
	Checkpoint        uint64
	PartitionSize     int
	HeartbeatInterval int
	DBPath            string
	Reset             bool
}

const ControllerUsage = "Usage: controller -p PORT -f SHADOW_FILE -u USERNAME -b HEARTBEAT_SECONDS -c PARTITION_SIZE -k CHECKPOINT_INTERVAL [-d SQLITE_DB_PATH] [-reset]"

func ParseController(args []string) (Controller, error) {
	var cfg Controller
	fs := flag.NewFlagSet("controller", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.IntVar(&cfg.Port, "p", 0, "port to bind")
	fs.StringVar(&cfg.Username, "u", "", "username")
	fs.StringVar(&cfg.ShadowFilePath, "f", "", "shadow file path")
	fs.IntVar(&cfg.HeartbeatInterval, "b", 0, "heartbeat interval in seconds")
	fs.Uint64Var(&cfg.Checkpoint, "k", 0, "checkpoint interval measured in candidate password attempts")
	fs.IntVar(&cfg.PartitionSize, "c", 1, "partition size for password space")
	fs.IntVar(&cfg.PartitionSize, "s", 1, "partition size for password space")
	fs.StringVar(&cfg.DBPath, "d", defaultDBPath(), "sqlite database file path")
	fs.BoolVar(&cfg.Reset, "reset", false, "reset persisted tracking state before startup")

	if err := fs.Parse(args); err != nil {
		return Controller{}, err
	}

	if cfg.Port <= 0 || cfg.Port > 65535 ||
		cfg.Checkpoint <= 0 ||
		cfg.PartitionSize <= 0 ||
		cfg.HeartbeatInterval <= 0 ||
		cfg.DBPath == "" ||
		cfg.ShadowFilePath == "" ||
		cfg.Username == "" {
		return Controller{}, fmt.Errorf(ControllerUsage)
	}

	return cfg, nil
}

func defaultDBPath() string {
	const fallback = "cracker.db"
	if path := os.Getenv("SQLITE_DB_PATH"); path != "" {
		return path
	}
	return fallback
}
