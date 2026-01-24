package models

// Severity levels for violations.
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// ViolationType represents a specific violation pattern.
type ViolationType string

const (
	// High severity
	ViolationRetryLoop           ViolationType = "retry_loop"
	ViolationIgnoredInstructions ViolationType = "ignored_instructions"
	ViolationTimeoutContinuation ViolationType = "timeout_continuation"

	// Medium severity
	ViolationToolMisuse         ViolationType = "tool_misuse"
	ViolationContextWaste       ViolationType = "context_waste"
	ViolationExcessiveThinking  ViolationType = "excessive_thinking"
	ViolationDuplicateWork      ViolationType = "duplicate_work"
	ViolationBrowserLoop        ViolationType = "browser_loop"
	ViolationGenerationRecovery ViolationType = "generation_recovery"

	// Low severity
	ViolationOverGeneration        ViolationType = "over_generation"
	ViolationIterativeInefficiency ViolationType = "iterative_inefficiency"
	ViolationThinkingWithoutAction ViolationType = "thinking_without_action"

	// Dynamic Context Discovery violations (Medium-High severity)
	ViolationScopeCreep              ViolationType = "scope_creep"
	ViolationIrrelevantContext       ViolationType = "irrelevant_context"
	ViolationUnnecessaryCodebaseScan ViolationType = "unnecessary_codebase_scan"
	ViolationFailedContextRetrieval  ViolationType = "failed_context_retrieval"
	ViolationStaleContextUsage       ViolationType = "stale_context_usage"
	ViolationContextHallucination    ViolationType = "context_hallucination"
)

// Severity returns the severity level for a violation type.
func (v ViolationType) Severity() Severity {
	switch v {
	case ViolationRetryLoop, ViolationIgnoredInstructions, ViolationTimeoutContinuation,
		ViolationContextHallucination, ViolationScopeCreep:
		return SeverityHigh
	case ViolationToolMisuse, ViolationContextWaste, ViolationExcessiveThinking,
		ViolationDuplicateWork, ViolationBrowserLoop, ViolationGenerationRecovery,
		ViolationIrrelevantContext, ViolationUnnecessaryCodebaseScan,
		ViolationFailedContextRetrieval, ViolationStaleContextUsage:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// BaseScore returns the base refund score for a violation type.
func (v ViolationType) BaseScore() int {
	switch v {
	case ViolationRetryLoop:
		return 85
	case ViolationIgnoredInstructions, ViolationContextHallucination:
		return 80
	case ViolationTimeoutContinuation, ViolationScopeCreep:
		return 75
	case ViolationToolMisuse, ViolationContextWaste, ViolationIrrelevantContext:
		return 65
	case ViolationExcessiveThinking, ViolationDuplicateWork, ViolationUnnecessaryCodebaseScan:
		return 60
	case ViolationBrowserLoop, ViolationGenerationRecovery, ViolationFailedContextRetrieval:
		return 55
	case ViolationStaleContextUsage:
		return 50
	default:
		return 40
	}
}

// LegalBasis returns the legal grounds for refund.
func (v ViolationType) LegalBasis() string {
	switch v {
	case ViolationRetryLoop, ViolationIgnoredInstructions, ViolationTimeoutContinuation:
		return "FTC Act Section 5, FCBA"
	case ViolationToolMisuse, ViolationContextWaste, ViolationDuplicateWork:
		return "FTC Act, State UDAP"
	case ViolationContextHallucination, ViolationScopeCreep:
		return "FTC Act Section 5 - Deceptive practices, FCBA"
	case ViolationIrrelevantContext, ViolationUnnecessaryCodebaseScan,
		ViolationFailedContextRetrieval, ViolationStaleContextUsage:
		return "FTC Act - Wasteful operations, Contract Theory"
	default:
		return "Contract Theory, Pattern Evidence"
	}
}

// Violation represents a detected violation.
type Violation struct {
	Type         ViolationType  `json:"type"`
	Severity     Severity       `json:"severity"`
	Confidence   float64        `json:"confidence"`
	Description  string         `json:"description"`
	Evidence     []string       `json:"evidence"`
	BaseScore    int            `json:"base_score"`
	LegalBasis   string         `json:"legal_basis,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
	StartEvent   int            `json:"start_event,omitempty"`
	EndEvent     int            `json:"end_event,omitempty"`
	WastedTokens int            `json:"wasted_tokens,omitempty"`
	WastedCost   float64        `json:"wasted_cost,omitempty"`
}
