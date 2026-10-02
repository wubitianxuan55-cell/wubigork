package docmd

// pdf_pages_consistency_test.go — IN3-11：PDF 页数三套口径（字面计数 raw /
// 文本流视图 content / 页规格渲染边界 pageBounds / OCR 产物页码）的差异探针、
// 单点函数守卫与诚实告警断言。
//
// 结论口径（实测，见下）：
//   - 对外 total 仍是「原始字节字面计数」（pdfTotalPages 单点，行为不变）；
//   - 文本路径页码来自「文本流视图」的页对象序列（decodeFlateStreams +
//     stripNonTextStreams 之后），与 raw 计数可以不等 → pdfTotalPages 告警；
//   - OCR 路径页码以 pdftoppm **实际产物**文件名后缀为唯一来源
//     （renderedPageJobs），不再用 first+i 假设。

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// slogCapture 收集默认 logger 的记录（诚实告警断言用）。
type slogCapture struct {
	mu   sync.Mutex
	msgs []string
}

func (c *slogCapture) Enabled(context.Context, slog.Level) bool { return true }

func (c *slogCapture) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	c.msgs = append(c.msgs, r.Message)
	c.mu.Unlock()
	return nil
}

func (c *slogCapture) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *slogCapture) WithGroup(string) slog.Handler      { return c }

func (c *slogCapture) contains(sub string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range c.msgs {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}

func (c *slogCapture) dump() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.msgs, " | ")
}

// captureSlog 把默认 logger 换成收集器，返回 (收集器, 还原函数)。
func captureSlog(t *testing.T) (*slogCapture, func()) {
	t.Helper()
	old := slog.Default()
	cap := &slogCapture{}
	slog.SetDefault(slog.New(cap))
	return cap, func() { slog.SetDefault(old) }
}

// phantomPagePDF 探针①：3 个真页对象（5/7 号各带 BT..ET 文本）+ 1 个伪
// /Type /Page 藏在非文本二进制 stream（4 号对象）里。原始字节字面计数会把它
// 数成第 4 页；文本流视图（stripNonTextStreams 之后）只剩 3 个页对象。
const phantomPagePDF = `%PDF-1.4
1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj
2 0 obj << /Type /Pages /Kids [3 0 R 5 0 R 7 0 R] /Count 3 >> endobj
3 0 obj << /Type /Page /Parent 2 0 R /Contents 4 0 R >> endobj
4 0 obj << /Length 32 >>
stream
GARBAGE-BINARY /Type /Page XX
endstream
endobj
5 0 obj << /Type /Page /Parent 2 0 R /Contents 6 0 R >> endobj
6 0 obj << /Length 44 >>
stream
BT /F1 12 Tf 72 720 Td (page-two) Tj ET
endstream
endobj
7 0 obj << /Type /Page /Parent 2 0 R /Contents 8 0 R >> endobj
8 0 obj << /Length 44 >>
stream
BT /F1 12 Tf 72 720 Td (page-three) Tj ET
endstream
endobj
`

// buildFlateTextPDF 探针②：单页 PDF + 一个 FlateDecode 内容流，流内文本前置
// 一行字面 "/Type /Page"（压缩在 zlib 里 → raw 字面计数看不到，解压后的文本流
// 视图看得到 → content 计数 = raw + 1）。
func buildFlateTextPDF(t *testing.T, payload string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf(`%%PDF-1.4
1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj
2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj
3 0 obj << /Type /Page /Parent 2 0 R /Contents 4 0 R >> endobj
4 0 obj << /Length %d /Filter /FlateDecode >>
stream
%s
endstream
endobj
`, buf.Len(), buf.String())
}

