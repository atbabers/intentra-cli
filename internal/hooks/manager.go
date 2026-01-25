package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Tool represents an AI coding tool.
type Tool string

const (
	ToolCursor     Tool = "cursor"
	ToolClaudeCode Tool = "claude"
	ToolGeminiCLI  Tool = "gemini"
)

// AllTools returns all supported tools.
func AllTools() []Tool {
	return []Tool{ToolCursor, ToolClaudeCode, ToolGeminiCLI}
}

// ToolStatus represents the installation status of a tool.
type ToolStatus struct {
	Tool      Tool
	Installed bool
	Path      string
	Error     error
}

// GetHooksDir returns the hooks directory for a tool.
func GetHooksDir(tool Tool) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch tool {
	case ToolCursor:
		return getCursorHooksDir(home)
	case ToolClaudeCode:
		return getClaudeCodeDir(home)
	case ToolGeminiCLI:
		return getGeminiCLIDir(home)
	default:
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
}

func getCursorHooksDir(home string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Cursor", "hooks"), nil
	default:
		return filepath.Join(home, ".cursor", "hooks"), nil
	}
}

func getClaudeCodeDir(home string) (string, error) {
	return filepath.Join(home, ".claude"), nil
}

func getGeminiCLIDir(home string) (string, error) {
	return filepath.Join(home, ".gemini"), nil
}

// Install installs hooks for the specified tool.
func Install(tool Tool, handlerPath string) error {
	switch tool {
	case ToolCursor:
		return installCursor(handlerPath)
	case ToolClaudeCode:
		return installClaudeCode(handlerPath)
	case ToolGeminiCLI:
		return installGeminiCLI(handlerPath)
	default:
		return fmt.Errorf("unknown tool: %s", tool)
	}
}

// InstallAll installs hooks for all supported tools.
func InstallAll(handlerPath string) map[Tool]error {
	results := make(map[Tool]error)
	for _, tool := range AllTools() {
		results[tool] = Install(tool, handlerPath)
	}
	return results
}

// Uninstall removes hooks for the specified tool.
func Uninstall(tool Tool) error {
	switch tool {
	case ToolCursor:
		return uninstallCursor()
	case ToolClaudeCode:
		return uninstallClaudeCode()
	case ToolGeminiCLI:
		return uninstallGeminiCLI()
	default:
		return fmt.Errorf("unknown tool: %s", tool)
	}
}

// UninstallAll removes hooks for all supported tools.
func UninstallAll() map[Tool]error {
	results := make(map[Tool]error)
	for _, tool := range AllTools() {
		results[tool] = Uninstall(tool)
	}
	return results
}

// Status returns installation status for all tools.
func Status() []ToolStatus {
	var statuses []ToolStatus
	for _, tool := range AllTools() {
		status := ToolStatus{Tool: tool}
		status.Installed, status.Path, status.Error = checkStatus(tool)
		statuses = append(statuses, status)
	}
	return statuses
}

func checkStatus(tool Tool) (bool, string, error) {
	dir, err := GetHooksDir(tool)
	if err != nil {
		return false, "", err
	}

	switch tool {
	case ToolCursor:
		hooksFile := filepath.Join(dir, "hooks.json")
		if _, err := os.Stat(hooksFile); os.IsNotExist(err) {
			return false, dir, nil
		}
		return true, dir, nil

	case ToolClaudeCode:
		settingsFile := filepath.Join(dir, "settings.json")
		if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
			return false, dir, nil
		}
		// Check if hooks are configured
		data, err := os.ReadFile(settingsFile)
		if err != nil {
			return false, dir, err
		}
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			return false, dir, err
		}
		if _, ok := settings["hooks"]; ok {
			return true, dir, nil
		}
		return false, dir, nil

	case ToolGeminiCLI:
		settingsFile := filepath.Join(dir, "settings.json")
		if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
			return false, dir, nil
		}
		data, err := os.ReadFile(settingsFile)
		if err != nil {
			return false, dir, err
		}
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			return false, dir, err
		}
		if hooks, ok := settings["hooks"].(map[string]any); ok {
			if enabled, ok := hooks["enabled"].(bool); ok && enabled {
				return true, dir, nil
			}
		}
		return false, dir, nil

	default:
		return false, "", fmt.Errorf("unknown tool: %s", tool)
	}
}

// --- Cursor ---

