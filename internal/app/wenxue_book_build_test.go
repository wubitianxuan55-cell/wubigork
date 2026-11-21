package app

// wenxue_book_build_test.go — 2026-11-21 成册三合一回归钉：机检闸（硬伤拒组
// 装/warn 不拦）、配额面（目标 vs 实测+每章 JSON）、md 组装序。docx 静态目录
// E2E 依赖 soffice+技能脚本，缺席即 skip（回落域目录契约由 office 侧同款钉
// 覆盖，板间共用核心 measureTocEntries）。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/docmd"
)

func wbbSetup(t *testing.T, chapters map[string]string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GAEA_TUWEN_ROOT", root)
	final := filepath.Join(root, "定稿")
	if err := os.MkdirAll(final, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range chapters {
		if err := os.WriteFile(filepath.Join(final, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func wbbRun(t *testing.T, args map[string]any) (string, error) {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return wenxueBookBuildTool(&App{}).Execute(context.Background(), b)
}

func TestWenxueBookBuildGateRejectsHards(t *testing.T) {
	wbbSetup(t, map[string]string{
		"001-开端.md": "# 第一章 开端\n\n正文。",
		"001-撞号.md": "# 又一章\n\n正文。",
		"003-占位.md": "# 第三章\n\n【待补充】内容。",
		"楔子.md":    "# 楔子\n\n序。",
	})
	for _, want := range []string{"重号", "占位残留", "未编号", "拒绝组装"} {
		_, err := wbbRun(t, map[string]any{"format": "md", "output_path": "整本.md"})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("硬伤 %q 应拒组装，got %v", want, err)
		}
	}
}

func TestWenxueBookBuildAssembleAndQuota(t *testing.T) {
	wbbSetup(t, map[string]string{
		"001-风起.md": "# 第一章 风起\n\n" + strings.Repeat("长风破浪会有时。", 20),
		"002-云涌.md": "# 第二章 云涌\n\n" + strings.Repeat("直挂云帆济沧海。", 20),
	})
	out, err := wbbRun(t, map[string]any{
		"format": "md", "output_path": "整本.md",
		"title": "沧浪集", "target_words_total": 1000000, "tolerance_pct": 30})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2 章", "配额 warn", "偏差", "每章字数 JSON", `"no":1`, `"no":2`} {
		if !strings.Contains(out, want) {
			t.Errorf("回执缺 %q：%s", want, out)
		}
	}
	root := os.Getenv("GAEA_TUWEN_ROOT")
	b, rerr := os.ReadFile(filepath.Join(root, "整本.md"))
	if rerr != nil {
		t.Fatalf("整本未落盘: %v", rerr)
	}
	joined := string(b)
	if strings.Index(joined, "第一章 风起") > strings.Index(joined, "第二章 云涌") {
		t.Fatal("章节应按章号序拼接")
	}
}

func TestWenxueBookBuildStaticTocE2E(t *testing.T) {
	if docmd.FindSoffice() == "" {
		t.Skip("LibreOffice 缺席，docx 静态目录 E2E 不可测")
	}
	root := t.TempDir()
	t.Setenv("GAEA_TUWEN_ROOT", root)
	// 技能脚本副本进临时工作区（wenxueFindSkillScript 首选位）。
	dst := filepath.Join(root, ".gaea", "skills", "docx", "scripts")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	src, serr := os.ReadFile(filepath.Join(ormRepoRootForSkills(t), ".gaea", "skills", "docx", "scripts", "create_docx.py"))
	if serr != nil {
		t.Skipf("仓内技能脚本不可读: %v", serr)
	}
	if err := os.WriteFile(filepath.Join(dst, "create_docx.py"), src, 0o644); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(root, "定稿")
	os.MkdirAll(final, 0o755)
	os.WriteFile(filepath.Join(final, "001-风起.md"), []byte("# 第一章 风起\n\n"+strings.Repeat("长风破浪会有时。", 60)), 0o644)
	os.WriteFile(filepath.Join(final, "002-云涌.md"), []byte("# 第二章 云涌\n\n"+strings.Repeat("直挂云帆济沧海。", 60)), 0o644)

	out, err := wbbRun(t, map[string]any{
		"format": "docx", "output_path": "整本.docx", "toc": true, "cover": true, "author": "测试作者"})
	if err != nil {
		t.Fatalf("docx 成册失败: %v", err)
	}
	if strings.Contains(out, "TOC 域回落") {
		t.Skipf("pdftotext 不可用，回落域目录（回执如实标注）：%s", out)
	}
	if !strings.Contains(out, "目录=静态页码（2 章渲染实测") {
		t.Fatalf("回执应标注静态目录：%s", out)
	}
	if !strings.Contains(out, "sha256=") {
		t.Fatalf("回执应含 sha256：%s", out)
	}
}
