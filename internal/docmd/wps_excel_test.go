package docmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// useFakeExcelCLI 打开 WPS 通道、清包级状态，并把 wpscli 指到假实现（转表子命令按 mode
// 产出，见 fakeWpsExcel；pdfinfo 报 pages 页）。单测不得真调 WPS。
func useFakeExcelCLI(t *testing.T, mode string, pages int) string {
	t.Helper()
	t.Setenv("GAEA_WPS_PDF", "1")
	resetWpsState(t)
	cli := fakeWpsExcelCLI(t, mode, pages)
	t.Setenv("GAEA_WPSCLI", cli)
	return cli
}

// moveProd 把临时目录里的产物搬到 out（ConvertToExcel 的 save 回调的最常见形态）。
func moveProd(out string) func(prod string) error {
	return func(prod string) error { return MoveOrCopyFile(prod, out) }
}

// buildMixedPDF 前 textPages 页有正文、后面全是页码戳（扫描页的替身）。
func buildMixedPDF(t testing.TB, textPages, scanPages int) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	for i := 1; i <= textPages; i++ {
		fmt.Fprintf(&sb, "/Type /Page\nBT /F1 12 Tf 72 720 Td (第 %d 页内容，正文若干字) Tj ET\n", i)
	}
	for i := 1; i <= scanPages; i++ {
		fmt.Fprintf(&sb, "/Type /Page\nBT /F1 12 Tf 72 720 Td (%d) Tj ET\n", i)
	}
	path := filepath.Join(t.TempDir(), fmt.Sprintf("mixed-%d-%d.pdf", textPages, scanPages))
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	return path
}

// 文字型 PDF：参数真到 wpscli 进程——不带 --range / --scanned，--timeout 给足；产物核对
// 出回执计数；save 把临时目录里的产物搬到目标位置。
func TestConvertToExcel_TextPDF(t *testing.T) {
	log := filepath.Join(t.TempDir(), "wpscli.log")
	t.Setenv("GAEAPROBE_WPS_LOG", log)
	useFakeExcelCLI(t, "table", 3)
	src := buildPagePDF(t, 3)
	// 目标目录由调用方先建好（工具层的职责）
	out := filepath.Join(t.TempDir(), "nested", "清单.xlsx")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatal(err)
	}

	rec, err := ConvertToExcel(src, ExcelOptions{}, moveProd(out))
	if err != nil {
		t.Fatalf("ConvertToExcel: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("产物没有搬到目标位置: %v", err)
	}
	calls := readWpsCallLog(t, log)
	if len(calls) != 1 {
		t.Fatalf("wpscli 调用数=%d，want 1：%q", len(calls), calls)
	}
	got := calls[0]
	for _, banned := range []string{"--range", "--scanned", "--ai-fix", "--multi-sheet"} {
		if strings.Contains(got, banned) {
			t.Fatalf("整本文字型不应带 %s：%s", banned, got)
		}
	}
	if !strings.HasPrefix(got, "pdf2excel "+src+" -o ") || !strings.HasSuffix(got, "--json --timeout 300") {
		t.Fatalf("参数不对：%s", got)
	}
	want := ExcelReceipt{Sheets: 1, Rows: 6, MaxCols: 6, Merged: 1, Cells: 24, Numbers: 8, NumberText: 3, Pages: 3}
	rec.Elapsed = 0
	if !reflect.DeepEqual(rec, want) {
		t.Fatalf("回执=%+v，want %+v", rec, want)
	}
}

// 显式页范围、一页一表、按文字型强制：各自落到参数上。
func TestConvertToExcel_OptionsReachArgs(t *testing.T) {
	log := filepath.Join(t.TempDir(), "wpscli.log")
	t.Setenv("GAEAPROBE_WPS_LOG", log)
	useFakeExcelCLI(t, "table", 3)
	src := buildPagePDF(t, 3)

	rec, err := ConvertToExcel(src, ExcelOptions{Pages: "2-3", PerPageSheet: true}, nil)
	if err != nil {
		t.Fatalf("ConvertToExcel: %v", err)
	}
	if !strings.Contains(readWpsCallLog(t, log)[0], "--json --range 2-3 --multi-sheet true --timeout ") || rec.Pages != 2 {
		t.Fatalf("显式页范围 / 一页一表没落到参数上：%q rec=%+v", readWpsCallLog(t, log), rec)
	}

	// 版面像扫描件（页码戳），但调用方知道它是文字型：走本地引擎
	_, err = ConvertToExcel(buildScanStampPDF(t, 3), ExcelOptions{Scanned: "false"}, nil)
	if err != nil {
		t.Fatalf("scanned=false: %v", err)
	}
	calls := readWpsCallLog(t, log)
	if strings.Contains(calls[1], "--scanned") {
		t.Fatalf("scanned=false 不应走在线识别：%s", calls[1])
	}
}

