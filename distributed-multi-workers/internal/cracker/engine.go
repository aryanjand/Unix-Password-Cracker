package cracker

func CrackChunk(start, end uint64, fullHash string) (string, error) {
	for i := start; i < end; i++ {
		candidate := generateNextPassword(i)
		matched, err := verifyCandidatePassword(candidate, fullHash)
		if err != nil {
			return "", err
		}

		if matched {
			return candidate, nil
		}
	}

	return "", nil
}

var charset = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"abcdefghijklmnopqrstuvwxyz" +
	"0123456789" +
	"@#%^&*()_+-=.,:;?")

func generateNextPassword(value uint64) string {
	base := uint64(len(charset))

	result := []rune{}
	for {
		result = append([]rune{charset[value%base]}, result...)
		if value < base {
			break
		}
		value = value/base - 1
	}

	return string(result)
}
