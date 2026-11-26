package docmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// TestMain 让单测默认关掉 WPS 通道（GAEA_WPS_PDF 未设时置 0）。WPS 优先后「提取不到
// 文字」的 PDF 都会先去拉 wpscli——装了 WPS 的开发机上，任何没打桩的用例都会真调
// WPS 云。要验证 WPS 行为的用例用 stubWps / stubWpsBoth（显式置 1 并打桩）。
// GAEAPROBE_WPS_STUB=1 时本测试二进制充当假 wpscli（子进程，见 fakeWpsCLI）。
func TestMain(m *testing.M) {
	if os.Getenv("GAEAPROBE_WPS_STUB") == "1" {
		os.Exit(fakeWpsCLI(os.Args[1:]))
	}
	if os.Getenv("GAEA_WPS_PDF") == "" {
		_ = os.Setenv("GAEA_WPS_PDF", "0")
	}
	os.Exit(m.Run())
}

// fakeWpsCLI 假 wpscli（由测试二进制的子进程扮演）：pdfinfo 报 GAEAPROBE_WPS_PAGES 页
// （GAEAPROBE_WPS_SCAN=1 时再报是扫描件）；pdf2excel / photo2excel 见 fakeWpsExcel；
// 其余子命令把完整参数行追加到 GAEAPROBE_WPS_LOG，再按 GAEAPROBE_WPS_EXIT 退出（默认 1）。
// GAEAPROBE_WPS_SLEEP（毫秒）让非 pdfinfo 的子命令先睡一会儿（测 gaea 侧超时用）。
func fakeWpsCLI(args []string) int {
	if len(args) == 0 {
		return 201
	}
	if args[0] == "pdfinfo" {
		pages := os.Getenv("GAEAPROBE_WPS_PAGES")
		if pages == "" {
			pages = "0"
		}
		scan := "false"
		if os.Getenv("GAEAPROBE_WPS_SCAN") == "1" {
			scan = "true"
		}
		fmt.Printf("{\"type\":\"completed\",\"status\":\"success\",\"page_count\":%s,\"is_scan_document\":%s}\n", pages, scan)
		return 0
	}
	if ms := os.Getenv("GAEAPROBE_WPS_SLEEP"); ms != "" {
		if d, err := time.ParseDuration(ms + "ms"); err == nil {
			time.Sleep(d)
		}
	}
	if args[0] == "pdf2excel" || args[0] == "photo2excel" {
		return fakeWpsExcel(args)
	}
	if log := os.Getenv("GAEAPROBE_WPS_LOG"); log != "" {
		if f, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			fmt.Fprintln(f, strings.Join(args, " "))
			_ = f.Close()
		}
	}
	return fakeWpsExitCode()
}

// fakeWpsExitCode GAEAPROBE_WPS_EXIT 指定的退出码（未设时为 1）。
func fakeWpsExitCode() int {
	code := 1
	if v := os.Getenv("GAEAPROBE_WPS_EXIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			code = n
		}
	}
	return code
}

// fakeWpsExcel 假 wpscli 的转表子命令（pdf2excel / photo2excel）。GAEAPROBE_WPS_EXIT 非 0 时
// 只记参数并按码退出（失败路径）；为 0 时按 GAEAPROBE_WPS_XLSX 往 -o 指的位置产出并报成功：
//   - table（默认）：一张带合并单元格的小表，数值格、文本数字都有；
//   - gap：清单页替身，第 6 行的工程量空着（疑似缺格 1 行）；
//   - rename：wpscli 不覆盖已有文件语义——产物另存成 <名>_1.xlsx，终态行报另存后的路径；
//   - empty / missing：0 字节 / 不产出（报成功但没产物）；
//   - garbage：写一堆不是 xlsx 的字节（产物损坏）；
//   - blank：能打开，但一个非空格子都没有。
func fakeWpsExcel(args []string) int {
	if log := os.Getenv("GAEAPROBE_WPS_LOG"); log != "" {
		if f, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			fmt.Fprintln(f, strings.Join(args, " "))
			_ = f.Close()
		}
	}
	if code := fakeWpsExitCode(); code != 0 {
		return code
	}
	out := ""
	for i := 1; i+1 < len(args); i++ {
		if args[i] == "-o" || args[i] == "--output" {
			out = args[i+1]
		}
	}
	if out == "" {
		return 201
	}
	mode := os.Getenv("GAEAPROBE_WPS_XLSX")
	reported := out
	if mode == "rename" {
		reported = strings.TrimSuffix(out, filepath.Ext(out)) + "_1" + filepath.Ext(out)
	}
	var err error
	switch mode {
	case "empty":
		err = os.WriteFile(out, nil, 0o644)
	case "missing":
	case "garbage":
		err = os.WriteFile(out, []byte("not an xlsx"), 0o644)
	default:
		err = fakeWpsWriteXLSX(reported, mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 214
	}
	fmt.Printf("{\"type\":\"completed\",\"status\":\"success\",\"output\":%q,\"elapsed_ms\":1234}\n",
		filepath.ToSlash(reported))
	return 0
}