// 扫描件：WPS 判是扫描件，或自写提取读不出可信文字层，都走在线识别——带 --scanned
// true、显式页范围，AI 修复默认开。pages 是假 wpscli 的 pdfinfo 报的页数，须与夹具
// 的真实页数一致（自动判断拿它当分母）。
func TestConvertToExcel_ScanRoute(t *testing.T) {
	cases := []struct {
		name      string
		src       func(testing.TB) string
		pages     int
		scanFlag  bool
		wantLocal bool
	}{
		{"WPS 判扫描件（文字层其实干净）", func(tb testing.TB) string { return buildPagePDF(tb, 3) }, 3, true, false},
		{"页码戳扫描件", func(tb testing.TB) string { return buildScanStampPDF(tb, 3) }, 3, false, false},
		{"字形号文字层", func(tb testing.TB) string { return buildGlyphIDPDF(tb, 3) }, 3, false, false},
		{"乱码文字层", func(tb testing.TB) string { return buildGarbledPDF(tb, true) }, 1, false, false},
		{"前 3 页正文 + 后 5 页扫描：过半不是文字 → 在线识别", func(tb testing.TB) string { return buildMixedPDF(tb, 3, 5) }, 8, false, false},
		{"前 5 页正文 + 后 3 页扫描：过半是文字 → 本地引擎", func(tb testing.TB) string { return buildMixedPDF(tb, 5, 3) }, 8, false, true},
		{"页对象在压缩对象流里：整档文字可信 → 本地引擎", func(tb testing.TB) string { return buildHiddenPagesPDF(tb) }, 2, false, true},
		{"干净文字型", func(tb testing.TB) string { return buildPagePDF(tb, 3) }, 3, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := filepath.Join(t.TempDir(), "wpscli.log")
			t.Setenv("GAEAPROBE_WPS_LOG", log)
			useFakeExcelCLI(t, "table", c.pages)
			if c.scanFlag {
				t.Setenv("GAEAPROBE_WPS_SCAN", "1")
			}
			rec, err := ConvertToExcel(c.src(t), ExcelOptions{AIFix: true}, nil)
			if err != nil {
				t.Fatalf("ConvertToExcel: %v", err)
			}
			calls := readWpsCallLog(t, log)
			if len(calls) != 1 {
				t.Fatalf("wpscli 调用数=%d，want 1：%q", len(calls), calls)
			}
			wantRange := fmt.Sprintf("1-%d", c.pages) // 单页不带短横
			if c.pages == 1 {
				wantRange = "1"
			}
			wantRange = "--scanned true --ai-fix true --range " + wantRange + " --timeout "
			if got := strings.Contains(calls[0], wantRange); got == c.wantLocal {
				t.Fatalf("路线不对（本地=%v）：%s", c.wantLocal, calls[0])
			}
			if rec.OnlineOCR == c.wantLocal {
				t.Fatalf("回执 OnlineOCR=%v，与路线不符：%+v", rec.OnlineOCR, rec)
			}
		})
	}
}

