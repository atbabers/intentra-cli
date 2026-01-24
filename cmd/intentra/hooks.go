package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/internal/hooks"
	"github.com/spf13/cobra"
)

// newHooksCmd returns a cobra.Command for managing AI tool hooks.
func newHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage AI tool hooks (Cursor, Claude Code)",
	}

	cmd.AddCommand(newHooksInstallCmd())
	cmd.AddCommand(newHooksUninstallCmd())
	cmd.AddCommand(newHooksStatusCmd())

	return cmd
}

// newHooksInstallCmd returns a cobra.Command for installing hooks.
func newHooksInstallCmd() *cobra.Command {
	var tool string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install hooks for AI tools",
		Long: `Install hooks for AI coding tools. Supported tools:
  - cursor: Cursor editor
  - claude: Claude Code CLI
  - all: All supported tools (default)

If --api-server, --api-key-id, and --api-secret are provided,
configuration is saved to ~/.config/intentra/config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Save API config if provided via flags
			if apiServer != "" && apiKeyID != "" && apiSecret != "" {
				if err := saveAPIConfig(apiServer, apiKeyID, apiSecret); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Println("✓ Saved API configuration")
			}

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}

			if tool == "all" || tool == "" {
				results := hooks.InstallAll(execPath)
				var errors []string
				for t, err := range results {
					if err != nil {
						errors = append(errors, fmt.Sprintf("%s: %v", t, err))
					} else {
						fmt.Printf("✓ Installed hooks for %s\n", t)
					}
				}
				if len(errors) > 0 {
					fmt.Println("\nSome installations failed:")
					for _, e := range errors {
						fmt.Printf("  ✗ %s\n", e)
					}
				}
				fmt.Println("\nPlease restart your AI tools for hooks to take effect.")
				return nil
			}

			t := hooks.Tool(tool)
			if err := hooks.Install(t, execPath); err != nil {
				return err
			}

			fmt.Printf("✓ Hooks installed for %s\n", tool)
			fmt.Printf("Please restart %s for hooks to take effect.\n", tool)
			return nil
		},
	}

	cmd.Flags().StringVarP(&tool, "tool", "t", "all", "Tool to install hooks for (cursor, claude, all)")

	return cmd
}

// saveAPIConfig writes the server configuration to the config file.
func saveAPIConfig(server, keyID, secret string) error {
	configDir := config.GetConfigDir()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	configContent := fmt.Sprintf(`server:
  enabled: true
  endpoint: "%s"
  timeout: 30s
  auth:
    mode: "hmac"
    hmac:
      key_id: "%s"
      secret: "%s"
      device_id: ""
`, server, keyID, secret)

	configPath := filepath.Join(configDir, "config.yaml")
	return os.WriteFile(configPath, []byte(configContent), 0600)
}

// newHooksUninstallCmd returns a cobra.Command for removing hooks.
func newHooksUninstallCmd() *cobra.Command {
	var tool string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove hooks from AI tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tool == "all" || tool == "" {
				// Uninstall from all tools
				results := hooks.UninstallAll()
				var errors []string
				for t, err := range results {
					if err != nil {
						errors = append(errors, fmt.Sprintf("%s: %v", t, err))
					} else {
						fmt.Printf("✓ Uninstalled hooks from %s\n", t)
					}
				}
				if len(errors) > 0 {
					fmt.Println("\nSome uninstallations had issues:")
					for _, e := range errors {
						fmt.Printf("  ✗ %s\n", e)
					}
				}
				fmt.Println("\nPlease restart your AI tools for changes to take effect.")
				return nil
			}

			// Uninstall from specific tool
			t := hooks.Tool(tool)
			if err := hooks.Uninstall(t); err != nil {
				return err
			}

			fmt.Printf("✓ Hooks uninstalled from %s\n", tool)
			fmt.Printf("Please restart %s for changes to take effect.\n", tool)
			return nil
		},
	}

	cmd.Flags().StringVarP(&tool, "tool", "t", "all", "Tool to uninstall hooks from (cursor, claude, all)")

	return cmd
}

// newHooksStatusCmd returns a cobra.Command for checking hook installation status.
func newHooksStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check hooks installation status",
		RunE: func(cmd *cobra.Command, args []string) error {
			statuses := hooks.Status()

			fmt.Println("Hook Installation Status:")
			fmt.Println(strings.Repeat("-", 50))

			for _, s := range statuses {
				status := "✗ Not installed"
				if s.Installed {
					status = "✓ Installed"
				}
				fmt.Printf("%-12s %s\n", s.Tool+":", status)
				if s.Path != "" {
					fmt.Printf("             Path: %s\n", s.Path)
				}
				if s.Error != nil {
					fmt.Printf("             Error: %v\n", s.Error)
				}
			}

			return nil
		},
	}
}
