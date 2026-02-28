// Package hooks provides Gemini CLI-specific event normalization.
package hooks

func init() {
	RegisterNormalizer(&tableNormalizer{
		tool: "gemini",
		mapping: map[string]NormalizedEventType{
			"SessionStart":        EventSessionStart,
			"SessionEnd":          EventSessionEnd,
			"BeforeAgent":         EventBeforePrompt,
			"AfterAgent":          EventAfterResponse,
			"BeforeModel":         EventBeforeModel,
			"AfterModel":          EventAfterModel,
			"BeforeToolSelection": EventToolSelection,
			"BeforeTool":          EventBeforeTool,
			"AfterTool":           EventAfterTool,
			"PreCompress":         EventPreCompact,
			"Notification":        EventNotification,
		},
	})
}
