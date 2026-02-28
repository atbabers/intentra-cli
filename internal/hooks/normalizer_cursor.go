// Package hooks provides Cursor-specific event normalization.
package hooks

func init() {
	RegisterNormalizer(&tableNormalizer{
		tool: "cursor",
		mapping: map[string]NormalizedEventType{
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
			"beforeReadFile":       EventBeforeFileRead,
			"afterFileEdit":        EventAfterFileEdit,
			"afterTabFileEdit":     EventAfterFileEdit,
			"preToolUse":           EventBeforeTool,
			"postToolUse":          EventAfterTool,
			"postToolUseFailure":   EventToolUseFailure,
			"preCompact":           EventPreCompact,
			"subagentStart":        EventSubagentStart,
			"subagentStop":         EventSubagentStop,
			"stop":                 EventStop,
		},
	})
}
