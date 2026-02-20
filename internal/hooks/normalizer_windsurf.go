// Package hooks provides Windsurf Cascade-specific event normalization.
package hooks

// WindsurfNormalizer normalizes Windsurf Cascade hook events to unified types.
type WindsurfNormalizer struct{}

func init() {
	RegisterNormalizer(&WindsurfNormalizer{})
}

// Tool returns the tool identifier.
func (n *WindsurfNormalizer) Tool() string { return "windsurf" }

var windsurfEventMapping = map[string]NormalizedEventType{
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
}

// NormalizeEventType converts Windsurf Cascade snake_case events to unified normalized types.
func (n *WindsurfNormalizer) NormalizeEventType(native string) NormalizedEventType {
	if normalized, ok := windsurfEventMapping[native]; ok {
		return normalized
	}
	return EventUnknown
}
