package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentrahq/intentra-cli/pkg/models"
)

func TestWriteSendPayload_SendScan(t *testing.T) {
	scan := &models.Scan{
		ID:             "scan_test123",
		Tool:           "claude",
		ConversationID: "conv-1",
	}

	path, err := writeSendPayload("send_scan", scan, "", "", "", 0)
	if err != nil {
		t.Fatalf("writeSendPayload failed: %v", err)
	}
	defer os.Remove(path)

	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read payload file: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}

	if payload["action"] != "send_scan" {
		t.Errorf("expected action send_scan, got: %v", payload["action"])
	}
	if payload["scan"] == nil {
		t.Error("expected scan in payload")
	}
}

func TestWriteSendPayload_PatchSessionEnd(t *testing.T) {
	path, err := writeSendPayload("patch_session_end", nil, "scan_abc", "key1", "user_exit", 45000)
	if err != nil {
		t.Fatalf("writeSendPayload failed: %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read payload file: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}

	if payload["action"] != "patch_session_end" {
		t.Errorf("expected action patch_session_end, got: %v", payload["action"])
	}
	if payload["scan_id"] != "scan_abc" {
		t.Errorf("expected scan_id scan_abc, got: %v", payload["scan_id"])
	}
	if payload["reason"] != "user_exit" {
		t.Errorf("expected reason user_exit, got: %v", payload["reason"])
	}
}

func TestCleanupStalePayloads(t *testing.T) {
	// Create a fake stale payload file
	f, err := os.CreateTemp("", "intentra_send_*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(`{"action":"test"}`)); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Backdate it beyond maxBufferAge (30 minutes)
	staleTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(f.Name(), staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	// Remove the cleanup marker to force a cleanup run
	os.Remove(filepath.Join(os.TempDir(), cleanupMarkerFile))

	cleanupStaleBuffers()

	if _, err := os.Stat(f.Name()); !os.IsNotExist(err) {
		t.Errorf("stale payload file should have been cleaned up: %s", f.Name())
		os.Remove(f.Name())
	}
}
