package protocol

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	TCP      = "tcp"
	IDLE     = "IDLE"
	BUSY     = "BUSY"
	READY    = "READY"
	FAILED   = "FAILED"
	SUCCESS  = "SUCCESS"
	SHUTDOWN = "SHUTDOWN"
)

type CrackingJob struct {
	Id       int
	Username string
	Setting  string
	FullHash string
}

type WorkerMetrics struct {
	TotalCrackingTimeNanos int64 `json:"total_cracking_time_ns"`
	WorkerReceiveJobNanos  int64 `json:"worker_receive_job_ns"`
	WorkerSentResultsNanos int64 `json:"worker_sent_results_ns"`
}

type CrackResult struct {
	Password string        `json:"password"`
	Metrics  WorkerMetrics `json:"metrics"`
}

type WorkerMessage struct {
	Status string `json:"status"`
}

func FindUserInShadow(filePath string, username string) (*CrackingJob, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open shadow file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		entry, err := parseShadowLine(scanner.Text())
		if err != nil {
			continue
		}

		if entry.Username == username {
			return entry, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading shadow file failed: %w", err)
	}

	return nil, fmt.Errorf("user %q not found in shadow file", username)
}

func parseShadowLine(line string) (*CrackingJob, error) {
	fields := strings.Split(line, ":")
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid shadow line format")
	}

	username := fields[0]
	fullHash := fields[1]

	// Locked / disabled accounts
	if fullHash == "!" || fullHash == "*" {
		return nil, fmt.Errorf("account %s has no valid password", username)
	}

	// crypt format always starts with $
	if !strings.HasPrefix(fullHash, "$") {
		return nil, fmt.Errorf("unsupported hash format for user %s", username)
	}

	// Remove trailing hash part to get the setting
	lastDollar := strings.LastIndex(fullHash, "$")
	setting := fullHash[:lastDollar]

	return &CrackingJob{
		Id:       1,
		Username: username,
		Setting:  setting,
		FullHash: fullHash,
	}, nil
}
