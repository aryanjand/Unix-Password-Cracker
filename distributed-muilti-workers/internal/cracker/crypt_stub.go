//go:build !linux || !cgo

package cracker

import "fmt"

func verifyCandidatePassword(candidate, hash string) (bool, error) {
	return false, fmt.Errorf("crypt verification requires linux with cgo enabled")
}
