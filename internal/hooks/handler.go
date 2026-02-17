// Package hooks manages integration with AI coding tools by installing and
// handling event hooks. It supports Cursor, Claude Code, Gemini CLI, GitHub
// Copilot, and Windsurf Cascade, providing real-time event capture and
// forwarding to the Intentra API.
package hooks

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/atbabers/intentra-cli/internal/api"
	"github.com/atbabers/intentra-cli/internal/auth"
	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/internal/debug"
	"github.com/atbabers/intentra-cli/internal/device"
	"github.com/atbabers/intentra-cli/internal/scanner"
	"github.com/atbabers/intentra-cli/pkg/models"
)

const maxBufferAge = 30 * time.Minute

type bufferedEvent struct {
	Event    *models.Event  `json:"event"`
	RawEvent map[string]any `json:"raw_event"`
}

func getBufferPath(sessionKey string) string {
	hash := sha256.Sum256([]byte(sessionKey))
	filename := "intentra_buffer_" + hex.EncodeToString(hash[:8]) + ".jsonl"
	return filepath.Join(os.TempDir(), filename)
}

func getLastScanPath(sessionKey string) string {
	hash := sha256.Sum256([]byte(sessionKey))
	filename := "intentra_lastscan_" + hex.EncodeToString(hash[:8]) + ".txt"
	return filepath.Join(os.TempDir(), filename)
}

func saveLastScanID(sessionKey, scanID string) {
	path := getLastScanPath(sessionKey)
	os.WriteFile(path, []byte(scanID), 0600)
}

func getLastScanID(sessionKey string) string {
	path := getLastScanPath(sessionKey)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func clearLastScanID(sessionKey string) {
	path := getLastScanPath(sessionKey)
	os.Remove(path)
}

func appendToBuffer(sessionKey string, event *models.Event, rawEvent map[string]any) error {
	bufferPath := getBufferPath(sessionKey)
	f, err := os.OpenFile(bufferPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open buffer: %w", err)
	}
	defer f.Close()

	entry := bufferedEvent{Event: event, RawEvent: rawEvent}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to buffer: %w", err)
	}

	return nil
}

func readAndClearBuffer(sessionKey string) ([]bufferedEvent, error) {
	bufferPath := getBufferPath(sessionKey)

	data, err := os.ReadFile(bufferPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read buffer: %w", err)
	}

	os.Remove(bufferPath)

	var events []bufferedEvent
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry bufferedEvent
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		events = append(events, entry)
	}

	return events, nil
}

func cleanupStaleBuffers() {
	patterns := []string{
		filepath.Join(os.TempDir(), "intentra_buffer_*.jsonl"),
		filepath.Join(os.TempDir(), "intentra_lastscan_*.txt"),
	}

	cutoff := time.Now().Add(-maxBufferAge)
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(f)
			}
		}
	}
}

func collectGitMetadata() (repoName, repoURLHash, branchName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if out, err := exec.CommandContext(ctx, "git", "remote", "get-url", "origin").Output(); err == nil {
		remoteURL := strings.TrimSpace(string(out))
		if remoteURL != "" {
			hash := sha256.Sum256([]byte(remoteURL))
			repoURLHash = hex.EncodeToString(hash[:])

			name := remoteURL
			if idx := strings.LastIndex(name, "/"); idx >= 0 {
				name = name[idx+1:]
			}
			if idx := strings.LastIndex(name, ":"); idx >= 0 {
				name = name[idx+1:]
			}
			name = strings.TrimSuffix(name, ".git")
			repoName = name
		}
	}

	if out, err := exec.CommandContext(ctx, "git", "branch", "--show-current").Output(); err == nil {
		branchName = strings.TrimSpace(string(out))
	}

	return
}

