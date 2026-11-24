package docmd

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildGarbledPDF 生成「子集字体字形号被当码点硬解」形态的合成 PDF（Word/WPS 导出中文
// PDF 的真实病因，见 wps.go）。hex 串是字形号对，自写提取会解成孟加拉文/天城文等非预期
// 文字（soft）；withInvalid 时再带一段含非法 UTF-8 字节的字面串（hard）。
func buildGarbledPDF(t testing.TB, withInvalid bool) string {
	t.Helper()
	var body strings.Builder
	for i := 0; i < 6; i++ {
		fmt.Fprintf(&body, "BT /F3 12 Tf 72 %d Td [<09870F28096A15791180> 0] TJ ET\n", 720-i*12)
	}
	if withInvalid {
		body.WriteString("BT /F3 12 Tf 72 600 Td (\xf4\xe0\xa4\x80) Tj ET\n")
	}
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n/Type /Page\n")
	fmt.Fprintf(&sb, "<< /Length %d >>\nstream\n%sendstream\nendobj\n%%%%EOF\n", body.Len(), body.String())
	path := filepath.Join(t.TempDir(), fmt.Sprintf("garbled-%v.pdf", withInvalid))
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	return path
}

// buildHiddenPagesPDF 「页对象进压缩对象流」PDF 的最小替身（Word 2016 起导出的默认
// 形态）：明文里只有 1 个页对象，另一个页对象在一个 Flate 压缩的对象流里，解压后才
// 数得到；正文在明文文字流里——自写提取读得到文字，却分不清它属于哪一页。
func buildHiddenPagesPDF(t testing.TB) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	fmt.Fprint(zw, "12 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n")
	_ = zw.Close()
	const body = "BT /F1 12 Tf 72 720 Td (hidden page marker) Tj ET\n"
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n/Type /Page\n")
	fmt.Fprintf(&sb, "<< /Length %d >>\nstream\n%sendstream\n", len(body), body)
	fmt.Fprintf(&sb, "<< /Type /ObjStm /Length %d /Filter /FlateDecode >>\nstream\n", buf.Len())
	sb.Write(buf.Bytes())
	sb.WriteString("\nendstream\nendobj\n%%EOF\n")
	path := filepath.Join(t.TempDir(), "hidden-pages.pdf")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	return path
}

// stubWps 堵住 WPS 通道：wpscli 指向不存在的文件（pdfinfo 立即失败、页数读数保持
// 自写计数），转换 seam 返回给定 Markdown 并记录是否被调用。单测不得真调 WPS。
func stubWps(t *testing.T, md string) *bool {
	t.Helper()
	t.Setenv("GAEA_WPS_PDF", "1") // 外层若关了 WPS 通道，这几条用例仍按「WPS 在」验证
	called := new(bool)
	origLookup, origFn := wpsCLILookup, wpsPDFMarkdownFn
	wpsCLILookup = func() string { return filepath.Join(t.TempDir(), "wpscli-missing.exe") }
	wpsPDFMarkdownFn = func(_, _, _ string, _ int) (string, error) {
		*called = true
		return md, nil
	}
	t.Cleanup(func() { wpsCLILookup, wpsPDFMarkdownFn = origLookup, origFn })
	return called
}

func forbidOCR(t *testing.T) {
	t.Helper()
	orig := pdftoppmLookup
	pdftoppmLookup = func() string { return "" }
	t.Cleanup(func() { pdftoppmLookup = orig })
}

