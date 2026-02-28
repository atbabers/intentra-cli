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
	ToolCopilot    Tool = "copilot"
	ToolWindsurf   Tool = "windsurf"
)

// AllTools returns all supported tools.
func AllTools() []Tool {
	return []Tool{ToolCursor, ToolClaudeCode, ToolGeminiCLI, ToolCopilot, ToolWindsurf}
}

// ToolStatus represents the installation status of a tool.
type ToolStatus struct {
	Tool      Tool
	Installed bool
	Path      string
	Error     error
}

// toolOps defines per-tool install, uninstall, and status-check operations.
type toolOps struct {
	install   func(string) error
	uninstall func() error
	checkFile string
	// checkHook inspects parsed JSON config to determine if hooks are installed.
	// Nil means file existence alone is sufficient.
	checkHook func(config map[string]any) bool
}

var toolRegistry = map[Tool]toolOps{
	ToolCursor: {
		install: installCursor, uninstall: uninstallCursor,
		checkFile: "hooks.json",
		checkHook: nil,
	},
	ToolClaudeCode: {
		install: installClaudeCode, uninstall: uninstallClaudeCode,
		checkFile: "settings.json",
		checkHook: func(c map[string]any) bool { _, ok := c["hooks"]; return ok },
	},
	ToolGeminiCLI: {
		install: installGeminiCLI, uninstall: uninstallGeminiCLI,
		checkFile: "settings.json",
		checkHook: func(c map[string]any) bool {
			hooks, ok := c["hooks"].(map[string]any)
			return ok && len(hooks) > 0
		},
	},
	ToolCopilot: {
		install: installCopilot, uninstall: uninstallCopilot,
		checkFile: "hooks.json",
		checkHook: func(c map[string]any) bool {
			hooks, ok := c["hooks"].(map[string]any)
			return ok && len(hooks) > 0
		},
	},
	ToolWindsurf: {
		install: installWindsurf, uninstall: uninstallWindsurf,
		checkFile: "hooks.json",
		checkHook: func(c map[string]any) bool {
			hooks, ok := c["hooks"].(map[string]any)
			return ok && len(hooks) > 0
		},
	},
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
	case ToolCopilot:
		return getCopilotHooksDir(home)
	case ToolWindsurf:
		return getWindsurfHooksDir(home)
	default:
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
}

func getCursorHooksDir(home string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "Cursor"), nil
	default:
		return filepath.Join(home, ".cursor"), nil
	}
}

func getClaudeCodeDir(home string) (string, error) {
	return filepath.Join(home, ".claude"), nil
}

func getGeminiCLIDir(home string) (string, error) {
	return filepath.Join(home, ".gemini"), nil
}

func getCopilotHooksDir(home string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "GitHub Copilot", "hooks"), nil
	default:
		return filepath.Join(home, ".config", "github-copilot", "hooks"), nil
	}
}

func getWindsurfHooksDir(home string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Windsurf"), nil
	default:
		return filepath.Join(home, ".codeium", "windsurf"), nil
	}
}

// Install installs hooks for the specified tool.
func Install(tool Tool, handlerPath string) error {
	ops, ok := toolRegistry[tool]
	if !ok {
		return fmt.Errorf("unknown tool: %s", tool)
	}
	return ops.install(handlerPath)
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
	ops, ok := toolRegistry[tool]
	if !ok {
		return fmt.Errorf("unknown tool: %s", tool)
	}
	return ops.uninstall()
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

// AnyHooksInstalled returns true if hooks are installed for any tool.
func AnyHooksInstalled() bool {
	for _, status := range Status() {
		if status.Installed {
			return true
		}
	}
	return false
}

func checkStatus(tool Tool) (bool, string, error) {
	dir, err := GetHooksDir(tool)
	if err != nil {
		return false, "", err
	}

	ops, ok := toolRegistry[tool]
	if !ok {
		return false, "", fmt.Errorf("unknown tool: %s", tool)
	}

	filePath := filepath.Join(dir, ops.checkFile)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, dir, nil
	}

	if ops.checkHook == nil {
		return true, dir, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, dir, err
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return false, dir, err
	}
	return ops.checkHook(config), dir, nil
}

