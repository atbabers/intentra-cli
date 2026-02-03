// Package api provides HTTP client functionality for communicating with the
// Intentra server. It supports JWT authentication (via intentra login) and
// API key authentication (Enterprise).
package api

import (
	"bytes"
	cryptoRand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/atbabers/intentra-cli/internal/auth"
	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/internal/debug"
	"github.com/atbabers/intentra-cli/internal/device"
	"github.com/atbabers/intentra-cli/pkg/models"
)

// ScansResponse represents the response from GET /scans.
type ScansResponse struct {
	Scans   []models.Scan `json:"scans"`
	Summary ScansSummary  `json:"summary"`
}

// ScansSummary contains aggregated scan statistics.
type ScansSummary struct {
	TotalScans          int     `json:"total_scans"`
	TotalCost           float64 `json:"total_cost"`
	ScansWithViolations int     `json:"scans_with_violations"`
}

// ScanDetailResponse represents the response from GET /scans/{id}.
type ScanDetailResponse struct {
	Scan             models.Scan       `json:"scan"`
	ViolationDetails map[string]string `json:"violation_details,omitempty"`
}

// Client handles communication with the Intentra API.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new API client configured with the provided settings.
func NewClient(cfg *config.Config) (*Client, error) {
	if !cfg.Server.Enabled {
		return nil, fmt.Errorf("server sync is not enabled")
	}

	httpClient := &http.Client{
		Timeout: cfg.Server.Timeout,
	}

	return &Client{
		cfg:        cfg,
		httpClient: httpClient,
	}, nil
}

