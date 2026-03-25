package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/intentrahq/intentra-cli/internal/api"
	"github.com/intentrahq/intentra-cli/internal/auth"
	"github.com/intentrahq/intentra-cli/internal/debug"
	"github.com/intentrahq/intentra-cli/internal/hooks"
	"github.com/intentrahq/intentra-cli/internal/queue"
	"github.com/intentrahq/intentra-cli/pkg/models"
	"github.com/spf13/cobra"
)

// newSendCmd returns the hidden __send subcommand used by detached child processes.
func newSendCmd() *cobra.Command {
	var payloadFile string

	cmd := &cobra.Command{
		Use:           "__send",
		Short:         "Send a deferred payload (internal use)",
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer os.Remove(payloadFile)

			data, err := os.ReadFile(payloadFile)
			if err != nil {
				return fmt.Errorf("failed to read payload file: %w", err)
			}

			var p models.SendPayload
			if err := json.Unmarshal(data, &p); err != nil {
				return fmt.Errorf("failed to parse payload: %w", err)
			}

			switch p.Action {
			case "send_scan":
				return deferredSendScan(p)
			case "patch_session_end":
				return deferredPatchSessionEnd(p.ScanID, p.Reason, p.DurationMs)
			default:
				return fmt.Errorf("unknown action: %s", p.Action)
			}
		},
	}

	cmd.Flags().StringVar(&payloadFile, "payload", "", "Path to JSON payload file")
	_ = cmd.MarkFlagRequired("payload")

	return cmd
}

// deferredSendScan sends a scan to the server, falling back to the offline queue.
func deferredSendScan(p models.SendPayload) error {
	scan := p.Scan
	if scan == nil {
		return fmt.Errorf("send_scan action requires a scan")
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	synced := false

	creds, credsErr := auth.GetValidCredentials()
	if credsErr != nil {
		debug.Warn("credential check failed: %v", credsErr)
	}

	if creds != nil {
		if err := api.SendScanWithJWT(scan, creds.AccessToken); err != nil {
			debug.Warn("JWT send failed: %v", err)
		} else {
			synced = true
		}
	}

	if !synced && cfg.Server.Enabled {
		client, err := api.NewClient(cfg)
		if err != nil {
			debug.Warn("failed to create API client: %v", err)
		} else {
			if err := client.SendScan(scan); err != nil {
				debug.Warn("client send failed: %v", err)
			} else {
				synced = true
			}
		}
	}

	if !synced {
		if err := queue.Enqueue(scan); err != nil {
			return fmt.Errorf("failed to enqueue scan: %w", err)
		}
		return nil
	}

	if creds != nil {
		queue.FlushWithJWT(creds.AccessToken)
	}

	if synced && scan.ID != "" && p.SessionKey != "" {
		hooks.SaveLastScanID(p.SessionKey, scan.ID)
	}

	return nil
}

// deferredPatchSessionEnd patches session-end metadata on an already-sent scan.
func deferredPatchSessionEnd(scanID, reason string, durationMs int64) error {
	creds, err := auth.GetValidCredentials()
	if err != nil {
		debug.Warn("credential check failed: %v", err)
	}

	if creds == nil {
		debug.Warn("skipping patch_session_end for %s: not authenticated", scanID)
		return nil
	}

	return api.PatchSessionEnd(scanID, creds.AccessToken, reason, durationMs)
}
