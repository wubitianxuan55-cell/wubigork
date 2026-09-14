package types

import "testing"

// TestCareerLabels 职业标签渲染（t5 §7.4-1 回灌用）：
// 主/副带阶渲染、无阶直渲染、CareerName 快照优先、缺失退 career_id、空跳过。
func TestCareerLabels(t *testing.T) {
	c := Character{
		MainCareerID:    "剑修",
		MainCareerStage: 3,
		SubCareers: []CharacterCareerRef{
			{CareerID: "cd-1", CareerName: "炼丹师", Stage: 2},
			{CareerID: "cd-2", Stage: 0},   // 无名快照退 ID、无阶直渲染
			{CareerID: "", CareerName: ""}, // 全空防御跳过
		},
	}
	if got := c.CareerMainLabel(); got != "剑修·3阶" {
		t.Errorf("CareerMainLabel = %q, want 剑修·3阶", got)
	}
	subs := c.CareerSubLabels()
	if len(subs) != 2 {
		t.Fatalf("CareerSubLabels len = %d, want 2（全空引用跳过）: %v", len(subs), subs)
	}
	if subs[0] != "炼丹师·2阶" {
		t.Errorf("subs[0] = %q, want 炼丹师·2阶", subs[0])
	}
	if subs[1] != "cd-2" {
		t.Errorf("subs[1] = %q, want cd-2（CareerName 缺失退 ID）", subs[1])
	}

	// 无主职业 / 零阶主职业
	if got := (Character{}).CareerMainLabel(); got != "" {
		t.Errorf("空角色 CareerMainLabel = %q, want 空", got)
	}
	if got := (Character{MainCareerID: "剑修"}).CareerMainLabel(); got != "剑修" {
		t.Errorf("零阶 CareerMainLabel = %q, want 剑修", got)
	}
	if got := (Character{}).CareerSubLabels(); len(got) != 0 {
		t.Errorf("空角色 CareerSubLabels = %v, want 空", got)
	}
}
