package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/atbabers/intentra-cli/internal/api"
	"github.com/atbabers/intentra-cli/internal/scanner"
	"github.com/atbabers/intentra-cli/pkg/models"
	"github.com/spf13/cobra"
)

// newScanCmd returns a cobra.Command for managing scans.
func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage scans",
	}

	cmd.AddCommand(newScanListCmd())
	cmd.AddCommand(newScanShowCmd())
	cmd.AddCommand(newScanTodayCmd())
	cmd.AddCommand(newScanAggregateCmd())

	return cmd
}

// newScanListCmd returns a cobra.Command for listing all scans.
func newScanListCmd() *cobra.Command {
	var jsonOutput bool
	var days int
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all scans",
		Long: `List scans from the server (if logged in and server enabled) or local storage.

When server mode is enabled, scans are fetched from the API.
When server mode is disabled (local-only), scans are read from local files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			var scans []models.Scan
			var source string

			if cfg.Server.Enabled {
				client, err := api.NewClient(cfg)
				if err != nil {
					return fmt.Errorf("failed to create API client: %w", err)
				}

				resp, err := client.GetScans(days, limit)
				if err != nil {
					return fmt.Errorf("failed to fetch scans from server: %w", err)
				}
				scans = resp.Scans
				source = "server"

				if !jsonOutput && resp.Summary.TotalScans > 0 {
					fmt.Printf("Summary: %d scans, $%.2f total cost, %d with violations\n\n",
						resp.Summary.TotalScans, resp.Summary.TotalCost, resp.Summary.ScansWithViolations)
				}
			} else {
				localScans, err := scanner.LoadScans()
				if err != nil {
					return err
				}
				scans = localScans
				source = "local"
			}

			if len(scans) == 0 {
				if source == "server" {
					fmt.Println("No scans found on server.")
				} else {
					fmt.Println("No scans found. Run 'intentra scan aggregate' to process events.")
				}
				return nil
			}

			if jsonOutput {
				data, err := json.MarshalIndent(scans, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal scans: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tEVENTS\tTOKENS\tCOST\tTIME")
			for _, s := range scans {
				id := s.ID
				if len(id) > 8 {
					id = id[:8]
				}
				status := string(s.Status)
				if status == "" {
					status = "-"
				}
				startTime := s.StartTime
				if startTime.IsZero() {
					startTime = time.Now()
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\t$%.4f\t%s\n",
					id,
					status,
					len(s.Events),
					s.TotalTokens,
					s.EstimatedCost,
					startTime.Format("2006-01-02 15:04"),
				)
			}
			if err := w.Flush(); err != nil {
				return fmt.Errorf("failed to flush output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().IntVar(&days, "days", 30, "Number of days to look back (server mode only)")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of scans to return (server mode only)")

	return cmd
}

// newScanShowCmd returns a cobra.Command for displaying scan details.
func newScanShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show scan details",
		Long: `Show detailed information about a specific scan.

When server mode is enabled, the scan is fetched from the API.
When server mode is disabled (local-only), the scan is read from local files.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scanID := args[0]

			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if cfg.Server.Enabled {
				client, err := api.NewClient(cfg)
				if err != nil {
					return fmt.Errorf("failed to create API client: %w", err)
				}

				resp, err := client.GetScan(scanID)
				if err != nil {
					return err
				}

				output := map[string]any{
					"scan": resp.Scan,
				}
				if len(resp.ViolationDetails) > 0 {
					output["violation_details"] = resp.ViolationDetails
				}

				data, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal scan: %w", err)
				}
				fmt.Println(string(data))
			} else {
				scan, err := scanner.LoadScan(scanID)
				if err != nil {
					return fmt.Errorf("scan not found: %s", scanID)
				}

				data, err := json.MarshalIndent(scan, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal scan: %w", err)
				}
				fmt.Println(string(data))
			}

			return nil
		},
	}
}

// newScanTodayCmd returns a cobra.Command for showing today's scans.
func newScanTodayCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "today",
		Short: "Show today's scans",
		Long:  `List scans from today only. Uses server or local storage based on configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			var scans []models.Scan
			today := time.Now().Truncate(24 * time.Hour)

			if cfg.Server.Enabled {
				client, err := api.NewClient(cfg)
				if err != nil {
					return fmt.Errorf("failed to create API client: %w", err)
				}

				resp, err := client.GetScans(1, 200)
				if err != nil {
					return fmt.Errorf("failed to fetch scans from server: %w", err)
				}
				scans = resp.Scans
			} else {
				localScans, err := scanner.LoadScans()
				if err != nil {
					return err
				}
				for _, s := range localScans {
					if !s.StartTime.Before(today) {
						scans = append(scans, s)
					}
				}
			}

			if len(scans) == 0 {
				fmt.Println("No scans found for today.")
				return nil
			}

			if jsonOutput {
				data, err := json.MarshalIndent(scans, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal scans: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			var totalCost float64
			var totalTokens int
			for _, s := range scans {
				totalCost += s.EstimatedCost
				totalTokens += s.TotalTokens
			}

			fmt.Printf("Today's scans: %d scans, %d tokens, $%.4f estimated cost\n\n",
				len(scans), totalTokens, totalCost)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTOKENS\tCOST\tTIME")
			for _, s := range scans {
				id := s.ID
				if len(id) > 8 {
					id = id[:8]
				}
				fmt.Fprintf(w, "%s\t%d\t$%.4f\t%s\n",
					id,
					s.TotalTokens,
					s.EstimatedCost,
					s.StartTime.Format("15:04"),
				)
			}
			if err := w.Flush(); err != nil {
				return fmt.Errorf("failed to flush output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// newScanAggregateCmd returns a cobra.Command for aggregating events into scans.
func newScanAggregateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "aggregate",
		Short: "Process events into scans",
		RunE: func(cmd *cobra.Command, args []string) error {
			events, err := scanner.LoadEvents()
			if err != nil {
				return fmt.Errorf("failed to load events: %w", err)
			}

			if len(events) == 0 {
				fmt.Println("No events found. Use Cursor with hooks installed to generate events.")
				return nil
			}

			scans := scanner.AggregateEvents(events)
			fmt.Printf("Found %d events, aggregated into %d scans\n", len(events), len(scans))

			for _, scan := range scans {
				if err := scanner.SaveScan(&scan); err != nil {
					fmt.Printf("Warning: failed to save scan %s: %v\n", scan.ID, err)
					continue
				}
				id := scan.ID
				if len(id) > 8 {
					id = id[:8]
				}
				fmt.Printf("Saved scan %s (%d events, %d tokens)\n", id, len(scan.Events), scan.TotalTokens)
			}

			return nil
		},
	}
}
