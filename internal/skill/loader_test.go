package skill

import (
	"testing"
)

// TestLoaderLoadsExistingSkills 回归：加载器应能解析仓库内置写作技能，
// 且 SKILL.md 多行 applies_to 列表（- item）能正确读出。
func TestLoaderLoadsExistingSkills(t *testing.T) {
	l := NewLoader("../../skills")
	list := l.List()
	t.Logf("loaded %d skills", len(list))
	for _, s := range list {
		t.Logf("name=%q desc=%q applies=%v version=%q", s.Name, s.Description, s.AppliesTo, s.Version)
	}
	if len(list) == 0 {
		t.Fatal("未能加载任何技能")
	}
	if l.Get("story-long-write") == nil {
		t.Fatal("story-long-write 未加载")
	}
	if l.Get("story-deslop") == nil {
		t.Fatal("story-deslop 未加载")
	}
	// 多行 applies_to 解析
	if got := l.Get("story-long-write").AppliesTo; len(got) != 2 {
		t.Fatalf("story-long-write applies_to = %v, want [chapter outline]", got)
	}
}
