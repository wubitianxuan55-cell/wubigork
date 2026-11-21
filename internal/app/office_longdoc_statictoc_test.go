package app

// office_longdoc_statictoc_test.go — 2026-11-21 反馈 P0-2 建议③回归钉：静态
// 目录两遍法。toc=true 的 docx 构建不留需要 Ctrl+A→F9 的 TOC 域——占位目录
// pass-1 → soffice→PDF → pdftotext 逐页文本 → 标题实测页回填重建。渲染链
// （LibreOffice/poppler）缺席时如实回落域目录（回执标注），不静默降级。

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/docmd"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

func TestLongdocBuildStaticTocTwoPass(t *testing.T) {
	if docmd.FindSoffice() == "" {
		t.Skip("LibreOffice 缺席，静态目录两遍法不可测")
	}
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ormRepoRootForSkills(t)}
	defer func() { ga.cfg = oldCfg }()
	// 产物落仓内 build/tmp（gitignore 面），测试自清。
	outDir := filepath.Join(ormRepoRootForSkills(t), "build", "tmp-statictoc-test")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(outDir) }()

	manifest := filepath.Join(outDir, "manifest.json")
	ch1 := filepath.Join(outDir, "01.md")
	if err := os.WriteFile(ch1, []byte("# 第一章 总体方案\n\n总体安排正文。\n\n## 1.1 编制依据\n\n依据清单正文。"), 0o644); err != nil {
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
		"chapters":     []map[string]any{{"no": 1, "title": "第一章 总体方案", "path": ch1, "requirements": []string{"R1"}}}})); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(outDir, "整册.docx")
	outRel, err := filepath.Rel(ormRepoRootForSkills(t), out)
	if err != nil {
		t.Fatal(err)
	}
	outRel = filepath.ToSlash(outRel) // 声明写路径=工作区相对
	btl := longdocBuildTool{a: &App{}}
	receipt, err := btl.Execute(context.Background(), ormJSON(t, map[string]any{
		"manifest": manifest, "format": "docx", "output_path": outRel,
		"toc": true, "cover": true, "author": "某某环境科技有限公司"}))
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if strings.Contains(receipt, "TOC 域回落") {
		t.Skipf("pdftotext 不可用，回落域目录（回执如实标注）：%s", receipt)
	}
	if !strings.Contains(receipt, "目录=静态页码") {
		t.Fatalf("回执应标注静态目录：%s", receipt)
	}
	// docx 实体断言：无 TOC 域指令；静态条目=标题+数字页码。
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var doc strings.Builder
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, oerr := f.Open()
			if oerr != nil {
				t.Fatal(oerr)
			}
			b, _ := io.ReadAll(rc)
			_ = rc.Close()
			doc.Write(b)
		}
	}
	d := doc.String()
	if strings.Contains(d, `TOC \o`) {
		t.Fatal("静态目录不应残留 TOC 域指令（需 F9 刷新的域目录=旧行为复发）")
	}
	if !strings.Contains(d, "第一章 总体方案") || !strings.Contains(d, "1.1 编制依据") {
		t.Fatalf("目录条目缺章/节标题：%s", d[:min(400, len(d))])
	}
	// 页码是数字（两遍法回填成功），不是占位符「—」。
	if strings.Contains(d, ">—<") {
		t.Fatal("目录页码仍是占位符「—」（量页腿未回填）")
	}
}
