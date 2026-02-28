// Package hooks provides Claude Code-specific event normalization.
package hooks

func init() {
	RegisterNormalizer(&tableNormalizer{
		tool: "claude",
		mapping: map[string]NormalizedEventType{
			"SessionStart":       EventSessionStart,
			"SessionEnd":         EventSessionEnd,
			"UserPromptSubmit":   EventBeforePrompt,
			"PreToolUse":         EventBeforeTool,
			"PostToolUse":        EventAfterTool,
			"PostToolUseFailure": EventToolUseFailure,
			"PermissionRequest":  EventPermissionRequest,
			"Notification":       EventNotification,
			"Stop":               EventStop,
			"SubagentStart":      EventBeforePrompt,
			"SubagentStop":       EventSubagentStop,
			"PreCompact":         EventPreCompact,
			"Setup":              EventSessionStart,
		},
	})
}
