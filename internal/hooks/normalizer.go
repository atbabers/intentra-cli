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
	EventSubagentStart     NormalizedEventType = "subagent_start"
	EventSubagentStop      NormalizedEventType = "subagent_stop"
	EventPreCompact        NormalizedEventType = "pre_compact"
	EventError             NormalizedEventType = "error"
	EventToolUseFailure    NormalizedEventType = "tool_use_failure"
	EventWorktreeSetup     NormalizedEventType = "worktree_setup"
	EventUnknown           NormalizedEventType = "unknown"
)

// Normalizer defines the interface for tool-specific event normalizers.
type Normalizer interface {
	// NormalizeEventType converts a tool-native event name (nativeType) to a
	// unified NormalizedEventType. Returns EventUnknown for unrecognized names.
	NormalizeEventType(nativeType string) NormalizedEventType
	// Tool returns the tool identifier string (e.g. "cursor", "claude").
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

// tableNormalizer implements Normalizer using a simple mapping table.
// Tool-specific normalizers register themselves with this in init().
type tableNormalizer struct {
	tool    string
	mapping map[string]NormalizedEventType
}

// Tool returns the tool identifier.
func (n *tableNormalizer) Tool() string { return n.tool }

// NormalizeEventType converts a tool-native event name to a unified type.
func (n *tableNormalizer) NormalizeEventType(native string) NormalizedEventType {
	if normalized, ok := n.mapping[native]; ok {
		return normalized
	}
	return EventUnknown
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
// Each tool has exactly ONE designated terminal event to prevent duplicate scans.
//
// NOTE: Windsurf does not provide a dedicated "stop" hook. We use
// EventAfterResponse as the best available proxy, but this means scans may
// be incomplete if the session continues after the last observed response.
// Windsurf sessions that end without a final response will not generate a scan.
func IsStopEvent(eventType NormalizedEventType, tool string) bool {
	switch tool {
	case "windsurf":
		return eventType == EventAfterResponse
	case "copilot", "gemini":
		return eventType == EventSessionEnd
	default:
		return eventType == EventStop
	}
}

// IsSessionEndEvent returns true if this event carries session-end metadata
// that should be PATCHed onto the last scan (not trigger a new scan).
func IsSessionEndEvent(eventType NormalizedEventType, tool string) bool {
	if tool == "windsurf" || tool == "copilot" {
		return false
	}
	return eventType == EventSessionEnd
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
