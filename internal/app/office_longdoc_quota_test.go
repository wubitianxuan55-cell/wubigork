package app

// office_longdoc_quota_test.go — 2026-11-21 反馈 P2-9 回归钉：check 附配额面
// （字数目标偏差+图/表题注数 vs 期望）与结构化 JSON 行——此前配额验收要自写
// 四套高度重叠的脚本。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLongdocCheckQuotaSummary(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "manifest.json")
	ch1 := filepath.Join(dir, "01.md")
	// 300 汉字目标给 30 字=偏差 −90%；图 2/表 1 题注。
	if err := os.WriteFile(ch1, []byte("# 第一章 总体方案\n\n图1-1 总平面图\n\n图1-2 工艺流程图\n\n表1-1 参数表\n\n三十个汉字的正文内容三十个汉字的正文内容三十个汉字"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := longdocManifestTool{a: &App{}}
	if _, err := tl.Execute(context.Background(), ormJSON(t, map[string]any{
		"action": "init", "manifest": manifest, "title": "修复技术标"})); err != nil {
		t.Fatal(err)
	}
	if _, err := tl.Execute(context.Background(), ormJSON(t, map[string]any{
		"action": "set", "manifest": manifest, "title": "修复技术标",
		"requirements": []map[string]any{{"id": "R1", "text": "评分点1"}},
		"chapters": []map[string]any{{
			"no": 1, "title": "第一章 总体方案", "path": ch1, "requirements": []string{"R1"},
			"words": 300, "figs": 2, "tables": 1}}})); err != nil {
		t.Fatal(err)
	}
	out := ormRun(t, tl, map[string]any{"action": "check", "manifest": manifest})
	if !strings.Contains(out, "机检全绿") {
		t.Fatalf("配额 warn 不拦组装（非硬伤）：\n%s", out)
	}
	if !strings.Contains(out, "字数偏差") || !strings.Contains(out, "第 1 章字数偏差") {
		t.Fatalf("超差应出配额 warn：\n%s", out)
	}
	if !strings.Contains(out, "配额摘要 JSON：") || !strings.Contains(out, `"figs":2`) || !strings.Contains(out, `"words_target":300`) {
		t.Fatalf("应附结构化配额 JSON：\n%s", out)
	}
	// 图数不符：期望 3 实际 2 → check warn 点名。
	ormRun(t, tl, map[string]any{"action": "set", "manifest": manifest, "title": "修复技术标",
		"requirements": []map[string]any{{"id": "R1", "text": "评分点1"}},
		"chapters": []map[string]any{{
			"no": 1, "title": "第一章 总体方案", "path": ch1, "requirements": []string{"R1"},
			"words": 300, "figs": 3, "tables": 1}}})
	out2 := ormRun(t, tl, map[string]any{"action": "check", "manifest": manifest})
	if !strings.Contains(out2, "图数期望 3，实际题注 2") {
		t.Fatalf("图数不符应 warn：\n%s", out2)
	}
}