// fakeWpsWriteXLSX 用 excelize 写一份小工作簿当 wpscli 的产物。
func fakeWpsWriteXLSX(path, mode string) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if mode == "blank" {
		return f.SaveAs(path) // 新建的工作簿没有任何格子
	}
	var rows [][]any
	if mode == "gap" {
		rows = [][]any{
			{"序号", "项目编码", "项目名称", "单位", "工程量"},
			{9, "030412001008", "配管", "m", 1000},
			{10, "030409001014", "电力电缆", "m", 1000},
			{11, "030404014004", "控制箱、同期小屏", "台", 9},
			{12, "030409001015", "电力电缆", "m", 500},
			{13, "030601004004", "流量仪表", "台", nil}, // 工程量「1」被在线识别漏掉
			{14, "030409001016", "电力电缆", "m", 100},
			{15, "030608008004", "专用线缆", "m", 100},
		}
	} else {
		rows = [][]any{
			{"序号", "项目编码", "项目名称", "单位", "工程量", "金额（元）"},
			{9, "030412001008", "配管", "m", 1000, 5776.41},
			{10, "030409001014", "电力电缆", "m", 1000, 4364.61},
			{11, "030404014004", "控制箱、同期小屏", "台", 9, nil},
		}
	}
	for r, row := range rows {
		for c, v := range row {
			if v == nil {
				continue
			}
			coord, _ := excelize.CoordinatesToCellName(c+1, r+1)
			if err := f.SetCellValue("Sheet1", coord, v); err != nil {
				return err
			}
		}
	}
	if mode != "gap" {
		if err := f.SetCellValue("Sheet1", "A6", "合计"); err != nil {
			return err
		}
		if err := f.MergeCell("Sheet1", "A6", "B6"); err != nil {
			return err
		}
	}
	return f.SaveAs(path)
}

// fakeWpsExcelCLI 假 wpscli：转表子命令报成功并按 mode 产出 xlsx（见 fakeWpsExcel）。
func fakeWpsExcelCLI(t *testing.T, mode string, pages int) string {
	t.Helper()
	t.Setenv("GAEAPROBE_WPS_XLSX", mode)
	return fakeWpsCLIPath(t, 0, pages)
}

// fakeWpsCLIPath 让本测试二进制充当假 wpscli：返回可执行路径；子进程行为经 t.Setenv
// 写进环境变量（随 exec 传给子进程）。调用方自己决定日志落点（GAEAPROBE_WPS_LOG）。
func fakeWpsCLIPath(t *testing.T, exitCode, pages int) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("取测试二进制路径: %v", err)
	}
	t.Setenv("GAEAPROBE_WPS_STUB", "1")
	t.Setenv("GAEAPROBE_WPS_EXIT", strconv.Itoa(exitCode))
	t.Setenv("GAEAPROBE_WPS_PAGES", strconv.Itoa(pages))
	return exe
}

// resetWpsState 换掉进程内 OCR/WPS 结果缓存与失败静默记录：这些是包级状态，同内容的
// 夹具 PDF 跨用例会互相命中（上一条用例缓存了 1-100 段的成功结果，这一条就不再拉
// wpscli）。走真实编排函数（wpsScannedPDFMarkdown / wpsPDFMarkdown）的用例开头调用。
func resetWpsState(t *testing.T) {
	t.Helper()
	origCache, origFail := defaultOCRCache, wpsFail
	defaultOCRCache, wpsFail = newOCRCache(), map[string]time.Time{}
	wpsSuspend(-time.Minute)
	t.Cleanup(func() {
		defaultOCRCache, wpsFail = origCache, origFail
		wpsSuspend(-time.Minute)
	})
}

