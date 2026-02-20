package models

import (
	"encoding/json"
	"testing"
)

func TestEventUnmarshal(t *testing.T) {
	jsonData := `{
		"hook_type": "afterAgentResponse",
		"normalized_type": "after_response",
		"timestamp": "2025-01-07T10:30:00Z",
		"conversation_id": "conv-123",
		"model": "claude-3-5-sonnet"
	}`

	var event Event
	if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if event.HookType != "afterAgentResponse" {
		t.Errorf("Expected afterAgentResponse, got %s", event.HookType)
	}
	if event.NormalizedType != "after_response" {
		t.Errorf("Expected after_response normalized type, got %s", event.NormalizedType)
	}
}

func TestEventUnmarshal_WithNewFields(t *testing.T) {
	jsonData := `{
		"hook_type": "afterTool",
		"normalized_type": "after_tool",
		"timestamp": "2025-01-07T10:30:00Z",
		"conversation_id": "conv-123",
		"generation_id": "gen-456",
		"error": "tool execution failed"
	}`

	var event Event
	if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if event.GenerationID != "gen-456" {
		t.Errorf("Expected gen-456, got %s", event.GenerationID)
	}
	if event.Error != "tool execution failed" {
		t.Errorf("Expected error field, got %s", event.Error)
	}
}

func TestScanMarshal(t *testing.T) {
	scan := Scan{
		ID:           "scan-123",
		Tool:         "cursor",
		Fingerprint:  "abc123",
		FilesHash:    "def456",
		ActionCounts: map[string]int{"edits": 5, "reads": 10},
	}

	data, err := json.Marshal(scan)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result Scan
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Fingerprint != "abc123" {
		t.Errorf("Expected fingerprint abc123, got %s", result.Fingerprint)
	}
	if result.FilesHash != "def456" {
		t.Errorf("Expected files_hash def456, got %s", result.FilesHash)
	}
	if result.ActionCounts["edits"] != 5 {
		t.Errorf("Expected edits count 5, got %d", result.ActionCounts["edits"])
	}
}
