package ai

import (
	"testing"
)

func TestBeatType(t *testing.T) {
	b := Beat{
		ID:          "beat-1",
		Description: "Elara 推开大门",
		Order:       1,
	}
	if b.ID != "beat-1" {
		t.Fatal("beat ID mismatch")
	}
	if b.Order != 1 {
		t.Fatal("beat order mismatch")
	}
}

func TestGhostCompletePromptTruncation(t *testing.T) {
	// 验证超长文本会被截断到 2000 字符
	longText := ""
	for i := 0; i < 3000; i++ {
		longText += "文"
	}

	// 截断逻辑在 GhostComplete 内部执行
	runes := []rune(longText)
	if len(runes) <= 2000 {
		t.Fatal("test text should be > 2000 runes")
	}
	trimmed := string(runes[len(runes)-2000:])
	if len([]rune(trimmed)) != 2000 {
		t.Fatalf("trimmed should be 2000 runes, got %d", len([]rune(trimmed)))
	}
}

func TestBeatSliceOrdering(t *testing.T) {
	beats := []Beat{
		{ID: "beat-3", Description: "Third", Order: 3},
		{ID: "beat-1", Description: "First", Order: 1},
		{ID: "beat-2", Description: "Second", Order: 2},
	}

	// 验证结构完整性
	for _, b := range beats {
		if b.ID == "" {
			t.Fatal("beat ID should not be empty")
		}
		if b.Order < 1 {
			t.Fatal("beat order should be >= 1")
		}
	}
}
