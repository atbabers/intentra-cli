package models

import (
	"encoding/json"
	"testing"
)

func TestEventUnmarshal(t *testing.T) {
	jsonData := `{
		"hook_type": "afterAgentResponse",
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
}

func TestViolationSeverity(t *testing.T) {
	v := Violation{Type: ViolationRetryLoop}
	if v.Type.Severity() != SeverityHigh {
		t.Errorf("retry_loop should be high severity")
	}
}
