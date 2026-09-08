package app

// pptx 真编辑刀2 测试（GaeaPptxSlideText，docs/gaea-pptx-edit-design-2026-09.md §4 刀2）：
//   - 正常路径：两页段落全文（含超 200 rune 长段不被截断——大纲 texts 是截断
//     预览，本绑定必须回全文）+ 裸文件名回退（resolvePreviewPath 口径）；
//   - 早退：非 .pptx 扩展名 / 文件不存在，均结构化报错不 panic。
// 夹具=测试内 zip.Writer 构造的最小 pptx（与 internal/office/pptxedit 既有
// helper 同款口径）；页码序 = presentation.xml sldIdLst 的 r:id → rels 映射
//（slideParts 口径），刻意与 slide 文件名逆序，锁「页码序非文件名/zip 条目序」。

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	slideTextPresXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<p:sldIdLst>
<p:sldId id="256" r:id="rId2"/>
<p:sldId id="257" r:id="rId3"/>
</p:sldIdLst>
</p:presentation>`

	// rId2 → slide2.xml（放映第 1 页）、rId3 → slide1.xml（放映第 2 页）：
	// 映射逆文件名序，页码序只由 sldIdLst 决定。
	slideTextPresRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://x/slideMaster" Target="slideMasters/slideMaster1.xml"/>
<Relationship Id="rId2" Type="http://x/slide" Target="slides/slide2.xml"/>
<Relationship Id="rId3" Type="http://x/slide" Target="slides/slide1.xml"/>
</Relationships>`

	// 放映第 1 页：两段普通文本
	slideTextPage1XML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
<p:cSld><p:spTree>
<p:sp>
<p:txBody><a:bodyPr/><a:p><a:r><a:rPr lang="zh-CN" sz="4000"/><a:t>首页标题段落</a:t></a:r></a:p>
<a:p><a:r><a:rPr lang="zh-CN"/><a:t>首页正文段落：成本下降 3%</a:t></a:r></a:p>
</p:txBody></p:sp>
</p:spTree></p:cSld>
</p:sld>`
)

// buildPptxSlideTextFixture 构造 2 页、每页 2 段的最小 pptx（zip 字节）。
// 放映第 2 页的第 2 段为 longPara（测试传 >200 rune，锁全文不被截断）。
func buildPptxSlideTextFixture(t *testing.T, longPara string) []byte {
	t.Helper()
	page2XML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
<p:cSld><p:spTree>
<p:sp>
<p:txBody><a:bodyPr/><a:p><a:r><a:rPr lang="zh-CN" sz="4000"/><a:t>次页标题段落</a:t></a:r></a:p>
<a:p><a:r><a:rPr lang="zh-CN"/><a:t>` + longPara + `</a:t></a:r></a:p>
</p:txBody></p:sp>
</p:spTree></p:cSld>
</p:sld>`
	entries := []struct{ name, body string }{
		{"[Content_Types].xml", "<Types/>"},
		{"ppt/presentation.xml", slideTextPresXML},
		{"ppt/_rels/presentation.xml.rels", slideTextPresRels},
		// 映射逆文件名序：slide2.xml=放映第 1 页（rId2）、slide1.xml=放映第 2 页（rId3）
		{"ppt/slides/slide1.xml", page2XML},
		{"ppt/slides/slide2.xml", slideTextPage1XML},
		{"ppt/theme/theme1.xml", "<theme/>"},
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		f, err := w.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestGaeaPptxSlideText 正常路径：两页段落全文 + 裸文件名回退 + 长段不截断。
func TestGaeaPptxSlideText(t *testing.T) {
	t.Chdir(t.TempDir())
	// 落 exports/（resolvePreviewPath 的常见输出目录），用裸文件名调用顺带
	// 覆盖「裸文件名回退」口径（与 GaeaPptxOutline 同款路径解析）。
	rel := filepath.Join("exports", "deck.pptx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("长", 300)
	if err := os.WriteFile(rel, buildPptxSlideTextFixture(t, long), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	got, err := a.GaeaPptxSlideText("deck.pptx")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("应返回两页，got %d", len(got))
	}
	if got[0].Index != 1 || got[1].Index != 2 {
		t.Fatalf("页码应 1-based 连续: %+v", got)
	}
	// 第 1 页（sldIdLst 首个 sldId → slide2.xml）段落全文、粒度=段落
	if p1 := strings.Join(got[0].Paragraphs, "\n"); p1 != "首页标题段落\n首页正文段落：成本下降 3%" {
		t.Errorf("第 1 页段落全文不符: %q", p1)
	}
	// 第 2 页：300 rune 长段必须全文返回（大纲侧 200 rune 截断预览，此处不截）
	if len(got[1].Paragraphs) != 2 {
		t.Fatalf("第 2 页应有 2 段: %+v", got[1].Paragraphs)
	}
	if got[1].Paragraphs[0] != "次页标题段落" {
		t.Errorf("第 2 页首段不符: %q", got[1].Paragraphs[0])
	}
	if got[1].Paragraphs[1] != long {
		t.Errorf("长段应全文返回（%d rune），得 %d rune", len([]rune(long)), len([]rune(got[1].Paragraphs[1])))
	}
}

// TestGaeaPptxSlideText_NotPptx 非 .pptx 扩展名 → 结构化报错（含另存提示）。
func TestGaeaPptxSlideText_NotPptx(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("readme.txt", []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	_, err := a.GaeaPptxSlideText("readme.txt")
	if err == nil || !strings.Contains(err.Error(), "仅支持 .pptx 文本提取") {
		t.Fatalf("应报仅支持 .pptx 文本提取: %v", err)
	}
	if !strings.Contains(err.Error(), ".ppt 旧格式请先另存为 .pptx") {
		t.Errorf("应含旧格式提示: %v", err)
	}
}

// TestGaeaPptxSlideText_Missing 文件不存在 → 报错不 panic。
func TestGaeaPptxSlideText_Missing(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	_, err := a.GaeaPptxSlideText("nope/deck.pptx")
	if err == nil || !strings.Contains(err.Error(), "文件不存在") {
		t.Fatalf("应报文件不存在: %v", err)
	}
}