func createAggregatedScan(events []bufferedEvent, tool string) *models.Scan {
	if len(events) == 0 {
		return nil
	}

	first := events[0]
	last := events[len(events)-1]

	scan := &models.Scan{
		Tool:           tool,
		ConversationID: first.Event.ConversationID,
		Status:         models.ScanStatusPending,
		StartTime:      first.Event.Timestamp,
		EndTime:        last.Event.Timestamp,
		DeviceID:       first.Event.DeviceID,
	}

	if scan.ConversationID == "" && first.Event.SessionID != "" {
		scan.ConversationID = first.Event.SessionID
	}

	hash := sha256.Sum256([]byte(scan.ConversationID + scan.StartTime.String()))
	scan.ID = "scan_" + hex.EncodeToString(hash[:])[:12]

	scan.Source = &models.ScanSource{
		Tool:      tool,
		SessionID: first.Event.SessionID,
	}

	const maxPreCompactEvents = 10
	preCompactCount := 0

	for _, entry := range events {
		ev := entry.Event
		normalizedType := NormalizedEventType(ev.NormalizedType)

		if normalizedType == EventPreCompact {
			preCompactCount++
			if preCompactCount > maxPreCompactEvents {
				continue
			}
		}

		scan.Events = append(scan.Events, *ev)

		rawEvent := entry.RawEvent
		if rawEvent == nil {
			rawEvent = make(map[string]any)
		}
		rawEvent["normalized_type"] = ev.NormalizedType
		scan.RawEvents = append(scan.RawEvents, rawEvent)

		scan.InputTokens += ev.InputTokens
		scan.OutputTokens += ev.OutputTokens
		scan.ThinkingTokens += ev.ThinkingTokens

		if IsLLMCallEvent(normalizedType) {
			scan.LLMCalls++
		}
		if IsToolCallEvent(normalizedType) {
			scan.ToolCalls++
		}
	}

	scan.TotalTokens = scan.InputTokens + scan.OutputTokens + scan.ThinkingTokens

	for _, entry := range events {
		if entry.Event.Model != "" {
			scan.Model = entry.Event.Model
			break
		}
	}

	for _, entry := range events {
		if entry.Event.GenerationID != "" {
			scan.GenerationID = entry.Event.GenerationID
			break
		}
	}

	modelPricing := map[string]float64{
		"claude-sonnet-4":   0.003,
		"claude-3-5-sonnet": 0.003,
		"claude-3-5-haiku":  0.00025,
		"claude-3-opus":     0.015,
		"gemini":            0.0001,
		"gpt-4o":            0.005,
		"gpt-4":             0.03,
		"gpt-3.5":           0.0005,
		"o1":                0.015,
	}
	price := 0.003
	if tool == "copilot" {
		price = 0.005
	} else if tool == "windsurf" {
		price = 0.003
	}
	model := scan.Model
	if model == "" {
		model = "claude-3-5-sonnet"
	}
	for prefix, p := range modelPricing {
		if strings.HasPrefix(model, prefix) {
			price = p
			break
		}
	}
	scan.EstimatedCost = float64(scan.TotalTokens) / 1000.0 * price

	scan.MCPToolUsage = aggregateMCPToolUsage(events, scan.EstimatedCost)

	repoName, repoURLHash, branchName := collectGitMetadata()
	scan.RepoName = repoName
	scan.RepoURLHash = repoURLHash
	scan.BranchName = branchName

	var allEvents []models.Event
	for _, entry := range events {
		allEvents = append(allEvents, *entry.Event)
	}
	scan.FilesModified = scanner.AggregateFilesModified(allEvents)

	if tool == "copilot" || tool == "gemini" {
		for i := len(events) - 1; i >= 0; i-- {
			entry := events[i]
			if NormalizedEventType(entry.Event.NormalizedType) == EventSessionEnd && entry.RawEvent != nil {
				if reason, ok := entry.RawEvent["reason"].(string); ok && reason != "" {
					scan.SessionEndReason = reason
				}
				switch v := entry.RawEvent["duration_ms"].(type) {
				case float64:
					scan.SessionDurationMs = int64(v)
				case json.Number:
					if n, err := v.Int64(); err == nil {
						scan.SessionDurationMs = n
					}
				}
				break
			}
		}
	}

	return scan
}

