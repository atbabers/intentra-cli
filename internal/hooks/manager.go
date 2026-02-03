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
		return filepath.Join(os.Getenv("APPDATA"), "Cursor"), nil
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
	case "darwin":
		return filepath.Join(home, ".codeium", "windsurf"), nil
	default:
		return filepath.Join(home, ".codeium", "windsurf"), nil
	}
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
	case ToolCopilot:
		return installCopilot(handlerPath)
	case ToolWindsurf:
		return installWindsurf(handlerPath)
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
	case ToolCopilot:
		return uninstallCopilot()
	case ToolWindsurf:
		return uninstallWindsurf()
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

	case ToolCopilot:
		hooksFile := filepath.Join(dir, "hooks.json")
		if _, err := os.Stat(hooksFile); os.IsNotExist(err) {
			return false, dir, nil
		}
		data, err := os.ReadFile(hooksFile)
		if err != nil {
			return false, dir, err
		}
		var config map[string]any
		if err := json.Unmarshal(data, &config); err != nil {
			return false, dir, err
		}
		if hooks, ok := config["hooks"].(map[string]any); ok && len(hooks) > 0 {
			return true, dir, nil
		}
		return false, dir, nil

	case ToolWindsurf:
		hooksFile := filepath.Join(dir, "hooks.json")
		if _, err := os.Stat(hooksFile); os.IsNotExist(err) {
			return false, dir, nil
		}
		data, err := os.ReadFile(hooksFile)
		if err != nil {
			return false, dir, err
		}
		var config map[string]any
		if err := json.Unmarshal(data, &config); err != nil {
			return false, dir, err
		}
		if hooks, ok := config["hooks"].(map[string]any); ok && len(hooks) > 0 {
			return true, dir, nil
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
			cleanedHooks := removeIntentraHooksFromMap(existingHooks)
			if newHooks, ok := newConfig["hooks"].(map[string]any); ok {
				existingConfig["hooks"] = mergeHookMaps(cleanedHooks, newHooks)
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
		cleanedHooks := removeIntentraHooksFromMap(hooks)
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
		existingHooks = removeIntentraHooks(existingHooks)
		settings["hooks"] = mergeHooks(existingHooks, newHooksConfig)
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

func removeIntentraHooks(hooks map[string]any) map[string]any {
	cleaned := make(map[string]any)
	for eventType, hookList := range hooks {
		if list, ok := hookList.([]any); ok {
			var filtered []any
			for _, item := range list {
				if itemMap, ok := item.(map[string]any); ok {
					if innerHooks, ok := itemMap["hooks"].([]any); ok {
						var filteredInner []any
						for _, h := range innerHooks {
							if hookEntry, ok := h.(map[string]any); ok {
								if cmd, ok := hookEntry["command"].(string); ok {
									if !strings.Contains(cmd, "intentra") {
										filteredInner = append(filteredInner, h)
									}
								}
							}
						}
						if len(filteredInner) > 0 {
							itemMap["hooks"] = filteredInner
							filtered = append(filtered, itemMap)
						}
					} else if cmd, ok := itemMap["command"].(string); ok {
						if !strings.Contains(cmd, "intentra") {
							filtered = append(filtered, item)
						}
					}
				}
			}
			if len(filtered) > 0 {
				cleaned[eventType] = filtered
			}
		}
	}
	return cleaned
}

func mergeHooks(existing, newHooks map[string]any) map[string]any {
	merged := make(map[string]any)
	for k, v := range existing {
		merged[k] = v
	}
	for eventType, newList := range newHooks {
		if existingList, ok := merged[eventType].([]any); ok {
			if newArr, ok := newList.([]map[string]any); ok {
				for _, item := range newArr {
					existingList = append(existingList, item)
				}
				merged[eventType] = existingList
			}
		} else {
			merged[eventType] = newList
		}
	}
	return merged
}

func removeIntentraHooksFromMap(hooks map[string]any) map[string]any {
	cleaned := make(map[string]any)
	for eventType, hookList := range hooks {
		if list, ok := hookList.([]any); ok {
			var filtered []any
			for _, item := range list {
				if itemMap, ok := item.(map[string]any); ok {
					if cmd, ok := itemMap["command"].(string); ok {
						if !strings.Contains(cmd, "intentra") {
							filtered = append(filtered, item)
						}
						continue
					}
					if bash, ok := itemMap["bash"].(string); ok {
						if !strings.Contains(bash, "intentra") {
							filtered = append(filtered, item)
						}
						continue
					}
				}
			}
			if len(filtered) > 0 {
				cleaned[eventType] = filtered
			}
		}
	}
	return cleaned
}

func mergeHookMaps(existing, newHooks map[string]any) map[string]any {
	merged := make(map[string]any)
	for k, v := range existing {
		merged[k] = v
	}
	for eventType, newList := range newHooks {
		if existingList, ok := merged[eventType].([]any); ok {
			if newArr, ok := newList.([]any); ok {
				existingList = append(existingList, newArr...)
				merged[eventType] = existingList
			}
		} else {
			merged[eventType] = newList
		}
	}
	return merged
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
		cleanedHooks := removeIntentraHooks(hooks)
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
		"BeforeTool", "AfterTool",
		"BeforeModel", "AfterModel",
		"SessionStart", "SessionEnd",
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
		cleanedHooks := removeIntentraHooksFromGemini(existingHooks)
		settings["hooks"] = mergeGeminiHooks(cleanedHooks, newHooks)
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

func removeIntentraHooksFromGemini(hooks map[string]any) map[string]any {
	cleaned := make(map[string]any)
	for eventType, hookList := range hooks {
		if list, ok := hookList.([]any); ok {
			var filtered []any
			for _, item := range list {
				if itemMap, ok := item.(map[string]any); ok {
					if innerHooks, ok := itemMap["hooks"].([]any); ok {
						var filteredInner []any
						for _, h := range innerHooks {
							if hookEntry, ok := h.(map[string]any); ok {
								isIntentra := false
								if name, ok := hookEntry["name"].(string); ok && strings.Contains(name, "intentra") {
									isIntentra = true
								}
								if cmd, ok := hookEntry["command"].(string); ok && strings.Contains(cmd, "intentra") {
									isIntentra = true
								}
								if !isIntentra {
									filteredInner = append(filteredInner, h)
								}
							}
						}
						if len(filteredInner) > 0 {
							itemMap["hooks"] = filteredInner
							filtered = append(filtered, itemMap)
						}
					}
				}
			}
			if len(filtered) > 0 {
				cleaned[eventType] = filtered
			}
		}
	}
	return cleaned
}

func mergeGeminiHooks(existing, newHooks map[string]any) map[string]any {
	merged := make(map[string]any)
	for k, v := range existing {
		merged[k] = v
	}
	for eventType, newList := range newHooks {
		if existingList, ok := merged[eventType].([]any); ok {
			if newArr, ok := newList.([]map[string]any); ok {
				for _, item := range newArr {
					existingList = append(existingList, item)
				}
				merged[eventType] = existingList
			}
		} else {
			merged[eventType] = newList
		}
	}
	return merged
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
		cleanedHooks := removeIntentraHooksFromGemini(hooks)
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
			cleanedHooks := removeIntentraHooksFromCopilot(existingHooks)
			if newHooks, ok := newConfig["hooks"].(map[string]any); ok {
				existingConfig["hooks"] = mergeHookMaps(cleanedHooks, newHooks)
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

func removeIntentraHooksFromCopilot(hooks map[string]any) map[string]any {
	cleaned := make(map[string]any)
	for eventType, hookList := range hooks {
		if list, ok := hookList.([]any); ok {
			var filtered []any
			for _, item := range list {
				if itemMap, ok := item.(map[string]any); ok {
					isIntentra := false
					if bash, ok := itemMap["bash"].(string); ok && strings.Contains(bash, "intentra") {
						isIntentra = true
					}
					if ps, ok := itemMap["powershell"].(string); ok && strings.Contains(ps, "intentra") {
						isIntentra = true
					}
					if !isIntentra {
						filtered = append(filtered, item)
					}
				}
			}
			if len(filtered) > 0 {
				cleaned[eventType] = filtered
			}
		}
	}
	return cleaned
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
		cleanedHooks := removeIntentraHooksFromCopilot(hooks)
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
			cleanedHooks := removeIntentraHooksFromMap(existingHooks)
			if newHooks, ok := newConfig["hooks"].(map[string]any); ok {
				existingConfig["hooks"] = mergeHookMaps(cleanedHooks, newHooks)
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
		cleanedHooks := removeIntentraHooksFromMap(hooks)
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
