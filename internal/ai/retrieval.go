package ai

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

// rankHelpCandidates fuses semantic and lexical ranks only after help-center
// authorization. Score remains the original cosine similarity: fusion is candidate
// selection, not an assertion that an article can answer the question.
func rankHelpCandidates(query string, semantic []models.SearchResult, k int) []models.SearchResult {
	if len(semantic) == 0 {
		return semantic
	}
	terms := retrievalTerms(query)
	tfs := make([]map[string]float64, len(semantic))
	lengths := make([]float64, len(semantic))
	df := map[string]int{}
	average := 0.0
	for i, hit := range semantic {
		tfs[i] = map[string]float64{}
		for _, term := range retrievalTerms(hit.ChunkText) {
			tfs[i][term]++
			lengths[i]++
		}
		// Titles carry topic signal even when the passage is procedural.
		if title, _, ok := strings.Cut(hit.ChunkText, "\n"); ok && strings.HasPrefix(title, "Title: ") {
			for _, term := range retrievalTerms(strings.TrimPrefix(title, "Title: ")) {
				tfs[i][term] += 2
			}
		}
		for term := range tfs[i] {
			df[term]++
		}
		average += lengths[i]
	}
	average = math.Max(average/float64(len(semantic)), 1)
	type candidate struct {
		index          int
		lexical, fused float64
	}
	ranked := make([]candidate, len(semantic))
	for i := range semantic {
		ranked[i] = candidate{index: i, fused: 1 / (20 + float64(i+1))}
		seen := map[string]bool{}
		for _, term := range terms {
			if seen[term] || tfs[i][term] == 0 {
				continue
			}
			seen[term] = true
			idf := math.Log(1 + (float64(len(semantic)-df[term])+0.5)/(float64(df[term])+0.5))
			tf := tfs[i][term]
			ranked[i].lexical += idf * tf * 2.2 / (tf + 1.2*(0.25+0.75*lengths[i]/average))
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].lexical > ranked[j].lexical })
	for i := range ranked {
		if ranked[i].lexical > 0 {
			ranked[i].fused += 1.2 / (20 + float64(i+1))
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].fused == ranked[j].fused {
			return ranked[i].index < ranked[j].index
		}
		return ranked[i].fused > ranked[j].fused
	})
	limit := len(ranked)
	if k > 0 && k < limit {
		limit = k
	}
	out := make([]models.SearchResult, 0, limit)
	seen := map[string]bool{}
	for _, candidate := range ranked {
		hit := semantic[candidate.index]
		key := retrievalContentKey(hit.ChunkText)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, hit)
		if len(out) == limit {
			break
		}
	}
	// Reserve a candidate for the strongest semantic match, including paraphrases
	// without shared words. Avoid replacing an equivalent duplicate passage.
	best := semantic[0]
	if !seen[retrievalContentKey(best.ChunkText)] && len(out) > 1 {
		out[len(out)-1] = best
	}
	return out
}

func retrievalContentKey(text string) string {
	if _, content, ok := strings.Cut(text, "Content:"); ok && strings.TrimSpace(content) != "" {
		text = content
	}
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

func retrievalTerms(text string) []string {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	out := make([]string, 0, len(words))
	for _, word := range words {
		switch word {
		case "a", "an", "and", "are", "as", "at", "be", "by", "can", "do", "for", "from", "how", "i", "in", "is", "it", "of", "on", "or", "that", "the", "this", "to", "was", "what", "when", "where", "which", "with", "you", "your", "title", "section", "content":
			continue
		}
		// Normalize ordinary plurals without changing words ending in ss.
		if len(word) > 3 && strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
			word = strings.TrimSuffix(word, "s")
		}
		out = append(out, word)
	}
	return out
}
