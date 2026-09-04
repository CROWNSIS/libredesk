package ai

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

// TestEvaluateRetrievalCorpus exports the real candidate-ranking implementation
// for an explicitly supplied corpus. It is an evaluation harness, not a quality
// assertion: the exported candidates must be assessed against labeled questions.
func TestEvaluateRetrievalCorpus(t *testing.T) {
	inputPath := os.Getenv("LIBREDESK_RETRIEVAL_CORPUS_INPUT")
	outputPath := os.Getenv("LIBREDESK_RETRIEVAL_CORPUS_OUTPUT")
	if inputPath == "" || outputPath == "" {
		t.Skip("set both LIBREDESK_RETRIEVAL_CORPUS_INPUT and LIBREDESK_RETRIEVAL_CORPUS_OUTPUT to evaluate a corpus")
	}
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	var corpus []struct {
		Question string                `json:"question"`
		Semantic []models.SearchResult `json:"semantic"`
	}
	if err := json.Unmarshal(input, &corpus); err != nil {
		t.Fatal(err)
	}
	type result struct {
		Question   string                `json:"question"`
		Candidates []models.SearchResult `json:"candidates"`
	}
	output := make([]result, 0, len(corpus))
	for _, item := range corpus {
		output = append(output, result{item.Question, rankHelpCandidates(item.Question, item.Semantic, 12)})
	}
	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// Exclusive creation protects an existing dataset or evidence artifact from
	// accidental replacement if the two environment paths are misconfigured.
	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(append(encoded, '\n')); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("exported candidates for %d corpus questions", len(output))
}
