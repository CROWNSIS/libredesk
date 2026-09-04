package ai

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

func TestStoredChunksRestoreDocumentOrderIndependentOfDatabaseRowOrder(t *testing.T) {
	vec := serializeEmbedding([]float32{1, 0})
	chunks := storedEmbeddingChunks([]models.Embedding{
		{ID: 30, SourceType: models.SourceHelpArticle, SourceID: 7, ChunkText: "Save", Embedding: vec},
		{ID: 10, SourceType: models.SourceHelpArticle, SourceID: 7, ChunkText: "Enter details", Embedding: vec},
		{ID: 20, SourceType: models.SourceSnippet, SourceID: 7, ChunkText: "Separate source", Embedding: vec},
		{ID: 25, SourceType: models.SourceHelpArticle, SourceID: 8, ChunkText: "Other article", Embedding: vec},
	})
	want := map[string]int{"Enter details": 0, "Save": 1, "Separate source": 0, "Other article": 0}
	for _, chunk := range chunks {
		if chunk.chunkOrder != want[chunk.chunkText] {
			t.Fatalf("%s order=%d want=%d", chunk.chunkText, chunk.chunkOrder, want[chunk.chunkText])
		}
	}
	if len(chunks) != len(want) {
		t.Fatalf("got %d chunks", len(chunks))
	}
}

func TestSearchKeepsDocumentOrderMetadataWhenRankingReversesSteps(t *testing.T) {
	ix := newEmbeddingIndex()
	ix.replaceAll([]indexedChunk{
		{sourceType: models.SourceHelpArticle, sourceID: 7, helpCenterID: 1, chunkText: "Enter details", chunkOrder: 0, vec: []float32{.8, .6}, norm: 1},
		{sourceType: models.SourceHelpArticle, sourceID: 7, helpCenterID: 1, chunkText: "Save", chunkOrder: 1, vec: []float32{1, 0}, norm: 1},
	})
	hits, _ := ix.searchHelpCenters([]float32{1, 0}, 2, map[int]bool{1: true})
	if len(hits) != 2 || hits[0].ChunkText != "Save" || hits[0].ChunkOrder != 1 || hits[1].ChunkOrder != 0 {
		t.Fatalf("ranking must preserve independent document positions: %#v", hits)
	}
	for _, hit := range rankHelpCandidates("save", hits, 2) {
		if hit.ChunkText == "Save" && hit.ChunkOrder != 1 || hit.ChunkText == "Enter details" && hit.ChunkOrder != 0 {
			t.Fatalf("hybrid ranking lost order: %#v", hit)
		}
	}
}
