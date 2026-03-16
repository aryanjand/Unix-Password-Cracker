package config

import (
	"flag"
	"fmt"
	"io"
)

type Controller struct {
	Port              int
	Username          string
	ShadowFilePath    string
	HeartbeatInterval int
	PartitionSize     int
}

const ControllerUsage = "Usage: controller -p PORT -f SHADOW_FILE -u USERNAME -b HEARTBEAT_SECONDS -c PARTITION_SIZE"

func ParseController(args []string) (Controller, error) {
	var cfg Controller
	fs := flag.NewFlagSet("controller", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.IntVar(&cfg.Port, "p", 0, "port to bind")
	fs.StringVar(&cfg.Username, "u", "", "username")
	fs.StringVar(&cfg.ShadowFilePath, "f", "", "shadow file path")
	fs.IntVar(&cfg.HeartbeatInterval, "b", 0, "heartbeat interval in seconds")
	fs.IntVar(&cfg.PartitionSize, "c", 1, "partition size for password space")
	fs.IntVar(&cfg.PartitionSize, "s", 1, "partition size for password space")

	if err := fs.Parse(args); err != nil {
		return Controller{}, err
	}

	if cfg.Port <= 0 || cfg.Port > 65535 ||
		cfg.PartitionSize <= 0 ||
		cfg.HeartbeatInterval <= 0 ||
		cfg.ShadowFilePath == "" ||
		cfg.Username == "" {
		return Controller{}, fmt.Errorf(ControllerUsage)
	}

	return cfg, nil
}