func readWpsCallLog(t *testing.T, log string) []string {
	t.Helper()
	data, err := os.ReadFile(log)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("读 wpscli 调用日志: %v", err)
	}
	var lines []string
	for _, ln := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if ln != "" {
			lines = append(lines, ln)
		}
	}
	return lines
}

// buildGlyphIDPDF 生成「字形号文字层」PDF：裸 <hex>Tj（CID 字体把字存成字形号，要靠
// 字体映射表才能还原）。extractPDFText 只认 (...)Tj 与 [...] TJ，这种形态整页提空——
// 2026-11-26 实弹那份 174 页扫描件就是它（JPEG 图像页 + 这种文字层）。
func buildGlyphIDPDF(t testing.TB, pages int) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	for i := 1; i <= pages; i++ {
		fmt.Fprintf(&sb, "/Type /Page\nBT /FT8 521 Tf 72 %d Td <11CD>Tj <07A4>Tj <09EF>Tj ET\n", 720-i*12)
	}
	path := filepath.Join(t.TempDir(), fmt.Sprintf("glyph-id-%d.pdf", pages))
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	return path
}

// 字形号文字层（自写提取整页提空）：先走 WPS 在线 OCR，不碰文字型通路；keepImages
// 决定图片去留；快速通道（preferWPS=false）与 format_convert（true）同此分流。
func TestPDFToMarkdownLimit_GlyphIDTextLayerGoesWpsOnlineOCR(t *testing.T) {
	forbidOCR(t)
	stubs := stubWpsBoth(t, "不该走文字型通路",
		"# WPS 在线 OCR\n\n字形号文字层还原出的正文\n\n![图](data:image/png;base64,AAAA)\n")
	for _, c := range []struct {
		keep, prefer bool
	}{{false, false}, {false, true}, {true, true}} {
		*stubs.text, *stubs.scanned = false, false
		md, total, _, err := pdfToMarkdownLimitImages(buildGlyphIDPDF(t, 2), "", 0, nil, c.keep, c.prefer)
		if err != nil {
			t.Fatalf("keep=%v prefer=%v: %v", c.keep, c.prefer, err)
		}
		if *stubs.text || !*stubs.scanned || !strings.Contains(md, "还原出的正文") {
			t.Fatalf("keep=%v prefer=%v: 应走 WPS 在线 OCR，得到 text=%v scanned=%v md=%q",
				c.keep, c.prefer, *stubs.text, *stubs.scanned, md)
		}
		if has := strings.Contains(md, "data:image/"); has != c.keep {
			t.Fatalf("keep=%v: 图片是否保留=%v", c.keep, has)
		}
		if total != 2 {
			t.Fatalf("total=%d, want 2", total)
		}
	}
}

// 扫描件替身（页码戳噪声文本层）：同样先走 WPS 在线 OCR。
func TestPDFToMarkdownLimit_NoTextLayerGoesWpsOnlineOCRFirst(t *testing.T) {
	forbidOCR(t)
	stubs := stubWpsBoth(t, "不该走文字型通路", "# WPS 在线 OCR\n\n扫描件正文\n")
	md, _, _, err := ConvertLimit(buildScanStampPDF(t, 3), "", 0)
	if err != nil || !*stubs.scanned || *stubs.text || !strings.Contains(md, "扫描件正文") {
		t.Fatalf("应走 WPS 在线 OCR，得到 text=%v scanned=%v md=%q err=%v", *stubs.text, *stubs.scanned, md, err)
	}
}

// WPS 在线 OCR 失败：落本地 OCR 兜底（证明走到了渲染路径）；兜底也失败时，报错把
// WPS 的失败原因一并带上——只报本地 OCR 的原因会把刚失败过的路再指一遍。
func TestPDFToMarkdownLimit_OnlineOCRFailsFallsToLocalOCR(t *testing.T) {
	forbidOCR(t)
	stubs := stubWpsBoth(t, "", "")
	md, _, _, err := ConvertLimit(buildScanStampPDF(t, 3), "", 0)
	if err == nil || !strings.Contains(err.Error(), "pdftoppm") {
		t.Fatalf("WPS 失败后应落本地 OCR（报 pdftoppm 缺失），得到 md=%q err=%v", md, err)
	}
	if !*stubs.scanned {
		t.Fatal("应先试过 WPS 在线 OCR")
	}
	for _, want := range []string{"WPS 在线 OCR 没成功", "WPS 在线 OCR 失败（测试桩）", "本地 OCR 兜底也没成功"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("报错应带 WPS 失败原因 %q：%v", want, err)
		}
	}
}

