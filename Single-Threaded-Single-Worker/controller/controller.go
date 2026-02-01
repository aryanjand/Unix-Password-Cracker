package main

import (
	"bufio" // Read file
	"encoding/json"
	"errors"  // Return Errors
	"flag"    // Parse args
	"fmt"     // Print statements
	"log"     // Logger Statments
	"net"     // Tcp connections
	"os"      // Open File
	"strings" // String manipulation functions
	"time"
)

const (
	shadowFieldCount = 9
	shadowDelimiter  = ":"
	tcp              = "tcp"
	READY_FOR_JOB    = "READY_FOR_JOB"
	PASSWORD_FOUND   = "PASSWORD_FOUND"
	NO_PASSWORD_LEFT = "NO_PASSWORD_LEFT"
	CLOSE_CONNCETION = "CLOSE_CONNCETION"
)

type ControllerArgs struct {
	Port       int
	ShadowFile string
	Username   string
}

type CrackingJob struct {
	JobId    int
	Username string
	Algo     string
	Cost     string
	Hash     string
	Salt     string
}

func main() {
	log.SetPrefix("[Controller] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	args := parseFlags()

	if err := validateArgs(args); err != nil {
		log.Fatal(err)
	}

	log.Printf("Port: %d", args.Port)
	log.Printf("Username: %s", args.Username)
	log.Printf("Shadow file: %s", args.ShadowFile)

	job, err := findUserInShadow(args)
	if err != nil {
		log.Fatal(err)
	}

	// Create a Server Conncetion
	address := fmt.Sprintf(":%d", args.Port)
	ln, err := net.Listen(tcp, address)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Println("Controller ready, waiting for workers on ", address)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		handleWorkerConnection(conn, job)
	}

}

func handleWorkerConnection(conn net.Conn, job *CrackingJob) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		log.Println("read error: ", err)
		return
	}

	if strings.TrimSpace(line) != READY_FOR_JOB {
		log.Println("unexpected message:", line)
		return
	}

	conn.SetReadDeadline(time.Time{})

	jobBytes, err := json.Marshal(job)
	if err != nil {
		log.Println("failed to marshal job:", err)
		return
	}

	_, err = conn.Write(jobBytes)
	if err != nil {
		log.Println("write error:", err)
		return
	}

	for {

		line, err = reader.ReadString('\n')
		if err != nil {
			log.Println("read error: ", err)
		}

		cmd := strings.TrimSpace(line)

		switch cmd {
		case PASSWORD_FOUND:
			password, err := reader.ReadString('\n')
			if err != nil {
				log.Println("failed to read password: ", err)
			}
			passwordParsed := strings.TrimSpace(password)
			log.Println("password found:", passwordParsed)
			return

		case NO_PASSWORD_LEFT:
			log.Println("worker exhausted keyspace")
			return

		default:
			log.Println("unknown command:", cmd)
		}

	}
}

func parseFlags() *ControllerArgs {
	port := flag.Int("p", 0, "port to bind to")
	shadowFile := flag.String("f", "", "path to shadow file")
	username := flag.String("u", "", "username to inspect")
	flag.Parse()

	return &ControllerArgs{
		Port:       *port,
		ShadowFile: *shadowFile,
		Username:   *username,
	}
}

func validateArgs(args *ControllerArgs) error {
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

func findUserInShadow(args *ControllerArgs) (*CrackingJob, error) {
	file, err := os.Open(args.ShadowFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open shadow file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		entry, err := parseShadowLine(scanner.Text())
		if err != nil {
			log.Println("Skipping line:", err)
			continue
		}

		if entry.Username == args.Username {
			return entry, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading shadow file: %w", err)
	}

	return nil, fmt.Errorf("user %q not found in shadow file", args.Username)
}

func parseShadowLine(line string) (*CrackingJob, error) {
	fields := strings.Split(line, shadowDelimiter)
	if len(fields) != shadowFieldCount {
		return nil, fmt.Errorf("invalid shadow line format")
	}

	username := fields[0]
	passwordField := fields[1]
	parts := strings.Split(passwordField, "$")

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid password hash format for user %s", username)
	}

	algo := parts[1]

	// 1. Handle case for yescrypt
	switch algo {
	case "y":
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid yescrypt hash for user %s", username)
		}
		return &CrackingJob{
			JobId:    1,
			Username: username,
			Algo:     "$" + parts[1] + "$",
			Cost:     parts[2],
			Salt:     parts[3],
			Hash:     parts[4],
		}, nil

	// 2. Hadle case for bcrypt
	case "2a", "2b", "2c":
		if len(parts) < 4 || len(parts[3]) < 53 {
			return nil, fmt.Errorf("invalid bcrypt hash for user %s", username)
		}
		return &CrackingJob{
			JobId:    1,
			Username: username,
			Algo:     "$" + parts[1] + "$",
			Cost:     parts[2],
			Salt:     parts[3][:22],
			Hash:     parts[3][22:],
		}, nil

	// Handle case for md5, sha512, sha256
	default:
		if len(parts) < 4 {
			return nil, fmt.Errorf("invalid password hash format for user %s", username)
		}

		return &CrackingJob{
			JobId:    1,
			Username: username,
			Algo:     "$" + parts[1] + "$",
			Salt:     parts[2],
			Hash:     parts[3],
		}, nil
	}

}
