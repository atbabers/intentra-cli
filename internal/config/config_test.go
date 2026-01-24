package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	dir := GetConfigDir()
	if dir == "" {
		t.Error("GetConfigDir returned empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("GetConfigDir returned relative path: %s", dir)
	}
}

func TestLoadConfig(t *testing.T) {
	// Use temp dir for test
	tmpDir := t.TempDir()
	os.Setenv("INTENTRA_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("INTENTRA_CONFIG_DIR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Local.AnthropicAPIKey != "" {
		t.Error("Expected empty API key for fresh config")
	}
	if cfg.Server.Enabled {
		t.Error("Expected server sync to be disabled by default")
	}
}
