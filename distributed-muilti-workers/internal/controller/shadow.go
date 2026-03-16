package controller

import (
	"bufio"
	"fmt"
	"os"

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
