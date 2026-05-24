//go:build windows

package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	lastConfigFileName     = "last-config.txt"
	maxLastConfigPathBytes = 32 * 1024
)

// lastConfigStatePath returns the path to the last-used config state file next to the executable.
func lastConfigStatePath() string {
	executable, err := os.Executable()
	if err != nil {
		return lastConfigFileName
	}
	return filepath.Join(filepath.Dir(executable), lastConfigFileName)
}

// loadLastConfigPath reads the config path stored in last-config.txt.
// Returns "" if the file is absent, unreadable, oversized, or contains a relative path.
func loadLastConfigPath() string {
	f, err := os.Open(lastConfigStatePath())
	if err != nil {
		return ""
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxLastConfigPathBytes+1))
	if err != nil {
		return ""
	}
	if len(data) > maxLastConfigPathBytes {
		return ""
	}
	path := strings.TrimSpace(string(data))
	if !filepath.IsAbs(path) {
		return ""
	}
	return path
}

// saveLastConfigPath persists configPath to last-config.txt.
// Errors are silently discarded — this is best-effort state.
func saveLastConfigPath(configPath string) {
	_ = os.WriteFile(lastConfigStatePath(), []byte(configPath), 0o600)
}
