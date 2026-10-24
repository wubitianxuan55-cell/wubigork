// cad_test.go — 内核纯函数单测：\U+ 转义解码 golden、最小 DXF 夹具的
// 解析/文本/摘要冒烟。渲染路径依赖字体与图框启发式，走 PoC 沙箱与真机
// 走查验收（docs/gaea-cad-poc-report-2026-10.md），不在单测断像素。
package cad

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gocad "github.com/unitedrhino/go-cad"
)

// miniDXF 最小可解析 DXF（R12 式：LTYPE/LAYER 表 + ENTITIES 段的 LINE 与
// 中文转义 TEXT；与 PoC 沙箱验证过的 mini2.dxf 同构——无层表时 go-cad 的
// SheetText.Layer 解析为空串，图层名断言需要层表在位）。
const miniDXF = "0\nSECTION\n2\nTABLES\n" +
	"0\nTABLE\n2\nLTYPE\n0\nLTYPE\n2\nCONTINUOUS\n70\n64\n3\nSolid line\n0\nENDTAB\n" +
	"0\nTABLE\n2\nLAYER\n0\nLAYER\n2\n\\U+56fe\\U+6846\n70\n0\n62\n7\n6\nCONTINUOUS\n0\nENDTAB\n" +
	"0\nENDSEC\n" +
	"0\nSECTION\n2\nENTITIES\n" +
	"0\nLINE\n8\n\\U+56fe\\U+6846\n10\n0.0\n20\n0.0\n11\n100.0\n21\n50.0\n" +
	"0\nTEXT\n8\n\\U+56fe\\U+6846\n10\n5.0\n20\n5.0\n40\n2.5\n1\n\\U+4e00\\U+5c42\\U+5e73\\U+9762\\U+56fe JS-01\n" +
	"0\nENDSEC\n0\nEOF\n"

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestDecodeUnicodeEscape 转义解码 golden：中文、无转义快路径、截断/非法
// 序列原样保留、孤立代理替换。
func TestDecodeUnicodeEscape(t *testing.T) {
	cases := []struct{ in, want string }{
		{`\U+4e00\U+5c42\U+5e73\U+9762\U+56fe`, "一层平面图"},
		{`JS-\U+0031\U+0032`, "JS-12"},
		{`无转义原文`, `无转义原文`},
		{`\U+56fe JS-01`, "图 JS-01"},       // 转义与明文混排
		{`\U+12`, `\U+12`},                 // 位数不足：原样
		{`\U+ZZZZ`, `\U+ZZZZ`},             // 非法十六进制：原样
		{`\U+D83D\uDE00`, "\uFFFD\\uDE00"}, // 孤立高代理：替换符占位，后继非 \U+ 序列原样
	}
	for _, c := range cases {
		if got := DecodeUnicodeEscape(c.in); got != c.want {
			t.Errorf("Decode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestOpenMiniDXF 最小夹具冒烟：判型/实体计数/文本提取与转义自修/图框兜底。
func TestOpenMiniDXF(t *testing.T) {
	p := writeTemp(t, "mini.dxf", miniDXF)
	if got := Sniff(p); got != KindDXF {
		t.Fatalf("Sniff = %q, want dxf", got)
	}
	d, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if d.Kind != KindDXF || d.entityCount < 2 {
		t.Errorf("Kind=%q entityCount=%d", d.Kind, d.entityCount)
	}
	texts, _ := d.Texts(0)
	if len(texts) == 0 || texts[len(texts)-1].Text != "一层平面图 JS-01" {
		t.Errorf("文本转义解码失败: %+v", texts)
	}
	if len(d.layers) == 0 || d.layers[0] != "图框" {
		t.Errorf("图层名转义解码失败: %v", d.layers)
	}
	s := d.Summary()
	if s.TextCount != len(texts) {
		t.Errorf("Summary.TextCount=%d want %d", s.TextCount, len(texts))
	}
}

// TestOpenMissingFile 错误口径：读取失败带上下文、不 panic。
func TestOpenMissingFile(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "不存在.dwg")); err == nil {
		t.Fatal("期望报错")
	}
}

// TestSniffUnknown 未知扩展名返回空串（绑定层据此拒绝）。
func TestSniffUnknown(t *testing.T) {
	for _, name := range []string{"a.txt", "b", "c.dwg.bak"} {
		if got := Sniff(name); got != "" {
			t.Errorf("Sniff(%q) = %q, want 空", name, got)
		}
	}
}

// countDXF 最小块统计夹具：SYM1 块（内含一线）+ 两个 INSERT 各带同排邻近
// 文字「暗装插座」（众数命名）+ 模型空间一条 LINE（层线长）。
const countDXF = "0\nSECTION\n2\nTABLES\n" +
	"0\nTABLE\n2\nLTYPE\n0\nLTYPE\n2\nCONTINUOUS\n70\n64\n3\nSolid line\n0\nENDTAB\n" +
	"0\nTABLE\n2\nLAYER\n0\nLAYER\n2\n\\U+56fe\\U+6846\n70\n0\n62\n7\n6\nCONTINUOUS\n0\nENDTAB\n" +
	"0\nENDSEC\n" +
	"0\nSECTION\n2\nBLOCKS\n" +
	"0\nBLOCK\n2\nSYM1\n70\n0\n10\n0.0\n20\n0.0\n3\nSYM1\n1\n\n" +
	"0\nLINE\n8\n0\n10\n0.0\n20\n0.0\n11\n10.0\n21\n0.0\n" +
	"0\nENDBLK\n0\nENDSEC\n" +
	"0\nSECTION\n2\nENTITIES\n" +
	"0\nLINE\n8\n\\U+56fe\\U+6846\n10\n0.0\n20\n0.0\n11\n100.0\n21\n0.0\n" +
	"0\nINSERT\n8\n\\U+56fe\\U+6846\n2\nSYM1\n10\n100.0\n20\n100.0\n" +
	"0\nTEXT\n8\n\\U+56fe\\U+6846\n10\n105.0\n20\n100.0\n40\n2.5\n1\n\\U+6697\\U+88c5\\U+63d2\\U+5ea7\n" +
	"0\nINSERT\n8\n\\U+56fe\\U+6846\n2\nSYM1\n10\n200.0\n20\n100.0\n" +
	"0\nTEXT\n8\n\\U+56fe\\U+6846\n10\n205.0\n20\n100.0\n40\n2.5\n1\n\\U+6697\\U+88c5\\U+63d2\\U+5ea7\n" +
	"0\nENDSEC\n0\nEOF\n"

// TestCountBlocksAndNaming 通用符号统计：块计数精确 + 邻近文字众数自动命名 +
// 模型空间层线长（块内线不计入——与内容密度判定同口径）。
func TestCountBlocksAndNaming(t *testing.T) {
	p := writeTemp(t, "count.dxf", countDXF)
	d, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	res := d.Count()
	if len(res.Items) != 1 {
		t.Fatalf("符号统计应恰 1 种块，实际 %d：%+v", len(res.Items), res.Items)
	}
	it := res.Items[0]
	if it.Block != "SYM1" || it.Count != 2 {
		t.Errorf("块名/计数异常: %+v", it)
	}
	if it.Name != "暗装插座" || it.Confidence != 1 {
		t.Errorf("众数命名异常: name=%q conf=%v", it.Name, it.Confidence)
	}
	if len(res.Layers) == 0 || res.Layers[0].Layer != "图框" {
		t.Fatalf("层线长图层名异常: %+v", res.Layers)
	}
	if math.Abs(res.Layers[0].Length-100) > 0.01 {
		t.Errorf("层线长应 100（模型空间 LINE），实际 %v", res.Layers[0].Length)
	}
}

func TestValidateInputRejectsOversizeAndDirectories(t *testing.T) {
	dxf := filepath.Join(t.TempDir(), "huge.dxf")
	f, err := os.Create(dxf)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxDWGOrDXFBytes + 1); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateInput(dxf); err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Fatalf("oversize DXF = %v, want 32 MiB limit", err)
	}
	pdf := filepath.Join(t.TempDir(), "huge.pdf")
	f, err = os.Create(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxPDFBytes + 1); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	_ = f.Close()
	if _, _, err := ValidateInput(pdf); err == nil || !strings.Contains(err.Error(), "128 MiB") {
		t.Fatalf("oversize PDF = %v, want 128 MiB limit", err)
	}
	dir := filepath.Join(t.TempDir(), "folder.dxf")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateInput(dir); err == nil || !strings.Contains(err.Error(), "普通文件") {
		t.Fatalf("directory input = %v, want regular-file rejection", err)
	}
}