// format_convert（preferWPS）：文字层干净的 PDF 也先经 WPS；快速通道同一份不碰 WPS。
func TestConvertPreservingImages_CleanTextPrefersWPS(t *testing.T) {
	stubs := stubWpsBoth(t, "# 来自 WPS\n\n| a | b |\n|---|---|\n| 1 | 2 |\n", "不该走在线 OCR")
	md, total, truncated, _, err := ConvertPreservingImages(buildPagePDF(t, 3), "", 0)
	if err != nil || !*stubs.text || *stubs.scanned || !strings.Contains(md, "来自 WPS") || total != 3 || truncated {
		t.Fatalf("format_convert 应 WPS 优先：text=%v scanned=%v total=%d truncated=%v err=%v md=%q",
			*stubs.text, *stubs.scanned, total, truncated, err, md)
	}
	*stubs.text = false
	md, _, _, err = ConvertLimit(buildPagePDF(t, 3), "", 0)
	if err != nil || *stubs.text || !strings.Contains(md, "第 2 页内容") {
		t.Fatalf("快速通道应走本地提取：text=%v err=%v md=%q", *stubs.text, err, md)
	}
}

// format_convert 的 WPS 转换失败：退回本地提取的文字，不比以前差。
func TestConvertPreservingImages_WPSFailureFallsBackToLocalText(t *testing.T) {
	forbidOCR(t)
	stubs := stubWpsBoth(t, "", "")
	md, _, _, _, err := ConvertPreservingImages(buildPagePDF(t, 3), "", 0)
	if err != nil || !*stubs.text || !strings.Contains(md, "第 1 页内容") {
		t.Fatalf("WPS 失败应退回本地提取：text=%v err=%v md=%q", *stubs.text, err, md)
	}
}

func TestWpsConvertArgs(t *testing.T) {
	cases := []struct {
		name              string
		first, last       int
		scanned           bool
		wantRange         string
		wantScanned, noRg bool
	}{
		{"文字型整本不带页范围", 1, 0, false, "", false, true},
		{"文字型带页范围", 3, 5, false, "--range 3-5", false, false},
		{"在线 OCR 必带 --scanned 与页范围", 101, 174, true, "--range 101-174", true, false},
	}
	for _, c := range cases {
		got := strings.Join(wpsConvertArgs("in.pdf", "out.docx", c.first, c.last, c.scanned), " ")
		if !strings.HasPrefix(got, "pdf2word in.pdf -o out.docx --json") {
			t.Errorf("%s: 前缀不对：%s", c.name, got)
		}
		if has := strings.Contains(got, "--scanned true"); has != c.wantScanned {
			t.Errorf("%s: --scanned=%v", c.name, has)
		}
		if has := strings.Contains(got, "--range"); has == c.noRg {
			t.Errorf("%s: --range 有无不对：%s", c.name, got)
		}
		if c.wantRange != "" && !strings.Contains(got, c.wantRange) {
			t.Errorf("%s: 缺 %s：%s", c.name, c.wantRange, got)
		}
	}
}

