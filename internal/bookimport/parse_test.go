package bookimport

import (
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// ── 编码链 ────────────────────────────────────────────────────

func TestDecode_UTF8PlainAndBOM(t *testing.T) {
	if got, name := Decode([]byte("第一章 测试内容")); got != "第一章 测试内容" || name != "utf-8" {
		t.Fatalf("UTF-8 直通失败: %q %q", got, name)
	}
	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte("第一章 带 BOM")...)
	if got, name := Decode(withBOM); got != "第一章 带 BOM" || name != "utf-8-sig" {
		t.Fatalf("UTF-8 BOM 识别失败: %q %q", got, name)
	}
	if got, name := Decode([]byte("plain ascii")); got != "plain ascii" || name != "utf-8" {
		t.Fatalf("纯 ASCII 失败: %q %q", got, name)
	}
}

func TestDecode_GB18030(t *testing.T) {
	src := "第一章 测试内容：他推开茶馆的门，雨还在下。"
	raw, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte(src))
	if err != nil {
		t.Fatalf("构造 GB18030 样本失败: %v", err)
	}
	got, name := Decode(raw)
	if got != src || name != "gb18030" {
		t.Fatalf("GB18030 解码失败: %q %q", got, name)
	}
}

func TestDecode_Big5_NotShadowedByGBK(t *testing.T) {
	// GB18030/GBK 解码 Big5 字节不报 error 而是产出含替换符的乱码；
	// 契约要求这种误判被跳过并落到 big5（否则繁体中文书整篇乱码）。
	src := "第一章 測試內容：他推開茶館的門，雨還在下。"
	raw, err := traditionalchinese.Big5.NewEncoder().Bytes([]byte(src))
	if err != nil {
		t.Fatalf("构造 Big5 样本失败: %v", err)
	}
	got, name := Decode(raw)
	if got != src || name != "big5" {
		t.Fatalf("Big5 解码失败: %q %q", got, name)
	}
}

func TestDecode_UTF16LEBOM(t *testing.T) {
	src := "第一章 记事本另存"
	raw := []byte{0xFF, 0xFE}
	for _, r := range src {
		u := uint16(r)
		raw = append(raw, byte(u), byte(u>>8))
	}
	got, name := Decode(raw)
	if got != src || name != "utf-16le" {
		t.Fatalf("UTF-16LE 解码失败: %q %q", got, name)
	}
}

func TestDecode_FallbackNeverPanics(t *testing.T) {
	got, name := Decode([]byte{0x80, 0x81, 0x82, 0x83})
	if name != "utf-8(ignore)" {
		t.Fatalf("兜底编码名 = %q, want utf-8(ignore)", name)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("兜底结果必须是合法 UTF-8: %q", got)
	}
}

// ── 清洗 ──────────────────────────────────────────────────────

func TestClean_OrderSensitiveRules(t *testing.T) {
	in := "第一行   \r\n\r\n\r\n\r\n第二行\u3000结尾\r第三行\uFEFF"
	got := Clean(in)
	if strings.Contains(got, "\r") {
		t.Fatalf("CR 未归一: %q", got)
	}
	if strings.Contains(got, "\uFEFF") {
		t.Fatalf("BOM 未清除: %q", got)
	}
	if !strings.Contains(got, "第二行  结尾") {
		t.Fatalf("全角空格应换成两个半角空格: %q", got)
	}
	if strings.Contains(got, "\n\n\n\n") {
		t.Fatalf("≥4 连续空行未压缩: %q", got)
	}
	if strings.HasSuffix(got, " ") || strings.HasSuffix(got, "\n") {
		t.Fatalf("首尾未 TrimSpace: %q", got)
	}
	if !strings.Contains(got, "第一行\n") {
		t.Fatalf("行尾空白未删除: %q", got)
	}
}

// ── 三级切分 ──────────────────────────────────────────────────

