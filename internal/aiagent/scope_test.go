package aiagent

import (
	"reflect"
	"strings"
	"testing"

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
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
	if len(got) != 2 || got[0].SourceID != 1 || got[1].SourceID != 2 {
		t.Fatalf("relevantGroundingMatches() = %#v", got)
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
