package app

// v4.28 B2「pptx 最小交互」测试：
//   - 解析层（parsePptxOutline）纯单测——CI 无 python 也锁行为；
//   - 绑定早退（路径不存在/非 pptx）——结构化错误不 panic；
//   - .pptx 预览分支——seam 注入（verifyConvertToPdf/verifyRenderPages），
//     锁缓存命中/页缩略/回退/失败透出，不依赖真实 LibreOffice/poppler；
//   - 真机冒烟（GAEA_SMOKE_EXPORT 门控，跟随 gaea_export_test.go 惯例）。

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// TestParsePptxOutlineOutput 解析层：截断/封顶/空白剔除/空页/结构化错误。
func TestParsePptxOutlineOutput(t *testing.T) {
	long := strings.Repeat("长", 300)
	out := fmt.Sprintf(`{"slides": [
		{"index": 1, "title": "季度总结", "texts": [%q, "  ", "要点A"], "shapeCount": 5},
		{"index": 2, "title": "", "texts": [], "shapeCount": 2}
	]}`, long)

	v := parsePptxOutline(out)
	if !v.Available || v.Error != "" {
		t.Fatalf("应可用: %+v", v)
	}
	if len(v.Slides) != 2 {
		t.Fatalf("slides = %d, want 2", len(v.Slides))
	}
	s1 := v.Slides[0]
	if s1.Index != 1 || s1.Title != "季度总结" || s1.ShapeCount != 5 {
		t.Errorf("第 1 页字段不符: %+v", s1)
	}
	// 超长文本按 rune 截断为上限 + 省略号；空白条剔除；末条保留
	if got := []rune(s1.Texts[0]); len(got) != pptxOutlineTextLimit+1 || got[pptxOutlineTextLimit] != '…' {
		t.Errorf("截断应为 %d rune+省略号, 得 %d rune", pptxOutlineTextLimit, len(got))
	}
	if len(s1.Texts) != 2 || s1.Texts[1] != "要点A" {
		t.Errorf("空白条应剔除/末条保留: %q", s1.Texts)
	}
	// 空页：Texts 恒非 nil（JSON null 会硌前端）
	if s2 := v.Slides[1]; s2.Texts == nil || len(s2.Texts) != 0 {
		t.Errorf("空页 Texts 应为非 nil 空切片: %#v", s2.Texts)
	}

	// 条数封顶：50 条只收前 pptxOutlineTextsMax 条
	var items []string
	for i := 0; i < 50; i++ {
		items = append(items, fmt.Sprintf("t%02d", i))
	}
	many := `{"slides": [{"index": 1, "title": "T", "texts": ["` + strings.Join(items, `","`) + `"], "shapeCount": 50}]}`
	if got := parsePptxOutline(many).Slides[0]; len(got.Texts) != pptxOutlineTextsMax || got.Texts[0] != "t00" {
		t.Errorf("条数应封顶 %d: %d", pptxOutlineTextsMax, len(got.Texts))
	}

	// 结构化错误：python-pptx 缺失走 {"error"}，退出码 0
	verr := parsePptxOutline(`{"error": "python-pptx 不可用"}`)
	if verr.Available || !strings.Contains(verr.Error, "python-pptx") {
		t.Errorf("error JSON 应透出: %+v", verr)
	}
	// 坏 JSON 同样结构化
	vbad := parsePptxOutline("not json")
	if vbad.Available || vbad.Error == "" {
		t.Errorf("坏 JSON 应结构化报错: %+v", vbad)
	}
	// 空演示文稿：slides 缺失也视为可用（0 页）
	vempty := parsePptxOutline("{}")
	if !vempty.Available || vempty.Slides == nil || len(vempty.Slides) != 0 {
		t.Errorf("空 slides 应可用且非 nil: %+v", vempty)
	}
}

// TestGaeaPptxOutline_Missing 绑定早退：路径不存在 → 结构化错误不 panic。
func TestGaeaPptxOutline_Missing(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	v := a.GaeaPptxOutline("nope/deck.pptx")
	if v.Available || v.Error == "" {
		t.Fatalf("应结构化报错: %+v", v)
	}
	if len(v.Slides) != 0 {
		t.Errorf("失败时不应带 slides: %+v", v.Slides)
	}
}

