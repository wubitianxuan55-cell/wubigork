package app

import "testing"

// TestNovelGhostSuggest 内联续写绑定（v4.444）：输出清理（空白/包裹引号/解释
// 前缀）、空建议拒绝、上下文不足拒绝。
func TestNovelGhostSuggest(t *testing.T) {
	a := newRewriteTestApp(t, "  接续：他推门而入，风雪灌了一身。  ")
	out, err := a.NovelGhostSuggest("夜里雪落无声，他在门外站了许久，终于抬手推门。")
	if err != nil {
		t.Fatalf("续写: %v", err)
	}
	if out != "他推门而入，风雪灌了一身。" {
		t.Fatalf("输出应清理包裹与前缀: %q", out)
	}

	empty := newRewriteTestApp(t, "   ")
	if _, err := empty.NovelGhostSuggest("足够长的上下文内容哦足够长的上下文内容哦"); err == nil {
		t.Fatalf("空建议必须拒绝")
	}

	if _, err := a.NovelGhostSuggest("太短"); err == nil {
		t.Fatalf("上下文不足必须拒绝")
	}
}
