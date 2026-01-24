package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/atbabers/intentra-cli/internal/scanner"
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
	cmd.AddCommand(newScanAggregateCmd())

	return cmd
}

// newScanListCmd returns a cobra.Command for listing all scans.
func newScanListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all scans",
		RunE: func(cmd *cobra.Command, args []string) error {
			scans, err := scanner.LoadScans()
			if err != nil {
				return err
			}

			if len(scans) == 0 {
				fmt.Println("No scans found. Run 'intentra scan aggregate' to process events.")
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
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\t$%.4f\t%s\n",
					id,
					s.Status,
					len(s.Events),
					s.TotalTokens,
					s.EstimatedCost,
					s.StartTime.Format("2006-01-02 15:04"),
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

// newScanShowCmd returns a cobra.Command for displaying scan details.
func newScanShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show scan details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scan, err := scanner.LoadScan(args[0])
			if err != nil {
				return fmt.Errorf("scan not found: %s", args[0])
			}

			data, err := json.MarshalIndent(scan, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal scan: %w", err)
			}
			fmt.Println(string(data))
			return nil
		},
	}
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
