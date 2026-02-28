// Package hooks provides Windsurf Cascade-specific event normalization.
package hooks

func init() {
	RegisterNormalizer(&tableNormalizer{
		tool: "windsurf",
		mapping: map[string]NormalizedEventType{
			"pre_user_prompt":       EventBeforePrompt,
			"post_cascade_response": EventAfterResponse,
			"pre_read_code":         EventBeforeFileRead,
			"post_read_code":        EventAfterFileRead,
			"pre_write_code":        EventBeforeFileEdit,
			"post_write_code":       EventAfterFileEdit,
			"pre_run_command":       EventBeforeShell,
			"post_run_command":      EventAfterShell,
			"pre_mcp_tool_use":      EventBeforeMCP,
			"post_mcp_tool_use":     EventAfterMCP,
			"post_setup_worktree":   EventWorktreeSetup,
		},
	})
}
