// Package models provides data structures and types used throughout Intentra.
// It defines events, scans, and their associated metadata for tracking AI
// coding tool activity.
package models

import (
	"encoding/json"
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

	// Content varies by hook type
	Prompt        string          `json:"prompt,omitempty"`
	Response      string          `json:"response,omitempty"`
	Thought       string          `json:"thought,omitempty"`
	ToolName      string          `json:"tool_name,omitempty"`
	ToolInput     json.RawMessage `json:"tool_input,omitempty"`
	ToolOutput    json.RawMessage `json:"tool_output,omitempty"`
	FilePath      string          `json:"file_path,omitempty"`
	Command       string          `json:"command,omitempty"`
	CommandOutput string          `json:"command_output,omitempty"`

	// Metrics
	InputTokens    int `json:"input_tokens,omitempty"`
	OutputTokens   int `json:"output_tokens,omitempty"`
	ThinkingTokens int `json:"thinking_tokens,omitempty"`
	DurationMs     int `json:"duration_ms,omitempty"`

	// Error tracking
	Error string `json:"error,omitempty"`
}

// EstimateTokens estimates tokens from text length.
func EstimateTokens(text string, charsPerToken int) int {
	if charsPerToken <= 0 {
		charsPerToken = 4
	}
	return len(text) / charsPerToken
}