func installCursor(handlerPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dir, err := getCursorHooksDir(home)
	if err != nil {
		return fmt.Errorf("failed to get hooks directory: %w", err)
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	hooksJSON, err := GenerateCursorHooksJSON(handlerPath)
	if err != nil {
		return fmt.Errorf("invalid handler path: %w", err)
	}
	hooksFile := filepath.Join(dir, "hooks.json")

	if err := os.WriteFile(hooksFile, []byte(hooksJSON), 0600); err != nil {
		return fmt.Errorf("failed to write hooks.json: %w", err)
	}

	return nil
}

func uninstallCursor() error {
	home, _ := os.UserHomeDir()
	dir, _ := getCursorHooksDir(home)
	hooksFile := filepath.Join(dir, "hooks.json")

	if _, err := os.Stat(hooksFile); os.IsNotExist(err) {
		return fmt.Errorf("no hooks.json found at %s", dir)
	}

	if err := os.Remove(hooksFile); err != nil {
		return fmt.Errorf("failed to remove hooks.json: %w", err)
	}

	return nil
}

// --- Claude Code ---

func installClaudeCode(handlerPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir, _ := getClaudeCodeDir(home)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	settingsFile := filepath.Join(dir, "settings.json")

	// Read existing settings if any
	var settings map[string]any
	if data, err := os.ReadFile(settingsFile); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = nil
		}
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	// Add hooks configuration
	hooksConfig, err := GenerateClaudeCodeHooks(handlerPath)
	if err != nil {
		return fmt.Errorf("invalid handler path: %w", err)
	}
	settings["hooks"] = hooksConfig

	// Write back
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	return nil
}

func uninstallClaudeCode() error {
	home, _ := os.UserHomeDir()
	dir, _ := getClaudeCodeDir(home)
	settingsFile := filepath.Join(dir, "settings.json")

	// Read existing settings
	data, err := os.ReadFile(settingsFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("no settings.json found at %s", dir)
	}
	if err != nil {
		return err
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	// Remove hooks
	delete(settings, "hooks")

	// Write back
	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, newData, 0600)
}

// --- Gemini CLI ---

func installGeminiCLI(handlerPath string) error {
	// Validate handler path to prevent command injection
	if err := validateHandlerPath(handlerPath); err != nil {
		return fmt.Errorf("invalid handler path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir, _ := getGeminiCLIDir(home)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create .gemini directory: %w", err)
	}

	settingsFile := filepath.Join(dir, "settings.json")

	// Read existing settings if any
	var settings map[string]any
	if data, err := os.ReadFile(settingsFile); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = nil
		}
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	// Ensure tools.enableHooks = true
	tools, ok := settings["tools"].(map[string]any)
	if !ok {
		tools = make(map[string]any)
	}
	tools["enableHooks"] = true
	settings["tools"] = tools

	// Build hooks configuration
	hooks := make(map[string]any)
	hooks["enabled"] = true

	// Quote the path for safe shell execution
	quotedPath := quotePathForShell(handlerPath)

	// All Gemini CLI hook events
	geminiEvents := []string{
		"SessionStart", "SessionEnd",
		"BeforeAgent", "AfterAgent",
		"BeforeModel", "AfterModel",
		"BeforeToolSelection",
		"BeforeTool", "AfterTool",
		"PreCompress", "Notification",
	}

	for _, event := range geminiEvents {
		hooks[event] = []map[string]string{
			{"command": fmt.Sprintf("%s audit --tool gemini_cli --event %s", quotedPath, event)},
		}
	}

	settings["hooks"] = hooks

	// Write back
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	return nil
}

func uninstallGeminiCLI() error {
	home, _ := os.UserHomeDir()
	dir, _ := getGeminiCLIDir(home)
	settingsFile := filepath.Join(dir, "settings.json")

	// Read existing settings
	data, err := os.ReadFile(settingsFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("no settings.json found at %s", dir)
	}
	if err != nil {
		return err
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	// Remove intentra hooks but preserve user hooks
	if hooks, ok := settings["hooks"].(map[string]any); ok {
		for event, hookList := range hooks {
			if event == "enabled" {
				continue
			}
			if list, ok := hookList.([]any); ok {
				var filtered []any
				for _, h := range list {
					if hookMap, ok := h.(map[string]any); ok {
						if cmd, ok := hookMap["command"].(string); ok {
							if !strings.Contains(cmd, "intentra") {
								filtered = append(filtered, h)
							}
						}
					}
				}
				if len(filtered) > 0 {
					hooks[event] = filtered
				} else {
					delete(hooks, event)
				}
			}
		}

		// If only "enabled" key remains, disable hooks
		if len(hooks) == 1 {
			hooks["enabled"] = false
		}
	}

	// Write back
	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, newData, 0600)
}

// --- Legacy compatibility ---

// GetCursorHooksDir returns the Cursor hooks directory (legacy).
func GetCursorHooksDir() (string, error) {
	home, _ := os.UserHomeDir()
	return getCursorHooksDir(home)
}

// GenerateHooksJSON generates Cursor hooks.json (legacy alias).
// Returns empty JSON on error for backward compatibility.
func GenerateHooksJSON(handlerPath string) string {
	result, err := GenerateCursorHooksJSON(handlerPath)
	if err != nil {
		return "{}"
	}
	return result
}
