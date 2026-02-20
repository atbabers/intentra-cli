// Package models provides data structures and types used throughout Intentra.
// It defines events, scans, and their associated metadata for tracking AI
// coding tool activity.
package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// HookType represents the native hook event type from AI coding tools.
// This is the raw event type as received from the tool (camelCase, PascalCase, or snake_case).
type HookType string

// Event represents a single hook event from AI coding tools.
type Event struct {
	HookType       HookType  `json:"hook_type"`
	NormalizedType string    `json:"normalized_type"`
	Timestamp      time.Time `json:"timestamp"`
	ScanID         string    `json:"scan_id,omitempty"`
	ConversationID string    `json:"conversation_id"`
	SessionID      string    `json:"session_id,omitempty"`
	GenerationID   string    `json:"generation_id,omitempty"`
	Model          string    `json:"model,omitempty"`
	UserEmail      string    `json:"user_email,omitempty"`
	DeviceID       string    `json:"device_id,omitempty"`
	Tool           string    `json:"tool,omitempty"`

	Prompt        string          `json:"prompt,omitempty"`
	Response      string          `json:"response,omitempty"`
	Thought       string          `json:"thought,omitempty"`
	ToolName      string          `json:"tool_name,omitempty"`
	ToolInput     json.RawMessage `json:"tool_input,omitempty"`
	ToolOutput    json.RawMessage `json:"tool_output,omitempty"`
	FilePath      string          `json:"file_path,omitempty"`
	Command       string          `json:"command,omitempty"`
	CommandOutput string          `json:"command_output,omitempty"`

	MCPServerName string `json:"mcp_server_name,omitempty"`
	MCPToolName   string `json:"mcp_tool_name,omitempty"`
	MCPServerURL  string `json:"mcp_server_url,omitempty"`
	MCPServerCmd  string `json:"mcp_server_cmd,omitempty"`

	InputTokens    int `json:"input_tokens,omitempty"`
	OutputTokens   int `json:"output_tokens,omitempty"`
	ThinkingTokens int `json:"thinking_tokens,omitempty"`
	DurationMs     int `json:"duration_ms,omitempty"`

	ContextUsagePercent int    `json:"context_usage_percent,omitempty"`
	ContextTokens       int    `json:"context_tokens,omitempty"`
	ContextWindowSize   int    `json:"context_window_size,omitempty"`
	MessageCount        int    `json:"message_count,omitempty"`
	MessagesToCompact   int    `json:"messages_to_compact,omitempty"`
	IsFirstCompaction   *bool  `json:"is_first_compaction,omitempty"`
	CompactionTrigger   string `json:"compaction_trigger,omitempty"`

	Error string `json:"error,omitempty"`
}

// IsMCPEvent returns true if this event is an MCP tool invocation.
func (e *Event) IsMCPEvent() bool {
	return e.MCPServerName != "" || e.MCPToolName != ""
}

// SanitizeMCPServerURL strips query parameters from a URL to prevent leaking API keys.
// Returns only scheme + host + path.
func SanitizeMCPServerURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

// SanitizeMCPServerCmd extracts only the binary name from a command path.
// Prevents leaking local directory structures.
func SanitizeMCPServerCmd(cmd string) string {
	if cmd == "" {
		return ""
	}
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	return filepath.Base(parts[0])
}

// ParseMCPDoubleUnderscoreName parses Claude Code and Gemini CLI MCP tool names
// in the format mcp__<server>__<tool>. Splits on the first two __ delimiters only.
func ParseMCPDoubleUnderscoreName(toolName string) (serverName, mcpToolName string, ok bool) {
	if !strings.HasPrefix(toolName, "mcp__") {
		return "", "", false
	}
	rest := toolName[5:]
	idx := strings.Index(rest, "__")
	if idx < 0 {
		return rest, "", true
	}
	return rest[:idx], rest[idx+2:], true
}

// MCPServerURLHash returns a short hash of the sanitized server URL or command.
// Used as a deduplication key alongside server name.
func MCPServerURLHash(serverURL, serverCmd string) string {
	input := serverURL + "|" + serverCmd
	if input == "|" {
		return ""
	}
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:8]
}
