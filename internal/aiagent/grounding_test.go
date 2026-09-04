package aiagent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
)

func TestAnswerPassagesRequireExplicitEvidenceSelection(t *testing.T) {
	matches := []aimodels.SearchResult{
		{SourceID: 135, Score: .8, ChunkText: "Title: Get Help\nContent: Contact support.", SourceURL: "https://help.test/support"},
		{SourceID: 35, Score: .7, SourceContext: "Teacher lesson planning", ChunkText: "Title: Lesson Plans\nContent: Select New Lesson, enter a title, and save.", SourceURL: "https://help.test/lessons"},
	}
	complete := func(_ context.Context, system, input string) (string, error) {
		if !strings.Contains(system, "untrusted") || !strings.Contains(system, "directly answer") {
			t.Fatal("selection must distinguish reference data from instructions and require answerability")
		}
		var data struct {
			Sources  map[string]string `json:"sources"`
			Question string            `json:"question"`
			Passages []struct {
				Number    int    `json:"number"`
				Text      string `json:"text"`
				SourceURL string `json:"source_url"`
			} `json:"passages"`
		}
		if err := json.Unmarshal([]byte(input), &data); err != nil || len(data.Passages) != 2 {
			t.Fatalf("invalid candidate request: %s", input)
		}
		if data.Passages[1].SourceURL != matches[1].SourceURL {
			t.Fatal("selector needs original source context to distinguish audience-specific workflows")
		}
		if data.Sources[matches[1].SourceURL] != matches[1].SourceContext {
			t.Fatal("selector lost source introduction")
		}
		return `{"passages":[2]}`, nil
	}
	got, err := selectAnswerPassages(context.Background(), "How do I create a lesson?", matches, complete)
	if err != nil || len(got) != 1 || got[0].SourceID != 35 {
		t.Fatalf("must use selected instructional evidence, not highest cosine: %#v, %v", got, err)
	}
}

func TestAnswerPassagesFailClosed(t *testing.T) {
	matches := []aimodels.SearchResult{{SourceID: 15, Score: .8, ChunkText: "Billing settings", SourceURL: "https://help.test/billing"}}
	for _, output := range []string{`{"passages":[]}`, `{"passages":[2]}`, `{"passages":[0]}`, `{"passages":[1,1]}`, `{"passages":[1],"answer":"invented instructions"}`, `{"passages":[1]} trailing`, `{"passages":[1.5]}`, `{}`, `not json`} {
		t.Run(output, func(t *testing.T) {
			got, _ := selectAnswerPassages(context.Background(), "Payroll in another product", matches, func(context.Context, string, string) (string, error) { return output, nil })
			if len(got) != 0 {
				t.Fatalf("must not fall back to a merely related passage: %#v", got)
			}
		})
	}
	_, err := selectAnswerPassages(context.Background(), "question", matches, func(context.Context, string, string) (string, error) { return "", errors.New("provider offline") })
	if err == nil {
		t.Fatal("provider failure must remain observable")
	}
}

func TestAnswerPassagesDoNotCallModelWithoutEligibleEvidence(t *testing.T) {
	got, err := selectAnswerPassages(context.Background(), "unrelated", []aimodels.SearchResult{{Score: .2, ChunkText: "weak match"}}, func(context.Context, string, string) (string, error) { t.Fatal("no eligible context"); return "", nil })
	if err != nil || len(got) != 0 {
		t.Fatalf("got %#v, %v", got, err)
	}
}

func TestGroundedPassageReplyPreservesNegationAndServerCitations(t *testing.T) {
	got := groundedPassageReply([]aimodels.SearchResult{
		{ChunkText: "Title: Rollover\nContent: Course sections do not roll over.", SourceURL: "https://help.test/rollover"},
		{ChunkText: "Title: Rollover\nContent: Create the next school year first.", SourceURL: "https://help.test/rollover"},
	})
	for _, want := range []string{"Course sections do not roll over.", "Create the next school year first.", "Source: https://help.test/rollover"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Count(got, "Source:") != 1 {
		t.Fatal("duplicate source links")
	}
}

func TestAnswerPassagesRestoreDocumentStepOrder(t *testing.T) {
	matches := []aimodels.SearchResult{
		{SourceID: 18, Score: .8, ChunkOrder: 8, ChunkText: "Save the payment", SourceURL: "https://help.test/payment"},
		{SourceID: 18, Score: .7, ChunkOrder: 4, ChunkText: "Enter the amount", SourceURL: "https://help.test/payment"},
	}
	got, err := selectAnswerPassages(context.Background(), "Record a payment", matches, func(context.Context, string, string) (string, error) { return `{"passages":[1,2]}`, nil })
	if err != nil || len(got) != 2 || got[0].ChunkText != "Enter the amount" {
		t.Fatalf("must not save before data entry: %#v %v", got, err)
	}
}

func TestArticleSelectionRestrictsEvidenceToOneWorkflow(t *testing.T) {
	matches := []aimodels.SearchResult{
		{SourceID: 1, SourceType: "help_article", Score: .9, SourceURL: "https://help.test/admin", SourceContext: "Administrator workflow", ChunkText: "Edit scores"},
		{SourceID: 2, SourceType: "help_article", Score: .7, SourceURL: "https://help.test/teacher", SourceContext: "Teacher workflow", ChunkText: "Open own class"},
		{SourceID: 2, SourceType: "help_article", Score: .6, SourceURL: "https://help.test/teacher", SourceContext: "Teacher workflow", ChunkText: "Save own class scores"},
	}
	got, err := selectArticleCandidates(context.Background(), "As a teacher", matches, func(_ context.Context, system, input string) (string, error) {
		if !strings.Contains(input, "Administrator workflow") || !strings.Contains(input, "Teacher workflow") {
			t.Fatal("missing audience context")
		}
		return `{"article":2}`, nil
	})
	if err != nil || len(got) != 2 || got[0].SourceID != 2 || got[1].SourceID != 2 {
		t.Fatalf("mixed workflows: %#v %v", got, err)
	}
	for _, response := range []string{`{"article":0}`, `{"article":3}`, `{"article":-1}`, `{}`, `{"article":null}`, `{"article":1,"other":2}`, `{"article":1} trailing`} {
		got, _ := selectArticleCandidates(context.Background(), "question", matches, func(context.Context, string, string) (string, error) { return response, nil })
		if len(got) != 0 {
			t.Fatalf("invalid selection must fail closed: %s", response)
		}
	}
}

func TestArticleSelectionPrefersCompleteDuplicateGuide(t *testing.T) {
	matches := []aimodels.SearchResult{
		{SourceID: 1, Score: .9, SourceURL: "https://help.test/copy", SourceContext: "Same guide introduction", ChunkText: "Before you begin"},
		{SourceID: 2, Score: .8, SourceURL: "https://help.test/full", SourceContext: "Same guide introduction", ChunkText: "Enter fields"},
		{SourceID: 2, Score: .7, SourceURL: "https://help.test/full", SourceContext: "Same guide introduction", ChunkText: "Save changes"},
	}
	got, err := selectArticleCandidates(context.Background(), "question", matches, func(_ context.Context, _, input string) (string, error) {
		if strings.Contains(input, "https://help.test/copy") {
			t.Fatal("fragmented duplicate displaced complete guide")
		}
		return `{"article":1}`, nil
	})
	if err != nil || len(got) != 2 || got[0].SourceID != 2 {
		t.Fatalf("lost complete guide: %#v %v", got, err)
	}
}
