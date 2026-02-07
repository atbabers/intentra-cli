// Package hooks provides Cursor-specific event normalization.
package hooks

// CursorNormalizer normalizes Cursor IDE hook events to unified types.
type CursorNormalizer struct{}

func init() {
	RegisterNormalizer(&CursorNormalizer{})
}

// Tool returns the tool identifier.
func (n *CursorNormalizer) Tool() string { return "cursor" }

// NormalizeEventType converts Cursor camelCase events to snake_case normalized types.
func (n *CursorNormalizer) NormalizeEventType(native string) NormalizedEventType {
	mapping := map[string]NormalizedEventType{
		"sessionStart":         EventSessionStart,
		"sessionEnd":           EventSessionEnd,
		"beforeSubmitPrompt":   EventBeforePrompt,
		"afterAgentResponse":   EventAfterResponse,
		"afterAgentThought":    EventAgentThought,
		"beforeShellExecution": EventBeforeShell,
		"afterShellExecution":  EventAfterShell,
		"beforeMCPExecution":   EventBeforeMCP,
		"afterMCPExecution":    EventAfterMCP,
		"beforeTabFileRead":    EventBeforeFileRead,
		"afterFileEdit":        EventAfterFileEdit,
		"afterTabFileEdit":     EventAfterFileEdit,
		"preToolUse":           EventBeforeTool,
		"postToolUse":          EventAfterTool,
		"preCompact":           EventPreCompact,
		"stop":                 EventStop,
	}
	if normalized, ok := mapping[native]; ok {
		return normalized
	}
	return EventUnknown
}
