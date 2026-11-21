package tool

import "testing"

// 功能组目录三守卫（2026-11-21 众神殿 pan_1795224680124 终案批A1）：
// ①组成员 ∈ RegistryMap（在册对账）；②一个可配置工具只属一组（跨组重复
// buildToolGroupOf 已 panic 钉死，此处验反查一致）；③未知组 fail-closed
// 判据（ValidToolGroup）。

func TestToolGroupsMembersInRegistryMap(t *testing.T) {
	for _, g := range ToolGroups() {
		for _, name := range ToolGroupMembers(g) {
			if _, ok := RegistryMap[name]; !ok {
				t.Fatalf("组 %s 成员 %s 不在 RegistryMap（目录与注册面漂移）", g, name)
			}
		}
	}
}

func TestToolGroupOfUnique(t *testing.T) {
	seen := map[string]ToolGroupName{}
	for _, g := range ToolGroups() {
		for _, name := range ToolGroupMembers(g) {
			if prev, ok := seen[name]; ok && prev != g {
				t.Fatalf("工具 %s 跨组归属（%s/%s）", name, prev, g)
			}
			seen[name] = g
			if got := ToolGroupOf(name); got != g {
				t.Fatalf("ToolGroupOf(%s)=%s ≠ %s", name, got, g)
			}
		}
	}
	if ToolGroupOf("read_file") != "" || ToolGroupOf("bash") != "" {
		t.Fatalf("core 工具不应有组归属：read_file=%q bash=%q", ToolGroupOf("read_file"), ToolGroupOf("bash"))
	}
	if ToolGroupOf("cost_project") != ToolGroupCost {
		t.Fatalf("cost_project 应归 cost 组")
	}
}

func TestValidToolGroup(t *testing.T) {
	for _, g := range ToolGroups() {
		if !ValidToolGroup(g) {
			t.Fatalf("在册组 %s 判定失败", g)
		}
	}
	for _, bad := range []ToolGroupName{"bogus", "", "Browser"} {
		if ValidToolGroup(bad) {
			t.Fatalf("未知/非法组 %q 应 fail-closed", bad)
		}
	}
}
