package protocol

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	TCP     = "tcp"
	IDLE    = "IDLE"
	BUSY    = "BUSY"
	READY   = "READY"
	FAILED  = "FAILED"
	SUCCESS = "SUCCESS"
)

type CrackingJob struct {
	Id       int
	Username string
	Setting  string
	FullHash string
}

const (
	shadowFieldCount = 9
	shadowDelimiter  = ":"
)

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
			log.Println("Skipping line:", err)
			continue
		}

		if entry.Username == username {
			return entry, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading shadow file: %w", err)
	}

	return nil, fmt.Errorf("user %q not found in shadow file", username)
}

func parseShadowLine(line string) (*CrackingJob, error) {
	fields := strings.Split(line, shadowDelimiter)
	if len(fields) != shadowFieldCount {
		return nil, fmt.Errorf("invalid shadow line format")
	}

	username := fields[0]
	fullHash := fields[1]
	fmt.Println(fields)

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
	if lastDollar == -1 {
		return nil, fmt.Errorf("invalid crypt format for user %s", username)
	}

	setting := fullHash[:lastDollar]

	return &CrackingJob{
		Id:       1,
		Username: username,
		Setting:  setting,
		FullHash: fullHash,
	}, nil
}
