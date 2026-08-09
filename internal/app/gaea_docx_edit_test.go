package app

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// docxWithText 构造一个含指定段落文本的最小 docx。
func docxWithText(t *testing.T, text string) []byte {
	t.Helper()
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>` + text + `</w:t></w:r></w:p>
  </w:body>
</w:document>`
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
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
	for name, content := range files {
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
	return buf.Bytes()
}

func TestGaeaDocxApplyEdit(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "edit-test.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, docxWithText(t, "合同期限为 30 天。"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	got, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "30 天", "60 天")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "docx" {
		t.Fatalf("kind = %q, want docx", got.Kind)
	}
	if !strings.HasPrefix(got.DataURL, "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,") {
		t.Error("预览 dataUrl 缺失")
	}

	// 落盘文件应包含修订标记
	r, err := zip.OpenReader(rel)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var docXML []byte
	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		docXML, _ = io.ReadAll(rc)
		rc.Close()
	}
	s := string(docXML)
	if !strings.Contains(s, "<w:del ") || !strings.Contains(s, "<w:ins ") {
		t.Errorf("docx 未写入修订标记: %s", s)
	}
	if !strings.Contains(s, "<w:delText>30 天</w:delText>") || !strings.Contains(s, ">60 天</w:t>") {
		t.Errorf("修订内容不正确: %s", s)
	}
}

func TestGaeaDocxApplyEdit_NotFound(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "edit-miss.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, docxWithText(t, "原始文本"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	if _, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "找不到", "替换"); err == nil {
		t.Fatal("期望未命中错误")
	}
}

func TestGaeaDocxAcceptChanges(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "accept-test.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, docxWithText(t, "合同期限为 30 天。"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	if _, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "30 天", "60 天"); err != nil {
		t.Fatal(err)
	}
	got, err := a.GaeaDocxAcceptChanges(filepath.ToSlash(rel), true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "docx" {
		t.Fatalf("kind = %q", got.Kind)
	}
	r, err := zip.OpenReader(rel)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var docXML []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			docXML, _ = io.ReadAll(rc)
			rc.Close()
		}
	}
	s := string(docXML)
	if strings.Contains(s, "<w:del ") || strings.Contains(s, "<w:ins ") {
		t.Error("接受后仍有修订标记")
	}
	if !strings.Contains(s, ">60 天</w:t>") {
		t.Error("接受后新文未生效")
	}
	// 无修订时再次接受应报错
	if _, err := a.GaeaDocxAcceptChanges(filepath.ToSlash(rel), true); err == nil {
		t.Fatal("无修订时接受应报错")
	}
}