// 在线 OCR 的页范围编排：整本按 100 页分段串行；显式页范围原样下发。
func TestWpsScannedPDFMarkdown_Chunks(t *testing.T) {
	resetWpsState(t)
	var got [][2]int
	orig := wpsScannedConvertFn
	wpsScannedConvertFn = func(_, _ string, first, last int) (string, error) {
		got = append(got, [2]int{first, last})
		return fmt.Sprintf("第 %d-%d 页", first, last), nil
	}
	t.Cleanup(func() { wpsScannedConvertFn = orig })
	path := buildPagePDF(t, 3) // 只借它的内容指纹，不走它的文字

	md, err := wpsScannedPDFMarkdown("cli", path, "", 174)
	if err != nil || len(got) != 2 || got[0] != [2]int{1, 100} || got[1] != [2]int{101, 174} ||
		!strings.Contains(md, "第 1-100 页") || !strings.Contains(md, "第 101-174 页") {
		t.Fatalf("整本 174 页应分成 1-100 / 101-174：got=%v md=%q err=%v", got, md, err)
	}

	got = nil
	if md, err = wpsScannedPDFMarkdown("cli", path, "3-5", 174); err != nil || len(got) != 1 || got[0] != [2]int{3, 5} || md != "第 3-5 页" {
		t.Fatalf("显式页范围应原样下发：got=%v md=%q err=%v", got, md, err)
	}

	if _, err = wpsScannedPDFMarkdown("cli", path, "999", 174); err == nil || !strings.Contains(err.Error(), "超出") {
		t.Fatalf("越界页范围应报错，得到 err=%v", err)
	}
	if _, err = wpsScannedPDFMarkdown("", path, "", 174); err == nil {
		t.Fatal("没有 wpscli 应报错")
	}
	// 读不到页数（pdfinfo 也读不到）：在线 OCR 必须有显式页范围，如实报错。
	if _, err = wpsScannedPDFMarkdown(filepath.Join(t.TempDir(), "wpscli-missing.exe"), path, "", 0); err == nil ||
		!strings.Contains(err.Error(), "页数") {
		t.Fatalf("读不到页数应报错，得到 err=%v", err)
	}
}

// 在线 OCR 没有显式页范围：不拉起 wpscli 直接报错（引擎的要求）。
func TestWpsConvertScannedRange_RequiresRange(t *testing.T) {
	if _, err := wpsConvertScannedRange("cli", "x.pdf", 1, 0); err == nil || !strings.Contains(err.Error(), "显式页范围") {
		t.Fatalf("应报需要显式页范围，得到 err=%v", err)
	}
}

// 真实参数落到 wpscli 进程：在线 OCR 带 --scanned true 与显式页范围；文字型整本不带。
func TestWpsConvert_ArgsReachCLIProcess(t *testing.T) {
	t.Setenv("GAEA_WPS_PDF", "1")
	resetWpsState(t)
	log := filepath.Join(t.TempDir(), "wpscli.log")
	t.Setenv("GAEAPROBE_WPS_LOG", log)
	cli := fakeWpsCLIPath(t, 1, 174) // pdf2word 一律失败：只看参数，不走 markitdown
	path := buildPagePDF(t, 3)

	if _, err := wpsScannedPDFMarkdown(cli, path, "", 174); err == nil || !strings.Contains(err.Error(), "WPS 在线 OCR 失败") {
		t.Fatalf("应如实报失败，得到 err=%v", err)
	}
	if _, err := wpsScannedPDFMarkdown(cli, path, "101-174", 174); err == nil {
		t.Fatal("应如实报失败")
	}
	calls := readWpsCallLog(t, log)
	want := []string{
		"--json --scanned true --range 1-100",
		"--json --scanned true --range 101-174",
	}
	if len(calls) != len(want) {
		t.Fatalf("wpscli 调用数=%d，want %d：%q", len(calls), len(want), calls)
	}
	for i, w := range want {
		if !strings.HasSuffix(calls[i], w) {
			t.Fatalf("第 %d 次调用应以 %s 结尾：%s", i+1, w, calls[i])
		}
	}
}

// 整本文字型转换的真实参数：不带 --range、不带 --scanned。
func TestWpsConvert_WholeBookArgsReachCLIProcess(t *testing.T) {
	t.Setenv("GAEA_WPS_PDF", "1")
	resetWpsState(t)
	log := filepath.Join(t.TempDir(), "wpscli.log")
	t.Setenv("GAEAPROBE_WPS_LOG", log)
	cli := fakeWpsCLIPath(t, 1, 0)
	// pdfinfo 读不到页数（0）：wpsPDFMarkdown 按「页数未知」整本转换，不带 --range。
	if _, err := wpsPDFMarkdown(cli, buildPagePDF(t, 3), "", 0); err == nil {
		t.Fatal("应如实报失败")
	}
	calls := readWpsCallLog(t, log)
	if len(calls) != 1 {
		t.Fatalf("wpscli 调用数=%d，want 1：%q", len(calls), calls)
	}
	if !strings.HasSuffix(calls[0], ".docx --json") || strings.Contains(calls[0], "--range") || strings.Contains(calls[0], "--scanned") {
		t.Fatalf("整本文字型不应带 --range / --scanned：%s", calls[0])
	}
}

