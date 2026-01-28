// Package hooks provides GitHub Copilot-specific event normalization.
package hooks

// CopilotNormalizer normalizes GitHub Copilot hook events to unified types.
type CopilotNormalizer struct{}

func init() {
	RegisterNormalizer(&CopilotNormalizer{})
}

// Tool returns the tool identifier.
func (n *CopilotNormalizer) Tool() string { return "copilot" }

// NormalizeEventType converts GitHub Copilot camelCase events to snake_case normalized types.
func (n *CopilotNormalizer) NormalizeEventType(native string) NormalizedEventType {
	mapping := map[string]NormalizedEventType{
		"sessionStart":        EventSessionStart,
		"sessionEnd":          EventSessionEnd,
		"userPromptSubmitted": EventBeforePrompt,
		"preToolUse":          EventBeforeTool,
		"postToolUse":         EventAfterTool,
		"errorOccurred":       EventError,
	}
	if normalized, ok := mapping[native]; ok {
		return normalized
	}
	return EventUnknown
}
