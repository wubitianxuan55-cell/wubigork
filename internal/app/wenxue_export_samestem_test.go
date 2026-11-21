package app

// wenxue_export_samestem_test.go — 2026-11-21 沿江反馈批对账发现的图文侧漂移
// 回归钉：office exportDocx 批6 A1/A2 已根修「同 stem 中间 md 吞源文件」，图文
// 副本仍落 <输出stem>.md 再删——同步后（随机临时名）导出成败皆不得触碰同 stem
// 源文件，也不得残留中间临时件。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

func TestWenxueExportDocxSameStemNeverTouchesSourceMD(t *testing.T) {
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ormRepoRootForSkills(t)} // 技能脚本走仓根
	defer func() { ga.cfg = oldCfg }()
	dir := t.TempDir()
	src := filepath.Join(dir, "第九章 终章.md")
	if err := os.WriteFile(src, []byte("章节源内容"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "第九章 终章.docx")
	_ = wenxueExportDocx(ExportDeliverableInput{Markdown: "整册内容", Template: "通用"}, "终章", out) // python 成败皆可：契约与成败无关
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("同 stem 源 md 被删除（A1/A2 图文侧复发）: %v", err)
	}
	if string(b) != "章节源内容" {
		t.Fatalf("同 stem 源 md 被覆写: %q", string(b))
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".gaea-tmp-") {
			t.Fatalf("中间临时件泄漏: %s", e.Name())
		}
	}
}
