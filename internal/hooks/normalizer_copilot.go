// Package hooks provides GitHub Copilot-specific event normalization.
package hooks

func init() {
	RegisterNormalizer(&tableNormalizer{
		tool: "copilot",
		mapping: map[string]NormalizedEventType{
			"sessionStart":        EventSessionStart,
			"sessionEnd":          EventSessionEnd,
			"userPromptSubmitted": EventBeforePrompt,
			"preToolUse":          EventBeforeTool,
			"postToolUse":         EventAfterTool,
			"errorOccurred":       EventError,
		},
	})
}
