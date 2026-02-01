package hooks

import (
	"bytes"
	"testing"

	"github.com/atbabers/intentra-cli/internal/config"
)

func TestProcessEvent_ParsesEvent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.Enabled = true
	cfg.Server.Endpoint = "http://localhost:9999/v1"
	cfg.Server.Auth.Mode = "hmac"
	cfg.Server.Auth.HMAC.KeyID = "test-key"
	cfg.Server.Auth.HMAC.Secret = "test-secret"

	promptInput := `{"conversation_id": "test-123"}`
	promptReader := bytes.NewBufferString(promptInput)
	err := ProcessEventWithEvent(promptReader, cfg, "cursor", "beforeSubmitPrompt")
	if err != nil {
		t.Errorf("Unexpected error buffering prompt event: %v", err)
	}

	stopInput := `{"conversation_id": "test-123"}`
	stopReader := bytes.NewBufferString(stopInput)
	err = ProcessEventWithEvent(stopReader, cfg, "cursor", "stop")
	if err != nil {
		t.Errorf("Unexpected error on stop event: %v", err)
	}
}

func TestRunHookHandlerWithTool_RequiresConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.Enabled = false

	emptyInput := bytes.NewBufferString("")
	err := ProcessEventWithEvent(emptyInput, cfg, "cursor", "stop")
	if err != nil {
		t.Errorf("Empty input should not return error, got: %v", err)
	}

	cfg.Server.Enabled = true
	cfg.Server.Endpoint = ""
	validInput := bytes.NewBufferString(`{"conversation_id": "test"}`)
	err = ProcessEventWithEvent(validInput, cfg, "cursor", "stop")
	if err != nil {
		t.Errorf("Should not error with empty endpoint (fails silently), got: %v", err)
	}
}
