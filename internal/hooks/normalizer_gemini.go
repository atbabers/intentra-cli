// Package hooks provides Gemini CLI-specific event normalization.
package hooks

// GeminiNormalizer normalizes Gemini CLI hook events to unified types.
type GeminiNormalizer struct{}

func init() {
	RegisterNormalizer(&GeminiNormalizer{})
}

// Tool returns the tool identifier.
func (n *GeminiNormalizer) Tool() string { return "gemini" }

var geminiEventMapping = map[string]NormalizedEventType{
	"SessionStart":        EventSessionStart,
	"SessionEnd":          EventSessionEnd,
	"BeforeAgent":         EventBeforePrompt,
	"AfterAgent":          EventAfterResponse,
	"BeforeModel":         EventBeforeModel,
	"AfterModel":          EventAfterModel,
	"BeforeToolSelection": EventToolSelection,
	"BeforeTool":          EventBeforeTool,
	"AfterTool":           EventAfterTool,
	"PreCompress":         EventPreCompact,
	"Notification":        EventNotification,
}

// NormalizeEventType converts Gemini CLI PascalCase events to snake_case normalized types.
func (n *GeminiNormalizer) NormalizeEventType(native string) NormalizedEventType {
	if normalized, ok := geminiEventMapping[native]; ok {
		return normalized
	}
	return EventUnknown
}