// ai_fix=false：不带 --ai-fix，单次页数上限放宽到 100；默认（AI 修复）上限 20 页，
// 超限在拉起转表之前就报错并指路分段。
func TestConvertToExcel_ScanCaps(t *testing.T) {
	log := filepath.Join(t.TempDir(), "wpscli.log")
	t.Setenv("GAEAPROBE_WPS_LOG", log)
	useFakeExcelCLI(t, "table", 30)
	src := buildScanStampPDF(t, 1)

	if _, err := ConvertToExcel(src, ExcelOptions{AIFix: true}, nil); err == nil ||
		!strings.Contains(err.Error(), "一次最多转 20 页，这次是 30 页") || !strings.Contains(err.Error(), "pages 分段") {
		t.Fatalf("AI 修复 30 页应超限报错，得到 err=%v", err)
	}
	if _, err := ConvertToExcel(src, ExcelOptions{AIFix: true, Pages: "11-30"}, nil); err != nil {
		t.Fatalf("20 页应放行，得到 err=%v", err)
	}
	if calls := readWpsCallLog(t, log); len(calls) != 1 || !strings.Contains(calls[0], "--range 11-30 --timeout 720") {
		t.Fatalf("放行的那一次参数不对（20 页 × 30 秒 + 120 秒 = 720 秒）：%q", calls)
	}

	if _, err := ConvertToExcel(src, ExcelOptions{}, nil); err != nil {
		t.Fatalf("不开 AI 修复 30 页应放行，得到 err=%v", err)
	}
	calls := readWpsCallLog(t, log)
	if strings.Contains(calls[1], "--ai-fix") || !strings.HasSuffix(calls[1], "--timeout 300") {
		t.Fatalf("ai_fix=false 的参数不对：%s", calls[1])
	}

	t.Setenv("GAEAPROBE_WPS_PAGES", "150")
	if _, err := ConvertToExcel(src, ExcelOptions{}, nil); err == nil ||
		!strings.Contains(err.Error(), "一次最多转 100 页，这次是 150 页") {
		t.Fatalf("不开 AI 修复 150 页应超限报错，得到 err=%v", err)
	}
}

// 图片与装着图片的目录走 photo2excel；目录按文件名自然序，多张图合成一个工作簿。
func TestConvertToExcel_Images(t *testing.T) {
	log := filepath.Join(t.TempDir(), "wpscli.log")
	t.Setenv("GAEAPROBE_WPS_LOG", log)
	useFakeExcelCLI(t, "table", 0)
	dir := t.TempDir()
	for _, name := range []string{"IMG_10.jpg", "IMG_2.png", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	rec, err := ConvertToExcel(dir, ExcelOptions{Pages: "2"}, nil)
	if err != nil {
		t.Fatalf("ConvertToExcel(目录): %v", err)
	}
	if rec.Pages != 2 || !rec.OnlineOCR {
		t.Fatalf("目录输入回执不对：%+v", rec)
	}
	calls := readWpsCallLog(t, log)
	want := "photo2excel " + filepath.Join(dir, "IMG_2.png") + " " + filepath.Join(dir, "IMG_10.jpg") + " -o "
	if len(calls) != 1 || !strings.HasPrefix(calls[0], want) ||
		!strings.HasSuffix(calls[0], ".xlsx --json --merge true --timeout 300") {
		t.Fatalf("目录输入应按自然序合并两张图，得到：%q", calls)
	}

	rec, err = ConvertToExcel(filepath.Join(dir, "IMG_2.png"), ExcelOptions{}, nil)
	if err != nil || rec.Pages != 1 {
		t.Fatalf("单张图：rec=%+v err=%v", rec, err)
	}
	calls = readWpsCallLog(t, log)
	if strings.Contains(calls[1], "--merge") || strings.Contains(calls[1], "--range") {
		t.Fatalf("单张图不该带 --merge / --range：%s", calls[1])
	}
}

// 「success」不可信：产物缺失、0 字节、损坏、没有内容，都如实报错；产物被另存成
// <名>_1 时跟着 wpscli 报的路径走；疑似缺格要数出来并给出行位置。
func TestConvertToExcel_ProductChecks(t *testing.T) {
	useFakeExcelCLI(t, "table", 1)
	src := buildPagePDF(t, 1)

	for _, c := range []struct{ mode, want string }{
		{"empty", "报成功但没有产出 Excel"},
		{"missing", "报成功但没有产出 Excel"},
		{"garbage", "打不开"},
		{"blank", "没有内容"},
	} {
		t.Setenv("GAEAPROBE_WPS_XLSX", c.mode)
		if _, err := ConvertToExcel(src, ExcelOptions{}, nil); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("mode=%s 应报「%s」，得到 err=%v", c.mode, c.want, err)
		}
	}

	t.Setenv("GAEAPROBE_WPS_XLSX", "rename")
	var got string
	if _, err := ConvertToExcel(src, ExcelOptions{}, func(prod string) error {
		got = prod
		return nil
	}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "/out_1.xlsx") {
		t.Fatalf("应跟着 wpscli 终态行报的另存路径走，得到 %q", got)
	}

	t.Setenv("GAEAPROBE_WPS_XLSX", "gap")
	rec, err := ConvertToExcel(src, ExcelOptions{}, nil)
	if err != nil {
		t.Fatalf("gap: %v", err)
	}
	if rec.SuspectCount != 1 || len(rec.Suspects) != 1 || rec.Suspects[0] != "Sheet1!6" {
		t.Fatalf("第 6 行工程量空着应判疑似缺格：%+v", rec)
	}

	boom := errors.New("搬不动")
	if _, err := ConvertToExcel(src, ExcelOptions{}, func(string) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("save 报的错应原样返回，得到 err=%v", err)
	}
}

