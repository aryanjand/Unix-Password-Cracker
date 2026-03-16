package controller

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
