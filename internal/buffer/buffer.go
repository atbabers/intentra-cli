// Package buffer provides local SQLite-based storage for scan data with
// automatic synchronization to the Intentra server. It enables offline
// operation and resilience during network outages.
package buffer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/atbabers/intentra-cli/internal/api"
	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/pkg/models"
)

// Buffer manages temporary scan storage with automatic sync.
type Buffer struct {
	db     *sql.DB
	cfg    *config.Config
	client *api.Client
	mu     sync.Mutex
}

// Status represents buffer status information.
type Status struct {
	PendingCount int
	SizeBytes    int64
	SizeHuman    string
	OldestAge    string
}

// New creates a new buffer instance.
func New(cfg *config.Config) (*Buffer, error) {
	if !cfg.Buffer.Enabled {
		return nil, fmt.Errorf("buffer is not enabled")
	}

	bufferPath := os.ExpandEnv(cfg.Buffer.Path)
	if bufferPath == "" {
		return nil, fmt.Errorf("buffer path is empty")
	}
	if bufferPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		bufferPath = filepath.Join(home, bufferPath[1:])
	}

	dir := filepath.Dir(bufferPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create buffer directory: %w", err)
	}

	db, err := sql.Open("sqlite3", bufferPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open buffer database: %w", err)
	}

	if err := initSchema(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to initialize schema: %w (also failed to close db: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	buf := &Buffer{
		db:  db,
		cfg: cfg,
	}

	if cfg.Server.Enabled {
		client, err := api.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: API client not available: %v\n", err)
		} else {
			buf.client = client
		}
	}

	if err := buf.purgeOld(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to purge old entries: %v\n", err)
	}

	return buf, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS scans (
		id TEXT PRIMARY KEY,
		payload BLOB NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		synced INTEGER DEFAULT 0,
		sync_attempts INTEGER DEFAULT 0,
		last_error TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_scans_synced ON scans(synced);
	CREATE INDEX IF NOT EXISTS idx_scans_created ON scans(created_at);
	`
	_, err := db.Exec(schema)
	return err
}

// Add adds a scan to the buffer.
func (b *Buffer) Add(scan *models.Scan) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if scan.ID == "" {
		scan.ID = "scan_" + uuid.New().String()[:12]
	}

	payload, err := json.Marshal(scan)
	if err != nil {
		return fmt.Errorf("failed to marshal scan: %w", err)
	}

	_, err = b.db.Exec(
		"INSERT OR REPLACE INTO scans (id, payload, synced) VALUES (?, ?, 0)",
		scan.ID, payload,
	)
	if err != nil {
		return fmt.Errorf("failed to insert scan: %w", err)
	}

	if b.client != nil {
		go b.trySync()
	}

	return nil
}

func (b *Buffer) trySync() {
	b.mu.Lock()
	defer b.mu.Unlock()

	scans, err := b.getPendingLocked(b.cfg.Buffer.FlushThreshold)
	if err != nil || len(scans) == 0 {
		return
	}

	if err := b.client.SendScans(scans); err != nil {
		for _, s := range scans {
			if markErr := b.markFailedLocked(s.ID, err.Error()); markErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to mark scan %s as failed: %v\n", s.ID, markErr)
			}
		}
		return
	}

	for _, s := range scans {
		if markErr := b.markSyncedLocked(s.ID); markErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to mark scan %s as synced: %v\n", s.ID, markErr)
		}
	}
}

func (b *Buffer) getPendingLocked(limit int) ([]*models.Scan, error) {
	rows, err := b.db.Query(
		"SELECT id, payload FROM scans WHERE synced = 0 ORDER BY created_at ASC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []*models.Scan
	for rows.Next() {
		var id string
		var payload []byte
		if err := rows.Scan(&id, &payload); err != nil {
			continue
		}

		var scan models.Scan
		if err := json.Unmarshal(payload, &scan); err != nil {
			continue
		}
		scans = append(scans, &scan)
	}

	return scans, nil
}

func (b *Buffer) markSyncedLocked(id string) error {
	_, err := b.db.Exec("DELETE FROM scans WHERE id = ?", id)
	return err
}

func (b *Buffer) markFailedLocked(id string, errMsg string) error {
	_, err := b.db.Exec(
		"UPDATE scans SET sync_attempts = sync_attempts + 1, last_error = ? WHERE id = ?",
		errMsg, id,
	)
	return err
}

// Flush forces sync of all pending scans.
func (b *Buffer) Flush() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.client == nil {
		return 0, fmt.Errorf("server sync is not configured")
	}

	scans, err := b.getPendingLocked(1000)
	if err != nil {
		return 0, err
	}

	if len(scans) == 0 {
		return 0, nil
	}

	if err := b.client.SendScans(scans); err != nil {
		return 0, err
	}

	for _, s := range scans {
		if err := b.markSyncedLocked(s.ID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to mark scan %s as synced: %v\n", s.ID, err)
		}
	}

	return len(scans), nil
}

// Status returns buffer status information.
func (b *Buffer) Status() (*Status, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var count int
	err := b.db.QueryRow("SELECT COUNT(*) FROM scans WHERE synced = 0").Scan(&count)
	if err != nil {
		return nil, err
	}

	var sizeBytes int64
	if fi, err := os.Stat(b.cfg.Buffer.Path); err == nil {
		sizeBytes = fi.Size()
	}

	var oldestAge string
	var oldestTime sql.NullString
	if err := b.db.QueryRow("SELECT MIN(created_at) FROM scans WHERE synced = 0").Scan(&oldestTime); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to query oldest scan: %v\n", err)
	} else if oldestTime.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", oldestTime.String); err == nil {
			oldestAge = time.Since(t).Round(time.Minute).String()
		}
	}

	return &Status{
		PendingCount: count,
		SizeBytes:    sizeBytes,
		SizeHuman:    humanSize(sizeBytes),
		OldestAge:    oldestAge,
	}, nil
}

// Clear removes all entries from the buffer.
func (b *Buffer) Clear() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var count int
	err := b.db.QueryRow("SELECT COUNT(*) FROM scans").Scan(&count)
	if err != nil {
		return 0, err
	}

	_, err = b.db.Exec("DELETE FROM scans")
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (b *Buffer) purgeOld() error {
	maxAge := time.Duration(b.cfg.Buffer.MaxAgeHours) * time.Hour
	cutoff := time.Now().Add(-maxAge).Format("2006-01-02 15:04:05")

	_, err := b.db.Exec("DELETE FROM scans WHERE created_at < ?", cutoff)
	return err
}

// Close closes the buffer database.
func (b *Buffer) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
