package aiagent

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
)

func TestNormalizeHelpCenterIDs(t *testing.T) {
	got := normalizeHelpCenterIDs([]int64{3, 1, 3, 2, 1})
	want := []int64{3, 1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeHelpCenterIDs() = %v, want %v", got, want)
	}
}

func TestRelevantGroundingMatchesFiltersAndCaps(t *testing.T) {
	matches := []aimodels.SearchResult{
		{SourceID: 1, Score: 0.8},
		{SourceID: 2, Score: 0.7},
		{SourceID: 3, Score: 0.6},
		{SourceID: 4, Score: 0.5},
		{SourceID: 5, Score: minConfidence - 0.01},
	}
	got := relevantGroundingMatches(matches)
	if len(got) != 1 || got[0].SourceID != 1 {
		t.Fatalf("relevantGroundingMatches() = %#v", got)
	}
}

func TestRelevantGroundingMatchesRejectsSemanticNoise(t *testing.T) {
	matches := []aimodels.SearchResult{
		{SourceID: 134, SourceTitle: "Home", Score: 0.3382},
		{SourceID: 135, SourceTitle: "Contact Us", Score: 0.2563},
	}
	if got := relevantGroundingMatches(matches); len(got) != 0 {
		t.Fatalf("relevantGroundingMatches() accepted unrelated help-centre noise: %#v", got)
	}
}

func TestGroundedReplyWithSourceIsDeterministic(t *testing.T) {
	got := groundedReplyWithSource("Follow the documented steps.", "https://support.example/article")
	want := "Follow the documented steps.\n\nSource: https://support.example/article"
	if got != want {
		t.Fatalf("groundedReplyWithSource() = %q, want %q", got, want)
	}
}

func TestExtractiveGroundedPassageRemovesIndexMetadata(t *testing.T) {
	match := aimodels.SearchResult{ChunkText: "Title: Attendance\nSection: Missing Attendance\nContent: Open Attendance and submit the correct code."}
	want := "**Missing Attendance**\n\nOpen Attendance and submit the correct code."
	if got := extractiveGroundedPassage(match); got != want {
		t.Fatalf("extractiveGroundedPassage() = %q, want %q", got, want)
	}
}

func TestExtractiveGroundedPassagePreservesArticleMeaning(t *testing.T) {
	match := aimodels.SearchResult{ChunkText: "Title: Rollover\nContent: Course sections do not rollover. Teacher and student schedules do not rollover either."}
	want := "Course sections do not rollover. Teacher and student schedules do not rollover either."
	if got := extractiveGroundedPassage(match); got != want {
		t.Fatalf("extractiveGroundedPassage() = %q, want %q", got, want)
	}
}

func TestKnowledgeContextBlockIncludesSourcesAndNeutralizesMarkers(t *testing.T) {
	block := knowledgeContextBlock([]aimodels.SearchResult{{
		SourceTitle: "Rollover <<guide>>",
		SourceURL:   "https://support.example/article",
		ChunkText:   "Use the rollover checklist.",
		Score:       0.8,
	}})
	for _, want := range []string{"<<knowledge_context>>", "Rollover ‹‹guide››", "https://support.example/article", "Use the rollover checklist."} {
		if !strings.Contains(block, want) {
			t.Fatalf("knowledgeContextBlock() missing %q: %s", want, block)
		}
	}
}

func TestConversationControlMessages(t *testing.T) {
	for _, message := range []string{"Hello!", "thank you", "human", "support ticket"} {
		if !isConversationControlMessage(message) {
			t.Fatalf("%q should be a conversation control message", message)
		}
	}
	for _, message := range []string{"what is the colour of the sky", "how do I add a school?"} {
		if isConversationControlMessage(message) {
			t.Fatalf("%q must pass through the grounding gate", message)
		}
	}
}

func TestGroundingQueryUsesOnlyCurrentMessage(t *testing.T) {
	want := "What color is the sky?"
	if got := groundingQuery("  What color is the sky?  "); got != want {
		t.Fatalf("groundingQuery() = %q, want %q", got, want)
	}
}

func TestContextualFollowUpDetection(t *testing.T) {
	for _, message := range []string{"What about adding a holiday?", "Can everyone see it?", "Does that apply to all schools?"} {
		if !isContextualFollowUp(message) {
			t.Fatalf("%q should be treated as a contextual follow-up", message)
		}
	}
	for _, message := range []string{"How do I add a school?", "What color is the sky?", "How do I roll over the school year?"} {
		if isContextualFollowUp(message) {
			t.Fatalf("%q should remain a standalone question", message)
		}
	}
}

