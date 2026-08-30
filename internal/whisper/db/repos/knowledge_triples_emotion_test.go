package repos

import (
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
)

// TestTripleEmotionRoundTrip 情绪维度随三元组落库/读回（SchemaV14 列 + repo）。
func TestTripleEmotionRoundTrip(t *testing.T) {
	root := mgTestRoot(t)
	tp := whisper.Triple{
		ID: "kg_emotion1", Subject: "生日", Predicate: "属性",
		Object: "我的生日是 5 月 20 日", Confidence: 0.9,
		SourceFactIDs: []string{"f1"}, CreatedAt: time.Now(),
		EmotionLabel: "正面", EmotionalIntensity: 0.8, Valence: 0.6,
	}
	if err := InsertTriple(root, tp); err != nil {
		t.Fatalf("InsertTriple: %v", err)
	}
	all, err := LoadTriplesFromDB(root)
	if err != nil {
		t.Fatalf("LoadTriplesFromDB: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("应 1 条, got %d", len(all))
	}
	got := all[0]
	if got.EmotionLabel != "正面" || got.EmotionalIntensity != 0.8 || got.Valence != 0.6 {
		t.Errorf("情绪维度未读回: %+v", got)
	}
	if got.Subject != "生日" || got.Object != "我的生日是 5 月 20 日" {
		t.Errorf("三元组内容不符: %+v", got)
	}
}
