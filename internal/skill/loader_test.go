package skill

import (
	"testing"
)

// TestLoaderParsesTestdataSkill 回归：加载器解析 testdata 夹具，且 SKILL.md
// 多行 applies_to 列表（- item）能正确读出。（原夹具为仓根 skills/ 写作域
// 两件 story-long-write/story-deslop；2026-11-23 两件随用户拍板退役，夹具改
// 自足 testdata，不再依赖仓内技能库。）
func TestLoaderParsesTestdataSkill(t *testing.T) {
	l := NewLoader("testdata")
	list := l.List()
	if len(list) == 0 {
		t.Fatal("未能加载任何技能")
	}
	sk := l.Get("demo-skill")
	if sk == nil {
		t.Fatal("demo-skill 未加载")
	}
	// 多行 applies_to 解析
	if got := sk.AppliesTo; len(got) != 2 || got[0] != "chapter" || got[1] != "outline" {
		t.Fatalf("demo-skill applies_to = %v, want [chapter outline]", got)
	}
}
