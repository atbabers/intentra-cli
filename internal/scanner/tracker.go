package scanner

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/pkg/models"
)

// validScanIDPattern validates scan IDs to prevent path traversal attacks.
// Only allows alphanumeric characters, underscores, and hyphens.
var validScanIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ErrInvalidScanID is returned when a scan ID contains invalid characters.
var ErrInvalidScanID = errors.New("invalid scan ID: must contain only alphanumeric characters, underscores, and hyphens")

// LoadEvents reads all events from events.jsonl.
func LoadEvents() ([]models.Event, error) {
	eventsFile := config.GetEventsFile()

	f, err := os.Open(eventsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []models.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var event models.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue // Skip malformed lines
		}
		events = append(events, event)
	}

	return events, scanner.Err()
}

// validateScanID checks if a scan ID is safe to use in file paths.
func validateScanID(id string) error {
	if id == "" {
		return ErrInvalidScanID
	}
	if len(id) > 128 {
		return errors.New("invalid scan ID: exceeds maximum length of 128 characters")
	}
	if !validScanIDPattern.MatchString(id) {
		return ErrInvalidScanID
	}
	return nil
}

// SaveScan writes a scan to the scans directory.
func SaveScan(scan *models.Scan) error {
	// Validate scan ID to prevent path traversal
	if err := validateScanID(scan.ID); err != nil {
		return err
	}

	scansDir := config.GetScansDir()
	if err := os.MkdirAll(scansDir, 0700); err != nil {
		return err
	}

	filename := filepath.Join(scansDir, scan.ID+".json")
	data, err := json.MarshalIndent(scan, "", "  ")
	if err != nil {
		return err
	}

	// Use 0600 for user-only read/write
	return os.WriteFile(filename, data, 0600)
}

// LoadScans reads all scans from the scans directory.
func LoadScans() ([]models.Scan, error) {
	scansDir := config.GetScansDir()

	entries, err := os.ReadDir(scansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var scans []models.Scan
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(scansDir, entry.Name()))
		if err != nil {
			continue
		}

		var scan models.Scan
		if err := json.Unmarshal(data, &scan); err != nil {
			continue
		}
		scans = append(scans, scan)
	}

	return scans, nil
}

// LoadScan reads a single scan by ID.
func LoadScan(id string) (*models.Scan, error) {
	// Validate scan ID to prevent path traversal attacks (e.g., "../../../etc/passwd")
	if err := validateScanID(id); err != nil {
		return nil, err
	}

	scansDir := config.GetScansDir()
	filename := filepath.Join(scansDir, id+".json")

	// Additional safety: verify the resolved path is within scansDir
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	absScansDir, err := filepath.Abs(scansDir)
	if err != nil {
		return nil, err
	}
	relPath, err := filepath.Rel(absScansDir, absFilename)
	if err != nil || relPath == ".." || len(relPath) > 2 && relPath[:3] == "../" {
		return nil, ErrInvalidScanID
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var scan models.Scan
	if err := json.Unmarshal(data, &scan); err != nil {
		return nil, err
	}

	return &scan, nil
}