func TestRenderWidthIsClamped(t *testing.T) {
	d, err := Open(writeTemp(t, "clamp.dxf", miniDXF))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	low, err := d.RenderSheet(0, dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(low) != "sheet-1-w64.png" {
		t.Errorf("low width path = %q, want w64", low)
	}
	high, err := d.RenderSheet(0, dir, 999999)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(high) != "sheet-1-w8000.png" {
		t.Errorf("high width path = %q, want w8000", high)
	}
}

func TestRenderSheetsRejectsFrameCountOverLimit(t *testing.T) {
	d := &Drawing{sheets: make([]gocad.Sheet, MaxSheets+1)}
	if _, err := d.RenderSheets(t.TempDir(), 0); err == nil || !strings.Contains(err.Error(), "上限") {
		t.Fatalf("RenderSheets over limit = %v, want frame-count rejection", err)
	}
}

func TestCorruptInputsReturnErrorsWithoutPanic(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"bad.dxf", "bad.dwg"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("not a drawing"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(path); err == nil {
			t.Errorf("%s should fail parsing", name)
		}
	}
	pdf := filepath.Join(dir, "bad.pdf")
	if err := os.WriteFile(pdf, []byte("not a pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenPDF(pdf, filepath.Join(dir, "cache"), 20, 5); err == nil {
		t.Error("corrupt PDF should fail without panic")
	}
}

func buildPDFPages(t *testing.T, pages int) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, pages+3)
	offsets[1] = sb.Len()
	sb.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	kids := make([]string, pages)
	for i := range kids {
		kids[i] = fmt.Sprintf("%d 0 R", i+3)
	}
	offsets[2] = sb.Len()
	fmt.Fprintf(&sb, "2 0 obj\n<< /Type /Pages /Kids [%s] /Count %d >>\nendobj\n", strings.Join(kids, " "), pages)
	for i := 0; i < pages; i++ {
		obj := i + 3
		offsets[obj] = sb.Len()
		fmt.Fprintf(&sb, "%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n", obj)
	}
	xref := sb.Len()
	fmt.Fprintf(&sb, "xref\n0 %d\n0000000000 65535 f \n", pages+3)
	for i := 1; i < pages+3; i++ {
		fmt.Fprintf(&sb, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&sb, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", pages+3, xref)
	path := filepath.Join(t.TempDir(), "pages.pdf")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenPDFRendersOnlyFiftyPages(t *testing.T) {
	path := buildPDFPages(t, 51)
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, "page-51.png"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := OpenPDF(path, cache, 20, 50)
	if err != nil {
		if strings.Contains(err.Error(), "pdftoppm") {
			t.Skipf("pdftoppm unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if len(res.Sheets) != 50 {
		t.Fatalf("rendered sheets = %d, want 50", len(res.Sheets))
	}
	if _, err := os.Stat(filepath.Join(cache, "sheet-51.png")); !os.IsNotExist(err) {
		t.Fatalf("page 51 must not be rendered, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(cache, "page-51.png")); !os.IsNotExist(err) {
		t.Fatalf("stale page 51 intermediate must be removed, stat err=%v", err)
	}
	if len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "前 50 页") {
		t.Fatalf("missing page-range warning: %+v", res.Warnings)
	}
}

func TestOpenPDFContextCancellation(t *testing.T) {
	path := buildPDFPages(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenPDFContext(ctx, path, t.TempDir(), 20, 50); err == nil {
		t.Fatal("cancelled context should abort PDF rendering")
	}
}
