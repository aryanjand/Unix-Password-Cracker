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
	Checkpoint        int
	PartitionSize     int
	HeartbeatInterval int
	MySQLDSN          string
}

const ControllerUsage = "Usage: controller -p PORT -f SHADOW_FILE -u USERNAME -b HEARTBEAT_SECONDS -c PARTITION_SIZE -k CHECKPOINT_INTERVAL [-d MYSQL_DSN]"

func ParseController(args []string) (Controller, error) {
	var cfg Controller
	fs := flag.NewFlagSet("controller", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.IntVar(&cfg.Port, "p", 0, "port to bind")
	fs.StringVar(&cfg.Username, "u", "", "username")
	fs.StringVar(&cfg.ShadowFilePath, "f", "", "shadow file path")
	fs.IntVar(&cfg.HeartbeatInterval, "b", 0, "heartbeat interval in seconds")
	fs.IntVar(&cfg.Checkpoint, "k", 0, "checkpoint interval measured in candidate password attempts")
	fs.IntVar(&cfg.PartitionSize, "c", 1, "partition size for password space")
	fs.IntVar(&cfg.PartitionSize, "s", 1, "partition size for password space")
	fs.StringVar(&cfg.MySQLDSN, "d", defaultMySQLDSN(), "mysql dsn")

	if err := fs.Parse(args); err != nil {
		return Controller{}, err
	}

	if cfg.Port <= 0 || cfg.Port > 65535 ||
		cfg.Checkpoint <= 0 ||
		cfg.PartitionSize <= 0 ||
		cfg.HeartbeatInterval <= 0 ||
		cfg.MySQLDSN == "" ||
		cfg.ShadowFilePath == "" ||
		cfg.Username == "" {
		return Controller{}, fmt.Errorf(ControllerUsage)
	}

	return cfg, nil
}

func defaultMySQLDSN() string {
	const fallback = "cracker:cracker_password@tcp(127.0.0.1:3306)/password_cracker?parseTime=true"
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		return dsn
	}
	return fallback
}
