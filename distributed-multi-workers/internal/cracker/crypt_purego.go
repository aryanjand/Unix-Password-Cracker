//go:build !cgo_crypt

package cracker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GehirnInc/crypt"
	"github.com/GehirnInc/crypt/md5_crypt"
	"github.com/GehirnInc/crypt/sha256_crypt"
	"github.com/GehirnInc/crypt/sha512_crypt"
	"golang.org/x/crypto/bcrypt"
)

func verifyCandidatePassword(candidate string, hash string) (bool, error) {
	switch {
	case strings.HasPrefix(hash, "$2a$"), strings.HasPrefix(hash, "$2b$"), strings.HasPrefix(hash, "$2y$"):
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(candidate))
		if err == nil {
			return true, nil
		}
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	case strings.HasPrefix(hash, "$1$"):
		return verifyWithCrypter(md5_crypt.New(), candidate, hash)
	case strings.HasPrefix(hash, "$5$"):
		return verifyWithCrypter(sha256_crypt.New(), candidate, hash)
	case strings.HasPrefix(hash, "$6$"):
		return verifyWithCrypter(sha512_crypt.New(), candidate, hash)
	case strings.HasPrefix(hash, "$y$"):
		return false, fmt.Errorf("yescrypt hashes are not supported without -tags cgo_crypt")
	default:
		return false, fmt.Errorf("unrecognized hash format: %q", hash)
	}
}

func verifyWithCrypter(crypter crypt.Crypter, candidate, hash string) (bool, error) {
	err := crypter.Verify(hash, []byte(candidate))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, crypt.ErrKeyMismatch) {
		return false, nil
	}
	return false, err
}
