package scanner

import (
	"testing"
	"time"

	"github.com/atbabers/intentra-cli/pkg/models"
)

func TestAggregateEvents(t *testing.T) {
	events := []models.Event{
		{HookType: models.HookBeforeSubmitPrompt, ConversationID: "conv-1", Timestamp: time.Now()},
		{HookType: models.HookAfterAgentResponse, ConversationID: "conv-1", Timestamp: time.Now().Add(time.Second)},
		{HookType: models.HookStop, ConversationID: "conv-1", Timestamp: time.Now().Add(2 * time.Second)},
	}

	scans := AggregateEvents(events)
	if len(scans) != 1 {
		t.Fatalf("Expected 1 scan, got %d", len(scans))
	}
	if len(scans[0].Events) != 3 {
		t.Errorf("Expected 3 events in scan, got %d", len(scans[0].Events))
	}
}

func TestAggregateMultipleConversations(t *testing.T) {
	events := []models.Event{
		{HookType: models.HookBeforeSubmitPrompt, ConversationID: "conv-1", Timestamp: time.Now()},
		{HookType: models.HookBeforeSubmitPrompt, ConversationID: "conv-2", Timestamp: time.Now()},
		{HookType: models.HookStop, ConversationID: "conv-1", Timestamp: time.Now().Add(time.Second)},
		{HookType: models.HookStop, ConversationID: "conv-2", Timestamp: time.Now().Add(time.Second)},
	}

	scans := AggregateEvents(events)
	if len(scans) != 2 {
		t.Fatalf("Expected 2 scans, got %d", len(scans))
	}
}