// mergeHookEntries merges incoming hook entries into existing hooks by event type.
// For each event type, if existing entries exist as []any, new entries are appended.
func mergeHookEntries(existing, incoming map[string]any) map[string]any {
	merged := make(map[string]any)
	for k, v := range existing {
		merged[k] = v
	}
	for eventType, newList := range incoming {
		existingList, ok := merged[eventType].([]any)
		if !ok {
			merged[eventType] = newList
			continue
		}
		switch nl := newList.(type) {
		case []any:
			existingList = append(existingList, nl...)
		case []map[string]any:
			for _, item := range nl {
				existingList = append(existingList, item)
			}
		}
		merged[eventType] = existingList
	}
	return merged
}

// isIntentraEntry returns true if any of the specified fields contain "intentra".
func isIntentraEntry(m map[string]any, fields ...string) bool {
	for _, f := range fields {
		if v, ok := m[f].(string); ok && strings.Contains(v, "intentra") {
			return true
		}
	}
	return false
}

// removeIntentraFromHooks removes all intentra entries from a hooks map.
// innerFields specifies fields to check within nested "hooks" arrays.
// outerFields specifies fields to check on top-level items.
func removeIntentraFromHooks(hooks map[string]any, innerFields, outerFields []string) map[string]any {
	cleaned := make(map[string]any)
	for eventType, hookList := range hooks {
		list, ok := hookList.([]any)
		if !ok {
			continue
		}
		var filtered []any
		for _, item := range list {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if innerHooks, ok := itemMap["hooks"].([]any); ok && len(innerFields) > 0 {
				var filteredInner []any
				for _, h := range innerHooks {
					if hookEntry, ok := h.(map[string]any); ok {
						if !isIntentraEntry(hookEntry, innerFields...) {
							filteredInner = append(filteredInner, h)
						}
					}
				}
				if len(filteredInner) > 0 {
					itemMap["hooks"] = filteredInner
					filtered = append(filtered, itemMap)
				}
			} else if len(outerFields) > 0 {
				if !isIntentraEntry(itemMap, outerFields...) {
					filtered = append(filtered, item)
				}
			}
		}
		if len(filtered) > 0 {
			cleaned[eventType] = filtered
		}
	}
	return cleaned
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

	hooksFile := filepath.Join(dir, "hooks.json")

	var existingConfig map[string]any
	if data, err := os.ReadFile(hooksFile); err == nil {
		if err := json.Unmarshal(data, &existingConfig); err != nil {
			existingConfig = nil
		}
	}

	newHooksJSON, err := GenerateCursorHooksJSON(handlerPath)
	if err != nil {
		return fmt.Errorf("invalid handler path: %w", err)
	}

	var newConfig map[string]any
	if err := json.Unmarshal([]byte(newHooksJSON), &newConfig); err != nil {
		return fmt.Errorf("failed to parse generated hooks config: %w", err)
	}

	if existingConfig != nil {
		if existingHooks, ok := existingConfig["hooks"].(map[string]any); ok {
			cleanedHooks := removeIntentraFromHooks(existingHooks, nil, []string{"command", "bash"})
			if newHooks, ok := newConfig["hooks"].(map[string]any); ok {
				existingConfig["hooks"] = mergeHookEntries(cleanedHooks, newHooks)
			}
		} else {
			existingConfig["hooks"] = newConfig["hooks"]
		}
	} else {
		existingConfig = newConfig
	}

	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hooks: %w", err)
	}

	if err := os.WriteFile(hooksFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write hooks.json: %w", err)
	}

	return nil
}

