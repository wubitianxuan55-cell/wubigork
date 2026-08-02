package app

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseChangelog 验证 CHANGELOG 解析：版本块切割、标题/日期/要点
func TestParseChangelog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	content := `# gaea

## v1.6.4「设置瘦身」(2026-08-02)

> 设置页删除重复引擎配置。

- 移除 EnginePanel tab
- 验证全绿

## v1.6.3「剧照模型可选」(2026-08-02)

> 补丁：角色剧照可选模型。

- 剧照弹窗可选 ComfyUI 本地模型
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir) // findChangelogPath 先查 cwd

	releases := parseChangelog(8)
	if len(releases) != 2 {
		t.Fatalf("releases = %d, want 2", len(releases))
	}
	r0 := releases[0]
	if r0["version"] != "v1.6.4" || r0["title"] != "设置瘦身" || r0["date"] != "2026-08-02" {
		t.Errorf("头部解析 = %v", r0)
	}
	if !contains(r0["points"], "移除 EnginePanel tab") {
		t.Errorf("要点缺失: %q", r0["points"])
	}
	if !contains(r0["intro"], "设置页删除重复引擎配置") {
		t.Errorf("简介缺失: %q", r0["intro"])
	}
}

func contains(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && (s == sub || len(s) > len(sub) && (len(sub) == 0 || indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
