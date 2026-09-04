package ai

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

func TestHelpCandidatesIncludeSpecificProcedure(t *testing.T) {
	hits := []models.SearchResult{
		{SourceID: 135, Score: .56, ChunkText: "Title: Get Help in SISOL\nContent: Contact support for help with CrownSIS."},
		{SourceID: 2, Score: .55, ChunkText: "Title: Schedule Lesson Plans\nContent: Schedule existing plans on your calendar."},
		{SourceID: 3, Score: .54, ChunkText: "Title: Create and Manage Lesson Plans\nContent: To create a lesson plan, open the planner and choose New."},
	}
	got := rankHelpCandidates("How do I create a lesson plan in CrownSIS?", hits, 2)
	if len(got) != 2 || got[0].SourceID != 3 {
		t.Fatalf("specific procedure not ranked first: %#v", got)
	}
	if got[0].Score != .54 {
		t.Fatalf("cosine score changed: %v", got[0].Score)
	}
}

func TestHelpCandidatesKeepSemanticParaphrase(t *testing.T) {
	hits := []models.SearchResult{
		{SourceID: 1, Score: .9, ChunkText: "Title: Credentials\nContent: Recover access by requesting a reset link."},
		{SourceID: 2, Score: .7, ChunkText: "Title: Password Policy\nContent: Password rules require length."},
		{SourceID: 3, Score: .6, ChunkText: "Title: Password Rotation\nContent: Password expiration can be configured."},
	}
	got := rankHelpCandidates("forgot my password", hits, 2)
	found := false
	for _, hit := range got {
		if hit.SourceID == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("lost semantic-only paraphrase: %#v", got)
	}
}

func TestHelpCandidatesDiversifyDuplicateContent(t *testing.T) {
	hits := []models.SearchResult{
		{SourceID: 35, Score: .9, ChunkText: "Title: Lesson Plans\nContent: Create a new lesson plan."},
		{SourceID: 130, Score: .89, ChunkText: "Title: Lesson Plan Guide\nContent: Create a new lesson plan."},
		{SourceID: 40, Score: .8, ChunkText: "Title: Lesson Schedule\nContent: Assign the lesson plan to a date."},
	}
	got := rankHelpCandidates("lesson plan", hits, 2)
	if len(got) != 2 || retrievalContentKey(got[0].ChunkText) == retrievalContentKey(got[1].ChunkText) {
		t.Fatalf("duplicates crowded candidates: %#v", got)
	}
}

func TestHelpCandidatesScopeBeforeFusion(t *testing.T) {
	ix := newEmbeddingIndex()
	ix.replaceAll([]indexedChunk{
		{sourceType: models.SourceHelpArticle, sourceID: 1, helpCenterID: 20, chunkText: "Title: Create Lesson Plans\nContent: Create lesson plans.", vec: []float32{1, 0}, norm: 1},
		{sourceType: models.SourceHelpArticle, sourceID: 2, helpCenterID: 10, chunkText: "Title: Planner\nContent: Use the planner.", vec: []float32{.8, .6}, norm: 1},
		{sourceType: models.SourceSnippet, sourceID: 3, helpCenterID: 10, chunkText: "Title: Lesson Plans\nContent: Private snippet.", vec: []float32{1, 0}, norm: 1},
	})
	hits, _ := ix.searchHelpCenters([]float32{1, 0}, 0, map[int]bool{10: true})
	got := rankHelpCandidates("create lesson plans", hits, 2)
	if len(got) != 1 || got[0].SourceID != 2 {
		t.Fatalf("ineligible result entered hybrid search: %#v", got)
	}
	hits, _ = ix.searchHelpCenters([]float32{1, 0}, 0, map[int]bool{})
	if got := rankHelpCandidates("create lesson plans", hits, 2); len(got) != 0 {
		t.Fatalf("empty scope must fail closed: %#v", got)
	}
}

func TestHelpCandidatesCarryIntroductionWithLaterProcedure(t *testing.T) {
	hits := []models.SearchResult{
		{SourceID: 1, ChunkOrder: 3, Score: .9, ChunkText: "Enter scores and submit."},
		{SourceID: 1, ChunkOrder: 0, Score: .4, ChunkText: "For authorized administrators acting on behalf of teachers."},
		{SourceID: 2, ChunkOrder: 0, Score: .3, ChunkText: "Teacher workflow for your own class."},
	}
	got := rankHelpCandidates("enter scores", hits, 1)
	if len(got) != 1 || got[0].SourceContext != hits[1].ChunkText {
		t.Fatalf("lost source audience while ranking procedural chunk: %#v", got)
	}
	if hits[0].SourceContext != "" {
		t.Fatal("ranking mutated its input")
	}
}
