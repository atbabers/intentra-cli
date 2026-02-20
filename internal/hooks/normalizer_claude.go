// Package hooks provides Claude Code-specific event normalization.
package hooks

// ClaudeNormalizer normalizes Claude Code hook events to unified types.
type ClaudeNormalizer struct{}

func init() {
	RegisterNormalizer(&ClaudeNormalizer{})
}

// Tool returns the tool identifier.
func (n *ClaudeNormalizer) Tool() string { return "claude" }

var claudeEventMapping = map[string]NormalizedEventType{
	"SessionStart":       EventSessionStart,
	"SessionEnd":         EventSessionEnd,
	"UserPromptSubmit":   EventBeforePrompt,
	"PreToolUse":         EventBeforeTool,
	"PostToolUse":        EventAfterTool,
	"PostToolUseFailure": EventAfterTool,
	"PermissionRequest":  EventPermissionRequest,
	"Notification":       EventNotification,
	"Stop":               EventStop,
	"SubagentStart":      EventBeforePrompt,
	"SubagentStop":       EventSubagentStop,
	"PreCompact":         EventPreCompact,
	"Setup":              EventSessionStart,
}

// NormalizeEventType converts Claude Code PascalCase events to snake_case normalized types.
func (n *ClaudeNormalizer) NormalizeEventType(native string) NormalizedEventType {
	if normalized, ok := claudeEventMapping[native]; ok {
		return normalized
	}
	return EventUnknown
}
