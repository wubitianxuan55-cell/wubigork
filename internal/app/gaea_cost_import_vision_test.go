package app

// 阶段 5 T5-5a（E1）成本库进料闭环测试：PDF/图片报价单本地识别入表
// （GaeaCostImportVisionPreview）与供应商比价（GaeaCostCompare）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	gconfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/pricefeed"
)

// newVisionTestApp 装配隔离 App：临时 HOME → 临时 Hephaestus.db，并注入成本库
// override（避免触碰真实用户库）。
func newVisionTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	gdb := db.GetDatabase(gconfig.MemoryUserDir())
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(gconfig.MemoryUserDir()) })
	SetCostStoreForTest(cost.Open(gdb))
	t.Cleanup(ResetCostStoreForTest)
	return &App{}
}

// injectVisionExtract 注入 PDF 文本提取函数并在测试结束后还原。
func injectVisionExtract(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	orig := visionExtractPDF
	visionExtractPDF = fn
	t.Cleanup(func() { visionExtractPDF = orig })
}

// injectVisionOCR 注入本地 OCR 函数并在测试结束后还原。
func injectVisionOCR(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	orig := visionOCRImage
	visionOCRImage = fn
	t.Cleanup(func() { visionOCRImage = orig })
}

// visionQuoteTable 报价单表格文本（TSV，供 OCR/PDF 提取 mock 复用）。
const visionQuoteTable = "序号\t材料名称\t规格型号\t单位\t单价(元)\n" +
	"1\tHP300 高频液压振动锤\t300kW\t台班\t3,200.00\n" +
	"2\tP.O 42.5 水泥\t\t吨\t480 元\n"

// ─── 规则解析纯函数 ───────────────────────────────────────────

func TestVisionParseTableTSV(t *testing.T) {
	cols, rows, ok := visionParseTable(visionQuoteTable)
	if !ok || len(rows) != 2 {
		t.Fatalf("ok=%v rows=%d, want 2", ok, len(rows))
	}
	if len(cols) != 5 || cols[1] != "材料名称" {
		t.Errorf("columns wrong: %v", cols)
	}
	r0 := rows[0]
	if r0.Title != "HP300 高频液压振动锤" || r0.Spec != "300kW" || r0.Unit != "台班" || r0.Price != 3200 {
		t.Errorf("row0 wrong: %+v", r0)
	}
	if r1 := rows[1]; r1.Title != "P.O 42.5 水泥" || r1.Price != 480 {
		t.Errorf("row1 wrong: %+v", r1)
	}
}

func TestVisionParseTableMarkdownPipe(t *testing.T) {
	text := "| 序号 | 材料名称 | 规格型号 | 单位 | 单价(元) |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| 1 | HP300 高频液压振动锤 | 300kW | 台班 | 3,200.00 |\n"
	_, rows, ok := visionParseTable(text)
	if !ok || len(rows) != 1 {
		t.Fatalf("ok=%v rows=%d", ok, len(rows))
	}
	if r := rows[0]; r.Title != "HP300 高频液压振动锤" || r.Price != 3200 || r.Unit != "台班" {
		t.Errorf("pipe row wrong: %+v", r)
	}
}

func TestVisionParseTableMultiSpace(t *testing.T) {
	text := "序号  材料名称  规格型号  单位  单价(元)\n" +
		"1  HP300 高频液压振动锤  300kW  台班  3,200.00\n"
	_, rows, ok := visionParseTable(text)
	if !ok || len(rows) != 1 {
		t.Fatalf("ok=%v rows=%d", ok, len(rows))
	}
	if r := rows[0]; r.Title != "HP300 高频液压振动锤" || r.Price != 3200 {
		t.Errorf("multispace row wrong: %+v", r)
	}
}

