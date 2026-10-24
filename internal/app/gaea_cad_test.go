// gaea_cad_test.go — CAD 绑定面单测：路径护栏（pickedFiles 白名单）、
// 打开/分幅/文本三绑定的 JSON 口径。缓存根注入 TempDir 防污染工作目录；
// 渲染走最小 DXF 夹具（真实图纸渲染验收在 PoC 沙箱/真机走查）。
package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cad"
)

// cadMiniDXF 与 cad 包单测同构的最小夹具（LINE + 中文转义 TEXT）。
const cadMiniDXF = "0\nSECTION\n2\nENTITIES\n" +
	"0\nLINE\n8\n0\n10\n0.0\n20\n0.0\n11\n100.0\n21\n50.0\n" +
	"0\nTEXT\n8\n0\n10\n5.0\n20\n5.0\n40\n2.5\n1\n\\U+4e00\\U+5c42 JS-01\n" +
	"0\nENDSEC\n0\nEOF\n"

func cadFixture(t *testing.T) string {
	t.Helper()
	orig := cadCacheRoot
	cache := filepath.Join(t.TempDir(), "cadcache")
	cadCacheRoot = func() string { return cache }
	t.Cleanup(func() { cadCacheRoot = orig })
	p := filepath.Join(t.TempDir(), "mini.dxf")
	if err := os.WriteFile(p, []byte(cadMiniDXF), 0o644); err != nil {
		t.Fatal(err)
	}
	registerPickedFile(p)
	return p
}

// TestGaeaCadOpenMiniDXF 打开口径：JSON 摘要字段 + 分幅 PNG 落盘 + 转义解码。
func TestGaeaCadOpenMiniDXF(t *testing.T) {
	p := cadFixture(t)
	a := &App{}
	raw, err := a.GaeaCadOpen(p)
	if err != nil {
		t.Fatalf("GaeaCadOpen: %v", err)
	}
	var out struct {
		Kind        string   `json:"kind"`
		EntityCount int      `json:"entityCount"`
		TextCount   int      `json:"textCount"`
		Layers      []string `json:"layers"`
		Sheets      []struct {
			Name string `json:"name"`
			PNG  string `json:"png"`
		} `json:"sheets"`
		Warnings []string `json:"warnings"`
		File     struct {
			Name string `json:"name"`
		} `json:"file"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("JSON 解析: %v\n%s", err, raw)
	}
	if out.Kind != cad.KindDXF || out.EntityCount < 2 || out.TextCount < 1 {
		t.Errorf("摘要口径异常: %+v", out)
	}
	if out.File.Name != "mini.dxf" {
		t.Errorf("file.name = %q", out.File.Name)
	}
	if len(out.Sheets) == 0 || out.Sheets[0].PNG == "" {
		t.Fatalf("图框分幅缺失: %s", raw)
	}
	if _, err := os.Stat(out.Sheets[0].PNG); err != nil {
		t.Errorf("分幅 PNG 未落盘: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(filepath.Dir(out.Sheets[0].PNG)))
	if err != nil || len(entries) != 1 {
		t.Fatalf("CAD cache entries = %v err=%v, want one content-key dir", entries, err)
	}
	metaPath := filepath.Join(cadCacheRoot(), entries[0].Name(), "meta.json")
	metaRaw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("cache metadata missing: %v", err)
	}
	var meta cadCacheMeta
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		t.Fatalf("cache metadata invalid: %v", err)
	}
	if meta.Name != "mini.dxf" || !filepath.IsAbs(meta.Source) {
		t.Errorf("cache metadata = %+v", meta)
	}
}

// TestGaeaCadRasterAndTexts 单框取图（含重渲）与文本清单口径。
func TestGaeaCadRasterAndTexts(t *testing.T) {
	p := cadFixture(t)
	a := &App{}
	if _, err := a.GaeaCadOpen(p); err != nil {
		t.Fatalf("Open: %v", err)
	}
	raw, err := a.GaeaCadRaster(p, 1, 0)
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	var r struct {
		PNGPath string `json:"pngPath"`
		DataURL string `json:"dataUrl"`
	}
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.DataURL, "data:image/png;base64,") {
		t.Errorf("dataUrl 前缀异常: %.40s", r.DataURL)
	}
	if _, err := a.GaeaCadRaster(p, 1, 4500); err != nil { // 加宽重渲
		t.Errorf("Raster(width=4500): %v", err)
	}
	highRaw, err := a.GaeaCadRaster(p, 1, 999999)
	if err != nil {
		t.Fatalf("Raster(width=999999): %v", err)
	}
	if !strings.Contains(highRaw, "sheet-1-w8000.png") {
		t.Errorf("render width was not clamped to 8000: %s", highRaw)
	}
	// 越界序号报错不 panic。
	if _, err := a.GaeaCadRaster(p, 99, 0); err == nil {
		t.Error("越界图框应报错")
	}

	traw, err := a.GaeaCadTexts(p)
	if err != nil {
		t.Fatalf("Texts: %v", err)
	}
	var tout struct {
		Total int `json:"total"`
		Texts []struct {
			Text string `json:"text"`
		} `json:"texts"`
	}
	if err := json.Unmarshal([]byte(traw), &tout); err != nil {
		t.Fatal(err)
	}
	if tout.Total < 1 || tout.Texts[len(tout.Texts)-1].Text != "一层 JS-01" {
		t.Errorf("文本清单/转义解码异常: %s", traw)
	}
}

// TestGaeaCadGuards 护栏：非白名单路径拒绝、未知扩展名拒绝、PDF 文本清单
// 明确报错口径。
func TestGaeaCadGuards(t *testing.T) {
	a := &App{}
	stranger := filepath.Join(t.TempDir(), "陌生.dxf")
	if err := os.WriteFile(stranger, []byte(cadMiniDXF), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaCadOpen(stranger); err == nil || !strings.Contains(err.Error(), "仅允许") {
		t.Errorf("非白名单路径应拒绝: %v", err)
	}
	p := cadFixture(t)
	if _, err := a.GaeaCadOpen(p + ".exe"); err == nil {
		t.Error("未知扩展名应拒绝")
	}
	// pickedFiles 挡 PDF 文本清单（换扩展名注册白名单后走到类型分支）。
	pdf := filepath.Join(t.TempDir(), "a.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	registerPickedFile(pdf)
	if _, err := a.GaeaCadTexts(pdf); err == nil || !strings.Contains(err.Error(), "问图纸") {
		t.Errorf("PDF 文本清单应引导问图纸: %v", err)
	}
}
