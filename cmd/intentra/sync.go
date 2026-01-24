package main

import (
	"fmt"

	"github.com/atbabers/intentra-cli/internal/api"
	"github.com/atbabers/intentra-cli/internal/scanner"
	"github.com/atbabers/intentra-cli/pkg/models"
	"github.com/spf13/cobra"
)

// runSyncNow syncs all pending scans to the configured server.
func runSyncNow(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if !cfg.Server.Enabled {
		return fmt.Errorf("server sync is not enabled. Set server.enabled=true in config or set INTENTRA_SERVER_ENDPOINT")
	}

	// Load pending scans
	scans, err := scanner.LoadScans()
	if err != nil {
		return fmt.Errorf("failed to load scans: %w", err)
	}

	if len(scans) == 0 {
		fmt.Println("No scans to sync. Run 'intentra scan aggregate' first to process events.")
		return nil
	}

	// Filter for pending/analyzed scans only
	var pending []*models.Scan
	for i := range scans {
		if scans[i].Status == models.ScanStatusPending || scans[i].Status == models.ScanStatusAnalyzing {
			pending = append(pending, &scans[i])
		}
	}

	if len(pending) == 0 {
		fmt.Println("No pending scans to sync. All scans have been reviewed.")
		return nil
	}

	fmt.Printf("Syncing %d scans to %s...\n", len(pending), cfg.Server.Endpoint)

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Send scans
	if err := client.SendScans(pending); err != nil {
		return fmt.Errorf("failed to sync scans: %w", err)
	}

	// Mark scans as reviewed
	for _, scan := range pending {
		scan.Status = models.ScanStatusReviewed
		if err := scanner.SaveScan(scan); err != nil {
			fmt.Printf("Warning: failed to update scan %s status: %v\n", scan.ID, err)
		}
	}

	fmt.Printf("✓ Successfully synced %d scans\n", len(pending))
	return nil
}
