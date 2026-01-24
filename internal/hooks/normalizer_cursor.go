package hooks

import (
	"time"

	"github.com/atbabers/intentra-cli/pkg/models"
)

// CursorNormalizer normalizes Cursor hook events
type CursorNormalizer struct{}

func (n *CursorNormalizer) Tool() string {
	return "cursor"
}

func (n *CursorNormalizer) Normalize(raw map[string]any) (*models.NormalizedEvent, error) {
	hookEvent := getString(raw, "hook_event_name")
	if hookEvent == "" {
		hookEvent = getString(raw, "event_type")
	}

	event := &models.NormalizedEvent{
		Tool:      "cursor",
		Timestamp: time.Now(),
		SessionID: getString(raw, "session_id"),
		RawEvent:  raw,
	}

	switch hookEvent {
	case "beforeShellExecution":
		event.EventType = models.EventBeforeShell
		event.Command = getString(raw, "command")

	case "afterShellExecution":
		event.EventType = models.EventAfterShell
		event.Command = getString(raw, "command")
		event.ExitCode = getInt(raw, "exit_code")
		event.Stdout = getString(raw, "stdout")
		event.Stderr = getString(raw, "stderr")

	case "beforeMCPExecution":
		event.EventType = models.EventBeforeMCP
		event.ToolName = getString(raw, "tool_name")
		event.ToolInput = getMap(raw, "tool_input")
		event.MCPServer = getString(raw, "server_name")

	case "afterMCPExecution":
		event.EventType = models.EventAfterMCP
		event.ToolName = getString(raw, "tool_name")
		event.ToolInput = getMap(raw, "tool_input")
		event.ToolOutput = getString(raw, "tool_output")
		event.MCPServer = getString(raw, "server_name")

	case "beforeTabFileRead":
		event.EventType = models.EventBeforeFileRead
		event.FilePath = getString(raw, "file_path")

	case "afterFileEdit", "afterTabFileEdit":
		event.EventType = models.EventAfterFileEdit
		event.FilePath = getString(raw, "file_path")
		event.Edits = n.extractEdits(raw)

	case "beforeSubmitPrompt":
		event.EventType = models.EventBeforePrompt
		event.Prompt = getString(raw, "prompt")

	case "afterAgentResponse":
		event.EventType = models.EventAfterResponse
		event.Response = getString(raw, "response")

	case "afterAgentThought":
		event.EventType = models.EventAgentThought
		event.Thought = getString(raw, "thought")

	case "stop":
		event.EventType = models.EventStop
		event.StopReason = getString(raw, "reason")

	default:
		event.EventType = hookEvent
	}

	return event, nil
}

func (n *CursorNormalizer) extractEdits(raw map[string]any) []models.Edit {
	editsRaw, ok := raw["edits"].([]any)
	if !ok {
		return nil
	}

	var edits []models.Edit
	for _, e := range editsRaw {
		if editMap, ok := e.(map[string]any); ok {
			edits = append(edits, models.Edit{
				OldString: getString(editMap, "old_string"),
				NewString: getString(editMap, "new_string"),
			})
		}
	}
	return edits
}