// SendScan sends a single scan to the API.
func (c *Client) SendScan(scan *models.Scan) error {
	deviceID, err := c.getDeviceID()
	if err != nil {
		return fmt.Errorf("failed to get device ID: %w", err)
	}

	durationMs := int64(0)
	if !scan.EndTime.IsZero() && !scan.StartTime.IsZero() {
		durationMs = scan.EndTime.Sub(scan.StartTime).Milliseconds()
	}

	sessionID := ""
	if scan.Source != nil {
		sessionID = scan.Source.SessionID
	}

	var events []map[string]any
	if len(scan.RawEvents) > 0 {
		events = scan.RawEvents
	} else {
		for _, ev := range scan.Events {
			evMap := map[string]any{
				"hook_type":       string(ev.HookType),
				"normalized_type": ev.NormalizedType,
				"timestamp":       ev.Timestamp.Format(time.RFC3339Nano),
				"tool_name":       ev.ToolName,
				"command":         ev.Command,
				"command_output":  ev.CommandOutput,
				"file_path":       ev.FilePath,
				"prompt":          ev.Prompt,
				"response":        ev.Response,
				"thought":         ev.Thought,
				"duration_ms":     ev.DurationMs,
				"conversation_id": ev.ConversationID,
				"session_id":      ev.SessionID,
				"tokens": map[string]int{
					"input":    ev.InputTokens,
					"output":   ev.OutputTokens,
					"thinking": ev.ThinkingTokens,
				},
			}
			if len(ev.ToolInput) > 0 {
				var toolInput map[string]any
				if err := json.Unmarshal(ev.ToolInput, &toolInput); err == nil {
					evMap["tool_input"] = toolInput
				}
			}
			if len(ev.ToolOutput) > 0 {
				var toolOutput any
				if err := json.Unmarshal(ev.ToolOutput, &toolOutput); err == nil {
					evMap["tool_output"] = toolOutput
				}
			}
			events = append(events, evMap)
		}
	}

	body := map[string]any{
		"tool":            scan.Tool,
		"started_at":      scan.StartTime.Format(time.RFC3339Nano),
		"ended_at":        scan.EndTime.Format(time.RFC3339Nano),
		"duration_ms":     durationMs,
		"llm_call_count":  scan.LLMCalls,
		"total_tokens":    scan.TotalTokens,
		"estimated_cost":  scan.EstimatedCost,
		"events":          events,
		"device_id":       deviceID,
		"conversation_id": scan.ConversationID,
		"session_id":      sessionID,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scan: %w", err)
	}

	url := c.cfg.Server.Endpoint + "/scans"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "intentra-cli/1.0")

	if err := c.addAuth(req, jsonBody); err != nil {
		return fmt.Errorf("failed to add auth: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		debug.LogHTTP("POST", url, 0)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	debug.LogHTTP("POST", url, resp.StatusCode)

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendScans sends a batch of scans to the API by calling SendScan for each.
func (c *Client) SendScans(scans []*models.Scan) error {
	for _, scan := range scans {
		if err := c.SendScan(scan); err != nil {
			return err
		}
	}
	return nil
}

// Health checks API connectivity.
func (c *Client) Health() error {
	url := c.cfg.Server.Endpoint + "/health"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		debug.LogHTTP("GET", url, 0)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	debug.LogHTTP("GET", url, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned: %d", resp.StatusCode)
	}

	return nil
}

// getDeviceID returns the device ID (auto-generated from hardware).
func (c *Client) getDeviceID() (string, error) {
	return device.GetDeviceID()
}

// addAuth adds authentication headers based on config.
// Priority: JWT credentials (from 'intentra login') > config auth mode (api_key)
func (c *Client) addAuth(req *http.Request, body []byte) error {
	creds := auth.GetValidCredentials()
	if creds != nil {
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
		deviceID, err := c.getDeviceID()
		if err != nil {
			return fmt.Errorf("failed to get device ID: %w", err)
		}
		req.Header.Set("X-Machine-ID", deviceID)
		return nil
	}

	switch c.cfg.Server.Auth.Mode {
	case "api_key":
		return c.addAPIKeyAuth(req)
	default:
		return fmt.Errorf("not authenticated - run 'intentra login' or configure api_key auth in config.yaml")
	}
}

// addAPIKeyAuth adds API key authentication headers for Enterprise organizations.
// Server expects: X-API-Key-ID, X-API-Key-Secret, X-API-Timestamp, X-API-Nonce
// The secret is sent directly and verified using bcrypt on the server.
func (c *Client) addAPIKeyAuth(req *http.Request) error {
	keyID := c.cfg.Server.Auth.APIKey.KeyID
	secret := c.cfg.Server.Auth.APIKey.Secret

	if keyID == "" || secret == "" {
		return fmt.Errorf("API key auth requires key_id and secret")
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	nonceBytes := make([]byte, 16)
	if _, err := io.ReadFull(cryptoRand.Reader, nonceBytes); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)

	req.Header.Set("X-API-Key-ID", keyID)
	req.Header.Set("X-API-Key-Secret", secret)
	req.Header.Set("X-API-Timestamp", timestamp)
	req.Header.Set("X-API-Nonce", nonce)

	return nil
}

// addJWTAuth adds JWT Bearer token authentication from stored credentials.
func (c *Client) addJWTAuth(req *http.Request) error {
	creds := auth.GetValidCredentials()
	if creds == nil {
		return fmt.Errorf("not authenticated - run 'intentra login' first")
	}

	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	deviceID, err := c.getDeviceID()
	if err != nil {
		return fmt.Errorf("failed to get device ID: %w", err)
	}
	req.Header.Set("X-Machine-ID", deviceID)

	return nil
}

// GetScans retrieves scans from the API.
func (c *Client) GetScans(days, limit int) (*ScansResponse, error) {
	if days <= 0 {
		days = 30
	}
	if limit <= 0 {
		limit = 50
	}

	url := fmt.Sprintf("%s/scans?days=%d&limit=%d", c.cfg.Server.Endpoint, days, limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "intentra-cli/1.0")

	if err := c.addJWTAuth(req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		debug.LogHTTP("GET", url, 0)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	debug.LogHTTP("GET", url, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed - run 'intentra login' to re-authenticate")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var result ScansResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetScan retrieves a single scan by ID from the API.
func (c *Client) GetScan(scanID string) (*ScanDetailResponse, error) {
	if scanID == "" {
		return nil, fmt.Errorf("scan ID is required")
	}

	url := fmt.Sprintf("%s/scans/%s", c.cfg.Server.Endpoint, scanID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "intentra-cli/1.0")

	if err := c.addJWTAuth(req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		debug.LogHTTP("GET", url, 0)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	debug.LogHTTP("GET", url, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed - run 'intentra login' to re-authenticate")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("scan not found: %s", scanID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var result ScanDetailResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