// 账号级失败（无会员 101）进全局静默期：之后别的 PDF 不再拉起 wpscli，直接走本地兜底。
func TestWpsConvert_AccountFailureSuspendsWPS(t *testing.T) {
	t.Setenv("GAEA_WPS_PDF", "1")
	resetWpsState(t) // 也清掉别的用例可能留下的静默期
	cli := fakeWpsCLIPath(t, 101, 1)
	if _, err := wpsConvertRange(cli, buildPagePDF(t, 1), 1, 0); err == nil || !strings.Contains(err.Error(), "会员") {
		t.Fatalf("应报无会员权限，得到 err=%v", err)
	}
	if !wpsSuspended() {
		t.Fatal("账号级失败后应进入静默期")
	}

	forbidOCR(t)
	stubs := stubWpsBoth(t, "不该被用到", "不该被用到")
	if _, _, _, err := ConvertLimit(buildGarbledPDF(t, true), "", 0); err == nil {
		t.Fatal("静默期内应跳过 WPS、走本地兜底并如实报错")
	}
	if *stubs.text || *stubs.scanned {
		t.Fatalf("静默期内不应调用 WPS：text=%v scanned=%v", *stubs.text, *stubs.scanned)
	}
}

// 本地 OCR 是兜底：超过 ocrMaxPages 直接拒绝，不渲染、不拉引擎。
func TestOCRPDFRange_RefusesOverCap(t *testing.T) {
	orig := pdftoppmLookup
	t.Cleanup(func() { pdftoppmLookup = orig })
	pdftoppmLookup = func() string {
		t.Error("超限不应进入渲染")
		return ""
	}

	_, err := ocrPDFRange("x.pdf", "", 1, ocrMaxPages+1, 200, nil)
	if err == nil || !strings.Contains(err.Error(), "最多识别 30 页") || !strings.Contains(err.Error(), "pages") {
		t.Fatalf("超限应拒绝并指路分段，得到 err=%v", err)
	}
	// 强制 OCR 入口（成本导入兜底）同样受限，且失败不进缓存。
	for i := 0; i < 2; i++ {
		if _, err := OCRPDFText(buildScanStampPDF(t, ocrMaxPages+10), 0); err == nil || !strings.Contains(err.Error(), "最多识别 30 页") {
			t.Fatalf("第 %d 次：应拒绝超限扫描件，得到 err=%v", i+1, err)
		}
	}
	// 恰到上限不拦：放行到渲染（pdftoppm 缺失才报错）。
	pdftoppmLookup = func() string { return "" }
	if _, err := ocrPDFRange("x.pdf", "", 1, ocrMaxPages, 200, nil); err == nil || strings.Contains(err.Error(), "最多识别") {
		t.Fatalf("恰到上限不应被拦，得到 err=%v", err)
	}
}

// 表格里的纯数字单元格被 OCR 拆成「3. 1」时并回，数字后直接贴着拉丁字母单位的同样并
// （115953. 12t）；带文字、加粗、中文单位的格子（「H/3, 1次/3d」并了就错），日期时间，
// 千分位写法，转义管道，以及非表格行，一律不动。
func TestRejoinSplitDecimals(t *testing.T) {
	in := "| 1 | 3. 1 | 115953. 12t | 30.\u00a011 | 3.5 m³ | 合计 102. 14 | **3.2** | a \\| b |\n" +
		"| 2 | 1, 234.5 | 2023/9/15 17:55:47 | 土方开挖至0~H/3, 1次/3d | 102. 14 m3 | 四川雅安市石棉县 | x | y |\n" +
		"正文里的 43268.84m3 与 3. 1 不动。\n"
	want := "| 1 | 3.1 | 115953.12t | 30.11 | 3.5 m³ | 合计 102. 14 | **3.2** | a \\| b |\n" +
		"| 2 | 1, 234.5 | 2023/9/15 17:55:47 | 土方开挖至0~H/3, 1次/3d | 102. 14 m3 | 四川雅安市石棉县 | x | y |\n" +
		"正文里的 43268.84m3 与 3. 1 不动。\n"
	if got := rejoinSplitDecimals(in); got != want {
		t.Fatalf("并回结果不对：\n got=%q\nwant=%q", got, want)
	}
	if got := rejoinSplitDecimals(want); got != want {
		t.Fatalf("再处理一遍应幂等：%q", got)
	}
}