// aggregateMCPToolUsage builds per-server/tool usage summaries from buffered events.
// Cost is attributed proportionally based on MCP call duration vs total scan duration.
func aggregateMCPToolUsage(events []bufferedEvent, totalScanCost float64) []models.MCPToolCall {
	type mcpKey struct {
		serverName string
		toolName   string
		urlHash    string
	}

	usage := make(map[mcpKey]*models.MCPToolCall)
	totalMCPDuration := 0
	totalScanDuration := 0

	for _, entry := range events {
		ev := entry.Event
		totalScanDuration += ev.DurationMs

		if !ev.IsMCPEvent() {
			continue
		}

		urlHash := models.MCPServerURLHash(ev.MCPServerURL, ev.MCPServerCmd)
		key := mcpKey{
			serverName: ev.MCPServerName,
			toolName:   ev.MCPToolName,
			urlHash:    urlHash,
		}

		call, exists := usage[key]
		if !exists {
			call = &models.MCPToolCall{
				ServerName:    ev.MCPServerName,
				ToolName:      ev.MCPToolName,
				ServerURLHash: urlHash,
			}
			usage[key] = call
		}

		call.CallCount++
		call.TotalDuration += ev.DurationMs
		totalMCPDuration += ev.DurationMs

		if ev.Error != "" {
			call.ErrorCount++
		}
	}

	if len(usage) == 0 {
		return nil
	}

	var result []models.MCPToolCall
	for _, call := range usage {
		if totalScanDuration > 0 && totalScanCost > 0 {
			proportion := float64(call.TotalDuration) / float64(totalScanDuration)
			call.EstimatedCost = totalScanCost * proportion
		}
		result = append(result, *call)
	}

	return result
}

func normalizeHookEvent(rawJSON []byte, tool, eventType string) (*models.Event, map[string]any, NormalizedEventType, error) {
	var raw map[string]any
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return nil, nil, EventUnknown, fmt.Errorf("failed to parse raw JSON: %w", err)
	}

	normalizer := GetNormalizer(tool)
	normalizedType := normalizer.NormalizeEventType(eventType)

	event := &models.Event{
		Tool:           tool,
		HookType:       models.HookType(eventType),
		NormalizedType: string(normalizedType),
	}

	if v, ok := raw["conversation_id"].(string); ok {
		event.ConversationID = v
	}
	if v, ok := raw["session_id"].(string); ok {
		event.SessionID = v
	}

	if v, ok := raw["trajectory_id"].(string); ok && event.ConversationID == "" {
		event.ConversationID = v
	}

	if v, ok := raw["generation_id"].(string); ok {
		event.GenerationID = v
	}
	if v, ok := raw["execution_id"].(string); ok && event.GenerationID == "" {
		event.GenerationID = v
	}
	if v, ok := raw["turn_id"].(string); ok && event.GenerationID == "" {
		event.GenerationID = v
	}

	if v, ok := raw["hook_event_name"].(string); ok && event.HookType == "" {
		event.HookType = models.HookType(v)
	}
	if v, ok := raw["agent_action_name"].(string); ok && event.HookType == "" {
		event.HookType = models.HookType(v)
	}

	if v, ok := raw["model"].(string); ok {
		event.Model = v
	}
	if v, ok := raw["user_email"].(string); ok {
		event.UserEmail = v
	}

	if v, ok := raw["tool_name"].(string); ok {
		event.ToolName = v
	}
	if v, ok := raw["toolName"].(string); ok && event.ToolName == "" {
		event.ToolName = v
	}

	if toolInput, ok := raw["tool_input"].(map[string]any); ok {
		if inputJSON, err := json.Marshal(toolInput); err == nil {
			event.ToolInput = inputJSON
		}
		if cmd, ok := toolInput["command"].(string); ok {
			event.Command = cmd
		}
		if fp, ok := toolInput["file_path"].(string); ok {
			event.FilePath = fp
		}
	}

	if toolArgs, ok := raw["toolArgs"].(string); ok && event.ToolInput == nil {
		event.ToolInput = json.RawMessage(toolArgs)
	}

	if toolOutput, ok := raw["tool_output"].(string); ok {
		event.ToolOutput = json.RawMessage(`"` + toolOutput + `"`)
	} else if toolOutput, ok := raw["tool_output"].(map[string]any); ok {
		if outputJSON, err := json.Marshal(toolOutput); err == nil {
			event.ToolOutput = outputJSON
		}
	}
	if toolResp, ok := raw["tool_response"].(map[string]any); ok {
		if respJSON, err := json.Marshal(toolResp); err == nil {
			event.ToolOutput = respJSON
		}
	}
	if toolResult, ok := raw["toolResult"].(map[string]any); ok {
		if resultJSON, err := json.Marshal(toolResult); err == nil {
			event.ToolOutput = resultJSON
		}
	}

	if toolInfo, ok := raw["tool_info"].(map[string]any); ok {
		if fp, ok := toolInfo["file_path"].(string); ok {
			event.FilePath = fp
		}
		if cmd, ok := toolInfo["command_line"].(string); ok {
			event.Command = cmd
		}
		if prompt, ok := toolInfo["user_prompt"].(string); ok {
			event.Prompt = prompt
		}
		if resp, ok := toolInfo["response"].(string); ok {
			event.Response = resp
		}
		if toolInfoJSON, err := json.Marshal(toolInfo); err == nil {
			if event.ToolInput == nil {
				event.ToolInput = toolInfoJSON
			}
		}
	}

	if v, ok := raw["command"].(string); ok && event.Command == "" {
		event.Command = v
	}
	if v, ok := raw["output"].(string); ok {
		event.CommandOutput = v
	}

	if v, ok := raw["prompt"].(string); ok {
		event.Prompt = v
	}
	if v, ok := raw["initialPrompt"].(string); ok && event.Prompt == "" {
		event.Prompt = v
	}
	if v, ok := raw["response"].(string); ok {
		event.Response = v
	}
	if v, ok := raw["thought"].(string); ok {
		event.Thought = v
	}
	if v, ok := raw["text"].(string); ok {
		if event.Response == "" {
			event.Response = v
		}
	}

	if v, ok := raw["file_path"].(string); ok && event.FilePath == "" {
		event.FilePath = v
	}
	if v, ok := raw["cwd"].(string); ok && event.FilePath == "" {
		event.FilePath = v
	}

	if v, ok := raw["duration"].(float64); ok {
		event.DurationMs = int(v)
	}
	if v, ok := raw["duration_ms"].(float64); ok {
		event.DurationMs = int(v)
	}

	if v, ok := raw["input_tokens"].(float64); ok {
		event.InputTokens = int(v)
	}
	if v, ok := raw["output_tokens"].(float64); ok {
		event.OutputTokens = int(v)
	}

	if errObj, ok := raw["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok {
			event.Response = "Error: " + msg
			event.Error = msg
		}
	} else if errStr, ok := raw["error"].(string); ok && errStr != "" {
		event.Error = errStr
	}

	extractMCPMetadata(event, raw, tool, normalizedType)
	extractCompactionMetadata(event, raw, normalizedType)

	return event, raw, normalizedType, nil
}

