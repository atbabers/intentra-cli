// Package config manages Intentra configuration loading, validation, and
// defaults. It supports file-based configuration, environment variables,
// and multiple authentication modes for server sync.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the intentra configuration.
type Config struct {
	// Debug mode enables HTTP request logging and local scan saving
	Debug bool `mapstructure:"debug"`

	// Server sync configuration (optional - for team deployments)
	Server ServerConfig `mapstructure:"server"`

	// Local settings
	Local LocalConfig `mapstructure:"local"`

	// Buffer configuration
	Buffer BufferConfig `mapstructure:"buffer"`

	// Logging configuration
	Log LogConfig `mapstructure:"logging"`
}

// ServerConfig contains API server settings for team deployments.
type ServerConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Endpoint string        `mapstructure:"endpoint"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Auth     AuthConfig    `mapstructure:"auth"`
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	Mode   string       `mapstructure:"mode"` // hmac, api_key, mtls
	HMAC   HMACConfig   `mapstructure:"hmac"`
	APIKey APIKeyConfig `mapstructure:"api_key"`
	MTLS   MTLSConfig   `mapstructure:"mtls"`
}

// HMACConfig contains HMAC authentication settings.
type HMACConfig struct {
	KeyID    string `mapstructure:"key_id"`    // API key identifier for server auth
	DeviceID string `mapstructure:"device_id"` // Device ID (auto-generated if empty)
	Secret   string `mapstructure:"secret"`    // Shared secret for HMAC signature
}

// APIKeyConfig contains API key authentication settings.
type APIKeyConfig struct {
	KeyID  string `mapstructure:"key_id"`
	Secret string `mapstructure:"secret"`
}

// MTLSConfig contains mTLS certificate settings for JAMF/MDM deployments.
type MTLSConfig struct {
	CertFile   string `mapstructure:"cert_file"`   // Client certificate path
	KeyFile    string `mapstructure:"key_file"`    // Client private key path
	CAFile     string `mapstructure:"ca_file"`     // CA certificate for server verification
	SkipVerify bool   `mapstructure:"skip_verify"` // Skip server cert verification (dev only)
}

// LocalConfig contains local-only settings.
type LocalConfig struct {
	AnthropicAPIKey  string        `mapstructure:"anthropic_api_key"`
	Model            string        `mapstructure:"model"`
	ScanTimeout      int           `mapstructure:"scan_timeout"`
	MinEventsPerScan int           `mapstructure:"min_events_per_scan"`
	CharsPerToken    int           `mapstructure:"chars_per_token"`
	Archive          ArchiveConfig `mapstructure:"archive"`
}

// ArchiveConfig contains local scan archive settings for benchmarking.
type ArchiveConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Path          string `mapstructure:"path"`
	Redacted      bool   `mapstructure:"redacted"`
	IncludeEvents bool   `mapstructure:"include_events"`
}

// BufferConfig contains local buffer settings.
type BufferConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	Path           string        `mapstructure:"path"`
	MaxSizeMB      int           `mapstructure:"max_size_mb"`
	MaxAgeHours    int           `mapstructure:"max_age_hours"`
	FlushInterval  time.Duration `mapstructure:"flush_interval"`
	FlushThreshold int           `mapstructure:"flush_threshold"`
}

// LogConfig contains logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	dataDir := GetDataDir()
	return &Config{
		Debug: false,
		Server: ServerConfig{
			Enabled:  false,
			Endpoint: "",
			Timeout:  30 * time.Second,
			Auth: AuthConfig{
				Mode: "hmac",
			},
		},
		Local: LocalConfig{
			Model:            "claude-3-5-haiku-latest",
			ScanTimeout:      30,
			MinEventsPerScan: 2,
			CharsPerToken:    4,
			Archive: ArchiveConfig{
				Enabled:       false,
				Path:          filepath.Join(dataDir, "archive"),
				Redacted:      true,
				IncludeEvents: false,
			},
		},
		Buffer: BufferConfig{
			Enabled:        false,
			Path:           filepath.Join(dataDir, "buffer.db"),
			MaxSizeMB:      50,
			MaxAgeHours:    24,
			FlushInterval:  30 * time.Second,
			FlushThreshold: 10,
		},
		Log: LogConfig{
			Level:  "warn",
			Format: "text",
		},
	}
}

// Load reads configuration from file and environment.
func Load() (*Config, error) {
	if err := EnsureDirectories(); err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	v := viper.New()

	// Config file locations
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(GetConfigDir())
	v.AddConfigPath("/etc/intentra")
	v.AddConfigPath(".")

	// Set defaults
	v.SetDefault("local.model", cfg.Local.Model)
	v.SetDefault("local.scan_timeout", cfg.Local.ScanTimeout)
	v.SetDefault("local.min_events_per_scan", cfg.Local.MinEventsPerScan)
	v.SetDefault("local.chars_per_token", cfg.Local.CharsPerToken)
	v.SetDefault("local.archive.enabled", cfg.Local.Archive.Enabled)
	v.SetDefault("local.archive.path", cfg.Local.Archive.Path)
	v.SetDefault("local.archive.redacted", cfg.Local.Archive.Redacted)
	v.SetDefault("local.archive.include_events", cfg.Local.Archive.IncludeEvents)
	v.SetDefault("buffer.enabled", cfg.Buffer.Enabled)
	v.SetDefault("buffer.path", cfg.Buffer.Path)
	v.SetDefault("buffer.max_size_mb", cfg.Buffer.MaxSizeMB)
	v.SetDefault("buffer.max_age_hours", cfg.Buffer.MaxAgeHours)
	v.SetDefault("buffer.flush_interval", cfg.Buffer.FlushInterval)
	v.SetDefault("buffer.flush_threshold", cfg.Buffer.FlushThreshold)

	// Environment variable overrides
	v.SetEnvPrefix("INTENTRA")
	v.AutomaticEnv()

	// Try to read config file (ignore if not exists)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	// Unmarshal
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	// Expand environment variables in sensitive fields
	cfg.Server.Auth.HMAC.KeyID = os.ExpandEnv(cfg.Server.Auth.HMAC.KeyID)
	cfg.Server.Auth.HMAC.DeviceID = os.ExpandEnv(cfg.Server.Auth.HMAC.DeviceID)
	cfg.Server.Auth.HMAC.Secret = os.ExpandEnv(cfg.Server.Auth.HMAC.Secret)
	cfg.Server.Auth.APIKey.KeyID = os.ExpandEnv(cfg.Server.Auth.APIKey.KeyID)
	cfg.Server.Auth.APIKey.Secret = os.ExpandEnv(cfg.Server.Auth.APIKey.Secret)
	cfg.Local.AnthropicAPIKey = os.ExpandEnv(cfg.Local.AnthropicAPIKey)

	// Environment variable overrides for HMAC auth
	if keyID := os.Getenv("INTENTRA_API_KEY_ID"); keyID != "" {
		cfg.Server.Auth.HMAC.KeyID = keyID
	}
	if secret := os.Getenv("INTENTRA_API_SECRET"); secret != "" {
		cfg.Server.Auth.HMAC.Secret = secret
	}

	// ANTHROPIC_API_KEY env takes precedence for local analysis
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.Local.AnthropicAPIKey = key
	}

	// INTENTRA_SERVER_ENDPOINT enables server sync
	if endpoint := os.Getenv("INTENTRA_SERVER_ENDPOINT"); endpoint != "" {
		cfg.Server.Enabled = true
		cfg.Server.Endpoint = endpoint
	}

	return cfg, nil
}

// LoadWithFile reads configuration from a specific file.
func LoadWithFile(cfgFile string) (*Config, error) {
	if err := EnsureDirectories(); err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	v := viper.New()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		return Load()
	}

	v.SetEnvPrefix("INTENTRA")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	// Expand environment variables
	cfg.Server.Auth.HMAC.KeyID = os.ExpandEnv(cfg.Server.Auth.HMAC.KeyID)
	cfg.Server.Auth.HMAC.DeviceID = os.ExpandEnv(cfg.Server.Auth.HMAC.DeviceID)
	cfg.Server.Auth.HMAC.Secret = os.ExpandEnv(cfg.Server.Auth.HMAC.Secret)
	cfg.Local.AnthropicAPIKey = os.ExpandEnv(cfg.Local.AnthropicAPIKey)

	// Environment variable overrides
	if keyID := os.Getenv("INTENTRA_API_KEY_ID"); keyID != "" {
		cfg.Server.Auth.HMAC.KeyID = keyID
	}
	if secret := os.Getenv("INTENTRA_API_SECRET"); secret != "" {
		cfg.Server.Auth.HMAC.Secret = secret
	}

	return cfg, nil
}

// Validate checks if the configuration is valid for server sync.
func (c *Config) Validate() error {
	if !c.Server.Enabled {
		return nil // Local-only mode, no validation needed
	}

	if c.Server.Endpoint == "" {
		return fmt.Errorf("server.endpoint is required when server sync is enabled")
	}

	switch c.Server.Auth.Mode {
	case "hmac":
		// Device ID is auto-generated if not provided, key_id and secret are required
		if c.Server.Auth.HMAC.KeyID == "" {
			return fmt.Errorf("hmac auth requires key_id")
		}
		if c.Server.Auth.HMAC.Secret == "" {
			return fmt.Errorf("hmac auth requires secret")
		}
	case "api_key":
		if c.Server.Auth.APIKey.KeyID == "" || c.Server.Auth.APIKey.Secret == "" {
			return fmt.Errorf("api_key auth requires key_id and secret")
		}
	case "mtls":
		if c.Server.Auth.MTLS.CertFile == "" || c.Server.Auth.MTLS.KeyFile == "" {
			return fmt.Errorf("mtls auth requires cert_file and key_file")
		}
	default:
		return fmt.Errorf("unknown auth mode: %s (supported: hmac, api_key, mtls)", c.Server.Auth.Mode)
	}

	return nil
}

// Print outputs the current configuration (redacting secrets).
func (c *Config) Print() {
	fmt.Println("=== Intentra Configuration ===")
	fmt.Println()

	fmt.Printf("Debug: %v\n", c.Debug)
	fmt.Println()

	fmt.Println("Server Sync:")
	fmt.Printf("  Enabled: %v\n", c.Server.Enabled)
	if c.Server.Enabled {
		fmt.Printf("  Endpoint: %s\n", c.Server.Endpoint)
		fmt.Printf("  Timeout: %s\n", c.Server.Timeout)
		fmt.Printf("  Auth Mode: %s\n", c.Server.Auth.Mode)
		if c.Server.Auth.Mode == "hmac" {
			fmt.Printf("  Device ID: %s\n", c.Server.Auth.HMAC.DeviceID)
			if c.Server.Auth.HMAC.Secret != "" {
				fmt.Printf("  Secret: [REDACTED]\n")
			}
		}
	}
	fmt.Println()

	fmt.Println("Local:")
	fmt.Printf("  Model: %s\n", c.Local.Model)
	if c.Local.AnthropicAPIKey != "" {
		fmt.Printf("  Anthropic API Key: [REDACTED]\n")
	}
	fmt.Println()

	fmt.Println("Archive:")
	fmt.Printf("  Enabled: %v\n", c.Local.Archive.Enabled)
	fmt.Printf("  Path: %s\n", c.Local.Archive.Path)
	fmt.Printf("  Redacted: %v\n", c.Local.Archive.Redacted)
	fmt.Printf("  Include Events: %v\n", c.Local.Archive.IncludeEvents)
	fmt.Println()

	fmt.Println("Buffer:")
	fmt.Printf("  Enabled: %v\n", c.Buffer.Enabled)
	fmt.Printf("  Path: %s\n", c.Buffer.Path)
	fmt.Printf("  Max Size: %d MB\n", c.Buffer.MaxSizeMB)
	fmt.Printf("  Flush Interval: %s\n", c.Buffer.FlushInterval)
}

// PrintSample outputs a sample configuration file.
func PrintSample() {
	sample := `# Intentra Configuration
