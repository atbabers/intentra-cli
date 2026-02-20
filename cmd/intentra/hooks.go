package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/internal/hooks"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Check hook installation status",
	}

	cmd.AddCommand(newHooksStatusCmd())

	return cmd
}

func saveAPIConfig(server, keyID, secret string) error {
	configDir := config.GetConfigDir()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	configData := map[string]any{
		"server": map[string]any{
			"enabled":  true,
			"endpoint": server,
			"timeout":  "30s",
			"auth": map[string]any{
				"mode": "api_key",
				"api_key": map[string]any{
					"key_id": keyID,
					"secret": secret,
				},
			},
		},
	}

	configContent, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	return os.WriteFile(configPath, configContent, 0600)
}

func newHooksStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "status",
		Short:         "Check hooks installation status",
		SilenceUsage:  true,
		SilenceErrors: true,
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
