package tool

import "testing"

// Registry.HiddenNames（2026-11-21 能力面板装配面缝）：升序快照、只含真实
// 注册过的名字（ghost 藏名不显）。
func TestRegistryHiddenNames(t *testing.T) {
	reg := NewRegistry()
	reg.Add(stubTool{name: "b_view"})
	reg.Add(stubTool{name: "a_hidden"})
	reg.Hide("a_hidden")
	reg.Hide("ghost_never_registered")
	got := reg.HiddenNames()
	if len(got) != 1 || got[0] != "a_hidden" {
		t.Fatalf("HiddenNames = %v，want [a_hidden]（ghost 不显、升序）", got)
	}
	// Unhide（同 key 重写 map——直接删验证：Hide 幂等 + Schemas 侧消失）
	schemas := reg.Schemas()
	if len(schemas) != 1 || schemas[0].Name != "b_view" {
		t.Fatalf("Schemas 应排除 hidden：%v", schemas)
	}
}