// TestPDFPageCountViewsDiverge 探针①：raw 字面计数（4）≠ 文本流视图计数（3），
// 对外 total 保持 raw 口径（行为不变）并发出诚实告警；文本路径最多只能出 3 页，
// 页码空间与对外总数分叉的事实被钉住。
func TestPDFPageCountViewsDiverge(t *testing.T) {
	rawCap, restore := captureSlog(t)
	defer restore()

	raw := phantomPagePDF
	content := stripNonTextStreams(decodeFlateStreams(raw))

	if got := countPDFPages(raw); got != 4 {
		t.Fatalf("raw 字面计数 = %d, want 4（含二进制流里的伪 /Type /Page）", got)
	}
	if got := countPDFPages(content); got != 3 {
		t.Fatalf("文本流视图计数 = %d, want 3（伪页已被剔除）", got)
	}
	if got := pdfTotalPages(raw, content); got != 4 {
		t.Fatalf("对外 total = %d, want 4（现状口径 = 原始字节，本刀不改行为）", got)
	}
	if !rawCap.contains("页数口径不一致") {
		t.Fatalf("视图不一致必须诚实告警，实际告警=%q", rawCap.dump())
	}

	// 文本路径页码空间（页对象序列）只有 3 页，却对外宣称 4 页 → 实测差异。
	if got := len(pdfPageTexts(content)); got != 3 {
		t.Fatalf("文本路径页对象数 = %d, want 3", got)
	}
	// 边界：无页对象声明 → total 下限 1（历史行为）。
	if got := pdfTotalPages("", ""); got != 1 {
		t.Fatalf("退化 PDF total = %d, want 1", got)
	}

	// 端到端：ConvertLimit 的 total 取 raw 口径（4），正文只有 2 页有文本。
	p := filepath.Join(t.TempDir(), "phantom.pdf")
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	md, total, _, err := ConvertLimit(p, "", 0)
	if err != nil {
		t.Fatalf("ConvertLimit: %v", err)
	}
	if total != 4 {
		t.Fatalf("ConvertLimit total = %d, want 4（raw 口径现状钉住）", total)
	}
	if !strings.Contains(md, "page-two") || !strings.Contains(md, "page-three") {
		t.Fatalf("正文提取缺失: %q", md)
	}
	if strings.Contains(md, "GARBAGE-BINARY") {
		t.Fatalf("伪页二进制内容混入正文: %q", md)
	}
}

// TestPDFPageCountContentViewExceedsRaw 探针②（反方向）：解压后的文本流视图里
// 出现 raw 里没有的 /Type /Page → 文本路径把正文归到第 2 页（真实只有 1 页），
// 且文本路径页数超过声明总页数时同样诚实告警。
func TestPDFPageCountContentViewExceedsRaw(t *testing.T) {
	rawCap, restore := captureSlog(t)
	defer restore()

	raw := buildFlateTextPDF(t, "/Type /Page\nBT /F1 12 Tf 72 720 Td (page-one) Tj ET\n")
	content := stripNonTextStreams(decodeFlateStreams(raw))

	if got := countPDFPages(raw); got != 1 {
		t.Fatalf("raw 字面计数 = %d, want 1（/Type /Page 在 zlib 流里，字面看不到）", got)
	}
	if got := countPDFPages(content); got != 2 {
		t.Fatalf("文本流视图计数 = %d, want 2（解压后可见）", got)
	}
	if got := pdfTotalPages(raw, content); got != 1 {
		t.Fatalf("对外 total = %d, want 1（raw 口径）", got)
	}
	if !rawCap.contains("页数口径不一致") {
		t.Fatalf("视图不一致必须告警，实际=%q", rawCap.dump())
	}

	// 文本路径页码错位实测：正文被判为第 2 页（第 1 页空）。
	texts := pdfPageTexts(content)
	if len(texts) != 2 {
		t.Fatalf("文本路径页数 = %d, want 2（正文错位到第 2 页）", len(texts))
	}
	if strings.TrimSpace(texts[0]) != "" || !strings.Contains(texts[1], "page-one") {
		t.Fatalf("文本路径归页不符：%q", texts)
	}

	// 端到端跑一次，触发「文本路径页数 > 声明总页数」的第二条诚实告警。
	p := filepath.Join(t.TempDir(), "inflate.pdf")
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, total, _, err := pdfToMarkdownLimit(p, "", 0, nil); total != 1 {
		t.Fatalf("pdfToMarkdownLimit total = %d, want 1 (err=%v)", total, err)
	}
	if !rawCap.contains("文本路径页对象数超过声明总页数") {
		t.Fatalf("文本路径页码分叉必须告警，实际=%v", rawCap.msgs)
	}
}

// TestPageBoundsSpecIgnoresDeclaredTotal 口径③现状钉住：pageBounds 只按页规格
// 算渲染边界，不夹到声明总页数（total 仅在空规格时参与）→ 渲染范围可以超出
// 声明页数；差异如实登记，不假装合一。
func TestPageBoundsSpecIgnoresDeclaredTotal(t *testing.T) {
	if f, l := pageBounds("3-5", 2); f != 3 || l != 5 {
		t.Fatalf("pageBounds(3-5, total=2) = (%d,%d), want (3,5)（规格口径不夹 total）", f, l)
	}
	if f, l := pageBounds("", 2); f != 1 || l != 2 {
		t.Fatalf("pageBounds 空规格 = (%d,%d), want (1,2)（仅此处用 total）", f, l)
	}
}