// TestGaeaPptxOutline_NotPptx 非 pptx 扩展名 → 结构化错误（含格式提示）。
func TestGaeaPptxOutline_NotPptx(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("readme.txt", []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	v := a.GaeaPptxOutline("readme.txt")
	if v.Available || !strings.Contains(v.Error, "仅支持 .pptx") {
		t.Fatalf("应提示仅支持 .pptx: %+v", v)
	}
}

// TestGaeaPreview_Pptx .pptx 预览分支（seam 注入）：kind=pdf + 逐页缩略 +
// outline 提示；二次预览命中缓存（转换只发生一次）。
func TestGaeaPreview_Pptx(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join("exports", "deck.pptx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, []byte("fake pptx"), 0o644); err != nil {
		t.Fatal(err)
	}

	png, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatal(err)
	}
	convCalls := 0
	renderCalls := 0
	oldConv, oldRender := verifyConvertToPdf, verifyRenderPages
	defer func() { verifyConvertToPdf, verifyRenderPages = oldConv, oldRender }()
	verifyConvertToPdf = func(src, out string) error {
		convCalls++
		return os.WriteFile(out, []byte("%PDF-fake"), 0o644)
	}
	verifyRenderPages = func(pdfPath, prefix string, dpi int) ([]string, error) {
		renderCalls++
		var paths []string
		for i := 1; i <= 2; i++ {
			p := fmt.Sprintf("%s-%d.png", prefix, i)
			if err := os.WriteFile(p, png, 0o644); err != nil {
				return nil, err
			}
			paths = append(paths, p)
		}
		return paths, nil
	}

	a := &App{}
	got := a.GaeaPreview(rel)
	if got.Kind != "pdf" {
		t.Fatalf("kind = %q, want pdf", got.Kind)
	}
	if got.Hint != pptxOutlineHint {
		t.Errorf("hint = %q, want %q", got.Hint, pptxOutlineHint)
	}
	if len(got.Pages) != 2 || got.Pages[0].Page != 1 || got.Pages[1].Page != 2 {
		t.Fatalf("pages = %+v, want 1/2 两页", got.Pages)
	}
	for i, p := range got.Pages {
		if !strings.HasPrefix(p.DataURL, "data:image/png;base64,") {
			t.Errorf("pages[%d] dataUrl 前缀不符: %.40s", i, p.DataURL)
		}
	}
	if got.Truncated || got.TotalPages != 2 {
		t.Errorf("truncated/totalPages = %v/%d, want false/2", got.Truncated, got.TotalPages)
	}
	if convCalls != 1 || renderCalls != 1 {
		t.Fatalf("首次预览应各调一次转换/渲染: conv=%d render=%d", convCalls, renderCalls)
	}

	// 二次预览：PDF 与页图缓存全命中，不再转换/渲染
	got2 := a.GaeaPreview(rel)
	if got2.Kind != "pdf" || len(got2.Pages) != 2 {
		t.Fatalf("二次预览异常: kind=%s pages=%d", got2.Kind, len(got2.Pages))
	}
	if convCalls != 1 || renderCalls != 1 {
		t.Errorf("缓存命中后不应重转/重渲: conv=%d render=%d", convCalls, renderCalls)
	}
}