// extractCompactionMetadata populates compaction-specific fields for pre_compact events.
// Cursor provides rich context window metrics; Claude Code and Gemini CLI provide only trigger type.
func extractCompactionMetadata(event *models.Event, raw map[string]any, normalizedType NormalizedEventType) {
	if normalizedType != EventPreCompact {
		return
	}

	if v, ok := raw["trigger"].(string); ok {
		if v == "auto" || v == "manual" {
			event.CompactionTrigger = v
		}
	}

	if v, ok := raw["context_usage_percent"].(float64); ok {
		pct := int(v)
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		event.ContextUsagePercent = pct
	}

	if v, ok := raw["context_tokens"].(float64); ok && v >= 0 {
		event.ContextTokens = int(v)
	}

	if v, ok := raw["context_window_size"].(float64); ok && v >= 0 {
		event.ContextWindowSize = int(v)
	}

	if v, ok := raw["message_count"].(float64); ok && v >= 0 {
		event.MessageCount = int(v)
	}

	if v, ok := raw["messages_to_compact"].(float64); ok && v >= 0 {
		event.MessagesToCompact = int(v)
	}

	if v, ok := raw["is_first_compaction"].(bool); ok {
		event.IsFirstCompaction = &v
	}
}

// extractMCPMetadata populates MCP-specific fields on the event based on the tool type.
// Each AI coding tool exposes MCP data in a different format.
func extractMCPMetadata(event *models.Event, raw map[string]any, tool string, normalizedType NormalizedEventType) {
	isMCPHook := normalizedType == EventBeforeMCP || normalizedType == EventAfterMCP
	isMCPToolUse := strings.HasPrefix(event.ToolName, "MCP:") || strings.HasPrefix(event.ToolName, "mcp__")

	if !isMCPHook && !isMCPToolUse {
		return
	}

	if isMCPToolUse && !isMCPHook {
		toolName := event.ToolName
		if strings.HasPrefix(toolName, "MCP:") {
			fullToolName := toolName[4:]
			event.MCPToolName = fullToolName
			event.MCPServerName = inferMCPServerName(fullToolName)
		} else if serverName, mcpTool, ok := models.ParseMCPDoubleUnderscoreName(toolName); ok {
			event.MCPServerName = serverName
			event.MCPToolName = mcpTool
		}
		return
	}

	switch tool {
	case "cursor":
		extractCursorMCP(event, raw)
	case "windsurf":
		extractWindsurfMCP(event, raw)
	case "claude":
		extractClaudeGeminiMCP(event, raw)
	case "gemini":
		extractClaudeGeminiMCP(event, raw)
	case "copilot":
		extractCopilotMCP(event, raw)
	default:
		if event.ToolName != "" {
			event.MCPToolName = event.ToolName
		}
	}
}

