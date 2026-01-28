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
