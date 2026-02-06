// Package scanner provides event aggregation and scan management for Intentra.
// It handles grouping events into scans, calculating metrics like token usage
// and cost estimates, and persisting scan data locally.
package scanner

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"

	"github.com/atbabers/intentra-cli/pkg/models"
)

// modelPricing contains pricing per 1K tokens for various models.
// Keys are prefixes that match model strings (e.g., "claude-3-5-sonnet" matches "claude-3-5-sonnet-latest").
var modelPricing = map[string]float64{
	"claude-sonnet-4":   0.003,
	"claude-3-5-sonnet": 0.003,
	"claude-3-5-haiku":  0.00025,
	"claude-3-opus":     0.015,
	"gpt-4o":            0.005,
	"gpt-4-turbo":       0.01,
	"gpt-4":             0.03,
	"gpt-3.5-turbo":     0.0005,
}

// AggregateEvents groups events by conversation into scans.
func AggregateEvents(events []models.Event) []models.Scan {
	// Group by conversation ID
	byConversation := make(map[string][]models.Event)
	for _, e := range events {
		if e.ConversationID == "" {
			continue
		}
		byConversation[e.ConversationID] = append(byConversation[e.ConversationID], e)
	}

	var scans []models.Scan
	for convID, convEvents := range byConversation {
		// Sort by timestamp
		sort.Slice(convEvents, func(i, j int) bool {
			return convEvents[i].Timestamp.Before(convEvents[j].Timestamp)
		})

		scan := createScan(convID, convEvents)
		scans = append(scans, scan)
	}

	// Sort scans by start time
	sort.Slice(scans, func(i, j int) bool {
		return scans[i].StartTime.Before(scans[j].StartTime)
	})

	return scans
}

func createScan(conversationID string, events []models.Event) models.Scan {
	scan := models.Scan{
		ConversationID: conversationID,
		Status:         models.ScanStatusPending,
		Events:         events,
	}

	// Generate ID from conversation ID and start time
	if len(events) > 0 {
		scan.StartTime = events[0].Timestamp
		scan.EndTime = events[len(events)-1].Timestamp

		hash := sha256.Sum256([]byte(conversationID + scan.StartTime.String()))
		scan.ID = hex.EncodeToString(hash[:8])
	}

	llmEvents := map[string]bool{
		"after_response": true, "after_tool": true, "after_file_edit": true,
		"after_file_read": true, "after_shell": true, "after_mcp": true, "after_model": true,
	}
	toolEvents := map[string]bool{
		"after_tool": true, "after_file_edit": true, "after_file_read": true,
		"after_shell": true, "after_mcp": true,
	}

	for _, e := range events {
		scan.InputTokens += e.InputTokens
		scan.OutputTokens += e.OutputTokens
		scan.ThinkingTokens += e.ThinkingTokens

		if llmEvents[e.NormalizedType] {
			scan.LLMCalls++
		}
		if toolEvents[e.NormalizedType] {
			scan.ToolCalls++
		}
	}

	scan.TotalTokens = scan.InputTokens + scan.OutputTokens + scan.ThinkingTokens
	scan.EstimatedCost = estimateCost(scan.TotalTokens, getModel(events))

	scan.Fingerprint = calculateFingerprint(events)
	scan.FilesHash = calculateFilesHash(events)
	scan.ActionCounts = calculateActionCounts(events)

	return scan
}

func getModel(events []models.Event) string {
	for _, e := range events {
		if e.Model != "" {
			return e.Model
		}
	}
	return "claude-3-5-sonnet"
}

func estimateCost(tokens int, model string) float64 {
	for prefix, price := range modelPricing {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			return float64(tokens) / 1000.0 * price
		}
	}
	return float64(tokens) / 1000.0 * 0.003
}

func calculateFingerprint(events []models.Event) string {
	var prompts []string
	for _, e := range events {
		if e.Prompt != "" {
			normalized := strings.ToLower(strings.TrimSpace(e.Prompt))
			if len(normalized) > 100 {
				normalized = normalized[:100]
			}
			if normalized != "" {
				prompts = append(prompts, normalized)
			}
		}
	}

	if len(prompts) == 0 {
		return ""
	}

	sort.Strings(prompts)
	combined := strings.Join(prompts, "|")
	hash := md5.Sum([]byte(combined))
	return hex.EncodeToString(hash[:])[:16]
}

func calculateFilesHash(events []models.Event) string {
	fileSet := make(map[string]bool)
	for _, e := range events {
		if e.FilePath != "" {
			fileSet[strings.ToLower(e.FilePath)] = true
		}
	}

	if len(fileSet) == 0 {
		return ""
	}

	var files []string
	for f := range fileSet {
		files = append(files, f)
	}
	sort.Strings(files)

	combined := strings.Join(files, "|")
	hash := md5.Sum([]byte(combined))
	return hex.EncodeToString(hash[:])[:8]
}

func calculateActionCounts(events []models.Event) map[string]int {
	counts := map[string]int{
		"edits":  0,
		"reads":  0,
		"shell":  0,
		"mcp":    0,
		"failed": 0,
	}

	editTypes := map[string]bool{"after_file_edit": true, "before_file_edit": true}
	readTypes := map[string]bool{"after_file_read": true, "before_file_read": true}
	shellTypes := map[string]bool{"after_shell": true, "before_shell": true}
	mcpTypes := map[string]bool{"after_mcp": true, "before_mcp": true}

	for _, e := range events {
		if editTypes[e.NormalizedType] {
			counts["edits"]++
		} else if readTypes[e.NormalizedType] {
			counts["reads"]++
		} else if shellTypes[e.NormalizedType] {
			counts["shell"]++
		} else if mcpTypes[e.NormalizedType] {
			counts["mcp"]++
		}

		if e.Error != "" {
			counts["failed"]++
		}
	}

	return counts
}

// CreateScanFromEvent creates a scan from a single event for immediate sync.
func CreateScanFromEvent(event models.Event) *models.Scan {
	// Generate scan ID
	idSource := event.ConversationID
	if idSource == "" {
		idSource = event.DeviceID + event.Timestamp.String()
	}
	hash := sha256.Sum256([]byte(idSource + event.Timestamp.String()))
	scanID := "scan_" + hex.EncodeToString(hash[:])[:12]

	scan := &models.Scan{
		ID:             scanID,
		DeviceID:       event.DeviceID,
		Timestamp:      event.Timestamp.Format(time.RFC3339Nano),
		Tool:           event.Tool,
		ConversationID: event.ConversationID,
		Status:         models.ScanStatusPending,
		StartTime:      event.Timestamp,
		EndTime:        event.Timestamp,
		Source: &models.ScanSource{
			Tool:      event.Tool,
			Event:     string(event.HookType),
			ToolName:  event.ToolName,
			SessionID: event.SessionID,
		},
		Content: &models.ScanContent{
			Prompt:    event.Prompt,
			Response:  event.Response,
			ToolInput: event.ToolInput,
		},
	}

	// Add the event (this also updates metrics)
	scan.AddEvent(event)

	// Calculate cost
	model := event.Model
	if model == "" {
		model = "claude-3-5-sonnet"
	}
	scan.EstimatedCost = estimateCost(scan.TotalTokens, model)

	// Calculate cross-scan detection metadata
	events := []models.Event{event}
	scan.Fingerprint = calculateFingerprint(events)
	scan.FilesHash = calculateFilesHash(events)
	scan.ActionCounts = calculateActionCounts(events)

	return scan
}
