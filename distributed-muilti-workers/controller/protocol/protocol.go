package protocol

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type Command string

const (
	MsgReady     Command = "ready"
	MsgJob       Command = "job"
	MsgHeartbeat Command = "heartbeat"
	MsgResult    Command = "result"
	MsgError     Command = "error"
	MsgShutdown  Command = "shutdown"
)

type Message struct {
	Command Command `json:"command"`

	Job       *CrackingJob       `json:"job,omitempty"`
	Result    *CrackResult       `json:"result,omitempty"`
	Heartbeat *HeartbeatResponse `json:"heartbeat,omitempty"`
	Error     string             `json:"error,omitempty"`
}

type CrackingJob struct {
	Id       int
	Interval int
	Username string
	Setting  string
	FullHash string
}

// CrackResult sent from Worker -> Controller
type CrackResult struct {
	Password string        `json:"password"`
	Metrics  WorkerMetrics `json:"metrics"`
}

type WorkerMetrics struct {
	TotalCrackingTimeNanos int64     `json:"total_cracking_time_ns"`
	WorkerReceiveJobNanos  time.Time `json:"worker_receive_job_ns"`
	WorkerSentResultsNanos time.Time `json:"worker_sent_results_ns"`
}

type HeartbeatResponse struct {
	DeltaTested   int64   `json:"delta_tested"`
	TotalTested   int64   `json:"total_tested"`
	ThreadsActive int64   `json:"threads_active"`
	CurrentRate   float64 `json:"current_rate"`
}

func FindUserInShadow(filePath string, username string) (*CrackingJob, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open shadow file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		crackingJob, err := parseShadowLine(scanner.Text())
		if err != nil {
			continue
		}

		if crackingJob.Username == username {
			return crackingJob, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading shadow file failed: %w", err)
	}

	return nil, fmt.Errorf("user %q not found in shadow file", username)
}

// Parse's Shadow line and creates CrackingJob
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
		Interval: 0.0,
		Username: username,
		Setting:  setting,
		FullHash: fullHash,
	}, nil
}
