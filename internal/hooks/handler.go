// Package hooks manages integration with AI coding tools by installing and
// handling event hooks. It supports Cursor, Claude Code, and Gemini CLI,
// providing real-time event capture and forwarding to the Intentra API.
package hooks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/atbabers/intentra-cli/internal/api"
	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/internal/device"
	"github.com/atbabers/intentra-cli/internal/scanner"
	"github.com/atbabers/intentra-cli/pkg/models"
)

// ProcessEvent reads an event from stdin and sends directly to API.
func ProcessEvent(reader io.Reader, cfg *config.Config, tool string) error {
	// Read input
	bufScanner := bufio.NewScanner(reader)
	if !bufScanner.Scan() {
		if err := bufScanner.Err(); err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		return fmt.Errorf("no input received")
	}

	// Parse event
	var event models.Event
	if err := json.Unmarshal(bufScanner.Bytes(), &event); err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Add device ID (HMAC-immutable identifier)
	if event.DeviceID == "" {
		deviceID, err := device.GetDeviceID()
		if err == nil {
			event.DeviceID = deviceID
		}
	}

	// Add tool identifier
	if tool != "" && event.Tool == "" {
		event.Tool = tool
	}

	// Create a scan from this single event and send immediately
	scan := scanner.CreateScanFromEvent(event)

	// Create API client and send
	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	if err := client.SendScan(scan); err != nil {
		return fmt.Errorf("failed to send scan: %w", err)
	}

	return nil
}

// RunHookHandler is the main entry point for hook processing.
func RunHookHandler() error {
	return RunHookHandlerWithTool("")
}

// RunHookHandlerWithTool processes hooks with tool identifier.
func RunHookHandlerWithTool(tool string) error {
	// Load and validate config - REQUIRED for hooks to work
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Server must be enabled and configured
	if !cfg.Server.Enabled {
		return fmt.Errorf("server sync is not enabled. Set INTENTRA_SERVER_ENDPOINT and INTENTRA_SERVER_ENABLED=true")
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w. Set INTENTRA_API_KEY_ID and INTENTRA_API_SECRET", err)
	}

	return ProcessEvent(os.Stdin, cfg, tool)
}