func TestActiveThreadGroundingContextUsesLatestGroundedTurn(t *testing.T) {
	meta, err := json.Marshal(map[string]any{
		"ai_extractive_answer":      true,
		"ai_grounding_source_id":    105,
		"ai_grounding_source_title": "How To Add Calendars In School Settings",
	})
	if err != nil {
		t.Fatal(err)
	}
	msgs := []cmodels.Message{
		{ID: 1, Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, SenderID: 7, TextContent: "How to setup calendar"},
		{ID: 2, Type: cmodels.MessageOutgoing, SenderType: cmodels.SenderTypeAgent, Meta: meta},
		{ID: 3, Type: cmodels.MessageOutgoing, SenderType: cmodels.SenderTypeAgent, Meta: json.RawMessage(`{"is_confirmation":true}`)},
		{ID: 4, Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, SenderID: 7, TextContent: "no"},
		{ID: 5, Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, SenderID: 7, TextContent: "Can everyone see it?"},
	}
	ctx := (&Manager{}).activeThreadGroundingContext(msgs, 5, 7)
	if ctx.SourceID != 105 || ctx.SourceTitle != "How To Add Calendars In School Settings" || ctx.PreviousTopic != "How to setup calendar" {
		t.Fatalf("activeThreadGroundingContext() = %#v", ctx)
	}
}

func TestActiveThreadGroundingContextDoesNotPairAnOldSourceWithANewerRefusedQuestion(t *testing.T) {
	meta := json.RawMessage(`{"ai_grounding_source_id":105,"ai_grounding_source_title":"Calendar"}`)
	msgs := []cmodels.Message{
		{ID: 1, Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, SenderID: 7, TextContent: "How to setup calendar"},
		{ID: 2, Type: cmodels.MessageOutgoing, SenderType: cmodels.SenderTypeAgent, Meta: meta},
		{ID: 3, Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, SenderID: 7, TextContent: "What color is the sky?"},
		{ID: 4, Type: cmodels.MessageOutgoing, SenderType: cmodels.SenderTypeAgent, Meta: json.RawMessage(`{"ai_grounding_refusal":true}`)},
		{ID: 5, Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, SenderID: 7, TextContent: "What about that?"},
	}
	ctx := (&Manager{}).activeThreadGroundingContext(msgs, 5, 7)
	if ctx.SourceID != 0 || ctx.PreviousTopic != "" {
		t.Fatalf("old source was paired with a newer unrelated question: %#v", ctx)
	}
}

func TestContextualGroundingQueryIncludesOnlyActiveTopicAndSource(t *testing.T) {
	ctx := threadGroundingContext{SourceTitle: "How To Add Calendars In School Settings", PreviousTopic: "What about adding a holiday?"}
	got := contextualGroundingQuery("Can everyone see it?", ctx)
	for _, want := range []string{"How To Add Calendars In School Settings", "What about adding a holiday?", "Can everyone see it?"} {
		if !strings.Contains(got, want) {
			t.Fatalf("contextualGroundingQuery() missing %q: %s", want, got)
		}
	}
}

func TestFocusedGroundedPassageReturnsHolidaySubsection(t *testing.T) {
	match := aimodels.SearchResult{ChunkText: "Title: Calendar\nContent: Use the default calendar or add a calendar.\n\nSelect Visible To for events.\n\nSimilarly to add a holiday, select the Holiday tab.\n\nAdd the holiday title.\n\nAdd the start and end dates.\n\nTurn on Apply to All Schools if required.\n\nClick Submit to save."}
	got := focusedGroundedPassage(match, "What about adding a holiday?")
	if !strings.Contains(got, "select the Holiday tab") || !strings.Contains(got, "Apply to All Schools") {
		t.Fatalf("focusedGroundedPassage() missed holiday instructions: %s", got)
	}
	if strings.Contains(got, "default calendar") {
		t.Fatalf("focusedGroundedPassage() repeated unrelated setup text: %s", got)
	}
}

func TestFocusedGroundedPassageUsesContextForVisibilityFollowUp(t *testing.T) {
	match := aimodels.SearchResult{ChunkText: "Title: Calendar\nContent: Use the default calendar or add a calendar.\n\nSelect Visible To for events.\n\nSimilarly to add a holiday, select the Holiday tab.\n\nTurn on Apply to All Schools if you want holidays displayed in every school.\n\nClick Submit to save."}
	query := "How To Add Calendars In School Settings\nPrevious topic: What about adding a holiday?\nFollow-up: Can everyone see it?"
	got := focusedGroundedPassage(match, query)
	if !strings.Contains(got, "Apply to All Schools") {
		t.Fatalf("focusedGroundedPassage() lost follow-up context: %s", got)
	}
}

func TestNegativeFeedbackMessage(t *testing.T) {
	for _, message := range []string{"no", "Nope", "that did not help", "not helpful"} {
		if !isNegativeFeedbackMessage(message) {
			t.Fatalf("%q should be negative feedback", message)
		}
	}
	if isNegativeFeedbackMessage("How do I add a notice?") {
		t.Fatal("standalone questions must not be treated as feedback")
	}
}