func uninstallCursor() error {
	home, _ := os.UserHomeDir()
	dir, _ := getCursorHooksDir(home)
	hooksFile := filepath.Join(dir, "hooks.json")

	data, err := os.ReadFile(hooksFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("no hooks.json found at %s", dir)
	}
	if err != nil {
		return err
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if hooks, ok := config["hooks"].(map[string]any); ok {
		cleanedHooks := removeIntentraFromHooks(hooks, nil, []string{"command", "bash"})
		if len(cleanedHooks) > 0 {
			config["hooks"] = cleanedHooks
		} else {
			delete(config, "hooks")
		}
	}

	if hooks, ok := config["hooks"].(map[string]any); !ok || len(hooks) == 0 {
		return os.Remove(hooksFile)
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(hooksFile, newData, 0600)
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

	var settings map[string]any
	if data, err := os.ReadFile(settingsFile); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = nil
		}
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	newHooksConfig, err := GenerateClaudeCodeHooks(handlerPath)
	if err != nil {
		return fmt.Errorf("invalid handler path: %w", err)
	}

	existingHooks, _ := settings["hooks"].(map[string]any)
	if existingHooks != nil {
		existingHooks = removeIntentraFromHooks(existingHooks, []string{"command"}, []string{"command"})
		settings["hooks"] = mergeHookEntries(existingHooks, newHooksConfig)
	} else {
		settings["hooks"] = newHooksConfig
	}

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

	if hooks, ok := settings["hooks"].(map[string]any); ok {
		cleanedHooks := removeIntentraFromHooks(hooks, []string{"command"}, []string{"command"})
		if len(cleanedHooks) > 0 {
			settings["hooks"] = cleanedHooks
		} else {
			delete(settings, "hooks")
		}
	}

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, newData, 0600)
}

// --- Gemini CLI ---

func installGeminiCLI(handlerPath string) error {
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

	var settings map[string]any
	if data, err := os.ReadFile(settingsFile); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = nil
		}
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	quotedPath := quotePathForShell(handlerPath)

	geminiEvents := []string{
		"SessionStart", "SessionEnd",
		"BeforeAgent", "AfterAgent",
		"BeforeModel", "AfterModel",
		"BeforeToolSelection",
		"BeforeTool", "AfterTool",
		"PreCompress", "Notification",
	}

	newHooks := make(map[string]any)
	for _, event := range geminiEvents {
		newHooks[event] = []map[string]any{
			{
				"matcher": ".*",
				"hooks": []map[string]any{
					{
						"name":    "intentra-" + event,
						"type":    "command",
						"command": fmt.Sprintf("%s hook --tool gemini --event %s", quotedPath, event),
						"timeout": 30000,
					},
				},
			},
		}
	}

	if existingHooks, ok := settings["hooks"].(map[string]any); ok {
		cleanedHooks := removeIntentraFromHooks(existingHooks, []string{"name", "command"}, nil)
		settings["hooks"] = mergeHookEntries(cleanedHooks, newHooks)
	} else {
		settings["hooks"] = newHooks
	}

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

	if hooks, ok := settings["hooks"].(map[string]any); ok {
		cleanedHooks := removeIntentraFromHooks(hooks, []string{"name", "command"}, nil)
		if len(cleanedHooks) > 0 {
			settings["hooks"] = cleanedHooks
		} else {
			delete(settings, "hooks")
		}
	}

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, newData, 0600)
}

// --- GitHub Copilot ---