func TestPDFTextGarbled(t *testing.T) {
	clean := strings.Repeat("第3章 项目实施安排：降水井布置间距 10m，沉降监测频率 1 次/天；污染物浓度 ±5%（苯并(a)芘 0.1mg/kg，pH 值 7.2，水温 25℃），面积 1600m²。", 20)
	cases := []struct {
		name       string
		text       string
		hard, soft bool
	}{
		{"空串", "", false, false},
		// 2026-11-24 实弹产物的前 420 字节（原样）
		{"真实乱码产物", "\x03\x03\x03\x03\x03\n\xe0\xa6\x87\xe1\x8b\xa8\xe0\xa5\xaa\xe1\x95\xa9\xe1\x86\x80.#4\xf4\xe0\xb4\x92\n)\n\xe0\xa0\xb6\xe0\xa5\xaa\n)\x1a\n\x10\n\x14\x1c\n\xe0\xb5\xa0\n\xe0\xb6\x87\xcb\x84\xe1\xa4\x8f\xe1\x94\xaa\xe5\x91\x91\xdf\x90\xe2\xb8\xa3\xe2\x8d\xae\xe0\xa0\xaa", true, true},
		{"字形号硬解成多脚本", strings.Repeat("ইየ४ᕩᆀഒ ඇ˄ᤏᔪ呑ߐ⸣⍮ ", 30), false, true},
		{"含 NUL 控制字符", strings.Repeat("正文段落内容。\x00\x01\x02", 40), true, false},
		{"正常中文报告", clean, false, false},
		{"正常英文", "The quick brown fox jumps over the lazy dog. Total suspended solids (TSS) ≤ 25 mg/L; pH 6.5–8.5.", false, false},
		{"中文里夹个别其他文字", clean + "한국어", false, false},
		// 小语种文档落 soft：有 WPS 就换（只会更好），没有时原样返回
		{"泰文文档", strings.Repeat("การประเมินความเสี่ยงต่อสุขภาพ ", 20), false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hard, soft := pdfTextGarbled(c.text)
			if hard != c.hard || soft != c.soft {
				t.Fatalf("pdfTextGarbled = hard %v soft %v, want %v %v", hard, soft, c.hard, c.soft)
			}
		})
	}
}

func TestMergeSplitTables(t *testing.T) {
	head := "| 序号 | 污染物 | 浓度 |\n| --- | --- | --- |\n"
	rows := func(a, b int) string {
		var sb strings.Builder
		for i := a; i <= b; i++ {
			fmt.Fprintf(&sb, "| %d | 苯并(a)芘 | %.1f |\n", i, float64(i))
		}
		return sb.String()
	}
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "续页重复表头 → 合成一张",
			in:   "前言\n\n" + head + rows(1, 3) + "\n" + head + rows(4, 6) + "\n后记",
			want: "前言\n\n" + head + rows(1, 3) + rows(4, 6) + "\n后记",
		},
		{
			name: "三页长表连续合并",
			in:   head + rows(1, 2) + "\n" + head + rows(3, 4) + "\n\n" + head + rows(5, 6),
			want: head + rows(1, 2) + rows(3, 4) + rows(5, 6),
		},
		{
			name: "表头不同 → 不动",
			in:   head + rows(1, 2) + "\n| 序号 | 污染物 | 备注 |\n| --- | --- | --- |\n| 3 | 砷 | — |\n",
		},
		{
			name: "中间隔着表题 → 不动",
			in:   head + rows(1, 2) + "\n表 3.1-2 续\n\n" + head + rows(3, 4) + "\n",
		},
		{
			name: "列数不同 → 不动",
			in:   head + rows(1, 2) + "\n| 序号 | 污染物 |\n| --- | --- |\n| 3 | 铅 |\n",
		},
		{
			name: "单元格里的转义竖线不拆列 → 仍按 3 列合并",
			in:   "| a | b \\| c | d |\n| --- | --- | --- |\n| 1 | x \\| y | 2 |\n\n| a | b \\| c | d |\n| --- | --- | --- |\n| 3 | z | 4 |\n",
			want: "| a | b \\| c | d |\n| --- | --- | --- |\n| 1 | x \\| y | 2 |\n| 3 | z | 4 |\n",
		},
		{
			name: "没有表格 → 原样",
			in:   "# 标题\n\n正文一段。\n\n- 列表项\n",
			want: "# 标题\n\n正文一段。\n\n- 列表项\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mergeSplitTables(c.in)
			want := c.want
			if want == "" {
				want = c.in
			}
			if got != want {
				t.Fatalf("mergeSplitTables:\n got: %q\nwant: %q", got, want)
			}
		})
	}
}

func TestCountStructure(t *testing.T) {
	md := "# 一级\n\n## 二级\n\n正文\n\n#### 四级\n\n    # 缩进代码里的井号\n\n####### 七个井号不算\n" +
		"| a | b |\n| --- | --- |\n| - | - |\n\n第二段\n\n| x |\n| :---: |\n| - |\n" +
		"![图1](data:image/png;base64,AAAA) ![图2](assets/a.png)\n"
	got := CountStructure(md)
	if got.Headings != 3 || got.Tables != 2 || got.Images != 2 {
		t.Fatalf("CountStructure = %+v, want {Headings:3 Tables:2 Images:2}", got)
	}
}

