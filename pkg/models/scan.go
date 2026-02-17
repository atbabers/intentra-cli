package models

import (
	"encoding/json"
	"time"
)

// ScanStatus represents the processing state of a scan.
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusAnalyzing ScanStatus = "analyzing"
	ScanStatusReviewed  ScanStatus = "reviewed"
	ScanStatusRejected  ScanStatus = "rejected"
)

// ScanSource identifies the origin of a scan event.
type ScanSource struct {
	Tool      string `json:"tool,omitempty"`
	Event     string `json:"event,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// ScanContent contains the actual prompt/response/tool data.
type ScanContent struct {
	Prompt    string          `json:"prompt,omitempty"`
	Response  string          `json:"response,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
}

// MCPToolCall represents aggregated usage of a single MCP server tool within a scan.
type MCPToolCall struct {
	ServerName    string  `json:"server_name"`
	ToolName      string  `json:"tool_name"`
	ServerURLHash string  `json:"server_url_hash,omitempty"`
	CallCount     int     `json:"call_count"`
	TotalDuration int     `json:"total_duration_ms"`
	EstimatedCost float64 `json:"estimated_cost"`
	ErrorCount    int     `json:"error_count"`
}

// Scan represents an aggregated conversation.
type Scan struct {
	ID             string       `json:"scan_id"`
	DeviceID       string       `json:"device_id"`
	Tool           string       `json:"tool,omitempty"`
	Timestamp      string       `json:"timestamp,omitempty"`
	ConversationID string       `json:"conversation_id,omitempty"`
	GenerationID   string       `json:"generation_id,omitempty"`
	Model          string       `json:"model,omitempty"`
	Status         ScanStatus   `json:"status,omitempty"`
	StartTime      time.Time    `json:"start_time,omitempty"`
	EndTime        time.Time    `json:"end_time,omitempty"`
	Source         *ScanSource  `json:"source,omitempty"`
	Content        *ScanContent `json:"content,omitempty"`
	Events         []Event      `json:"events,omitempty"`

	TotalTokens    int     `json:"total_tokens"`
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	ThinkingTokens int     `json:"thinking_tokens"`
	LLMCalls       int     `json:"llm_calls"`
	ToolCalls      int     `json:"tool_calls"`
	EstimatedCost  float64 `json:"estimated_cost"`

	RefundLikelihood int     `json:"refund_likelihood,omitempty"`
	RefundAmount     float64 `json:"refund_amount,omitempty"`
	Summary          string  `json:"summary,omitempty"`

	RawEvents []map[string]any `json:"raw_events,omitempty"`

	Fingerprint  string         `json:"fingerprint,omitempty"`
	FilesHash    string         `json:"files_hash,omitempty"`
	ActionCounts map[string]int `json:"action_counts,omitempty"`

	MCPToolUsage []MCPToolCall `json:"mcp_tool_usage,omitempty"`

	SessionEndReason  string `json:"session_end_reason,omitempty"`
	SessionDurationMs int64  `json:"session_duration_ms,omitempty"`

	RepoName      string                   `json:"repo_name,omitempty"`
	RepoURLHash   string                   `json:"repo_url_hash,omitempty"`
	BranchName    string                   `json:"branch_name,omitempty"`
	FilesModified []map[string]interface{} `json:"files_modified,omitempty"`
	AcceptedLines int                      `json:"accepted_lines,omitempty"`
}

// Duration returns the scan duration.
func (s *Scan) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

// AddEvent adds an event and updates metrics.
// Uses NormalizedType for event classification.
func (s *Scan) AddEvent(e Event) {
	e.ScanID = s.ID
	s.Events = append(s.Events, e)
	s.InputTokens += e.InputTokens
	s.OutputTokens += e.OutputTokens
	s.ThinkingTokens += e.ThinkingTokens
	s.TotalTokens = s.InputTokens + s.OutputTokens + s.ThinkingTokens

	llmEvents := map[string]bool{
		"after_response": true, "after_tool": true, "after_file_edit": true,
		"after_file_read": true, "after_shell": true, "after_mcp": true, "after_model": true,
	}
	toolEvents := map[string]bool{
		"after_tool": true, "after_file_edit": true, "after_file_read": true,
		"after_shell": true, "after_mcp": true,
	}

	if llmEvents[e.NormalizedType] {
		s.LLMCalls++
	}
	if toolEvents[e.NormalizedType] {
		s.ToolCalls++
	}
}