# ~/.intentra/config.yaml

# Debug mode (logs HTTP requests, saves scans locally)
debug: false

# Server sync (optional - for team deployments)
server:
  enabled: false
  endpoint: "https://api.intentra.example.com/v1"
  timeout: 30s
  auth:
    mode: "hmac"  # hmac, api_key, or mtls

    # HMAC authentication (default - device ID auto-generated from hardware)
    hmac:
      key_id: "${INTENTRA_API_KEY_ID}"  # API key identifier from server
      device_id: ""                      # Leave empty for auto-generated HMAC device ID
      secret: "${INTENTRA_API_SECRET}"   # Shared secret for HMAC signature

    # API key authentication
    # api_key:
    #   key_id: "${INTENTRA_API_KEY_ID}"
    #   secret: "${INTENTRA_API_SECRET}"

    # mTLS authentication (for JAMF/MDM deployments)
    # mtls:
    #   cert_file: "/etc/intentra/client.crt"   # Client certificate
    #   key_file: "/etc/intentra/client.key"    # Client private key
    #   ca_file: "/etc/intentra/ca.crt"         # CA certificate (optional)
    #   skip_verify: false                       # Dev only

# Local settings
local:
  anthropic_api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-3-5-haiku-latest"
  scan_timeout: 30
  min_events_per_scan: 2
  chars_per_token: 4

  # Local scan archive (for benchmarking)
  archive:
    enabled: false              # Enable local archiving
    path: ~/.intentra/archive   # Directory for archived scans
    redacted: true              # Strip prompt/response/thought content (default)
    include_events: false       # Include redacted event list in archive

