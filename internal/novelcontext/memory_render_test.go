package novelcontext

import (
	"strings"
	"testing"

)

// TestRenderMemories t3-P2：相关记忆区段渲染 + 单行截断 + 区段预算。
func TestRenderMemories(t *testing.T) {
	b := &SceneBible{
		Setting: "蒸汽纪元",
		Memories: []string{
			"- (相关度:0.82) " + strings.Repeat("忆", 200), // 超 memoryLineMax 截断
			"- (相关度:0.51) 短记忆",
			"", // 空行跳过
		},
	}
	r := b.Render(0)
	if !strings.Contains(r, "## 相关记忆（按相关度）") {
		t.Fatalf("缺记忆区段:\n%s", r)
	}
	if strings.Count(r, "忆") > memoryLineMax {
		t.Fatalf("单行应截断到 %d: %d", memoryLineMax, strings.Count(r, "忆"))
	}
	if !strings.Contains(r, "短记忆") {
		t.Fatalf("短记忆应保留:\n%s", r)
	}

	// 区段整体预算：大量记忆条目不超过 memoryBudget（加省略号容差）
	b.Memories = nil
	for i := 0; i < 20; i++ {
		b.Memories = append(b.Memories, "- (相关度:0.90) 记忆条目"+strings.Repeat("容", memoryLineMax))
	}
	r = b.Render(0)
	seg := r[strings.Index(r, "## 相关记忆（按相关度）"):]
	if end := strings.Index(seg, "\n\n"); end >= 0 {
		seg = seg[:end]
	}
	if len([]rune(seg)) > memoryBudget+10 { // +10 = 截断省略号与标题行容差
		t.Fatalf("记忆区段应 ≤%d rune，实际 %d", memoryBudget, len([]rune(seg)))
	}
}

// TestRenderMemories_Empty 空记忆不出区段标题（空段不输出）。
func TestRenderMemories_Empty(t *testing.T) {
	b := &SceneBible{Setting: "x"}
	if r := b.Render(0); strings.Contains(r, "相关记忆") {
		t.Fatalf("空记忆不应输出区段:\n%s", r)
	}
}
