package utils

import (
	"errors"
	"flag"
	"fmt"
)

type ControllerArgs struct {
	Port       int
	ShadowFile string
	Username   string
}

type WorkerArgs struct {
	ControllerHost string
	ControllerPort int
}

func ParseControllerFlags() (*ControllerArgs, error) {
	port := flag.Int("p", 0, "port to bind to")
	shadowFile := flag.String("f", "", "path to shadow file")
	username := flag.String("u", "", "username to inspect")
	flag.Parse()

	args := &ControllerArgs{
		Port:       *port,
		ShadowFile: *shadowFile,
		Username:   *username,
	}

	err := validateControllerArgs(args)

	return args, err
}

func validateControllerArgs(args *ControllerArgs) error {
	if args.Port <= 0 {
		return errors.New("port must be greater than 0")
	}
	if args.ShadowFile == "" {
		return errors.New("shadow file path is required")
	}
	if args.Username == "" {
		return errors.New("username is required")
	}

	if flag.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %v", flag.Args())
	}

	return nil
}

func ParseWorkerFlags() (*WorkerArgs, error) {
	port := flag.Int("p", 0, "server port")
	controller := flag.String("p", "", "server hostname")

	args := &WorkerArgs{
		ControllerPort: *port,
		ControllerHost: *controller,
	}
	err := validateWorkerArgs(args)
	return args, err
}

func validateWorkerArgs(args *WorkerArgs) error {

	if args.ControllerPort <= 0 {
		return errors.New("port must be greater than 0")
	}
	if args.ControllerHost == "" {
		return errors.New("valid sever hostname or ip address required")
	}

	if flag.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %v", flag.Args())
	}

	return nil
}