// 没有可用路线时如实报错，不产出半个文件：账号级失败进静默期、通道关闭、没装 WPS、
// 源文件缺失、格式不支持、参数无效、页范围越界或不连续、读不到页数。
func TestConvertToExcel_Failures(t *testing.T) {
	useFakeExcelCLI(t, "table", 3)
	src := buildPagePDF(t, 3)

	_, err := ConvertToExcel(src, ExcelOptions{Scanned: "yes"}, nil)
	if err == nil || !strings.Contains(err.Error(), "scanned 取值无效") {
		t.Fatalf("scanned 非法取值应报错，得到 err=%v", err)
	}
	for pages, want := range map[string]string{"1,3": "只收一段连续页", "9": "超出这份 PDF 的页数", "0": "超出这份 PDF 的页数"} {
		if _, err := ConvertToExcel(src, ExcelOptions{Pages: pages}, nil); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("pages=%q 应报「%s」，得到 err=%v", pages, want, err)
		}
	}
	if _, err := ConvertToExcel(filepath.Join(t.TempDir(), "nope.pdf"), ExcelOptions{}, nil); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("源文件缺失应透传 not-exist，得到 err=%v", err)
	}
	bad := filepath.Join(t.TempDir(), "x.docx")
	if err := os.WriteFile(bad, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ConvertToExcel(bad, ExcelOptions{}, nil); err == nil || !strings.Contains(err.Error(), "不支持的文件格式") {
		t.Fatalf("不支持的扩展名应报错，得到 err=%v", err)
	}

	// 读不到页数的扫描件：在线识别要显式页范围，如实报错
	t.Setenv("GAEAPROBE_WPS_PAGES", "0")
	if _, err := ConvertToExcel(buildScanStampPDF(t, 3), ExcelOptions{}, nil); err == nil ||
		!strings.Contains(err.Error(), "读不到 PDF 页数") {
		t.Fatalf("读不到页数的扫描件应报错，得到 err=%v", err)
	}

	// 没装 WPS：直接堵查找 seam（findWpsCLI 会回落到 PATH / 安装目录，装了 WPS 的开发机上
	// 不能靠环境变量模拟「没装」）
	orig := wpsCLILookup
	t.Cleanup(func() { wpsCLILookup = orig })
	wpsCLILookup = func() string { return "" }
	if _, err := ConvertToExcel(src, ExcelOptions{}, nil); err == nil || !strings.Contains(err.Error(), "未找到 WPS") {
		t.Fatalf("没装 WPS 应报错，得到 err=%v", err)
	}
	t.Setenv("GAEA_WPS_PDF", "0")
	if _, err := ConvertToExcel(src, ExcelOptions{}, nil); err == nil || !strings.Contains(err.Error(), "已关闭") {
		t.Fatalf("通道关闭应报错，得到 err=%v", err)
	}
}

// 账号级失败（无会员 101）：翻译成可执行的原因，并进全局静默期。
func TestConvertToExcel_AccountFailureSuspends(t *testing.T) {
	t.Setenv("GAEA_WPS_PDF", "1")
	resetWpsState(t)
	t.Setenv("GAEAPROBE_WPS_XLSX", "table")
	cli := fakeWpsCLIPath(t, 101, 1) // pdf2excel 一律退出 101
	t.Setenv("GAEA_WPSCLI", cli)
	src := buildPagePDF(t, 1)
	if _, err := ConvertToExcel(src, ExcelOptions{}, nil); err == nil || !strings.Contains(err.Error(), "会员") {
		t.Fatalf("应报无会员权限，得到 err=%v", err)
	}
	if !wpsSuspended() {
		t.Fatal("账号级失败后应进入静默期")
	}
	if _, err := ConvertToExcel(src, ExcelOptions{}, nil); err == nil || !strings.Contains(err.Error(), "静默期") {
		t.Fatalf("静默期内应直接报错、不再拉起 wpscli，得到 err=%v", err)
	}
}

