package memory

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/util"
)

func TestTokenize_Chinese(t *testing.T) {
	tokens := tokenize("青云宗坐落于苍山")
	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d: %v", len(tokens), tokens)
	}
	// 应该有 2-gram: "青云", "云宗", "坐落", "落于", "于苍", "苍山"
	hasBigram := false
	for _, tok := range tokens {
		if tok == "青云" || tok == "苍山" {
			hasBigram = true
		}
	}
	if !hasBigram {
		t.Fatalf("expected Chinese bigrams, got: %v", tokens)
	}
}

func TestTokenize_Mixed(t *testing.T) {
	tokens := tokenize("Elara walked into 青云宗")
	if len(tokens) < 3 {
		t.Fatalf("expected at least 3 tokens, got %d: %v", len(tokens), tokens)
	}
	hasEnglish := false
	hasChinese := false
	for _, tok := range tokens {
		if tok == "elara" || tok == "walked" {
			hasEnglish = true
		}
		if strings.ContainsAny(tok, "青云宗") || tok == "云宗" {
			hasChinese = true
		}
	}
	if !hasEnglish || !hasChinese {
		t.Fatalf("expected mixed tokens, got: %v", tokens)
	}
}

func TestBM25_AddSearch(t *testing.T) {
	idx := NewIndex()

	idx.Add(Memory{ID: "ch1", ChapterNum: 1, Text: "Elara enters the 青云宗 for the first time"})
	idx.Add(Memory{ID: "ch2", ChapterNum: 2, Text: "青云宗 holds a grand ceremony"})
	idx.Add(Memory{ID: "ch3", ChapterNum: 3, Text: "Kael trains in the mountains"})

	// 搜索 "青云宗"
	results := idx.Search("青云宗", 3)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results for 青云宗, got %d", len(results))
	}
	if results[0].ChapterNum != 2 && results[0].ChapterNum != 1 {
		t.Fatalf("highest scoring should be ch1 or ch2, got ch%d", results[0].ChapterNum)
	}

	// 搜索不存在的词
	results2 := idx.Search("zzzznonexistent", 3)
	if len(results2) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results2))
	}
}

func TestBM25_EmptyIndex(t *testing.T) {
	idx := NewIndex()
	results := idx.Search("anything", 5)
	if len(results) != 0 {
		t.Fatalf("expected 0 results from empty index, got %d", len(results))
	}
}

func TestEstimateTokens(t *testing.T) {
	chinese := "这是一段中文测试文本"
	tokens := util.EstimateTokens(chinese)
	if tokens < 5 || tokens > 20 {
		t.Fatalf("unexpected token estimate for Chinese: %d", tokens)
	}

	english := "This is a test sentence for token estimation"
	tokens2 := util.EstimateTokens(english)
	if tokens2 < 5 || tokens2 > 25 {
		t.Fatalf("unexpected token estimate for English: %d", tokens2)
	}
}