func installCopilot(handlerPath string) error {
	if err := validateHandlerPath(handlerPath); err != nil {
		return fmt.Errorf("invalid handler path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dir, err := getCopilotHooksDir(home)
	if err != nil {
		return fmt.Errorf("failed to get hooks directory: %w", err)
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	hooksFile := filepath.Join(dir, "hooks.json")

	var existingConfig map[string]any
	if data, err := os.ReadFile(hooksFile); err == nil {
		if err := json.Unmarshal(data, &existingConfig); err != nil {
			existingConfig = nil
		}
	}

	newHooksJSON, err := GenerateCopilotHooksJSON(handlerPath)
	if err != nil {
		return fmt.Errorf("failed to generate hooks config: %w", err)
	}

	var newConfig map[string]any
	if err := json.Unmarshal([]byte(newHooksJSON), &newConfig); err != nil {
		return fmt.Errorf("failed to parse generated hooks config: %w", err)
	}

	if existingConfig != nil {
		if existingHooks, ok := existingConfig["hooks"].(map[string]any); ok {
			cleanedHooks := removeIntentraFromHooks(existingHooks, nil, []string{"bash", "powershell"})
			if newHooks, ok := newConfig["hooks"].(map[string]any); ok {
				existingConfig["hooks"] = mergeHookEntries(cleanedHooks, newHooks)
			}
		} else {
			existingConfig["hooks"] = newConfig["hooks"]
		}
		existingConfig["version"] = newConfig["version"]
	} else {
		existingConfig = newConfig
	}

	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hooks: %w", err)
	}

	if err := os.WriteFile(hooksFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write hooks.json: %w", err)
	}

	return nil
}

func uninstallCopilot() error {
	home, _ := os.UserHomeDir()
	dir, _ := getCopilotHooksDir(home)
	hooksFile := filepath.Join(dir, "hooks.json")

	data, err := os.ReadFile(hooksFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("no hooks.json found at %s", dir)
	}
	if err != nil {
		return err
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if hooks, ok := config["hooks"].(map[string]any); ok {
		cleanedHooks := removeIntentraFromHooks(hooks, nil, []string{"bash", "powershell"})
		if len(cleanedHooks) > 0 {
			config["hooks"] = cleanedHooks
		} else {
			delete(config, "hooks")
		}
	}

	if hooks, ok := config["hooks"].(map[string]any); !ok || len(hooks) == 0 {
		return os.Remove(hooksFile)
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(hooksFile, newData, 0600)
}

// --- Windsurf Cascade ---

func installWindsurf(handlerPath string) error {
	if err := validateHandlerPath(handlerPath); err != nil {
		return fmt.Errorf("invalid handler path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dir, err := getWindsurfHooksDir(home)
	if err != nil {
		return fmt.Errorf("failed to get hooks directory: %w", err)
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	hooksFile := filepath.Join(dir, "hooks.json")

	var existingConfig map[string]any
	if data, err := os.ReadFile(hooksFile); err == nil {
		if err := json.Unmarshal(data, &existingConfig); err != nil {
			existingConfig = nil
		}
	}

	newHooksJSON, err := GenerateWindsurfHooksJSON(handlerPath)
	if err != nil {
		return fmt.Errorf("failed to generate hooks config: %w", err)
	}

	var newConfig map[string]any
	if err := json.Unmarshal([]byte(newHooksJSON), &newConfig); err != nil {
		return fmt.Errorf("failed to parse generated hooks config: %w", err)
	}

	if existingConfig != nil {
		if existingHooks, ok := existingConfig["hooks"].(map[string]any); ok {
			cleanedHooks := removeIntentraFromHooks(existingHooks, nil, []string{"command", "bash"})
			if newHooks, ok := newConfig["hooks"].(map[string]any); ok {
				existingConfig["hooks"] = mergeHookEntries(cleanedHooks, newHooks)
			}
		} else {
			existingConfig["hooks"] = newConfig["hooks"]
		}
	} else {
		existingConfig = newConfig
	}

	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hooks: %w", err)
	}

	if err := os.WriteFile(hooksFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write hooks.json: %w", err)
	}

	return nil
}

func uninstallWindsurf() error {
	home, _ := os.UserHomeDir()
	dir, _ := getWindsurfHooksDir(home)
	hooksFile := filepath.Join(dir, "hooks.json")

	data, err := os.ReadFile(hooksFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("no hooks.json found at %s", dir)
	}
	if err != nil {
		return err
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if hooks, ok := config["hooks"].(map[string]any); ok {
		cleanedHooks := removeIntentraFromHooks(hooks, nil, []string{"command", "bash"})
		if len(cleanedHooks) > 0 {
			config["hooks"] = cleanedHooks
		} else {
			delete(config, "hooks")
		}
	}

	if hooks, ok := config["hooks"].(map[string]any); !ok || len(hooks) == 0 {
		return os.Remove(hooksFile)
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(hooksFile, newData, 0600)
}
