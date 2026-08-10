package fileindex

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanAndExtract(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "docs"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, ".gaea"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "docs", "说明.md"), []byte("振动锤选型要点"), 0o644)
	_ = os.WriteFile(filepath.Join(root, ".gaea", "内部.md"), []byte("内部状态"), 0o644)
	big := make([]byte, MaxFileBytes+1)
	_ = os.WriteFile(filepath.Join(root, "超大.txt"), big, 0o644)
	_ = os.WriteFile(filepath.Join(root, "图片.png"), []byte("binary"), 0o644)

	docs, skipped, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].Path != "docs/说明.md" {
		t.Fatalf("docs = %+v, want only docs/说明.md", docs)
	}
	if skipped < 1 { // 超大.txt（>2MB）应计入跳过；.gaea 目录整体跳过、png 不支持不计
		t.Errorf("skipped = %d, want >=1", skipped)
	}
}

func TestExtractCaps(t *testing.T) {
	long := strings.Repeat("内容", 30000)
	got := capRunes(long, MaxIndexChars)
	if len([]rune(got)) > MaxIndexChars {
		t.Errorf("cap failed: %d runes", len([]rune(got)))
	}
}

func TestExtractPPTX(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "汇报.pptx")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("ppt/slides/slide1.xml")
	_, _ = w.Write([]byte(`<?xml version="1.0"?><p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:sp><a:t>振动锤选型汇报</a:t><a:t>HP300 高频液压振动锤</a:t></p:sp></p:sld>`))
	_ = zw.Close()
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if !Supported(p) {
		t.Fatal("pptx should be supported")
	}
	text, err := Extract(p)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if !strings.Contains(text, "振动锤选型汇报") || !strings.Contains(text, "HP300") {
		t.Errorf("pptx text = %q", text)
	}
}