// extractCursorMCP handles Cursor's beforeMCPExecution / afterMCPExecution format.
// Input contains tool_name directly, plus server url or command.
func extractCursorMCP(event *models.Event, raw map[string]any) {
	if v, ok := raw["tool_name"].(string); ok {
		event.MCPToolName = v
	}

	if v, ok := raw["url"].(string); ok {
		event.MCPServerURL = models.SanitizeMCPServerURL(v)
	}
	if v, ok := raw["command"].(string); ok && event.MCPServerURL == "" {
		event.MCPServerCmd = models.SanitizeMCPServerCmd(v)
	}

	if event.MCPServerName == "" {
		if event.MCPServerURL != "" {
			event.MCPServerName = extractHostFromURL(event.MCPServerURL)
		} else if event.MCPServerCmd != "" {
			event.MCPServerName = event.MCPServerCmd
		} else if event.MCPToolName != "" {
			event.MCPServerName = inferMCPServerName(event.MCPToolName)
		}
	}
}

// extractWindsurfMCP handles Windsurf's pre_mcp_tool_use / post_mcp_tool_use format.
// Data is nested inside tool_info with explicit mcp_server_name and mcp_tool_name.
func extractWindsurfMCP(event *models.Event, raw map[string]any) {
	toolInfo, ok := raw["tool_info"].(map[string]any)
	if !ok {
		return
	}

	if v, ok := toolInfo["mcp_server_name"].(string); ok {
		event.MCPServerName = v
	}
	if v, ok := toolInfo["mcp_tool_name"].(string); ok {
		event.MCPToolName = v
	}
}

// extractClaudeGeminiMCP handles Claude Code and Gemini CLI MCP tool format.
// Tool names follow the pattern mcp__<server>__<tool>.
func extractClaudeGeminiMCP(event *models.Event, raw map[string]any) {
	toolName := event.ToolName
	if toolName == "" {
		if v, ok := raw["tool_name"].(string); ok {
			toolName = v
		}
	}

	if serverName, mcpTool, ok := models.ParseMCPDoubleUnderscoreName(toolName); ok {
		event.MCPServerName = serverName
		event.MCPToolName = mcpTool
	}
}

// extractCopilotMCP handles GitHub Copilot MCP calls.
// Copilot does not expose server names, so we use a pseudo-server.
func extractCopilotMCP(event *models.Event, raw map[string]any) {
	event.MCPServerName = "copilot-mcp"
	if event.ToolName != "" {
		event.MCPToolName = event.ToolName
	}
}