// TestGaeaPreview_Pptx_ConvertFail soffice 转换失败 → Kind=error 带原因。
func TestGaeaPreview_Pptx_ConvertFail(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := "deck.pptx"
	if err := os.WriteFile(rel, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldConv, oldRender := verifyConvertToPdf, verifyRenderPages
	defer func() { verifyConvertToPdf, verifyRenderPages = oldConv, oldRender }()
	verifyConvertToPdf = func(src, out string) error {
		return fmt.Errorf("未找到 LibreOffice（soffice），请安装 LibreOffice 后重试")
	}
	verifyRenderPages = func(pdfPath, prefix string, dpi int) ([]string, error) {
		t.Fatal("转换失败后不应渲染")
		return nil, nil
	}

	a := &App{}
	got := a.GaeaPreview(rel)
	if got.Kind != "error" || !strings.Contains(got.Error, "LibreOffice") {
		t.Fatalf("应 Kind=error 带原因: %+v", got)
	}
}

// TestGaeaPreview_Pptx_RenderFallback pdftoppm 缺失（渲染 seam 失败）→
// 回退整本 PDF dataUrl（WebView 内嵌查看器），kind 仍为 pdf。
func TestGaeaPreview_Pptx_RenderFallback(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := "deck.pptx"
	if err := os.WriteFile(rel, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldConv, oldRender := verifyConvertToPdf, verifyRenderPages
	defer func() { verifyConvertToPdf, verifyRenderPages = oldConv, oldRender }()
	verifyConvertToPdf = func(src, out string) error {
		return os.WriteFile(out, []byte("%PDF-fallback"), 0o644)
	}
	verifyRenderPages = func(pdfPath, prefix string, dpi int) ([]string, error) {
		return nil, fmt.Errorf("视觉 diff 需要 poppler 渲染（pdftoppm），但未找到")
	}

	a := &App{}
	got := a.GaeaPreview(rel)
	if got.Kind != "pdf" {
		t.Fatalf("kind = %q, want pdf（回退而非报错）", got.Kind)
	}
	if len(got.Pages) != 0 {
		t.Errorf("回退时不应带 pages: %d", len(got.Pages))
	}
	if !strings.HasPrefix(got.DataURL, "data:application/pdf;base64,") {
		t.Errorf("应回退整本 PDF dataUrl: %.60s", got.DataURL)
	}
}

// TestGaeaPptxOutline_Smoke 真机冒烟：导出真实 pptx 后解析大纲（默认跳过）：
//   GAEA_SMOKE_EXPORT=1 go test ./internal/app -run TestGaeaPptxOutline_Smoke -v
func TestGaeaPptxOutline_Smoke(t *testing.T) {
	if os.Getenv("GAEA_SMOKE_EXPORT") == "" {
		t.Skip("未设置 GAEA_SMOKE_EXPORT")
	}
	orig, _ := os.Getwd()
	root := orig
	for {
		if _, err := os.Stat(filepath.Join(root, ".gaea", "skills", "pptx", "scripts", "create_pptx.py")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("未找到仓库 .gaea/skills")
		}
		root = parent
	}
	ga.cfg = &gaeaConfig.Config{Workspace: root}
	t.Chdir(t.TempDir())
	a := &App{}
	got, err := a.GaeaExportDeliverable(ExportDeliverableInput{
		Markdown: "# 封面页\n\n# 数据页\n\n- 营收增长 12%\n- 成本下降 3%\n", Format: "pptx", Title: "大纲冒烟",
	})
	if err != nil {
		t.Fatalf("pptx 导出失败: %v", err)
	}
	v := a.GaeaPptxOutline(got.Path)
	if !v.Available {
		t.Fatalf("大纲应可用: %+v", v)
	}
	if len(v.Slides) < 3 {
		t.Fatalf("slides = %d, want >=3（封面+两内容页）", len(v.Slides))
	}
	// create_pptx.py 用空白版式（无标题占位符）：标题回退为首个单行短文本框
	if v.Slides[0].Title != "大纲冒烟" || v.Slides[1].Title != "封面页" {
		t.Errorf("标题回退不符: %+v", v.Slides)
	}
	var dataSlide *PptxSlideOutline
	for i := range v.Slides {
		if v.Slides[i].Title == "数据页" {
			dataSlide = &v.Slides[i]
		}
		if s := v.Slides[i]; s.Index != i+1 {
			t.Errorf("Index 应为 1-based 连续页码: slides[%d].Index=%d", i, s.Index)
		}
	}
	if dataSlide == nil {
		t.Fatalf("未找到「数据页」: %+v", v.Slides)
	}
	joined := strings.Join(dataSlide.Texts, "\n")
	if !strings.Contains(joined, "营收增长") {
		t.Errorf("正文文本缺失: %+v", dataSlide.Texts)
	}
}
