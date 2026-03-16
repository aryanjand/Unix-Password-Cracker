package controller

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

// Finds the user in shadow file and returns the protocol.ShadowEntry
func FindUserInShadow(filePath string, username string) (protocol.ShadowEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return protocol.ShadowEntry{}, fmt.Errorf("failed to open shadow file: %w", err)
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
		return protocol.ShadowEntry{}, fmt.Errorf("reading shadow file failed: %w", err)
	}

	return protocol.ShadowEntry{}, fmt.Errorf("user %q not found in shadow file", username)
}

// Parse's Shadow line and creates protocol.ShadowEntry
func parseShadowLine(line string) (protocol.ShadowEntry, error) {
	fields := strings.Split(line, ":")
	if len(fields) < 2 {
		return protocol.ShadowEntry{}, fmt.Errorf("invalid shadow line format")
	}

	username := fields[0]
	fullHash := fields[1]

	// Locked / disabled accounts
	if fullHash == "!" || fullHash == "*" {
		return protocol.ShadowEntry{}, fmt.Errorf("account %s has no valid password", username)
	}

	// crypt format always starts with $
	if !strings.HasPrefix(fullHash, "$") {
		return protocol.ShadowEntry{}, fmt.Errorf("unsupported hash format for user %s", username)
	}

	// Remove trailing hash part to get the setting
	lastDollar := strings.LastIndex(fullHash, "$")
	setting := fullHash[:lastDollar]

	entry := protocol.ShadowEntry{
		Username: username,
		Settings: setting,
		FullHash: fullHash,
	}

	return entry, nil
}
