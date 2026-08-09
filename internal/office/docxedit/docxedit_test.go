package docxedit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildDocx 构造一个最小 docx，document.xml 使用给定 body 片段。
func buildDocx(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "t.docx")
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>` + body + `
  </w:body>
</w:document>`
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	entries := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": docXML,
	}
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readDocXML(t *testing.T, path string) string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	t.Fatal("缺少 word/document.xml")
	return ""
}

// assertWellFormed 验证 XML 可被解析（修订结构合法）。
func assertWellFormed(t *testing.T, xmlStr string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(xmlStr))
	for {
		_, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			t.Fatalf("XML 不合法: %v", err)
		}
	}
}

func TestApplyTrackedReplace_SingleRunPartial(t *testing.T) {
	path := buildDocx(t, `
    <w:p><w:r><w:t>合同期限为 30 天。</w:t></w:r></w:p>`)
	if err := ApplyTrackedReplace(path, "30 天", "60 天", "测试员"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	for _, want := range []string{
		`<w:del w:id="1" w:author="测试员"`,
		`<w:delText>30 天</w:delText>`,
		`<w:ins w:id="2" w:author="测试员"`,
		`<w:t>60 天</w:t>`,
		`>合同期限为 </w:t>`,
		`>。</w:t>`,
	} {
		if !strings.Contains(xmlStr, want) {
			t.Errorf("缺少 %q", want)
		}
	}
	// w:del/w:ins 必须是 w:r 的兄弟，不能嵌套在 run 内。
	if strings.Contains(xmlStr, "<w:r><w:t>30 天</w:t><w:del") {
		t.Error("w:del 被错误嵌套在 run 内")
	}
}

func TestApplyTrackedReplace_AcrossRuns(t *testing.T) {
	// "Hel" + "lo world" 两个 run 拼成 "Hello world"，替换 "lo world" 跨 run 边界。
	path := buildDocx(t, `
    <w:p>
      <w:r><w:rPr><w:b/></w:rPr><w:t>Hel</w:t></w:r>
      <w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">lo world</w:t></w:r>
    </w:p>`)
	if err := ApplyTrackedReplace(path, "lo world", "lo gaea", "测试员"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if !strings.Contains(xmlStr, `>lo world</w:delText>`) {
		t.Error("缺少删除的跨 run 文本")
	}
	if !strings.Contains(xmlStr, `>lo gaea</w:t>`) {
		t.Error("缺少插入文本")
	}
	// 删除前的 run 应保留
	if !strings.Contains(xmlStr, `<w:t>Hel</w:t>`) {
		t.Error("选区前文本丢失")
	}
	// 插入 run 应复用原 run 的 rPr（加粗）
	if !strings.Contains(xmlStr, `<w:ins `) || !strings.Contains(xmlStr, `<w:b/>`) {
		t.Error("插入未保留格式")
	}
}

func TestApplyTrackedReplace_ReplacementEscaping(t *testing.T) {
	path := buildDocx(t, `
    <w:p><w:r><w:t>价格 &lt; 100</w:t></w:r></w:p>`)
	if err := ApplyTrackedReplace(path, "价格 < 100", "价格 < 200 & 含税", "测试员"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if !strings.Contains(xmlStr, `<w:t>价格 &lt; 200 &amp; 含税</w:t>`) {
		t.Errorf("替换文本未正确转义: %s", xmlStr)
	}
}

func TestApplyTrackedReplace_WhitespaceFallback(t *testing.T) {
	// 渲染折叠空白：原文 "a  b"（两个空格），用户选中渲染后的 "a b"。
	path := buildDocx(t, `
    <w:p><w:r><w:t xml:space="preserve">a  b</w:t></w:r></w:p>`)
	if err := ApplyTrackedReplace(path, "a b", "c d", "测试员"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if !strings.Contains(xmlStr, `<w:delText`) || !strings.Contains(xmlStr, `>c d</w:t>`) {
		t.Errorf("空白兜底替换失败: %s", xmlStr)
	}
}

func TestApplyTrackedReplace_NotFound(t *testing.T) {
	path := buildDocx(t, `
    <w:p><w:r><w:t>原始文本</w:t></w:r></w:p>`)
	err := ApplyTrackedReplace(path, "不存在的文本", "替换", "测试员")
	if err == nil || !strings.Contains(err.Error(), "定位") {
		t.Fatalf("期望未命中错误，得到 %v", err)
	}
}

func TestApplyTrackedReplace_SpecialElementRejected(t *testing.T) {
	// 选区覆盖图片（drawing）时拒绝，避免破坏版式。
	path := buildDocx(t, `
    <w:p>
      <w:r><w:t>带图</w:t></w:r>
      <w:r><w:drawing><w:inline/></w:drawing></w:r>
      <w:r><w:t>片段落</w:t></w:r>
    </w:p>`)
	err := ApplyTrackedReplace(path, "带图片段落", "替换", "测试员")
	if err == nil || !strings.Contains(err.Error(), "特殊格式") {
		t.Fatalf("期望特殊格式错误，得到 %v", err)
	}
}

func TestApplyTrackedReplace_KeepsOtherBytes(t *testing.T) {
	path := buildDocx(t, `
    <w:p><w:r><w:t>第一段</w:t></w:r></w:p>
    <w:p><w:r><w:t>目标段</w:t></w:r></w:p>
    <w:p><w:r><w:t>第三段</w:t></w:r></w:p>`)
	if err := ApplyTrackedReplace(path, "目标段", "修改后", "测试员"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	if !strings.Contains(xmlStr, `<w:t>第一段</w:t>`) || !strings.Contains(xmlStr, `<w:t>第三段</w:t>`) {
		t.Error("非目标段落被改动")
	}
}

func TestApplyTrackedReplace_PrefersNonHyperlink(t *testing.T) {
	// 目标文本同时出现在目录超链接与正文时，优先修订正文（docx-preview
	// 不渲染 hyperlink 内的 ins/del，正文修订才能所见即所得）。
	path := buildDocx(t, `
    <w:p><w:hyperlink w:history="true" w:anchor="bookmark1"><w:r><w:t>目录目标</w:t></w:r></w:hyperlink></w:p>
    <w:p><w:r><w:t>正文目标段</w:t></w:r></w:p>`)
	if err := ApplyTrackedReplace(path, "目标", "目标X", "测试员"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if !strings.Contains(xmlStr, `>目标X</w:t>`) || !strings.Contains(xmlStr, `>正文</w:t>`) || !strings.Contains(xmlStr, `>段</w:t>`) {
		t.Error("正文段落未应用修订")
	}
	if strings.Contains(xmlStr, `<w:hyperlink`) && strings.Contains(xmlStr, `<w:del `) {
		// 超链接段落应保持原样：目录目标 不应被删除包裹
		if strings.Contains(xmlStr, `<w:delText>目录目标</w:delText>`) {
			t.Error("超链接（目录）内的文本被修订，应优先修订正文")
		}
	}
}

func TestApplyTrackedReplace_InTableCell(t *testing.T) {
	// 合同常见场景：修订落在表格单元格内（w:tbl > w:tr > w:tc > w:p）。
	path := buildDocx(t, `
    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>付款期限</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>30 天内</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>`)
	if err := ApplyTrackedReplace(path, "30 天内", "60 天内", "gaea AI"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if !strings.Contains(xmlStr, `<w:delText>30 天内</w:delText>`) || !strings.Contains(xmlStr, `>60 天内</w:t>`) {
		t.Errorf("表格单元格内修订失败: %s", xmlStr)
	}
	// 接受后表格内新文落地
	if err := AcceptChanges(path, "gaea AI"); err != nil {
		t.Fatal(err)
	}
	xmlStr = readDocXML(t, path)
	if strings.Contains(xmlStr, "<w:del ") || !strings.Contains(xmlStr, `>60 天内</w:t>`) {
		t.Errorf("表格内接受修订失败: %s", xmlStr)
	}
}

// TestSmokeRealDocx 用真实 Word 文档做端到端冒烟（默认跳过）：
//   GAEA_SMOKE_DOCX=<真实 docx 路径> go test ./internal/office/docxedit -run TestSmokeRealDocx -v
func TestSmokeRealDocx(t *testing.T) {
	path := os.Getenv("GAEA_SMOKE_DOCX")
	if path == "" {
		t.Skip("未设置 GAEA_SMOKE_DOCX，跳过真实文档冒烟")
	}
	if err := ApplyTrackedReplace(path, "申报类型", "申报类型（冒烟测试修订）", "gaea AI"); err != nil {
		t.Fatalf("真实文档修订失败: %v", err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if !strings.Contains(xmlStr, "<w:del ") || !strings.Contains(xmlStr, "<w:ins ") {
		t.Error("真实文档未写入修订标记")
	}
	if !strings.Contains(xmlStr, ">申报类型（冒烟测试修订）</w:t>") {
		t.Error("插入内容缺失")
	}
	// 接受全部 gaea 修订：新文落地为正文、修订标记清空
	if err := AcceptChanges(path, "gaea AI"); err != nil {
		t.Fatalf("真实文档接受修订失败: %v", err)
	}
	xmlStr = readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if strings.Contains(xmlStr, `w:author="gaea AI"`) {
		t.Error("接受后仍存在 gaea 修订标记")
	}
	if !strings.Contains(xmlStr, ">申报类型（冒烟测试修订）</w:t>") {
		t.Error("接受后插入内容未落地")
	}
}

func TestAcceptChanges(t *testing.T) {
	path := buildDocx(t, `
    <w:p><w:r><w:t>合同期限为 </w:t></w:r><w:del w:id="1" w:author="gaea AI" w:date="2026-08-09T00:00:00Z"><w:r><w:delText>30 天</w:delText></w:r></w:del><w:ins w:id="2" w:author="gaea AI" w:date="2026-08-09T00:00:00Z"><w:r><w:t>60 天</w:t></w:r></w:ins><w:r><w:t>。</w:t></w:r></w:p>`)
	if err := AcceptChanges(path, "gaea AI"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if strings.Contains(xmlStr, "<w:del ") || strings.Contains(xmlStr, "<w:ins ") {
		t.Error("接受后仍存在修订标记")
	}
	if !strings.Contains(xmlStr, ">60 天</w:t>") {
		t.Error("接受后新文未生效")
	}
	if strings.Contains(xmlStr, "30 天") {
		t.Error("接受后原文未移除")
	}
}

func TestRejectChanges(t *testing.T) {
	path := buildDocx(t, `
    <w:p><w:r><w:t>合同期限为 </w:t></w:r><w:del w:id="1" w:author="gaea AI" w:date="2026-08-09T00:00:00Z"><w:r><w:rPr><w:b/></w:rPr><w:delText>30 天</w:delText></w:r></w:del><w:ins w:id="2" w:author="gaea AI" w:date="2026-08-09T00:00:00Z"><w:r><w:t>60 天</w:t></w:r></w:ins><w:r><w:t>。</w:t></w:r></w:p>`)
	if err := RejectChanges(path, "gaea AI"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	assertWellFormed(t, xmlStr)
	if strings.Contains(xmlStr, "<w:del ") || strings.Contains(xmlStr, "<w:ins ") {
		t.Error("拒绝后仍存在修订标记")
	}
	if !strings.Contains(xmlStr, ">30 天</w:t>") {
		t.Error("拒绝后原文未恢复")
	}
	if strings.Contains(xmlStr, "60 天") {
		t.Error("拒绝后新文未移除")
	}
	// 还原的 run 保留 rPr（加粗）
	if !strings.Contains(xmlStr, `<w:rPr><w:b/></w:rPr><w:t>30 天</w:t>`) {
		t.Error("还原内容未保留格式")
	}
}

func TestFlattenChanges_NoChanges(t *testing.T) {
	path := buildDocx(t, `
    <w:p><w:r><w:t>没有修订</w:t></w:r></w:p>`)
	if err := AcceptChanges(path, "gaea AI"); err == nil || !strings.Contains(err.Error(), "没有") {
		t.Fatalf("期望无修订错误，得到 %v", err)
	}
}

func TestFlattenChanges_OnlyOwnAuthor(t *testing.T) {
	path := buildDocx(t, `
    <w:p><w:r><w:t>原</w:t></w:r><w:del w:id="1" w:author="别人" w:date="2026-08-09T00:00:00Z"><w:r><w:delText>甲</w:delText></w:r></w:del><w:ins w:id="2" w:author="gaea AI" w:date="2026-08-09T00:00:00Z"><w:r><w:t>乙</w:t></w:r></w:ins></w:p>`)
	if err := AcceptChanges(path, "gaea AI"); err != nil {
		t.Fatal(err)
	}
	xmlStr := readDocXML(t, path)
	// gaea 的 ins 应被保留，别人的 del 原样不动
	if !strings.Contains(xmlStr, "<w:del ") || !strings.Contains(xmlStr, `w:author="别人"`) {
		t.Error("他人的修订不应被触碰")
	}
	if strings.Contains(xmlStr, `w:author="gaea AI"`) {
		t.Error("gaea 的修订应被扁平化")
	}
	if !strings.Contains(xmlStr, "<w:t>乙</w:t>") {
		t.Error("接受后 gaea 新文未生效")
	}
}