func TestPageSpans(t *testing.T) {
	cases := []struct {
		spec  string
		total int
		want  string
	}{
		{"", 0, "[[1 0]]"},   // 整本：不带页数
		{"", 233, "[[1 0]]"}, // 整本：不把页数写进 --range
		{"20-23", 233, "[[20 23]]"},
		{"1-2,5", 10, "[[1 2] [5 5]]"},
		{"300-400", 233, "[]"},          // 全部越界：一段都不留（调用方报错，不回退成整本）
		{"7", 0, "[[7 7]]"},             // 页数未知：按规格原样
		{"2-4,300-400", 233, "[[2 4]]"}, // 越界段丢弃，界内段保留
	}
	for _, c := range cases {
		if got := fmt.Sprint(pageSpans(c.spec, c.total)); got != c.want {
			t.Fatalf("pageSpans(%q,%d) = %s, want %s", c.spec, c.total, got, c.want)
		}
	}
}

func TestStripDataImages(t *testing.T) {
	in := "前 ![](data:image/png;base64,iVBORw0KGgo=) 中 ![说明](data:image/jpeg;base64,/9j/4AAQ) 后 ![路径](assets/a.png)"
	want := "前  中  后 ![路径](assets/a.png)"
	if got := stripDataImages(in); got != want {
		t.Fatalf("stripDataImages = %q, want %q", got, want)
	}
}

func TestWpsVersionAtLeast(t *testing.T) {
	lo := `C:\Users\u\AppData\Local\Kingsoft\WPS Office\12.1.0.29680\clitool\wpscli.exe`
	hi := `C:\Users\u\AppData\Local\Kingsoft\WPS Office\12.1.0.29690\clitool\wpscli.exe`
	if !wpsVersionAtLeast(hi, lo) || wpsVersionAtLeast(lo, hi) || !wpsVersionAtLeast(lo, "") {
		t.Fatalf("版本比较错误")
	}
}

