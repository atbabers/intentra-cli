// Package hooks provides event normalization across AI coding tools.
// This file defines the unified event type schema and normalizer interface.
package hooks

// NormalizedEventType represents a unified event type across all AI coding tools.
// All tool-specific event names are normalized to these snake_case types.
type NormalizedEventType string

const (
	EventSessionStart NormalizedEventType = "session_start"
	EventSessionEnd   NormalizedEventType = "session_end"

	EventBeforePrompt  NormalizedEventType = "before_prompt"
	EventAfterResponse NormalizedEventType = "after_response"
	EventAgentThought  NormalizedEventType = "agent_thought"

	EventBeforeTool NormalizedEventType = "before_tool"
	EventAfterTool  NormalizedEventType = "after_tool"

	EventBeforeFileRead NormalizedEventType = "before_file_read"
	EventAfterFileRead  NormalizedEventType = "after_file_read"
	EventBeforeFileEdit NormalizedEventType = "before_file_edit"
	EventAfterFileEdit  NormalizedEventType = "after_file_edit"

	EventBeforeShell NormalizedEventType = "before_shell"
	EventAfterShell  NormalizedEventType = "after_shell"

	EventBeforeMCP NormalizedEventType = "before_mcp"
	EventAfterMCP  NormalizedEventType = "after_mcp"

	EventBeforeModel NormalizedEventType = "before_model"
	EventAfterModel  NormalizedEventType = "after_model"

	EventToolSelection     NormalizedEventType = "tool_selection"
	EventPermissionRequest NormalizedEventType = "permission_request"
	EventNotification      NormalizedEventType = "notification"
	EventStop              NormalizedEventType = "stop"
	EventSubagentStop      NormalizedEventType = "subagent_stop"
	EventPreCompact        NormalizedEventType = "pre_compact"
	EventError             NormalizedEventType = "error"
	EventWorktreeSetup     NormalizedEventType = "worktree_setup"
	EventUnknown           NormalizedEventType = "unknown"
)

// Normalizer defines the interface for tool-specific event normalizers.
type Normalizer interface {
	NormalizeEventType(nativeType string) NormalizedEventType
	Tool() string
}

var normalizers = map[string]Normalizer{}

// RegisterNormalizer registers a normalizer for a specific tool.
func RegisterNormalizer(n Normalizer) {
	normalizers[n.Tool()] = n
}

// GetNormalizer returns the normalizer for the specified tool.
// Returns GenericNormalizer if no specific normalizer is registered.
func GetNormalizer(tool string) Normalizer {
	if n, ok := normalizers[tool]; ok {
		return n
	}
	return &GenericNormalizer{}
}

// GenericNormalizer handles unknown tools by returning EventUnknown.
type GenericNormalizer struct{}

// Tool returns empty string for generic normalizer.
func (n *GenericNormalizer) Tool() string { return "" }

// NormalizeEventType returns EventUnknown for unrecognized events.
func (n *GenericNormalizer) NormalizeEventType(native string) NormalizedEventType {
	return EventUnknown
}

// IsStopEvent returns true if the event type marks the end of a scan.
func IsStopEvent(eventType NormalizedEventType) bool {
	return eventType == EventStop ||
		eventType == EventSubagentStop ||
		eventType == EventSessionEnd ||
		eventType == EventAfterResponse ||
		eventType == EventError
}

// IsLLMCallEvent returns true if the event represents an LLM call.
// Every "after" action event involves the LLM making a decision.
func IsLLMCallEvent(eventType NormalizedEventType) bool {
	return eventType == EventAfterResponse ||
		eventType == EventAfterTool ||
		eventType == EventAfterFileEdit ||
		eventType == EventAfterFileRead ||
		eventType == EventAfterShell ||
		eventType == EventAfterMCP ||
		eventType == EventAfterModel ||
		eventType == EventAgentThought
}

// IsToolCallEvent returns true if the event represents a tool execution.
// Tool calls are a subset of LLM calls.
func IsToolCallEvent(eventType NormalizedEventType) bool {
	return eventType == EventAfterTool ||
		eventType == EventAfterFileEdit ||
		eventType == EventAfterFileRead ||
		eventType == EventAfterShell ||
		eventType == EventAfterMCP
}
