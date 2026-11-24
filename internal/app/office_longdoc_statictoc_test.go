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
	"regexp"
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
	// 章正文带题注+引用（2026-11-24 编号联动：docx 层题注转 SEQ 域、引用转 REF 域）。
	if err := os.WriteFile(ch1, []byte("# 第一章 总体方案\n\n总体安排见图1-1。\n\n图1-1 总体安排流程\n\n总体安排正文。\n\n## 1.1 编制依据\n\n依据清单正文。"), 0o644); err != nil {
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
	// 登记格式模板（2026-11-24 方案格式模板线）：工作区 .gaea/templates/docx/
	// 登记一份（2 级标题改 15pt 以证模板面真生效），build 直接传模板名。
	tplDir := filepath.Join(ormRepoRootForSkills(t), ".gaea", "templates", "docx")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tplPath := filepath.Join(tplDir, "测试方案模板.json")
	if err := os.WriteFile(tplPath, []byte(`{"name":"测试方案模板","base":"标书","headings":{"2":{"size":15}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tplPath) }()
	btl := longdocBuildTool{a: &App{}}
	receipt, err := btl.Execute(context.Background(), ormJSON(t, map[string]any{
		"manifest": manifest, "format": "docx", "output_path": outRel, "template": "测试方案模板",
		"toc": true, "cover": true, "author": "某某环境科技有限公司"}))
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if strings.Contains(receipt, "TOC 域回落") {
		t.Skipf("pdftotext 不可用，回落域目录（回执如实标注）：%s", receipt)
	}
	if !strings.Contains(receipt, "目录=TOC 域+实测缓存") {
		t.Fatalf("回执应标注域化目录：%s", receipt)
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
	// 目录=TOC 域带实测缓存（2026-11-24 联动化）：域指令在档（WPS F9 全联动
	// 重排），缓存条目=两遍法实测页码（打开即准确，不吃「旧页码」坑）。
	if !strings.Contains(d, `TOC \o "1-2"`) {
		t.Fatal("目录应为 TOC 域（缓存静态条目但域可重排——纯静态无域=旧行为复发）")
	}
	if !strings.Contains(d, `w:fldCharType="separate"`) || !strings.Contains(d, `w:fldCharType="end"`) {
		t.Fatal("TOC 复杂域应带 separate/end（缓存结果结构不完整）")
	}
	// 编号联动（2026-11-24）：标题字面前缀剥净（md 写「第一章」，编号引擎按
	// 标书口径渲染「第1章」），目录条目文本=计算编号；字面量页 key 不中走裸
	// 题名兜底，页码仍回填成功。
	if strings.Contains(d, "第一章") {
		t.Fatalf("标题字面编号前缀应已剥净（编号=样式层多级列表）：%s", d[:min(400, len(d))])
	}
	if !strings.Contains(d, "第1章 总体方案") || !strings.Contains(d, "1.1 编制依据") {
		t.Fatalf("目录条目缺计算编号章/节标题：%s", d[:min(400, len(d))])
	}
	// 页码是数字（两遍法回填成功），不是占位符「—」。
	if strings.Contains(d, ">—<") {
		t.Fatal("目录页码仍是占位符「—」（量页腿未回填）")
	}
	// 题注=SEQ 域（序列名编码章号）+书签；正文引用=REF 域。
	if !strings.Contains(d, `SEQ 图1 \* ARABIC`) || !strings.Contains(d, `w:name="_Gaea图1_1"`) {
		t.Fatalf("题注应为按章 SEQ 域+书签：%s", d[:min(1200, len(d))])
	}
	if !strings.Contains(d, `REF _Gaea图1_1 \h`) {
		t.Fatal("正文引用应为 REF 域（F9 联动）")
	}
	// 登记模板面：工作区模板生效——Heading 2 字号按模板 15pt（styles.xml sz=30 半点）。
	var sty strings.Builder
	for _, f := range zr.File {
		if f.Name == "word/styles.xml" {
			rc, oerr := f.Open()
			if oerr != nil {
				t.Fatal(oerr)
			}
			b, _ := io.ReadAll(rc)
			_ = rc.Close()
			sty.Write(b)
		}
	}
	h2 := regexp.MustCompile(`w:styleId="Heading2".*?</w:style>`).FindString(sty.String())
	if !strings.Contains(h2, `w:sz w:val="30"`) {
		t.Fatalf("登记模板 Heading 2=15pt 未生效（styles.xml Heading2 缺 sz=30）：%s", h2)
	}
}
