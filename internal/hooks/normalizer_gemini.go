package hooks

import (
	"time"

	"github.com/atbabers/intentra-cli/pkg/models"
)

// GeminiCLINormalizer normalizes Gemini CLI hook events
type GeminiCLINormalizer struct{}

func (n *GeminiCLINormalizer) Tool() string {
	return "gemini_cli"
}

func (n *GeminiCLINormalizer) Normalize(raw map[string]any) (*models.NormalizedEvent, error) {
	hookEvent := getString(raw, "hook_event_name")
	if hookEvent == "" {
		hookEvent = getString(raw, "event_type")
	}

	event := &models.NormalizedEvent{
		Tool:      "gemini_cli",
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

	case "BeforeAgent":
		event.EventType = models.EventBeforePrompt
		event.Prompt = getString(raw, "prompt")

	case "AfterAgent":
		event.EventType = models.EventAfterResponse
		event.Response = getString(raw, "response")

	case "BeforeModel":
		event.EventType = models.EventBeforeModel
		event.LLMRequest = getMap(raw, "llm_request")
		if event.LLMRequest != nil {
			event.ModelName = getString(event.LLMRequest, "model")
			event.ModelConfig = getMap(event.LLMRequest, "config")
		}

	case "AfterModel":
		event.EventType = models.EventAfterModel
		event.LLMRequest = getMap(raw, "llm_request")
		event.LLMResponse = getMap(raw, "llm_response")
		if event.LLMRequest != nil {
			event.ModelName = getString(event.LLMRequest, "model")
		}

	case "BeforeToolSelection":
		event.EventType = models.EventToolSelection
		event.LLMRequest = getMap(raw, "llm_request")

	case "BeforeTool":
		event.ToolName = getString(raw, "tool_name")
		event.ToolInput = getMap(raw, "tool_input")
		event.EventType = n.classifyToolEvent(event.ToolName, true)
		n.extractToolDetails(event)

	case "AfterTool":
		event.ToolName = getString(raw, "tool_name")
		event.ToolInput = getMap(raw, "tool_input")
		event.ToolOutput = getString(raw, "tool_response")
		if event.ToolOutput == "" {
			event.ToolOutput = getString(raw, "tool_output")
		}
		event.EventType = n.classifyToolEvent(event.ToolName, false)
		n.extractToolDetails(event)

	case "PreCompress":
		event.EventType = models.EventPreCompact

	case "Notification":
		event.EventType = models.EventNotification

	default:
		event.EventType = hookEvent
	}

	return event, nil
}

func (n *GeminiCLINormalizer) classifyToolEvent(toolName string, isBefore bool) string {
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
	if isMCPTool(toolName) {
		if isBefore {
			return models.EventBeforeMCP
		}
		return models.EventAfterMCP
	}
	if isBefore {
		return models.EventBeforeTool
	}
	return models.EventAfterTool
}

func (n *GeminiCLINormalizer) extractToolDetails(event *models.NormalizedEvent) {
	if event.ToolInput == nil {
		return
	}
	// Extract command for shell tools
	if isShellTool(event.ToolName) {
		event.Command = getString(event.ToolInput, "command")
		if event.Command == "" {
			event.Command = getString(event.ToolInput, "cmd")
		}
	}
	// Extract file path for file tools
	if isFileTool(event.ToolName) {
		event.FilePath = getString(event.ToolInput, "file_path")
		if event.FilePath == "" {
			event.FilePath = getString(event.ToolInput, "path")
		}
	}
}
