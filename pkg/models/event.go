// Package models provides data structures and types used throughout Intentra.
// It defines events, scans, violations, and their associated metadata for
// tracking AI coding tool activity and detecting policy violations.
package models

import (
	"encoding/json"
	"time"
)

// HookType represents the type of Cursor hook event.
type HookType string

const (
	HookBeforeSubmitPrompt  HookType = "beforeSubmitPrompt"
	HookAfterAgentThought   HookType = "afterAgentThought"
	HookAfterAgentResponse  HookType = "afterAgentResponse"
	HookAfterMCPExecution   HookType = "afterMCPExecution"
	HookAfterShellExecution HookType = "afterShellExecution"
	HookAfterFileEdit       HookType = "afterFileEdit"
	HookAfterTabFileEdit    HookType = "afterTabFileEdit"
	HookStop                HookType = "stop"
)

// Event represents a single hook event from Cursor or Claude Code.
type Event struct {
	HookType       HookType  `json:"hook_type"`
	Timestamp      time.Time `json:"timestamp"`
	ScanID         string    `json:"scan_id,omitempty"` // Parent scan ID for tracing
	ConversationID string    `json:"conversation_id"`
	SessionID      string    `json:"session_id,omitempty"`
	Model          string    `json:"model,omitempty"`
	UserEmail      string    `json:"user_email,omitempty"`
	DeviceID       string    `json:"device_id,omitempty"` // HMAC-immutable device identifier
	Tool           string    `json:"tool,omitempty"`      // cursor, claude

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
}

// EstimateTokens estimates tokens from text length.
func EstimateTokens(text string, charsPerToken int) int {
	if charsPerToken <= 0 {
		charsPerToken = 4
	}
	return len(text) / charsPerToken
}

// Normalized event type constants.
const (
	EventSessionStart   = "session_start"
	EventSessionEnd     = "session_end"
	EventBeforePrompt   = "before_prompt"
	EventAfterResponse  = "after_response"
	EventAgentThought   = "agent_thought"
	EventBeforeTool     = "before_tool"
	EventAfterTool      = "after_tool"
	EventToolSelection  = "tool_selection"
	EventBeforeShell    = "before_shell"
	EventAfterShell     = "after_shell"
	EventBeforeFileRead = "before_file_read"
	EventAfterFileEdit  = "after_file_edit"
	EventBeforeMCP      = "before_mcp"
	EventAfterMCP       = "after_mcp"
	EventBeforeModel    = "before_model"
	EventAfterModel     = "after_model"
	EventPermission     = "permission_request"
	EventNotification   = "notification"
	EventStop           = "stop"
	EventSubagentStop   = "subagent_stop"
	EventPreCompact     = "pre_compact"
)

// NormalizedEvent is the tool-agnostic event format for detectors.
type NormalizedEvent struct {
	// Source identification
	Tool      string    `json:"tool"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id,omitempty"`

	// Prompt/Response
	Prompt   string `json:"prompt,omitempty"`
	Response string `json:"response,omitempty"`
	Thought  string `json:"thought,omitempty"`

	// Tool execution
	ToolName   string         `json:"tool_name,omitempty"`
	ToolInput  map[string]any `json:"tool_input,omitempty"`
	ToolOutput string         `json:"tool_output,omitempty"`

	// Shell (extracted from tool_input when applicable)
	Command  string `json:"command,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`

	// File operations
	FilePath    string `json:"file_path,omitempty"`
	FileContent string `json:"file_content,omitempty"`
	Edits       []Edit `json:"edits,omitempty"`

	// MCP operations
	MCPServer string `json:"mcp_server,omitempty"`

	// Model layer (Gemini)
	ModelName   string         `json:"model_name,omitempty"`
	ModelConfig map[string]any `json:"model_config,omitempty"`
	LLMRequest  map[string]any `json:"llm_request,omitempty"`
	LLMResponse map[string]any `json:"llm_response,omitempty"`

	// Control flow
	StopReason string `json:"stop_reason,omitempty"`
	Permission string `json:"permission,omitempty"`

	// Raw for advanced detection
	RawEvent map[string]any `json:"raw_event,omitempty"`
}

// Edit represents a file edit operation.
type Edit struct {
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}
