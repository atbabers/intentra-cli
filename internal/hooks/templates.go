package hooks

import (
	"encoding/json"
	"errors"
	"regexp"
	"runtime"
	"strings"
)

// ErrInvalidHandlerPath is returned when the handler path contains unsafe characters.
var ErrInvalidHandlerPath = errors.New("invalid handler path: contains unsafe characters")

// safePathPattern validates handler paths to prevent command injection.
// Allows alphanumeric, underscores, hyphens, dots, forward/back slashes, and colons (for Windows drives).
var safePathPattern = regexp.MustCompile(`^[a-zA-Z0-9/_\-\.:\\]+$`)

// validateHandlerPath checks if a handler path is safe to use in shell commands.
func validateHandlerPath(path string) error {
	if path == "" {
		return ErrInvalidHandlerPath
	}
	if len(path) > 4096 {
		return errors.New("invalid handler path: exceeds maximum length")
	}
	if !safePathPattern.MatchString(path) {
		return ErrInvalidHandlerPath
	}
	// Block common shell metacharacters and injection patterns
	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "<", ">", "!", "~", "*", "?", "[", "]", "#", "\n", "\r", "'", "\""}
	for _, d := range dangerous {
		if strings.Contains(path, d) {
			return ErrInvalidHandlerPath
		}
	}
	return nil
}

// quotePathForShell safely quotes a path for shell execution.
// This provides defense-in-depth even though validateHandlerPath should catch issues.
func quotePathForShell(path string) string {
	if runtime.GOOS == "windows" {
		// Windows: wrap in double quotes and escape internal double quotes
		escaped := strings.ReplaceAll(path, "\"", "\\\"")
		return "\"" + escaped + "\""
	}
	// Unix: wrap in single quotes and escape internal single quotes
	escaped := strings.ReplaceAll(path, "'", "'\"'\"'")
	return "'" + escaped + "'"
}

// CursorHookConfig represents Cursor's hooks.json structure.
type CursorHookConfig struct {
	Hooks map[string][]CursorHookEntry `json:"hooks"`
}

type CursorHookEntry struct {
	Command string `json:"command"`
}

// ClaudeCodeHooks represents Claude Code's hooks configuration.
type ClaudeCodeHooks struct {
	PreToolExecution  []ClaudeHookEntry `json:"PreToolExecution,omitempty"`
	PostToolExecution []ClaudeHookEntry `json:"PostToolExecution,omitempty"`
	Notification      []ClaudeHookEntry `json:"Notification,omitempty"`
	Stop              []ClaudeHookEntry `json:"Stop,omitempty"`
}

type ClaudeHookEntry struct {
	Matcher string `json:"matcher"`
	Hooks   []struct {
		Type    string `json:"type"`
		Command string `json:"command"`
	} `json:"hooks"`
}

// cursorHookTypes contains all available hooks per https://cursor.com/docs/agent/hooks.
var cursorHookTypes = []string{
	// Agent hooks (Cmd+K/Agent Chat)
	"beforeSubmitPrompt",
	"afterAgentThought",
	"afterAgentResponse",
	"beforeShellExecution",
	"afterShellExecution",
	"beforeMCPExecution",
	"afterMCPExecution",
	"beforeReadFile",
	"afterFileEdit",
	"stop",
	// Tab hooks (Inline Completions)
	"beforeTabFileRead",
	"afterTabFileEdit",
}

// GenerateCursorHooksJSON creates the Cursor hooks.json content.
// Returns an error if the handler path contains unsafe characters.
func GenerateCursorHooksJSON(handlerPath string) (string, error) {
	// Validate handler path to prevent command injection
	if err := validateHandlerPath(handlerPath); err != nil {
		return "", err
	}

	config := CursorHookConfig{
		Hooks: make(map[string][]CursorHookEntry),
	}

	for _, hookType := range cursorHookTypes {
		cmd := handlerPath
		if runtime.GOOS == "windows" {
			cmd = handlerPath + ".exe"
		}
		// Quote the path for safe shell execution
		quotedCmd := quotePathForShell(cmd)
		// Include event type in command for proper categorization
		config.Hooks[hookType] = []CursorHookEntry{{
			Command: quotedCmd + " hook --tool cursor --event " + hookType,
		}}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "{}", nil
	}
	return string(data), nil
}

// claudeCodeHookTypes contains all available hooks per https://code.claude.com/docs/en/hooks.
var claudeCodeHookTypes = []string{
	// Tool hooks (matcher applies)
	"PreToolUse",
	"PostToolUse",
	"PostToolUseFailure",
	"PermissionRequest",
	// Session lifecycle hooks
	"SessionStart",
	"SessionEnd",
	"Stop",
	// User interaction hooks
	"UserPromptSubmit",
	"Notification",
	// Subagent hooks
	"SubagentStart",
	"SubagentStop",
	// Other hooks
	"PreCompact",
}

// GenerateClaudeCodeHooks creates the Claude Code hooks configuration.
// Returns an error if the handler path contains unsafe characters.
func GenerateClaudeCodeHooks(handlerPath string) (map[string]any, error) {
	// Validate handler path to prevent command injection
	if err := validateHandlerPath(handlerPath); err != nil {
		return nil, err
	}

	cmd := handlerPath
	if runtime.GOOS == "windows" {
		cmd = handlerPath + ".exe"
	}

	// Quote the path for safe shell execution
	quotedCmd := quotePathForShell(cmd)

	// Claude Code uses a different hook structure
	// Hooks are defined per event type with matchers
	// matcher: ".*" matches all tools/events
	hooks := make(map[string]any)

	for _, hookType := range claudeCodeHookTypes {
		hooks[hookType] = []map[string]any{
			{
				"matcher": ".*",
				"hooks": []map[string]string{
					{
						"type":    "command",
						"command": quotedCmd + " hook --tool claude --event " + hookType,
					},
				},
			},
		}
	}

	return hooks, nil
}
