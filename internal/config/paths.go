package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigDir returns the OS-appropriate config directory.
func GetConfigDir() string {
	// Allow override for testing
	if dir := os.Getenv("INTENTRA_CONFIG_DIR"); dir != "" {
		return dir
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "intentra")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".intentra")
	}
}

// GetDataDir returns the data directory (same as config for now).
func GetDataDir() string {
	return GetConfigDir()
}

// GetEventsFile returns the path to events.jsonl.
func GetEventsFile() string {
	return filepath.Join(GetDataDir(), "events.jsonl")
}

// GetScansDir returns the scans directory.
func GetScansDir() string {
	return filepath.Join(GetDataDir(), "scans")
}

// GetEvidenceDir returns the evidence directory.
func GetEvidenceDir() string {
	return filepath.Join(GetDataDir(), "evidence")
}

// EnsureDirectories creates all required directories.
func EnsureDirectories() error {
	dirs := []string{
		GetConfigDir(),
		GetScansDir(),
		GetEvidenceDir(),
		filepath.Join(GetEvidenceDir(), "reviewed"),
		filepath.Join(GetEvidenceDir(), "rejected"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}