// gaea 侧上限先到：如实报成超时（被杀的 wpscli 在 Windows 上是退出码 1，按码翻译会
// 误报成「转换失败」）。
func TestWpsRunTimed_GaeaSideDeadline(t *testing.T) {
	t.Setenv("GAEAPROBE_WPS_STUB", "1")
	t.Setenv("GAEAPROBE_WPS_SLEEP", "5000")
	t.Setenv("GAEAPROBE_WPS_LOG", "")
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("取测试二进制路径: %v", err)
	}
	start := time.Now()
	_, _, _, err = wpsRunTimed(500*time.Millisecond, exe, "pdf2excel", "x.pdf", "-o", "y.xlsx", "--json")
	if err == nil || !strings.Contains(err.Error(), "仍未结束，已终止（wpscli pdf2excel）") ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("gaea 侧超时应如实报超时，得到 err=%v", err)
	}
	if time.Since(start) > 4*time.Second {
		t.Fatalf("上限到点应立即返回，实际 %s", time.Since(start))
	}
}

func TestWpsExcelTimeout(t *testing.T) {
	cases := []struct {
		pages       int
		online, ai  bool
		wantSeconds int
	}{
		{0, false, false, 300}, // 页数未知：下限
		{12, false, false, 300},
		{839, false, false, 2637}, // 839 × 3 + 120
		{20, true, true, 720},     // 20 × 30 + 120
		{20, true, false, 300},
		{100, true, false, 720},
	}
	for _, c := range cases {
		if got := int(wpsExcelTimeout(c.pages, c.online, c.ai) / time.Second); got != c.wantSeconds {
			t.Errorf("wpsExcelTimeout(%d,%v,%v) = %ds, want %ds", c.pages, c.online, c.ai, got, c.wantSeconds)
		}
	}
}

func TestNaturalLess(t *testing.T) {
	ordered := []string{"IMG_2.png", "IMG_10.png", "img_2.png", "报价单2.jpg", "报价单（第2张）.jpg", "报价单（第10张）.jpg"}
	for i := 0; i+1 < len(ordered); i++ {
		if !naturalLess(ordered[i], ordered[i+1]) || naturalLess(ordered[i+1], ordered[i]) {
			t.Errorf("naturalLess(%q, %q) 顺序不对", ordered[i], ordered[i+1])
		}
	}
	if naturalLess("a", "a") || naturalLess("ab", "a") || !naturalLess("a", "ab") {
		t.Error("前缀相同按长度：短的在前")
	}
}

// 疑似缺格：主键列=数字格最多的一列；数据行=主键列里是数字的行；数据行里过半有值的
// 其余数字列，缺了就计。表题 / 表头 / 合计行（主键列是文字）不算数据行。
func TestXlsxSuspectRows(t *testing.T) {
	const (
		seq  = 1 << 0 // 序号（数值）
		code = 1 << 1 // 项目编码（文本数字）
		name = 1 << 2
		qty  = 1 << 3 // 工程量（数值）
		amt  = 1 << 4 // 金额（数值，投标前全空）
	)
	row := func(nonEmpty, numberLike uint64) xlsxRowMask { return xlsxRowMask{nonEmpty, numberLike} }
	masks := []xlsxRowMask{
		row(seq|code|name|qty|amt, 0),                // 1 表头
		row(seq|code|name|qty|amt, seq|code|qty|amt), // 2
		row(seq|code|name|qty, seq|code|qty),         // 3
		row(seq|code|name|amt, seq|code),             // 4 ← 工程量空着
		row(seq|code|name|qty, seq|code|qty),         // 5
		row(seq|code|name|qty, seq|code|qty),         // 6
		row(seq|name, 0),                             // 7 合计（主键列是文字，不算数据行）
	}
	if got := xlsxSuspectRows(masks); len(got) != 1 || got[0] != 4 {
		t.Fatalf("只有第 4 行疑似缺格，得到 %v", got)
	}
	// 金额列只在 1 行有值：稀稀拉拉的可选数字列不算
	if got := xlsxSuspectRows(masks[:6]); len(got) != 1 {
		t.Fatalf("可选数字列不该触发疑似缺格，得到 %v", got)
	}
	// 数据行不足 3 行、或除主键列外没有别的数字列：不判
	if got := xlsxSuspectRows(masks[1:3]); got != nil {
		t.Fatalf("数据行不足不判，得到 %v", got)
	}
	textOnly := []xlsxRowMask{row(name, 0), row(name, 0), row(name, 0)}
	if got := xlsxSuspectRows(textOnly); got != nil {
		t.Fatalf("没有数字列不判，得到 %v", got)
	}
}
