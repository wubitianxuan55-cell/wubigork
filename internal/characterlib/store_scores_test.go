package characterlib

import (
	"strings"
	"testing"
)

// v4.411 一致性评分写回：reference_scores 列往返 + 超限剔除时同索引对齐。
func TestReferenceScoresRoundTrip(t *testing.T) {
	s := newTestStore(t)
	c := &Character{
		ID:              "c-scored",
		Name:            "评分角色",
		Kind:            KindCustom,
		ReferenceImages: []string{"ref-a", "ref-b", "ref-c"},
		ReferenceScores: []int{91, 0, 47}, // ref-b 未评分
	}
	if err := s.Upsert(c); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := s.Get("c-scored")
	if err != nil || got == nil {
		t.Fatalf("Get: %v, %v", got, err)
	}
	if len(got.ReferenceScores) != 3 || got.ReferenceScores[0] != 91 || got.ReferenceScores[1] != 0 || got.ReferenceScores[2] != 47 {
		t.Fatalf("分数往返失真: %v", got.ReferenceScores)
	}
}

func TestReferenceScoresCapAlignment(t *testing.T) {
	s := newTestStore(t)
	huge := "data:image/png;base64," + strings.Repeat("A", maxPortraitDataURL+10)
	c := &Character{
		ID:              "c-cap",
		Name:            "超限对齐",
		Kind:            KindCustom,
		ReferenceImages: []string{huge, "ref-keep", "ref-tiny-data"},
		ReferenceScores: []int{88, 72, 55},
	}
	if err := s.Upsert(c); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := s.Get("c-cap")
	if err != nil || got == nil {
		t.Fatalf("Get: %v, %v", got, err)
	}
	// 巨型 data URL（88 分那张）被剔除后，分数须同索引对齐：72/55 前移
	if len(got.ReferenceImages) != 2 {
		t.Fatalf("参考图应剩 2 张: %v", got.ReferenceImages)
	}
	if len(got.ReferenceScores) != 2 || got.ReferenceScores[0] != 72 || got.ReferenceScores[1] != 55 {
		t.Fatalf("分数应对齐为 [72 55]: %v", got.ReferenceScores)
	}
}

func TestReferenceScoresShorterThanImages(t *testing.T) {
	s := newTestStore(t)
	c := &Character{
		ID:              "c-short",
		Name:            "短分数表",
		Kind:            KindCustom,
		ReferenceImages: []string{"ref-a", "ref-b"},
		ReferenceScores: []int{80}, // 新增图未评分（sin 侧追加场景）
	}
	if err := s.Upsert(c); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := s.Get("c-short")
	if err != nil || got == nil {
		t.Fatalf("Get: %v, %v", got, err)
	}
	if len(got.ReferenceScores) != 2 || got.ReferenceScores[0] != 80 || got.ReferenceScores[1] != 0 {
		t.Fatalf("短分数表应补 0 对齐: %v", got.ReferenceScores)
	}
}