func TestSplit_StrongHeadings(t *testing.T) {
	text := "《测试之书》\n作者：佚名\n\n第一章 初遇\n\n雨夜，她推开茶馆的门。\n\n第二章 风波\n\n城外的消息传开了。\n\n第三章 归途\n\n灯火渐明。"
	chs, strategy := Split(text)
	if strategy != StrategyStrong {
		t.Fatalf("策略 = %q, want strong", strategy)
	}
	if len(chs) != 3 {
		t.Fatalf("章数 = %d, want 3（书名/作者不足 200 字并入首章）", len(chs))
	}
	if chs[0].Title != "第一章 初遇" || !strings.Contains(chs[0].Content, "《测试之书》") ||
		!strings.Contains(chs[0].Content, "雨夜") {
		t.Fatalf("首章应含前置书名与正文: %+v", chs[0])
	}
	if chs[2].Title != "第三章 归途" || !strings.Contains(chs[2].Content, "灯火渐明") {
		t.Fatalf("末章解析异常: %+v", chs[2])
	}
}

func TestSplit_MarkdownHeadings(t *testing.T) {
	chs, strategy := Split("# 卷一\n\n内容甲\n\n## 卷二\n\n内容乙")
	if strategy != StrategyStrong {
		t.Fatalf("策略 = %q, want strong（markdown 视同强标题）", strategy)
	}
	if len(chs) != 2 || chs[0].Title != "卷一" || chs[1].Title != "卷二" {
		t.Fatalf("markdown 标题切分异常: %+v", chs)
	}
}

func TestSplit_WeakHeadings(t *testing.T) {
	chs, strategy := Split("开篇的话\n\n雨夜\n\n她推开门。\n\n晨光\n\n城外的消息传开了。")
	if strategy != StrategyWeak {
		t.Fatalf("策略 = %q, want weak", strategy)
	}
	if len(chs) != 3 {
		t.Fatalf("弱标题应切出 3 章: %+v", chs)
	}
	if chs[1].Title != "雨夜" || !strings.Contains(chs[1].Content, "她推开门") {
		t.Fatalf("弱标题章内容异常: %+v", chs[1])
	}
}

func TestSplit_WeakHeadingRejectsPunctuatedAndLongLines(t *testing.T) {
	text := "她推开门，雨还在下。\n\n这一行没有标点但是长度明显超过二十五个字符所以不能当作章节标题\n\n正文继续。"
	chs, _ := Split(text)
	for _, ch := range chs {
		if strings.HasPrefix(ch.Title, "她推开门") || strings.HasPrefix(ch.Title, "这一行没有标点") {
			t.Fatalf("正文行被误判成标题: %+v", chs)
		}
	}
}

func TestSplit_WindowFallback(t *testing.T) {
	sentence := "他推开门，雨还在下，屋檐下的灯影在水面上摇晃了很久很久。"
	var sb strings.Builder
	for utf8.RuneCountInString(sb.String()) < 8000 {
		sb.WriteString(sentence)
		sb.WriteString("。")
	}
	chs, strategy := Split(sb.String())
	if strategy != StrategyWindow {
		t.Fatalf("策略 = %q, want window", strategy)
	}
	if len(chs) < 2 {
		t.Fatalf("8000 字无标题应切出 ≥2 章: %d", len(chs))
	}
	if chs[0].Title != "第1章" || chs[1].Title != "第2章" {
		t.Fatalf("窗口切分标题应伪造为第N章: %q %q", chs[0].Title, chs[1].Title)
	}
	for _, ch := range chs {
		if n := utf8.RuneCountInString(ch.Content); n > maxWindowRunes {
			t.Fatalf("窗口章超过上界 %d: %d", maxWindowRunes, n)
		}
	}
}

func TestSplit_SingleShortNoHeading(t *testing.T) {
	chs, strategy := Split("没有章节标记的一段文字")
	if strategy != StrategySingle || len(chs) != 1 || chs[0].Title != "全文" {
		t.Fatalf("短文本无标题应归为单章全文: %q %+v", strategy, chs)
	}
	if chs, _ := Split("   \n\n  "); len(chs) != 0 {
		t.Fatalf("空白文本应无章节: %+v", chs)
	}
}

func TestSplit_PrefaceKeptWhenLongEnough(t *testing.T) {
	text := strings.Repeat("雨", 250) + "\n\n第一章 初遇\n\n正文甲。"
	chs, _ := Split(text)
	if len(chs) != 2 || chs[0].Title != "前言" || utf8.RuneCountInString(chs[0].Content) != 250 {
		t.Fatalf("前置内容 ≥200 字应成「前言」章: %d %+v", len(chs), chs)
	}
}