func TestFindWpsCLI_EnvOverride(t *testing.T) {
	fake := filepath.Join(t.TempDir(), "wpscli.exe")
	if err := os.WriteFile(fake, []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GAEA_WPSCLI", fake)
	if got := findWpsCLI(); got != fake {
		t.Fatalf("findWpsCLI = %q, want %q", got, fake)
	}
}

func TestWpsError(t *testing.T) {
	cases := map[int]string{
		100: "未登录", 101: "会员", 429: "配额", 3: "超时", 207: "200MB", 209: "密码", 1: "退出码 1",
	}
	for code, want := range cases {
		if got := wpsError("pdf2word", code, "boom").Error(); !strings.Contains(got, want) {
			t.Fatalf("wpsError(%d) = %q, want contains %q", code, got, want)
		}
	}
}

// 乱码 PDF：WPS 在 → 用 WPS 的结果；keepImages 决定图片去留。
func TestPDFToMarkdownLimit_GarbledTextRescuedByWPS(t *testing.T) {
	forbidOCR(t)
	const wpsMD = "# 来自 WPS\n\n正文正确。\n\n![图](data:image/png;base64,AAAA)\n"
	called := stubWps(t, wpsMD)
	for _, keep := range []bool{false, true} {
		*called = false
		md, total, _, err := pdfToMarkdownLimitImages(buildGarbledPDF(t, true), "", 0, nil, keep)
		if err != nil {
			t.Fatalf("keep=%v: %v", keep, err)
		}
		if !*called || !strings.Contains(md, "正文正确。") {
			t.Fatalf("keep=%v: 应改走 WPS，得到 md=%q called=%v", keep, md, *called)
		}
		if has := strings.Contains(md, "data:image/"); has != keep {
			t.Fatalf("keep=%v: 图片是否保留=%v", keep, has)
		}
		if total != 1 {
			t.Fatalf("total=%d, want 1", total)
		}
	}
}

// 确定乱码且 WPS 不可用：OCR 也不通时如实报错，绝不把乱码当正文返回。
func TestPDFToMarkdownLimit_GarbledTextWithoutWPS_FailsHonestly(t *testing.T) {
	forbidOCR(t)
	t.Setenv("GAEA_WPS_PDF", "0")
	md, _, _, err := ConvertLimit(buildGarbledPDF(t, true), "", 0)
	if err == nil {
		t.Fatalf("确定乱码不得当正文返回，得到 md=%q", md)
	}
	if md != "" || !strings.Contains(err.Error(), "文字层无法解码") {
		t.Fatalf("应报「文字层无法解码」且不返回正文，得到 md=%q err=%v", md, err)
	}
}

// 疑似乱码（只是落在非预期文字区）且没有 WPS：原样返回，不比以前差。
func TestPDFToMarkdownLimit_SoftGarbleWithoutWPSKeepsText(t *testing.T) {
	forbidOCR(t)
	t.Setenv("GAEA_WPS_PDF", "0")
	md, _, _, err := ConvertLimit(buildGarbledPDF(t, false), "", 0)
	if err != nil || md == "" {
		t.Fatalf("疑似乱码无 WPS 时应原样返回，得到 md=%q err=%v", md, err)
	}
}

// 文字层干净的 PDF 不碰 WPS（本地快速通道不变）。
func TestPDFToMarkdownLimit_CleanTextNeverCallsWPS(t *testing.T) {
	called := stubWps(t, "不该被用到")
	md, total, truncated, err := ConvertLimit(buildPagePDF(t, 3), "", 5)
	if err != nil || !strings.Contains(md, "第 2 页内容") || total != 3 || truncated {
		t.Fatalf("md=%q total=%d truncated=%v err=%v", md, total, truncated, err)
	}
	if *called {
		t.Fatal("干净文字层不应调用 WPS")
	}
}

// 指定了页范围、而页对象只在压缩视图里数得到：自写提取分不清页归属，交给 WPS；
// 没指定页范围（整本）时不打扰。
func TestPDFToMarkdownLimit_PageRangeOnHiddenPagesUsesWPS(t *testing.T) {
	forbidOCR(t)
	path := buildHiddenPagesPDF(t)
	if d := countObjStmPages(decodeFlateStreams(readFileT(t, path))); d != 1 {
		t.Fatalf("夹具应在对象流里数到 1 个页对象，得到 %d", d)
	}
	called := stubWps(t, "WPS 结果")
	if md, _, _, err := ConvertLimit(path, "", 0); err != nil || *called || !strings.Contains(md, "hidden page marker") {
		t.Fatalf("整本转换不应调用 WPS，且正文照常读出：called=%v md=%q err=%v", *called, md, err)
	}
	*called = false
	md, total, _, err := ConvertLimit(path, "1", 0)
	if err != nil || !*called || md != "WPS 结果" {
		t.Fatalf("指定页范围应改走 WPS：called=%v md=%q total=%d err=%v", *called, md, total, err)
	}
	if total != 2 {
		t.Fatalf("总页数应取解压视图的读数 2，得到 %d", total)
	}
}

// format_convert 的图片通道：经 WPS 的 PDF 图片以 data URI 计入。
func TestConvertPreservingImages_PDFViaWPSCountsImages(t *testing.T) {
	forbidOCR(t)
	stubWps(t, "正文\n\n![a](data:image/png;base64,AAAA)\n![b](data:image/jpeg;base64,BBBB)\n")
	_, total, _, images, err := ConvertPreservingImages(buildGarbledPDF(t, true), "", 0)
	if err != nil || images != 2 || total != 1 {
		t.Fatalf("images=%d total=%d err=%v, want 2/1/nil", images, total, err)
	}
	// 非法扩展名仍是原来的「不支持的文件格式」，不被当成 PDF 解析
	if _, _, _, _, err := ConvertPreservingImages(filepath.Join(t.TempDir(), "x.xyz"), "", 0); err == nil || !strings.Contains(err.Error(), "不支持的文件格式") {
		t.Fatalf("非法扩展名应报不支持的文件格式，得到 err=%v", err)
	}
}

func readFileT(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var sb strings.Builder
	if _, err := bufio.NewReader(f).WriteTo(&sb); err != nil {
		t.Fatal(err)
	}
	return sb.String()
}
