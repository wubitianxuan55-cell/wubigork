package pptxedit

// pptxedit 刀1 回归：夹具=手工构造的最小 pptx（zip 条目字节级可控）。
// 覆盖：slide 序映射、跨 run 命中替换（rPr 继承）、表格内命中、blocker 拒绝
//（a:br / a:fld）、未命中/页码越界、原子写回与未知条目字节零扰动。

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const presXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<p:sldIdLst>
<p:sldId id="256" r:id="rId2"/>
<p:sldId id="257" r:id="rId3"/>
</p:sldIdLst>
</p:presentation>`

const presRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://x/slideMaster" Target="slideMasters/slideMaster1.xml"/>
<Relationship Id="rId2" Type="http://x/slide" Target="slides/slide1.xml"/>
<Relationship Id="rId3" Type="http://x/slide" Target="slides/slide2.xml"/>
</Relationships>`

// slide1：跨 run 文本（预算总 + 额：120 万）、a:br blocker、a:fld blocker
const slide1XML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
<p:cSld><p:spTree>
<p:sp>
<p:txBody><a:bodyPr/><a:p>
<a:r><a:rPr lang="zh-CN" sz="4000" b="1"/><a:t>预算总额</a:t></a:r>
<a:r><a:rPr lang="zh-CN" sz="2400"/><a:t>：120 万元</a:t></a:r>
</a:p>
<a:p><a:r><a:rPr lang="zh-CN"/><a:t>第一行</a:t></a:r><a:br/><a:r><a:rPr lang="zh-CN"/><a:t>第二行</a:t></a:r></a:p>
<a:p><a:fld id="{GUID}" type="slidenum"><a:rPr lang="zh-CN"/><a:t>1</a:t></a:fld><a:r><a:rPr lang="zh-CN"/><a:t> 页</a:t></a:r></a:p>
</p:txBody></p:sp>
</p:spTree></p:cSld>
</p:sld>`

// slide2：表格内文本（解析器自然覆盖）+ 图片 run 混排
const slide2XML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
<p:cSld><p:spTree>
<p:graphicFrame><a:graphic><a:graphicData uri="x"><a:tbl>
<a:tr><a:tc><a:txBody><a:p><a:r><a:rPr lang="zh-CN" sz="1800"/><a:t>科目：人工费</a:t></a:r></a:p></a:txBody></a:tc></a:tr>
</a:tbl></a:graphicData></a:graphic></p:graphicFrame>
</p:spTree></p:cSld>
</p:sld>`

func buildPPTX(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "deck.pptx")
	entries := []struct{ name, body string }{
		{"[Content_Types].xml", "<Types/>"},
		{"ppt/presentation.xml", presXML},
		{"ppt/_rels/presentation.xml.rels", presRels},
		{"ppt/slides/slide1.xml", slide1XML},
		{"ppt/slides/slide2.xml", slide2XML},
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
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLocateTextSlidesInOrder(t *testing.T) {
	slides, err := LocateText(buildPPTX(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(slides) != 2 {
		t.Fatalf("应映射 2 页，got %d", len(slides))
	}
	if slides[0].Index != 1 || len(slides[0].Paragraphs) == 0 {
		t.Fatalf("第 1 页应有段落文本: %+v", slides[0])
	}
	joined := strings.Join(slides[0].Paragraphs, "\n")
	if !strings.Contains(joined, "预算总额：120 万元") {
		t.Errorf("跨 run 文本应拼接命中: %q", joined)
	}
	joined2 := strings.Join(slides[1].Paragraphs, "\n")
	if !strings.Contains(joined2, "科目：人工费") {
		t.Errorf("表格内文本应自然覆盖: %q", joined2)
	}
}

func TestApplyReplaceCrossRunInheritsRPr(t *testing.T) {
	path := buildPPTX(t)
	summary, err := ApplyTextReplace(path, 1, "预算总额：120 万元", "预算总额：180 万元")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "第 1 页") {
		t.Errorf("摘要口径: %q", summary)
	}
	doc, err := readPptx(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(doc.files["ppt/slides/slide1.xml"])
	if strings.Contains(raw, "120 万元") {
		t.Error("旧文本应已被替换")
	}
	// rPr 原字节继承：替换文本所在 run 应保留 sz="2400"（最后受影响 run 的 rPr）
	if !strings.Contains(raw, `sz="2400"/>`) {
		t.Error("最后受影响 run 的 rPr 原字节应保留")
	}
	// run1 被选区完全覆盖且非最后受影响 → 整体省略（sz=4000 消失是设计语义）
	if strings.Contains(raw, `sz="4000"`) {
		t.Error("完全覆盖的非最后 run 应整体省略")
	}
	// 未知条目字节零扰动
	if string(doc.files["ppt/theme/theme1.xml"]) != "<theme/>" {
		t.Error("主题等未知条目应字节零扰动")
	}
}

func TestApplyReplaceTableText(t *testing.T) {
	path := buildPPTX(t)
	if _, err := ApplyTextReplace(path, 2, "人工费", "材料费"); err != nil {
		t.Fatal(err)
	}
	slides, err := LocateText(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(slides[1].Paragraphs, "\n"), "科目：材料费") {
		t.Error("表格内替换应生效")
	}
}

func TestApplyRejectsBlockersAndMisses(t *testing.T) {
	path := buildPPTX(t)
	// a:br blocker：跨「第一行/第二行」的目标被拒
	if _, err := ApplyTextReplace(path, 1, "第一行第二行", "x"); err == nil || !strings.Contains(err.Error(), "特殊元素") {
		t.Errorf("跨 a:br 应拒绝，got %v", err)
	}
	// a:fld blocker：覆盖页码字段的选区被拒
	if _, err := ApplyTextReplace(path, 1, "1 页", "x"); err == nil || !strings.Contains(err.Error(), "特殊元素") {
		t.Errorf("覆盖 a:fld 应拒绝，got %v", err)
	}
	// 未命中
	if _, err := ApplyTextReplace(path, 1, "不存在的文本", "x"); err == nil {
		t.Error("未命中应报错")
	}
	// 页码越界
	if _, err := ApplyTextReplace(path, 9, "预算", "x"); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Errorf("页码越界应报错，got %v", err)
	}
}

func TestApplyOnlyTouchesTargetSlide(t *testing.T) {
	path := buildPPTX(t)
	if _, err := ApplyTextReplace(path, 1, "第一行", "首行"); err != nil {
		t.Fatal(err)
	}
	slides, err := LocateText(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(slides[0].Paragraphs, "\n"), "首行") {
		t.Error("第 1 页应有替换")
	}
	if strings.Contains(strings.Join(slides[1].Paragraphs, "\n"), "首行") {
		t.Error("第 2 页不应被扰动")
	}
}