// 无表头/无单价列 → 回退整行解析。
func TestVisionParseQuotationFallsBackToLines(t *testing.T) {
	rows, cols := visionParseQuotation("热轧光圆钢筋 HPB300 Φ12 ￥3181.00\n水泥 P.O42.5 480元/吨")
	if len(rows) != 2 {
		t.Fatalf("rows=%d, want 2: %+v", len(rows), rows)
	}
	if rows[0].Title != "热轧光圆钢筋 HPB300 Φ12" || rows[0].Price != 3181 {
		t.Errorf("row0 wrong: %+v", rows[0])
	}
	if rows[1].Title != "水泥 P.O42.5" || rows[1].Price != 480 {
		t.Errorf("row1 wrong: %+v", rows[1])
	}
	if cols != nil {
		t.Errorf("fallback 不应返回表头列: %v", cols)
	}
}

// 整行回退：序号前缀剥离 + 行首序号不被误当价格 + 页码行跳过。
func TestVisionParseFallbackLineCases(t *testing.T) {
	rows := visionParseFallback("1. HP300 高频液压振动锤 3200 元\n第 1 页 共 3 页\n水泥 480元/吨")
	if len(rows) != 2 {
		t.Fatalf("rows=%d, want 2: %+v", len(rows), rows)
	}
	if rows[0].Title != "HP300 高频液压振动锤" || rows[0].Price != 3200 {
		t.Errorf("row0 wrong: %+v", rows[0])
	}
	if rows[1].Title != "水泥" || rows[1].Price != 480 {
		t.Errorf("row1 wrong: %+v", rows[1])
	}
}

