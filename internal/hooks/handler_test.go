package hooks

import (
	"bytes"
	"strings"
	"testing"

	"github.com/atbabers/intentra-cli/internal/config"
)

func TestProcessEvent_ParsesEvent(t *testing.T) {
	// Create a config (will fail on API send, but should parse event)
	cfg := config.DefaultConfig()
	cfg.Server.Enabled = true
	cfg.Server.Endpoint = "http://localhost:9999/v1" // Non-existent endpoint
	cfg.Server.Auth.Mode = "hmac"
	cfg.Server.Auth.HMAC.KeyID = "test-key"
	cfg.Server.Auth.HMAC.Secret = "test-secret"

	input := `{"hook_type": "afterAgentResponse", "conversation_id": "test-123"}`
	reader := bytes.NewBufferString(input)

	// This will fail on API send (connection refused), but should parse the event
	err := ProcessEvent(reader, cfg, "cursor")
	if err == nil {
		t.Error("Expected error due to non-existent API endpoint")
	}
	// Should fail on connection, not parsing
	if !strings.Contains(err.Error(), "failed to send scan") {
		t.Errorf("Expected 'failed to send scan' error, got: %v", err)
	}
}

func TestRunHookHandlerWithTool_RequiresConfig(t *testing.T) {
	// Clear environment to ensure no config
	t.Setenv("INTENTRA_SERVER_ENDPOINT", "")
	t.Setenv("INTENTRA_SERVER_ENABLED", "")

	err := RunHookHandlerWithTool("cursor")
	if err == nil {
		t.Error("Expected error when server not configured")
	}
}