var mcpToolToServer = map[string]string{
	"browser_click":            "cursor-browser",
	"browser_close":            "cursor-browser",
	"browser_console_messages": "cursor-browser",
	"browser_fill":             "cursor-browser",
	"browser_fill_form":        "cursor-browser",
	"browser_get_attribute":    "cursor-browser",
	"browser_get_bounding_box": "cursor-browser",
	"browser_get_input_value":  "cursor-browser",
	"browser_handle_dialog":    "cursor-browser",
	"browser_highlight":        "cursor-browser",
	"browser_hover":            "cursor-browser",
	"browser_is_checked":       "cursor-browser",
	"browser_is_enabled":       "cursor-browser",
	"browser_is_visible":       "cursor-browser",
	"browser_lock":             "cursor-browser",
	"browser_navigate":         "cursor-browser",
	"browser_navigate_back":    "cursor-browser",
	"browser_navigate_forward": "cursor-browser",
	"browser_network_requests": "cursor-browser",
	"browser_press_key":        "cursor-browser",
	"browser_reload":           "cursor-browser",
	"browser_resize":           "cursor-browser",
	"browser_run_code":         "cursor-browser",
	"browser_scroll":           "cursor-browser",
	"browser_search":           "cursor-browser",
	"browser_select_option":    "cursor-browser",
	"browser_snapshot":         "cursor-browser",
	"browser_tabs":             "cursor-browser",
	"browser_take_screenshot":  "cursor-browser",
	"browser_type":             "cursor-browser",
	"browser_unlock":           "cursor-browser",
	"browser_wait_for":         "cursor-browser",

	"navigate_page":   "chrome-devtools",
	"evaluate_script": "chrome-devtools",
	"list_pages":      "chrome-devtools",
	"new_page":        "chrome-devtools",
	"take_snapshot":   "chrome-devtools",
	"take_screenshot": "chrome-devtools",
	"navigate":        "chrome-devtools",
	"select_page":     "chrome-devtools",
	"close_page":      "chrome-devtools",
	"click":           "chrome-devtools",
	"hover":           "chrome-devtools",
	"fill":            "chrome-devtools",
	"fill_form":       "chrome-devtools",
	"drag":            "chrome-devtools",
	"press_key":       "chrome-devtools",
	"upload_file":     "chrome-devtools",
	"wait_for":        "chrome-devtools",
	"handle_dialog":   "chrome-devtools",
	"emulate":         "chrome-devtools",
	"resize_page":     "chrome-devtools",

	"search_issues":      "sentry",
	"get_issue_details":  "sentry",
	"find_organizations": "sentry",
	"find_projects":      "sentry",
	"find_releases":      "sentry",
	"search_events":      "sentry",
	"update_issue":       "sentry",

	"event-definitions-list": "posthog",
	"organizations-get":      "posthog",
	"projects-get":           "posthog",
	"list_teams":             "posthog",
	"list_projects":          "posthog",
	"list-errors":            "posthog",
}

func inferMCPServerName(toolName string) string {
	if server, ok := mcpToolToServer[toolName]; ok {
		return server
	}

	if strings.HasPrefix(toolName, "browser_") {
		return "cursor-browser"
	}

	if strings.Contains(toolName, "__") {
		parts := strings.SplitN(toolName, "__", 2)
		if len(parts) == 2 {
			return parts[0]
		}
	}

	return "mcp"
}

func extractHostFromURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return host
}

// ProcessEvent reads an event from stdin and sends directly to API.
func ProcessEvent(reader io.Reader, cfg *config.Config, tool string) error {
	return ProcessEventWithEvent(reader, cfg, tool, "")
}