func TestVisionParsePriceCases(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"3,200.00", 3200, true},
		{"3200 元", 3200, true},
		{"￥3181.00", 3181, true},
		{"3200元/台班", 3200, true},
		{"480 元/吨", 480, true},
		{"HP300", 0, false},
		{"", 0, false},
		{"—", 0, false},
		// "1." 这类序号在行级（visionLineNamePrice）被跳过，价格解析层仍可解析。
		{"1.", 1, true},
	}
	for _, c := range cases {
		got, ok := visionParsePrice(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("visionParsePrice(%q) = (%v,%v), want (%v,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestVisionStripIndexCases(t *testing.T) {
	cases := []struct{ in, want string }{
		{"1. HP300 高频液压振动锤", "HP300 高频液压振动锤"},
		{"2、水泥", "水泥"},
		{"（1）热轧光圆钢筋", "热轧光圆钢筋"},
		{"300kW 发电机", "300kW 发电机"}, // 非序号数字前缀保留
		{"HP300", "HP300"},
	}
	for _, c := range cases {
		if got := visionStripIndex(c.in); got != c.want {
			t.Errorf("visionStripIndex(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestVisionLineNamePrice(t *testing.T) {
	name, price, ok := visionLineNamePrice("热轧光圆钢筋 HPB300 Φ12 ￥3181.00")
	if !ok || name != "热轧光圆钢筋 HPB300 Φ12" || price != 3181 {
		t.Errorf("got (%q,%v,%v)", name, price, ok)
	}
	// 价格在行首：取其后第一个非价格词作为名称片段。
	name, price, ok = visionLineNamePrice("3200 HP300 高频液压振动锤")
	if !ok || name != "HP300" || price != 3200 {
		t.Errorf("leading price got (%q,%v,%v)", name, price, ok)
	}
	// 无价格行。
	if _, _, ok := visionLineNamePrice("材料报价清单"); ok {
		t.Error("无价格行不应命中")
	}
}

// ─── GaeaCostImportVisionPreview（mock 注入）──────────────────

// 图片 → 本地 OCR → source=image；AI 不可用（cfg 为 nil）降级注明。
func TestGaeaCostImportVisionPreviewImage(t *testing.T) {
	newVisionTestApp(t)
	img := filepath.Join(t.TempDir(), "quote.png")
	if err := os.WriteFile(img, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	injectVisionOCR(t, func(string) (string, error) { return visionQuoteTable, nil })

	pv, err := (&App{}).GaeaCostImportVisionPreview(img)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if pv.Source != visionSourceImage {
		t.Errorf("source = %q, want %q", pv.Source, visionSourceImage)
	}
	if len(pv.Rows) != 2 || pv.Rows[0].Title != "HP300 高频液压振动锤" || pv.Rows[0].Price != 3200 {
		t.Errorf("rows wrong: %+v", pv.Rows)
	}
	if pv.AIUsed {
		t.Error("AI 不可用不应标记 aiUsed")
	}
	if !strings.Contains(pv.Message, "未做 AI 归一化") {
		t.Errorf("message 应注明未做 AI 归一化: %s", pv.Message)
	}
}

// 文本型 PDF → source=pdf_text。
func TestGaeaCostImportVisionPreviewPDFText(t *testing.T) {
	newVisionTestApp(t)
	pdf := filepath.Join(t.TempDir(), "quote.pdf")
	if err := os.WriteFile(pdf, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	injectVisionExtract(t, func(string) (string, error) { return visionQuoteTable, nil })
	injectVisionOCR(t, func(string) (string, error) { return "", nil })

	pv, err := (&App{}).GaeaCostImportVisionPreview(pdf)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if pv.Source != visionSourcePDFText {
		t.Errorf("source = %q, want %q", pv.Source, visionSourcePDFText)
	}
	if len(pv.Rows) != 2 {
		t.Errorf("rows=%d, want 2", len(pv.Rows))
	}
}

// 扫描件 PDF：ConvertLimit 已 OCR（输出带 OCR 标记前缀）→ source=pdf_scan。
func TestGaeaCostImportVisionPreviewPDFScanOCRMarker(t *testing.T) {
	newVisionTestApp(t)
	pdf := filepath.Join(t.TempDir(), "scan.pdf")
	if err := os.WriteFile(pdf, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	injectVisionExtract(t, func(string) (string, error) {
		return "（以下内容由 OCR 识别，可能存在误差）\n\n" + visionQuoteTable, nil
	})

	pv, err := (&App{}).GaeaCostImportVisionPreview(pdf)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if pv.Source != visionSourcePDFScan {
		t.Errorf("source = %q, want %q", pv.Source, visionSourcePDFScan)
	}
}

// 扫描件 PDF：文本极少（提取不可用）→ 本地 OCR 兜底 → source=pdf_scan。
func TestGaeaCostImportVisionPreviewPDFScanFallbackOCR(t *testing.T) {
	newVisionTestApp(t)
	pdf := filepath.Join(t.TempDir(), "scan2.pdf")
	if err := os.WriteFile(pdf, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	injectVisionExtract(t, func(string) (string, error) { return "乱码??", nil })
	injectVisionOCR(t, func(string) (string, error) { return visionQuoteTable, nil })

	pv, err := (&App{}).GaeaCostImportVisionPreview(pdf)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if pv.Source != visionSourcePDFScan {
		t.Errorf("source = %q, want %q", pv.Source, visionSourcePDFScan)
	}
	if len(pv.Rows) != 2 {
		t.Errorf("rows=%d, want 2", len(pv.Rows))
	}
}

// 与既有条目匹配（复用 costimport.MatchRows）→ 覆盖更新提示。
func TestGaeaCostImportVisionPreviewExistingMatch(t *testing.T) {
	a := newVisionTestApp(t)
	store := a.hubCostStore()
	if err := store.Save(cost.Entry{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Source: "市场询价", Status: "现行"}); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(t.TempDir(), "quote.png")
	if err := os.WriteFile(img, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	injectVisionOCR(t, func(string) (string, error) { return visionQuoteTable, nil })

	pv, err := a.GaeaCostImportVisionPreview(img)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	r0 := pv.Rows[0]
	if r0.ExistingName != "hp300" || !strings.Contains(r0.MatchNote, "覆盖更新") {
		t.Errorf("期望命中既有条目并提示覆盖更新: %+v", r0)
	}
	if r1 := pv.Rows[1]; r1.MatchNote != "新增" {
		t.Errorf("row1 应为新增: %+v", r1)
	}
}

// AI 不可用降级：sensitive_local 开启但本地引擎缺失 → 规则解析 + 注明。
func TestGaeaCostImportVisionPreviewAIDegrades(t *testing.T) {
	newVisionTestApp(t)
	a := &App{core: &core{cfg: &config.Config{SensitiveLocal: true}}} // engineMgr 为 nil
	img := filepath.Join(t.TempDir(), "quote.png")
	if err := os.WriteFile(img, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	injectVisionOCR(t, func(string) (string, error) { return visionQuoteTable, nil })

	pv, err := a.GaeaCostImportVisionPreview(img)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if pv.AIUsed {
		t.Error("本地引擎缺失时不应标记 aiUsed")
	}
	if !strings.Contains(pv.Message, "未做 AI 归一化") {
		t.Errorf("message 应注明未做 AI 归一化: %s", pv.Message)
	}
	if len(pv.Rows) != 2 {
		t.Errorf("降级后应保留规则解析结果: %d 行", len(pv.Rows))
	}
}

// 文件不存在 / 不支持的扩展名 → 明确报错。
func TestGaeaCostImportVisionPreviewErrors(t *testing.T) {
	newVisionTestApp(t)
	if _, err := (&App{}).GaeaCostImportVisionPreview(filepath.Join(t.TempDir(), "missing.pdf")); err == nil {
		t.Error("缺失文件应报错")
	}
	txt := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(txt, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (&App{}).GaeaCostImportVisionPreview(txt); err == nil || !strings.Contains(err.Error(), "暂不支持") {
		t.Errorf("不支持扩展名应报错: %v", err)
	}
}

// ─── GaeaCostCompare（三源聚合）───────────────────────────────

func seedCompareData(t *testing.T, a *App) {
	t.Helper()
	// 成本库现价。
	store := a.hubCostStore()
	if err := store.Save(cost.Entry{Name: cost.SlugName("HP300 高频液压振动锤"), Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Source: "市场询价", Status: "现行"}); err != nil {
		t.Fatal(err)
	}
	// 价格源抓取候选（pending 记录）。
	ps := a.hubPriceStore()
	if err := ps.SaveFetch(pricefeed.FetchRecord{
		ID:         "fetch-1",
		SourceID:   "src-1",
		SourceName: "造价信息网A",
		URL:        "https://example.com/price",
		Period:     "2026年3月",
		FetchedAt:  "2026-03-10T00:00:00Z",
		Status:     "pending",
		Candidates: []pricefeed.Candidate{{Title: "HP300 高频液压振动锤", Spec: "300kW", Unit: "台班", Price: 3500, ExistingName: cost.SlugName("HP300 高频液压振动锤"), ExistingPrice: 3200, Status: "更新"}},
	}); err != nil {
		t.Fatal(err)
	}
	// 价格历史快照。
	if err := ps.AddHistory(pricefeed.History{
		Name: cost.SlugName("HP300 高频液压振动锤"), Title: "HP300 高频液压振动锤", Unit: "台班", Price: 3000,
		Source: "造价信息网", Period: "2026年2月", FetchedAt: "2026-02-01T00:00:00Z", Note: "价格源更新",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGaeaCostCompareThreeSources(t *testing.T) {
	a := newVisionTestApp(t)
	seedCompareData(t, a)

	rows, err := a.GaeaCostCompare("HP300 高频液压振动锤")
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows=%d, want 3: %+v", len(rows), rows)
	}
	kinds := map[string]bool{}
	for _, r := range rows {
		kinds[r.Kind] = true
		switch r.Kind {
		case compareKindCurrent:
			if r.Price != 3200 || r.Period != "现行" || r.Source != "市场询价" || r.DiffPct != 0 {
				t.Errorf("current row wrong: %+v", r)
			}
		case compareKindFetch:
			if r.Price != 3500 || r.Source != "造价信息网A" || r.Period != "2026年3月" || r.FetchedAt != "2026-03-10T00:00:00Z" {
				t.Errorf("fetch row wrong: %+v", r)
			}
			if r.DiffPct != 9.38 { // (3500-3200)/3200*100 = 9.375 → 9.38
				t.Errorf("fetch diffPct = %v, want 9.38", r.DiffPct)
			}
		case compareKindHistory:
			if r.Price != 3000 || r.Source != "造价信息网" || r.Period != "2026年2月" {
				t.Errorf("history row wrong: %+v", r)
			}
			// 与 pricefeed.round2 一致的截断取整（-625.0+0.5 截断为 -624）。
			if r.DiffPct != -6.24 { // (3000-3200)/3200*100 = -6.25
				t.Errorf("history diffPct = %v, want -6.24", r.DiffPct)
			}
		}
	}
	for _, k := range []string{compareKindCurrent, compareKindFetch, compareKindHistory} {
		if !kinds[k] {
			t.Errorf("缺少 %s 来源", k)
		}
	}
}

// 无现价时 diffPct 保持 0（不报错）。
func TestGaeaCostCompareNoCurrent(t *testing.T) {
	a := newVisionTestApp(t)
	ps := a.hubPriceStore()
	if err := ps.SaveFetch(pricefeed.FetchRecord{
		ID: "fetch-x", SourceID: "src-x", SourceName: "信息网B", URL: "https://e.com",
		Period: "2026年4月", FetchedAt: "2026-04-01T00:00:00Z", Status: "pending",
		Candidates: []pricefeed.Candidate{{Title: "水泥", Price: 500}},
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := a.GaeaCostCompare("水泥")
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(rows) != 1 || rows[0].Kind != compareKindFetch || rows[0].DiffPct != 0 {
		t.Errorf("rows wrong: %+v", rows)
	}
}

// 按 fetchedAt 倒序（无现价干扰，时间确定）。
func TestGaeaCostCompareSortDesc(t *testing.T) {
	a := newVisionTestApp(t)
	ps := a.hubPriceStore()
	base := pricefeed.FetchRecord{SourceID: "src-s", SourceName: "信息网", URL: "https://e.com", Status: "pending",
		Candidates: []pricefeed.Candidate{{Title: "水泥", Price: 500}},
	}
	base.ID, base.Period, base.FetchedAt = "fetch-1", "2026年1月", "2026-01-05T00:00:00Z"
	if err := ps.SaveFetch(base); err != nil {
		t.Fatal(err)
	}
	base.ID, base.Period, base.FetchedAt = "fetch-2", "2026年3月", "2026-03-05T00:00:00Z"
	if err := ps.SaveFetch(base); err != nil {
		t.Fatal(err)
	}
	rows, err := a.GaeaCostCompare("水泥")
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].Period != "2026年3月" || rows[1].Period != "2026年1月" {
		t.Errorf("应按 fetchedAt 倒序: %+v", rows)
	}
}

// 匹配不到 → 空数组（非 nil，不报错）。
func TestGaeaCostCompareNoMatch(t *testing.T) {
	a := newVisionTestApp(t)
	seedCompareData(t, a)
	rows, err := a.GaeaCostCompare("不存在的材料")
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if rows == nil || len(rows) != 0 {
		t.Errorf("期望空数组（非 nil），got %#v", rows)
	}
}

// 空查询 → 报错。
func TestGaeaCostCompareEmptyQuery(t *testing.T) {
	a := newVisionTestApp(t)
	if _, err := a.GaeaCostCompare("  "); err == nil {
		t.Error("空查询应报错")
	}
}

func TestVisionTitleMatch(t *testing.T) {
	cases := []struct{ title, query string; want bool }{
		{"HP300 高频液压振动锤", "HP300 高频液压振动锤", true},
		{"HP300 高频液压振动锤", "HP300", true},
		{"HP300", "HP300 高频液压振动锤", true},
		{"水泥", "钢材", false},
		{"", "水泥", false},
	}
	for _, c := range cases {
		if got := visionTitleMatch(c.title, c.query); got != c.want {
			t.Errorf("visionTitleMatch(%q,%q) = %v, want %v", c.title, c.query, got, c.want)
		}
	}
}

// 时间格式/解析辅助。
func TestVisionParseTime(t *testing.T) {
	ts := "2026-03-10T00:00:00Z"
	if got := visionParseTime(ts); got.IsZero() || got.Format(time.RFC3339) != ts {
		t.Errorf("parse %q = %v", ts, got)
	}
	if got := visionParseTime("bad"); !got.IsZero() {
		t.Errorf("bad time 应返回零值: %v", got)
	}
}