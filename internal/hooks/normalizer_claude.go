package hooks

import (
	"time"

	"github.com/atbabers/intentra-cli/pkg/models"
)

// ClaudeCodeNormalizer normalizes Claude Code hook events
type ClaudeCodeNormalizer struct{}

func (n *ClaudeCodeNormalizer) Tool() string {
	return "claude_code"
}

func (n *ClaudeCodeNormalizer) Normalize(raw map[string]any) (*models.NormalizedEvent, error) {
	hookEvent := getString(raw, "hook_event_name")
	if hookEvent == "" {
		hookEvent = getString(raw, "event_type")
	}

	event := &models.NormalizedEvent{
		Tool:      "claude_code",
		Timestamp: time.Now(),
		SessionID: getString(raw, "session_id"),
		RawEvent:  raw,
	}

	switch hookEvent {
	case "SessionStart":
		event.EventType = models.EventSessionStart

	case "SessionEnd":
		event.EventType = models.EventSessionEnd
		event.StopReason = getString(raw, "reason")

	case "UserPromptSubmit":
		event.EventType = models.EventBeforePrompt
		event.Prompt = getString(raw, "prompt")

	case "PreToolUse":
		event.ToolName = getString(raw, "tool_name")
		event.ToolInput = getMap(raw, "tool_input")
		event.EventType = n.classifyToolEvent(event.ToolName, true)
		n.extractToolDetails(event)

	case "PostToolUse":
		event.ToolName = getString(raw, "tool_name")
		event.ToolInput = getMap(raw, "tool_input")
		event.ToolOutput = getString(raw, "tool_output")
		event.EventType = n.classifyToolEvent(event.ToolName, false)
		n.extractToolDetails(event)

	case "PermissionRequest":
		event.EventType = models.EventPermission
		event.Permission = getString(raw, "permission")
		event.ToolName = getString(raw, "tool_name")

	case "Notification":
		event.EventType = models.EventNotification

	case "Stop":
		event.EventType = models.EventStop
		event.StopReason = getString(raw, "reason")

	case "SubagentStop":
		event.EventType = models.EventSubagentStop

	case "PreCompact":
		event.EventType = models.EventPreCompact

	default:
		event.EventType = hookEvent
	}

	return event, nil
}

func (n *ClaudeCodeNormalizer) classifyToolEvent(toolName string, isBefore bool) string {
	if isShellTool(toolName) {
		if isBefore {
			return models.EventBeforeShell
		}
		return models.EventAfterShell
	}
	if isFileTool(toolName) {
		if isBefore {
			return models.EventBeforeFileRead
		}
		return models.EventAfterFileEdit
	}
	if isBefore {
		return models.EventBeforeTool
	}
	return models.EventAfterTool
}

func (n *ClaudeCodeNormalizer) extractToolDetails(event *models.NormalizedEvent) {
	if event.ToolInput == nil {
		return
	}
	// Extract command for shell tools
	if isShellTool(event.ToolName) {
		event.Command = getString(event.ToolInput, "command")
	}
	// Extract file path for file tools
	if isFileTool(event.ToolName) {
		event.FilePath = getString(event.ToolInput, "file_path")
		if event.FilePath == "" {
			event.FilePath = getString(event.ToolInput, "path")
		}
	}
}