// ProcessEventWithEvent buffers events and sends aggregated scan on stop events.
func ProcessEventWithEvent(reader io.Reader, cfg *config.Config, tool, eventType string) error {
	cleanupStaleBuffers()

	bufScanner := bufio.NewScanner(reader)
	if !bufScanner.Scan() {
		if err := bufScanner.Err(); err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		return nil
	}

	rawJSON := bufScanner.Bytes()
	if len(rawJSON) == 0 {
		return nil
	}

	event, rawMap, normalizedType, err := normalizeHookEvent(rawJSON, tool, eventType)
	if err != nil {
		return fmt.Errorf("failed to normalize event: %w", err)
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if event.DeviceID == "" {
		deviceID, err := device.GetDeviceID()
		if err == nil {
			event.DeviceID = deviceID
		}
	}

	baseKey := event.ConversationID
	if baseKey == "" {
		baseKey = event.SessionID
	}
	if baseKey == "" {
		baseKey = event.DeviceID + "_default"
	}
	sessionKey := tool + "_" + baseKey

	if tool == "claude" {
		cursorKey := "cursor_" + baseKey
		cursorBufferPath := getBufferPath(cursorKey)
		if _, err := os.Stat(cursorBufferPath); err == nil {
			debug.Log("Claude event has matching Cursor session, treating as Cursor")
			sessionKey = cursorKey
			tool = "cursor"
		}
	}

	if IsStopEvent(normalizedType, tool) {
		if err := appendToBuffer(sessionKey, event, rawMap); err != nil {
			return fmt.Errorf("failed to buffer event: %w", err)
		}

		bufferedEvents, err := readAndClearBuffer(sessionKey)
		if err != nil {
			return fmt.Errorf("failed to read buffer: %w", err)
		}

		if len(bufferedEvents) == 0 {
			return nil
		}

		scan := createAggregatedScan(bufferedEvents, tool)
		if scan == nil {
			return nil
		}

		synced := false

		creds := auth.GetValidCredentials()
		if creds != nil {
			if err := syncScanWithJWT(scan, creds.AccessToken); err != nil {
				debug.Warn("failed to sync to api.intentra.sh: %v", err)
			} else {
				debug.Log("Synced to https://api.intentra.sh")
				synced = true
			}
		}

		if !synced && cfg.Server.Enabled {
			client, err := api.NewClient(cfg)
			if err == nil {
				debug.Log("Syncing to %s (config auth)", cfg.Server.Endpoint)
				if err := client.SendScan(scan); err != nil {
					debug.Warn("sync failed: %v", err)
				}
			}
		}

		if synced && scan.ID != "" {
			saveLastScanID(sessionKey, scan.ID)
		}

		if debug.Enabled {
			if err := scanner.SaveScan(scan); err != nil {
				debug.Warn("failed to save scan locally: %v", err)
			} else {
				debug.Log("Saved scan locally: %s", scan.ID)
			}
		}

		return nil
	}

	if IsSessionEndEvent(normalizedType, tool) {
		lastScanID := getLastScanID(sessionKey)
		if lastScanID == "" {
			debug.Log("sessionEnd event but no lastScanID for session %s, ignoring", sessionKey)
			return nil
		}

		creds := auth.GetValidCredentials()
		if creds == nil {
			debug.Log("sessionEnd event but no valid credentials, ignoring")
			return nil
		}

		reason := ""
		durationMs := int64(0)
		if rawMap != nil {
			if r, ok := rawMap["reason"].(string); ok {
				reason = r
			}
			switch v := rawMap["duration_ms"].(type) {
			case float64:
				durationMs = int64(v)
			case json.Number:
				if n, err := v.Int64(); err == nil {
					durationMs = n
				}
			}
		}

		if err := patchSessionEnd(lastScanID, creds.AccessToken, reason, durationMs); err != nil {
			debug.Warn("failed to PATCH session end: %v", err)
		} else {
			debug.Log("PATCHed session end for scan %s", lastScanID)
		}

		clearLastScanID(sessionKey)
		return nil
	}

	if err := appendToBuffer(sessionKey, event, rawMap); err != nil {
		return fmt.Errorf("failed to buffer event: %w", err)
	}

	return nil
}

// RunHookHandler is the main entry point for hook processing.
func RunHookHandler() error {
	return RunHookHandlerWithToolAndEvent("", "")
}

// RunHookHandlerWithTool processes hooks with tool identifier.
func RunHookHandlerWithTool(tool string) error {
	return RunHookHandlerWithToolAndEvent(tool, "")
}

// RunHookHandlerWithToolAndEvent processes hooks with tool and event identifiers.
func RunHookHandlerWithToolAndEvent(tool, event string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	debug.Enabled = cfg.Debug

	return ProcessEventWithEvent(os.Stdin, cfg, tool, event)
}

const defaultAPIEndpoint = "https://api.intentra.sh"

// syncScanWithJWT sends a scan to the default API endpoint using JWT auth.
func syncScanWithJWT(scan *models.Scan, accessToken string) error {
	deviceID, err := device.GetDeviceID()
	if err != nil {
		return fmt.Errorf("failed to get device ID: %w", err)
	}

	durationMs := int64(0)
	if !scan.EndTime.IsZero() && !scan.StartTime.IsZero() {
		durationMs = scan.EndTime.Sub(scan.StartTime).Milliseconds()
	}

	sessionID := ""
	if scan.Source != nil {
		sessionID = scan.Source.SessionID
	}

	var events []map[string]any
	if len(scan.RawEvents) > 0 {
		events = scan.RawEvents
	} else {
		for _, ev := range scan.Events {
			evMap := map[string]any{
				"hook_type":       string(ev.HookType),
				"normalized_type": ev.NormalizedType,
				"timestamp":       ev.Timestamp.Format(time.RFC3339Nano),
				"tool_name":       ev.ToolName,
				"command":         ev.Command,
				"command_output":  ev.CommandOutput,
				"file_path":       ev.FilePath,
				"prompt":          ev.Prompt,
				"response":        ev.Response,
				"thought":         ev.Thought,
				"duration_ms":     ev.DurationMs,
				"conversation_id": ev.ConversationID,
				"session_id":      ev.SessionID,
				"tokens": map[string]int{
					"input":    ev.InputTokens,
					"output":   ev.OutputTokens,
					"thinking": ev.ThinkingTokens,
				},
			}
			if ev.CompactionTrigger != "" {
				evMap["compaction_trigger"] = ev.CompactionTrigger
			}
			if ev.ContextUsagePercent > 0 {
				evMap["context_usage_percent"] = ev.ContextUsagePercent
			}
			if ev.ContextTokens > 0 {
				evMap["context_tokens"] = ev.ContextTokens
			}
			if ev.ContextWindowSize > 0 {
				evMap["context_window_size"] = ev.ContextWindowSize
			}
			if ev.MessageCount > 0 {
				evMap["message_count"] = ev.MessageCount
			}
			if ev.MessagesToCompact > 0 {
				evMap["messages_to_compact"] = ev.MessagesToCompact
			}
			if ev.IsFirstCompaction != nil {
				evMap["is_first_compaction"] = *ev.IsFirstCompaction
			}
			if len(ev.ToolInput) > 0 {
				var toolInput map[string]any
				if err := json.Unmarshal(ev.ToolInput, &toolInput); err == nil {
					evMap["tool_input"] = toolInput
				}
			}
			if len(ev.ToolOutput) > 0 {
				var toolOutput any
				if err := json.Unmarshal(ev.ToolOutput, &toolOutput); err == nil {
					evMap["tool_output"] = toolOutput
				}
			}
			events = append(events, evMap)
		}
	}

	body := map[string]any{
		"tool":            scan.Tool,
		"started_at":      scan.StartTime.Format(time.RFC3339Nano),
		"ended_at":        scan.EndTime.Format(time.RFC3339Nano),
		"duration_ms":     durationMs,
		"llm_call_count":  scan.LLMCalls,
		"total_tokens":    scan.TotalTokens,
		"estimated_cost":  scan.EstimatedCost,
		"events":          events,
		"device_id":       deviceID,
		"conversation_id": scan.ConversationID,
		"session_id":      sessionID,
		"generation_id":   scan.GenerationID,
		"model":           scan.Model,
	}

	if len(scan.MCPToolUsage) > 0 {
		body["mcp_tool_usage"] = scan.MCPToolUsage
	}

	if scan.SessionEndReason != "" {
		body["session_end_reason"] = scan.SessionEndReason
	}
	if scan.SessionDurationMs > 0 {
		body["session_duration_ms"] = scan.SessionDurationMs
	}

	if scan.RepoName != "" {
		body["repo_name"] = scan.RepoName
	}
	if scan.RepoURLHash != "" {
		body["repo_url_hash"] = scan.RepoURLHash
	}
	if scan.BranchName != "" {
		body["branch_name"] = scan.BranchName
	}
	if len(scan.FilesModified) > 0 {
		body["files_modified"] = scan.FilesModified
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scan: %w", err)
	}

	url := defaultAPIEndpoint + "/scans"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "intentra-cli/1.0")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Machine-ID", deviceID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		debug.LogHTTP("POST", url, 0)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	debug.LogHTTP("POST", url, resp.StatusCode)

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func patchSessionEnd(scanID, accessToken, reason string, durationMs int64) error {
	deviceID, err := device.GetDeviceID()
	if err != nil {
		return fmt.Errorf("failed to get device ID: %w", err)
	}

	body := map[string]any{}
	if reason != "" {
		body["session_end_reason"] = reason
	}
	if durationMs > 0 {
		body["session_duration_ms"] = durationMs
	}

	if len(body) == 0 {
		return nil
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal session end body: %w", err)
	}

	patchURL := defaultAPIEndpoint + "/scans/" + url.PathEscape(scanID) + "/session"
	req, err := http.NewRequest("PATCH", patchURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create PATCH request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "intentra-cli/1.0")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Machine-ID", deviceID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		debug.LogHTTP("PATCH", patchURL, 0)
		return fmt.Errorf("PATCH request failed: %w", err)
	}
	defer resp.Body.Close()
	debug.LogHTTP("PATCH", patchURL, resp.StatusCode)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PATCH returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
