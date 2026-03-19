// Package hooks provides event normalization across AI coding tools.
// Event type constants are defined in pkg/models/event.go and referenced
// directly via the models package.
package hooks

import "github.com/intentrahq/intentra-cli/pkg/models"

// NormalizedEventType is an alias for models.NormalizedEventType.
type NormalizedEventType = models.NormalizedEventType

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
	return models.EventUnknown
}

// GenericNormalizer handles unknown tools by returning EventUnknown.
type GenericNormalizer struct{}

// Tool returns empty string for generic normalizer.
func (n *GenericNormalizer) Tool() string { return "" }

// NormalizeEventType returns EventUnknown for unrecognized events.
func (n *GenericNormalizer) NormalizeEventType(native string) NormalizedEventType {
	return models.EventUnknown
}

// toolMappings defines the event type mappings for all supported AI coding tools.
// Each tool maps its native event names to unified NormalizedEventType constants.
var toolMappings = map[string]map[string]NormalizedEventType{
	string(ToolCursor): {
		"sessionStart":         models.EventSessionStart,
		"sessionEnd":           models.EventSessionEnd,
		"beforeSubmitPrompt":   models.EventBeforePrompt,
		"afterAgentResponse":   models.EventAfterResponse,
		"afterAgentThought":    models.EventAgentThought,
		"beforeShellExecution": models.EventBeforeShell,
		"afterShellExecution":  models.EventAfterShell,
		"beforeMCPExecution":   models.EventBeforeMCP,
		"afterMCPExecution":    models.EventAfterMCP,
		"beforeTabFileRead":    models.EventBeforeFileRead,
		"beforeReadFile":       models.EventBeforeFileRead,
		"afterFileEdit":        models.EventAfterFileEdit,
		"afterTabFileEdit":     models.EventAfterFileEdit,
		"preToolUse":           models.EventBeforeTool,
		"postToolUse":          models.EventAfterTool,
		"postToolUseFailure":   models.EventToolUseFailure,
		"preCompact":           models.EventPreCompact,
		"subagentStart":        models.EventSubagentStart,
		"subagentStop":         models.EventSubagentStop,
		"stop":                 models.EventStop,
	},
	string(ToolClaudeCode): {
		"SessionStart":        models.EventSessionStart,
		"SessionEnd":          models.EventSessionEnd,
		"UserPromptSubmit":    models.EventBeforePrompt,
		"PreToolUse":          models.EventBeforeTool,
		"PostToolUse":         models.EventAfterTool,
		"PostToolUseFailure":  models.EventToolUseFailure,
		"PermissionRequest":   models.EventPermissionRequest,
		"Notification":        models.EventNotification,
		"Stop":                models.EventStop,
		"SubagentStart":       models.EventSubagentStart,
		"SubagentStop":        models.EventSubagentStop,
		"PreCompact":          models.EventPreCompact,
		"PostCompact":         models.EventPostCompact,
		"TeammateIdle":        models.EventTeammateIdle,
		"TaskCompleted":       models.EventTaskCompleted,
		"InstructionsLoaded":  models.EventInstructionsLoaded,
		"ConfigChange":        models.EventConfigChange,
		"WorktreeCreate":      models.EventWorktreeCreate,
		"WorktreeRemove":      models.EventWorktreeRemove,
		"Elicitation":         models.EventElicitation,
		"ElicitationResult":   models.EventElicitationResult,
	},
	string(ToolCopilot): {
		"sessionStart":        models.EventSessionStart,
		"sessionEnd":          models.EventSessionEnd,
		"userPromptSubmitted": models.EventBeforePrompt,
		"preToolUse":          models.EventBeforeTool,
		"postToolUse":         models.EventAfterTool,
		"agentStop":           models.EventStop,
		"subagentStop":        models.EventSubagentStop,
		"errorOccurred":       models.EventError,
	},
	string(ToolWindsurf): {
		"pre_user_prompt":                        models.EventBeforePrompt,
		"post_cascade_response":                  models.EventAfterResponse,
		"post_cascade_response_with_transcript":  models.EventResponseWithTranscript,
		"pre_read_code":                          models.EventBeforeFileRead,
		"post_read_code":                         models.EventAfterFileRead,
		"pre_write_code":                         models.EventBeforeFileEdit,
		"post_write_code":                        models.EventAfterFileEdit,
		"pre_run_command":                        models.EventBeforeShell,
		"post_run_command":                       models.EventAfterShell,
		"pre_mcp_tool_use":                       models.EventBeforeMCP,
		"post_mcp_tool_use":                      models.EventAfterMCP,
		"post_setup_worktree":                    models.EventWorktreeSetup,
	},
	string(ToolGeminiCLI): {
		"SessionStart":        models.EventSessionStart,
		"SessionEnd":          models.EventSessionEnd,
		"BeforeAgent":         models.EventBeforePrompt,
		"AfterAgent":          models.EventAfterResponse,
		"BeforeModel":         models.EventBeforeModel,
		"AfterModel":          models.EventAfterModel,
		"BeforeToolSelection": models.EventBeforeToolSelection,
		"BeforeTool":          models.EventBeforeTool,
		"AfterTool":           models.EventAfterTool,
		"PreCompress":         models.EventPreCompress,
		"Notification":        models.EventNotification,
	},
}

func init() {
	for tool, mapping := range toolMappings {
		RegisterNormalizer(&tableNormalizer{tool: tool, mapping: mapping})
	}
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
	case string(ToolWindsurf):
		return eventType == models.EventAfterResponse
	case string(ToolCopilot), string(ToolGeminiCLI):
		return eventType == models.EventSessionEnd
	default:
		return eventType == models.EventStop
	}
}

// IsSessionEndEvent returns true if this event carries session-end metadata
// that should be PATCHed onto the last scan (not trigger a new scan).
func IsSessionEndEvent(eventType NormalizedEventType, tool string) bool {
	if tool == string(ToolWindsurf) || tool == string(ToolCopilot) {
		return false
	}
	return eventType == models.EventSessionEnd
}

// IsLLMCallEvent delegates to models.IsLLMCallEvent.
func IsLLMCallEvent(eventType NormalizedEventType) bool {
	return models.IsLLMCallEvent(eventType)
}

// IsToolCallEvent delegates to models.IsToolCallEvent.
func IsToolCallEvent(eventType NormalizedEventType) bool {
	return models.IsToolCallEvent(eventType)
}
