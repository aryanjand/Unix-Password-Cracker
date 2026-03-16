package config

import (
	"flag"
	"fmt"
	"io"
)

type Worker struct {
	ControllerHost string
	ControllerPort int
	Threads        int
	PartitionSize  int
}

const WorkerUsage = "Usage: worker -c HOST -p PORT -t THREADS"

func ParseWorker(args []string) (Worker, error) {
	var cfg Worker
	fs := flag.NewFlagSet("worker", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.IntVar(&cfg.ControllerPort, "p", 0, "controller port")
	fs.StringVar(&cfg.ControllerHost, "c", "", "controller host")
	fs.IntVar(&cfg.Threads, "t", 0, "number of threads")
	fs.IntVar(&cfg.PartitionSize, "s", 1, "partition size for password space")

	if err := fs.Parse(args); err != nil {
		return Worker{}, err
	}

	if cfg.ControllerHost == "" ||
		cfg.ControllerPort <= 0 ||
		cfg.ControllerPort > 65535 ||
		cfg.PartitionSize <= 0 ||
		cfg.Threads <= 0 {
		return Worker{}, fmt.Errorf(WorkerUsage)
	}

	return cfg, nil
}