// ── 报告与告警 ────────────────────────────────────────────────

func TestParse_ReportCarriesEncodingStrategyAndCounts(t *testing.T) {
	src := "第一章 甲\n\n" + strings.Repeat("内容", 200) + "\n\n第二章 乙\n\n" + strings.Repeat("正文", 200)
	raw, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	res := Parse(raw, ParseOptions{})
	if res.Report.Encoding != "gb18030" || res.Report.SplitStrategy != StrategyStrong {
		t.Fatalf("报告编码/策略异常: %+v", res.Report)
	}
	if res.Report.TotalChapters != 2 || res.Report.SelectedChapters != 2 || len(res.Chapters) != 2 {
		t.Fatalf("报告章数异常: %+v", res.Report)
	}
}

func TestParse_Warnings(t *testing.T) {
	text := "第一章 短章\n\n" + strings.Repeat("短", 100) +
		"\n\n第二章 重复\n\n" + strings.Repeat("长", 400) +
		"\n\n第二章 重复\n\n" + strings.Repeat("长", 400) +
		"\n\n第三章 超长\n\n" + strings.Repeat("长", longChapterRunes+1)
	res := Parse([]byte(text), ParseOptions{})
	codes := map[string]int{}
	for _, w := range res.Report.Warnings {
		codes[w.Code]++
		if w.Message == "" || w.Level == "" {
			t.Fatalf("告警缺少说明或级别: %+v", w)
		}
	}
	if codes[WarningChapterTooShort] != 1 {
		t.Fatalf("应有 1 条过短告警: %+v", res.Report.Warnings)
	}
	if codes[WarningChapterTooLong] != 1 {
		t.Fatalf("应有 1 条过长告知: %+v", res.Report.Warnings)
	}
	if codes[WarningDuplicateTitle] != 1 {
		t.Fatalf("重复标题只该报一次: %+v", res.Report.Warnings)
	}
}

func TestApplyExtractMode_TailSelection(t *testing.T) {
	var chs []ParsedChapter
	for i := 1; i <= 30; i++ {
		chs = append(chs, ParsedChapter{Title: "第" + itoa(i) + "章", Content: strings.Repeat("文", 400)})
	}
	kept, warns := applyExtractMode(chs, ParseOptions{ExtractMode: ExtractTail, TailChapterCount: 10})
	if len(kept) != 10 || kept[0].Title != "第21章" || kept[9].Title != "第30章" {
		t.Fatalf("tail=10 应取末尾 10 章: %d %q", len(kept), kept[0].Title)
	}
	if len(warns) != 1 || warns[0].Code != WarningTrimmedForMode {
		t.Fatalf("应有一条裁剪告知: %+v", warns)
	}
	if kept, _ := applyExtractMode(chs, ParseOptions{ExtractMode: ExtractTail, TailChapterCount: 7}); len(kept) != 10 {
		t.Fatalf("tail=7 应向上取整到 10: %d", len(kept))
	}
	if kept, warns := applyExtractMode(chs, ParseOptions{ExtractMode: ExtractTail, TailChapterCount: 55}); len(kept) != 30 || warns != nil {
		t.Fatalf("tail=55 应降级 full: %d %+v", len(kept), warns)
	}
	if kept, warns := applyExtractMode(chs, ParseOptions{ExtractMode: ExtractTail, TailChapterCount: 0}); len(kept) != 30 || warns != nil {
		t.Fatalf("tail=0 应不裁: %d %+v", len(kept), warns)
	}
}

func TestParse_TailModeReportCounts(t *testing.T) {
	var sb strings.Builder
	for i := 1; i <= 12; i++ {
		sb.WriteString("第" + itoa(i) + "章 标题\n\n")
		sb.WriteString(strings.Repeat("正文", 200))
		sb.WriteString("\n\n")
	}
	res := Parse([]byte(sb.String()), ParseOptions{ExtractMode: ExtractTail, TailChapterCount: 5})
	if res.Report.TotalChapters != 12 || res.Report.SelectedChapters != 5 || len(res.Chapters) != 5 {
		t.Fatalf("tail 报告计数异常: %+v", res.Report)
	}
}