# Buffer for offline resilience
buffer:
  enabled: true
  path: ~/.intentra/buffer.db
  max_size_mb: 50
  max_age_hours: 24
  flush_interval: 30s
  flush_threshold: 10

# Logging
logging:
  level: warn    # debug, info, warn, error
  format: text   # text, json
`
	fmt.Print(sample)
}

// GetConfigPath returns the path to the config file.
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.yaml")
}

// ConfigExists returns true if the config file exists.
func ConfigExists() bool {
	_, err := os.Stat(GetConfigPath())
	return err == nil
}

// SaveConfig writes the configuration to the config file.
// It preserves existing values and only updates specified fields.
func SaveConfig(cfg *Config) error {
	configPath := GetConfigPath()

	if err := os.MkdirAll(GetConfigDir(), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigType("yaml")

	if _, err := os.Stat(configPath); err == nil {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			v = viper.New()
			v.SetConfigType("yaml")
		}
	}

	v.Set("debug", cfg.Debug)
	v.Set("server.enabled", cfg.Server.Enabled)
	v.Set("server.endpoint", cfg.Server.Endpoint)
	v.Set("server.timeout", cfg.Server.Timeout.String())
	v.Set("server.auth.mode", cfg.Server.Auth.Mode)
	v.Set("local.model", cfg.Local.Model)
	v.Set("local.scan_timeout", cfg.Local.ScanTimeout)
	v.Set("local.min_events_per_scan", cfg.Local.MinEventsPerScan)
	v.Set("local.chars_per_token", cfg.Local.CharsPerToken)
	v.Set("local.archive.enabled", cfg.Local.Archive.Enabled)
	v.Set("local.archive.path", cfg.Local.Archive.Path)
	v.Set("local.archive.redacted", cfg.Local.Archive.Redacted)
	v.Set("local.archive.include_events", cfg.Local.Archive.IncludeEvents)
	v.Set("logging.level", cfg.Log.Level)
	v.Set("logging.format", cfg.Log.Format)

	return v.WriteConfigAs(configPath)
}

// --- Legacy compatibility ---

// APIKey returns the Anthropic API key (legacy compatibility).
func (c *Config) APIKey() string {
	return c.Local.AnthropicAPIKey
}

// Model returns the model name (legacy compatibility).
func (c *Config) Model() string {
	return c.Local.Model
}
