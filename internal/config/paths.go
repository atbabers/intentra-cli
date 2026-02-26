package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigDir returns the OS-appropriate config directory.
// Panics if the home directory cannot be determined and no override is set,
// since all downstream callers depend on a valid path.
func GetConfigDir() string {
	// Allow override for testing
	if dir := os.Getenv("INTENTRA_CONFIG_DIR"); dir != "" {
		return dir
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "intentra")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot determine home directory: %v\n", err)
			fmt.Fprintf(os.Stderr, "Set INTENTRA_CONFIG_DIR to override.\n")
			os.Exit(1)
		}
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

// GetCredentialsFile returns the path to the credentials file.
func GetCredentialsFile() string {
	return filepath.Join(GetConfigDir(), "credentials.json")
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
