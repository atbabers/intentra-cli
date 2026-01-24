// Package main implements the intentra CLI for monitoring AI coding assistants.
//
// Intentra provides commands for:
//   - Installing hooks into AI tools (Cursor, Claude Code)
//   - Managing and aggregating scan data
//   - Syncing scans to a central server
package main

import (
	"fmt"
	"os"

	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/internal/hooks"
	"github.com/spf13/cobra"
)

var (
	// version is set at build time via ldflags.
	version = "dev"

	// cfgFile holds the path to the configuration file.
	cfgFile string

	// apiServer, apiKeyID, and apiSecret are CLI flag overrides for server config.
	apiServer string
	apiKeyID  string
	apiSecret string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "intentra",
		Short: "AI usage monitoring and violation detection",
		Long: `Intentra monitors AI coding assistants (Cursor, Claude Code) for policy violations,
tracks usage metrics, and optionally syncs data to a central server.`,
		Version: version,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ~/.config/intentra/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiServer, "api-server", "", "API server endpoint (e.g., https://app.example.com/api/v1)")
	rootCmd.PersistentFlags().StringVar(&apiKeyID, "api-key-id", "", "API key ID for authentication")
	rootCmd.PersistentFlags().StringVar(&apiSecret, "api-secret", "", "API secret for authentication")

	// Add commands
	rootCmd.AddCommand(newHooksCmd())
	rootCmd.AddCommand(newScanCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newSyncCmd())

	// Hidden hook command (called by AI tools via installed hooks).
	var hookTool string
	hookCmd := &cobra.Command{
		Use:    "hook",
		Short:  "Process a hook event (internal use)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hooks.RunHookHandlerWithTool(hookTool)
		},
	}
	hookCmd.Flags().StringVar(&hookTool, "tool", "", "AI tool (cursor, claude)")
	rootCmd.AddCommand(hookCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newConfigCmd returns a cobra.Command for managing configuration.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	// config show
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			cfg.Print()
			return nil
		},
	}

	// config init
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Generate sample configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			config.PrintSample()
			return nil
		},
	}

	// config validate
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration for server sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			fmt.Println("✓ Configuration is valid")
			if cfg.Server.Enabled {
				fmt.Printf("  Server: %s\n", cfg.Server.Endpoint)
				fmt.Printf("  Auth: %s\n", cfg.Server.Auth.Mode)
			} else {
				fmt.Println("  Server sync: disabled (local-only mode)")
			}
			return nil
		},
	}

	cmd.AddCommand(showCmd, initCmd, validateCmd)
	return cmd
}

// newSyncCmd returns a cobra.Command for syncing scans to a server.
func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync scans to server",
		Long: `Sync locally buffered scans to the configured server.
Requires server sync to be enabled in config.`,
	}

	// sync now
	nowCmd := &cobra.Command{
		Use:   "now",
		Short: "Force sync all pending scans",
		RunE:  runSyncNow,
	}

	// sync status
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show sync status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			fmt.Println("Sync Status:")
			if cfg.Server.Enabled {
				fmt.Printf("  Server: %s\n", cfg.Server.Endpoint)
				fmt.Printf("  Buffer: %s\n", cfg.Buffer.Path)
				// TODO: Show pending count from buffer
			} else {
				fmt.Println("  Server sync: disabled")
				fmt.Println("  Running in local-only mode")
			}
			return nil
		},
	}

	cmd.AddCommand(nowCmd, statusCmd)
	return cmd
}

// loadConfig returns the configuration, applying file and CLI flag overrides.
func loadConfig() (*config.Config, error) {
	var cfg *config.Config
	var err error

	if cfgFile != "" {
		cfg, err = config.LoadWithFile(cfgFile)
	} else {
		cfg, err = config.Load()
	}
	if err != nil {
		return nil, err
	}

	// CLI flags override config file and environment variables
	if apiServer != "" {
		cfg.Server.Enabled = true
		cfg.Server.Endpoint = apiServer
	}
	if apiKeyID != "" {
		cfg.Server.Auth.HMAC.KeyID = apiKeyID
	}
	if apiSecret != "" {
		cfg.Server.Auth.HMAC.Secret = apiSecret
	}

	return cfg, nil
}
