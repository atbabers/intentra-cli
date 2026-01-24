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

// Scan represents an aggregated conversation.
type Scan struct {
	ID             string       `json:"scan_id"`
	DeviceID       string       `json:"device_id"`
	Tool           string       `json:"tool,omitempty"`
	Timestamp      string       `json:"timestamp,omitempty"`
	ConversationID string       `json:"conversation_id,omitempty"`
	Status         ScanStatus   `json:"status,omitempty"`
	StartTime      time.Time    `json:"start_time,omitempty"`
	EndTime        time.Time    `json:"end_time,omitempty"`
	Source         *ScanSource  `json:"source,omitempty"`
	Content        *ScanContent `json:"content,omitempty"`
	Events         []Event      `json:"events,omitempty"`
	Violations     []Violation  `json:"violations,omitempty"`

	// Aggregated metrics
	TotalTokens    int     `json:"total_tokens"`
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	ThinkingTokens int     `json:"thinking_tokens"`
	LLMCalls       int     `json:"llm_calls"`
	ToolCalls      int     `json:"tool_calls"`
	Retries        int     `json:"retries"`
	EstimatedCost  float64 `json:"estimated_cost"`

	// Analysis results
	RefundLikelihood int     `json:"refund_likelihood,omitempty"`
	RefundAmount     float64 `json:"refund_amount,omitempty"`
	Summary          string  `json:"summary,omitempty"`
}

// Duration returns the scan duration.
func (s *Scan) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

// AddEvent adds an event and updates metrics.
func (s *Scan) AddEvent(e Event) {
	e.ScanID = s.ID // Mark event with parent scan ID
	s.Events = append(s.Events, e)
	s.InputTokens += e.InputTokens
	s.OutputTokens += e.OutputTokens
	s.ThinkingTokens += e.ThinkingTokens
	s.TotalTokens = s.InputTokens + s.OutputTokens + s.ThinkingTokens

	switch e.HookType {
	case HookAfterAgentResponse:
		s.LLMCalls++
	case HookAfterMCPExecution, HookAfterShellExecution:
		s.ToolCalls++
	}
}
