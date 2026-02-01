package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateHooksJSON(t *testing.T) {
	// Test Cursor hooks with valid path
	json, err := GenerateCursorHooksJSON("/path/to/handler")
	if err != nil {
		t.Errorf("GenerateCursorHooksJSON failed: %v", err)
	}
	if json == "" {
		t.Error("GenerateCursorHooksJSON returned empty string")
	}
	if !contains(json, "sessionStart") {
		t.Error("Missing sessionStart hook")
	}
	if !contains(json, "\"version\": 1") {
		t.Error("Missing version field")
	}

	// Test Claude Code hooks with valid path
	claudeHooks, err := GenerateClaudeCodeHooks("/path/to/handler")
	if err != nil {
		t.Errorf("GenerateClaudeCodeHooks failed: %v", err)
	}
	if claudeHooks == nil {
		t.Error("GenerateClaudeCodeHooks returned nil")
	}
	if len(claudeHooks) == 0 {
		t.Error("GenerateClaudeCodeHooks returned empty map")
	}
}

func TestGenerateHooksJSON_InvalidPath(t *testing.T) {
	// Test with command injection attempt
	_, err := GenerateCursorHooksJSON("/path/to/handler; rm -rf /")
	if err == nil {
		t.Error("GenerateCursorHooksJSON should reject path with semicolon")
	}

	// Test with backtick injection
	_, err = GenerateCursorHooksJSON("/path/to/handler`id`")
	if err == nil {
		t.Error("GenerateCursorHooksJSON should reject path with backticks")
	}

	// Test with pipe injection
	_, err = GenerateCursorHooksJSON("/path/to/handler | cat /etc/passwd")
	if err == nil {
		t.Error("GenerateCursorHooksJSON should reject path with pipe")
	}

	// Test with empty path
	_, err = GenerateCursorHooksJSON("")
	if err == nil {
		t.Error("GenerateCursorHooksJSON should reject empty path")
	}

	// Test Claude Code with invalid path
	_, err = GenerateClaudeCodeHooks("/path/$(whoami)/handler")
	if err == nil {
		t.Error("GenerateClaudeCodeHooks should reject path with command substitution")
	}
}

func TestGetHooksDir(t *testing.T) {
	// Test Cursor
	cursorDir, err := GetHooksDir(ToolCursor)
	if err != nil {
		t.Errorf("GetHooksDir(cursor) failed: %v", err)
	}
	if cursorDir == "" {
		t.Error("GetHooksDir(cursor) returned empty string")
	}

	// Test Claude Code
	claudeDir, err := GetHooksDir(ToolClaudeCode)
	if err != nil {
		t.Errorf("GetHooksDir(claude) failed: %v", err)
	}
	if claudeDir == "" {
		t.Error("GetHooksDir(claude) returned empty string")
	}

	// Test Gemini CLI
	geminiDir, err := GetHooksDir(ToolGeminiCLI)
	if err != nil {
		t.Errorf("GetHooksDir(gemini) failed: %v", err)
	}
	if geminiDir == "" {
		t.Error("GetHooksDir(gemini) returned empty string")
	}
}

func TestStatus(t *testing.T) {
	statuses := Status()
	if len(statuses) != 5 {
		t.Errorf("Expected 5 tool statuses, got %d", len(statuses))
	}

	tools := make(map[Tool]bool)
	for _, s := range statuses {
		tools[s.Tool] = true
	}

	if !tools[ToolCursor] {
		t.Error("Missing Cursor in status")
	}
	if !tools[ToolClaudeCode] {
		t.Error("Missing Claude Code in status")
	}
	if !tools[ToolGeminiCLI] {
		t.Error("Missing Gemini CLI in status")
	}
}

func TestInstallCursor(t *testing.T) {
	tmpDir := t.TempDir()
	handlerPath := filepath.Join(tmpDir, "handler")

	// Create fake handler
	if err := os.WriteFile(handlerPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create test handler: %v", err)
	}

	// We can't easily test Install without mocking the home directory
	// Just verify the JSON generation works
	json, err := GenerateCursorHooksJSON(handlerPath)
	if err != nil {
		t.Errorf("GenerateCursorHooksJSON failed: %v", err)
	}
	if json == "" {
		t.Error("GenerateCursorHooksJSON returned empty string")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
