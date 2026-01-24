package hooks

import (
	"fmt"

	"github.com/atbabers/intentra-cli/pkg/models"
)

// Normalizer converts tool-specific events to NormalizedEvent
type Normalizer interface {
	Normalize(rawEvent map[string]any) (*models.NormalizedEvent, error)
	Tool() string
}

// GetNormalizer returns the appropriate normalizer for a tool
func GetNormalizer(tool string) (Normalizer, error) {
	switch tool {
	case "claude_code", "claude":
		return &ClaudeCodeNormalizer{}, nil
	case "cursor":
		return &CursorNormalizer{}, nil
	case "gemini_cli", "gemini":
		return &GeminiCLINormalizer{}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}
}

// Helper functions for all normalizers

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

func getInt(m map[string]any, key string) *int {
	if v, ok := m[key].(float64); ok {
		i := int(v)
		return &i
	}
	if v, ok := m[key].(int); ok {
		return &v
	}
	return nil
}

func isShellTool(name string) bool {
	shellTools := map[string]bool{
		"shell": true, "bash": true, "terminal": true,
		"Bash": true, "Shell": true, "execute_command": true,
	}
	return shellTools[name]
}

func isFileTool(name string) bool {
	fileTools := map[string]bool{
		"read_file": true, "write_file": true, "edit_file": true,
		"Read": true, "Write": true, "Edit": true,
		"file_read": true, "file_write": true, "file_edit": true,
	}
	return fileTools[name]
}

func isMCPTool(name string) bool {
	// MCP tools typically have server prefix like "mcp__server__tool"
	return len(name) > 4 && name[:4] == "mcp_"
}