// TestRenderedPageJobsArtifactNumbers OCR 页码来源：pdftoppm 实际产物的文件名
// 后缀即绝对页码；跳页（只生成 -1/-2/-4）时页号仍正确，页码过滤用绝对页码。
func TestRenderedPageJobsArtifactNumbers(t *testing.T) {
	jobs, fromArtifacts := renderedPageJobs([]string{
		filepath.Join("tmp", "page-1.png"),
		filepath.Join("tmp", "page-2.png"),
		filepath.Join("tmp", "page-4.png"), // pdftoppm 跳过损坏的第 3 页
	}, 1)
	if !fromArtifacts {
		t.Fatal("pdftoppm 产物文件名应可解析页码")
	}
	var nums []int
	for _, j := range jobs {
		nums = append(nums, j.num)
	}
	if fmt.Sprint(nums) != "[1 2 4]" {
		t.Fatalf("产物页码 = %v, want [1 2 4]（first+i 旧口径会给出 [1 2 3]）", nums)
	}

	// 页范围过滤按绝对页码：请求「1-3」只应识别真实第 1/2 页，第 4 页被排除。
	var got []int
	if _, _, _, err := ocrPageLoop([]string{"page-1.png", "page-2.png", "page-4.png"}, 1, "1-3",
		func(pageNum int, _ string) (string, error) {
			got = append(got, pageNum)
			return "text", nil
		}, nil); err != nil {
		t.Fatalf("ocrPageLoop: %v", err)
	}
	if fmt.Sprint(got) != "[1 2]" {
		t.Fatalf("页范围 1-3 识别的页码 = %v, want [1 2]（第 4 页不得被当成第 3 页混入）", got)
	}

	// 请求真实第 4 页：旧口径（first+i）会找不到任何页并报错。
	got = nil
	if _, _, _, err := ocrPageLoop([]string{"page-1.png", "page-2.png", "page-4.png"}, 1, "4",
		func(pageNum int, _ string) (string, error) {
			got = append(got, pageNum)
			return "text", nil
		}, nil); err != nil {
		t.Fatalf("请求第 4 页应命中实际产物 page-4.png: %v", err)
	}
	if fmt.Sprint(got) != "[4]" {
		t.Fatalf("请求第 4 页识别到的页码 = %v, want [4]", got)
	}
}

// TestOCRPageOrderNumericNotLexicographic 目录读取是字典序（page-10 在 page-2
// 之前）：产物页码必须按数值升序，否则第 10 页会被 OCR 成第 3 页。
func TestOCRPageOrderNumericNotLexicographic(t *testing.T) {
	files := []string{
		"page-1.png", "page-10.png", "page-11.png", "page-12.png",
		"page-2.png", "page-3.png",
	}
	var order []int
	var paths []string
	if _, _, _, err := ocrPageLoop(files, 1, "", func(pageNum int, png string) (string, error) {
		order = append(order, pageNum)
		paths = append(paths, png)
		return "text", nil
	}, nil); err != nil {
		t.Fatalf("ocrPageLoop: %v", err)
	}
	want := "[1 2 3 10 11 12]"
	if fmt.Sprint(order) != want {
		t.Fatalf("OCR 顺序/页码 = %v, want %s（字典序会给出 1,10,11,12,2,3）", order, want)
	}
	if paths[3] != "page-10.png" {
		t.Fatalf("第 4 个识别目标 = %s, want page-10.png", paths[3])
	}
}

// TestRenderedPageJobsFallback 文件名不可解析时整体回退 first+i（历史口径），
// 并标记 fromArtifacts=false 供调用方告警。
func TestRenderedPageJobsFallback(t *testing.T) {
	jobs, fromArtifacts := renderedPageJobs([]string{"p1.png", "p2.png"}, 3)
	if fromArtifacts {
		t.Fatal("不可解析的文件名不应声称来自产物页码")
	}
	if len(jobs) != 2 || jobs[0].num != 3 || jobs[1].num != 4 {
		t.Fatalf("回退编号 = %+v, want 3/4", jobs)
	}
}
