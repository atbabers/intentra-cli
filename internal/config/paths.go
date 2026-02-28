package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigDir returns the OS-appropriate config directory.
// Returns an error if the home directory cannot be determined and no override is set.
func GetConfigDir() (string, error) {
	if dir := os.Getenv("INTENTRA_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "intentra"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory (set INTENTRA_CONFIG_DIR to override): %w", err)
		}
		return filepath.Join(home, ".intentra"), nil
	}
}

// GetDataDir returns the data directory (same as config for now).
func GetDataDir() (string, error) {
	return GetConfigDir()
}

// GetEventsFile returns the path to events.jsonl.
func GetEventsFile() (string, error) {
	dir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "events.jsonl"), nil
}

// GetScansDir returns the scans directory.
func GetScansDir() (string, error) {
	dir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "scans"), nil
}

// GetEvidenceDir returns the evidence directory.
func GetEvidenceDir() (string, error) {
	dir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "evidence"), nil
}

// GetCredentialsFile returns the path to the credentials file.
func GetCredentialsFile() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

// GetConfigPath returns the path to the config file.
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// EnsureDirectories creates all required directories.
func EnsureDirectories() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	scansDir, err := GetScansDir()
	if err != nil {
		return err
	}
	evidenceDir, err := GetEvidenceDir()
	if err != nil {
		return err
	}

	dirs := []string{
		configDir,
		scansDir,
		evidenceDir,
		filepath.Join(evidenceDir, "reviewed"),
		filepath.Join(evidenceDir, "rejected"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}
