package app

// 阶段 5 T5-5a（任务 E1）：成本库进料闭环——PDF/图片报价单本地识别入表 + 供应商比价。
//
// A 部分 GaeaCostImportVisionPreview：按扩展名分流（.pdf 文本提取 / 扫描件与
// 图片走本地 OCR），提取文本 → 规则解析为候选成本条目（表格线优先，整行回退）→
// 可选 AI 字段归一化（sensitive_local 开启时走本地通道，不可用则降级只做规则
// 解析）→ 复用 costimport.MatchRows 做既有匹配；落地沿用 GaeaCostImportApply，
// 「无确认不落库」纪律不变。
//
// B 部分 GaeaCostCompare：聚合 cost_entries 现价 / price_fetch 抓取候选 /
// cost_price_history 历史快照，输出供应商比价（相对现价的单期跳幅）。

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costimport"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/office/docmd"
)

// 报价单识别来源标记（CostImportPreview.Source 的 json 取值）。
const (
	visionSourcePDFText = "pdf_text" // 文本型 PDF（直接提取）
	visionSourcePDFScan = "pdf_scan" // 扫描件 PDF（OCR 识别）
	visionSourceImage   = "image"    // 图片（OCR 识别）
)

// visionExtractPDF / visionOCRImage 是可注入的文本提取/OCR 函数（测试替换用）。
// 默认走 docmd 本地链路：ConvertLimit 对文本 PDF 直接提取、对扫描件自动
// OvisOCR2→tesseract 分页 OCR；OCRImageText 用常驻 OvisOCR2 本地服务识别图片
// （与 GaeaOCRText 的本地兜底同链路）。
var (
	visionExtractPDF = func(path string) (string, error) {
		md, _, _, err := docmd.ConvertLimit(path, "", docmd.DefaultMaxPDFPages)
		if err != nil {
			return "", err
		}
		return md, nil
	}
	visionOCRImage = docmd.OCRImageText
)

// GaeaCostImportVisionPreview 识别 PDF/图片报价单并产出可确认的候选成本条目。
// 与 GaeaCostImportPreview 同构（复用 costimport 匹配与前端预览确认），落地
// 沿用 GaeaCostImportApply——无确认不落库。source 标记识别来源：pdf_text /
// pdf_scan / image。
func (a *App) GaeaCostImportVisionPreview(path string) (CostImportPreview, error) {
	abs, _ := resolvePreviewPath(path)
	if abs == "" || !fileExists(abs) {
		return CostImportPreview{}, fmt.Errorf("文件不存在: %s", path)
	}
	ext := strings.ToLower(filepath.Ext(abs))

	var text, source string
	var err error
	switch ext {
	case ".pdf":
		text, source, err = a.visionExtractPDFText(abs)
	case ".png", ".jpg", ".jpeg", ".webp", ".bmp":
		text, err = visionOCRImage(abs)
		if err != nil {
			return CostImportPreview{}, fmt.Errorf("图片 OCR 识别失败: %w", err)
		}
		if !pdfTextUsable(text) {
			return CostImportPreview{}, fmt.Errorf("未能从图片中识别出有效文本，请确认图片为清晰报价单后重试")
		}
		source = visionSourceImage
	default:
		return CostImportPreview{}, fmt.Errorf("暂不支持 %s 格式识别，请使用 PDF/PNG/JPG/WEBP/BMP 报价单（表格类可继续用 xlsx/csv 导入）", ext)
	}
	if err != nil {
		return CostImportPreview{}, err
	}

	// 规则解析：表格线优先（TSV/竖线/多空格对齐），失败回退整行文本。
	rows, columns := visionParseQuotation(text)

	// AI 字段归一化（可选增强）：sensitive_local 开启时用本地通道；本地不可用
	// 则跳过，只用规则解析并在 message 注明「未做 AI 归一化」——绝不阻断预览。
	aiUsed := false
	if normalized, used := a.visionAINormalize(text, rows); used {
		rows, aiUsed = normalized, true
	}

	matched := costimport.MatchRows(rows, a.hubCostStore())
	msg := "本地识别完成，请核对后确认导入。"
	if !aiUsed {
		msg += "（未做 AI 归一化，仅规则解析）"
	}
	pv := &costimport.Preview{
		Path:     filepath.ToSlash(abs),
		FileName: filepathBase(abs),
		Columns:  columns,
		Rows:     matched,
		Message:  msg,
	}
	out := toCostImportPreview(pv, aiUsed)
	out.Source = source
	return out, nil
}

// visionExtractPDFText 提取 PDF 文本并判定来源：文本型（pdf_text）/ 扫描件
// （pdf_scan）。ConvertLimit 对扫描件已内置 OCR，输出带 OCR 标记前缀；文本
// 极少时再走本地 OCR 兜底（OvisOCR2）。
func (a *App) visionExtractPDFText(abs string) (string, string, error) {
	text, err := visionExtractPDF(abs)
	if err != nil {
		return "", "", fmt.Errorf("PDF 文本提取失败: %w", err)
	}
	source := visionSourcePDFText
	if strings.HasPrefix(strings.TrimSpace(text), "（以下内容由 OCR 识别") {
		source = visionSourcePDFScan
	}
	if !pdfTextUsable(text) {
		// 文本极少 → 扫描件判定：本地 OCR 兜底（OvisOCR2）。
		if ocr, oerr := visionOCRImage(abs); oerr == nil && pdfTextUsable(ocr) {
			text, source = ocr, visionSourcePDFScan
		}
	}
	if !pdfTextUsable(text) {
		return "", "", fmt.Errorf("未能从 PDF 中识别出有效文本（扫描件请确认本地 OCR 已安装）")
	}
	return text, source, nil
}

// ─── 规则解析（纯函数，可单测）─────────────────────────────────

// visionParseQuotation 把报价单文本解析为候选成本条目。
// 先尝试表格线解析（TSV / 竖线 / 多空格对齐，表头含「名称/规格/单位/单价/价格」），
// 失败回退「整行文本 → 一条候选（名称=行首片段，价格=行内首个数字）」。
func visionParseQuotation(text string) (rows []costimport.Row, columns []string) {
	if cols, rs, ok := visionParseTable(text); ok && len(rs) > 0 {
		return rs, cols
	}
	return visionParseFallback(text), nil
}

// 表头列字段编号（对齐 costimport 的列映射思路，轻量启发式）。
const (
	visionFieldNone = iota
	visionFieldTitle
	visionFieldSpec
	visionFieldUnit
	visionFieldPrice
	visionFieldSource
	visionFieldCategory
)

// visionFieldKeywords 表头关键词 → 成本字段（长词优先，与 costimport 一致）。
var visionFieldKeywords = []struct {
	field int
	kws   []string
}{
	{visionFieldTitle, []string{"材料名称", "项目名称", "名称规格", "品名", "材料", "物资", "项目", "设备", "机械", "名称", "内容", "子目"}},
	{visionFieldSpec, []string{"规格型号", "规格", "型号", "材质", "参数"}},
	{visionFieldUnit, []string{"单位"}},
	{visionFieldPrice, []string{"含税单价", "单价", "市场价", "信息价", "报价", "价格", "金额"}},
	{visionFieldSource, []string{"供应商", "来源", "品牌", "产地"}},
	{visionFieldCategory, []string{"分类", "类别"}},
}

// visionMultiSpaceRe 多空格对齐分隔（≥2 连续空格视为列分隔）。
var visionMultiSpaceRe = regexp.MustCompile(" {2,}")

// visionSplitCells 按行内分隔符拆单元格：TSV > 竖线（Markdown 表格）> 多空格
// 对齐；无法识别分隔（普通文本行）返回 nil，交由整行回退解析。
func visionSplitCells(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	var cells []string
	switch {
	case strings.Contains(line, "\t"):
		cells = strings.Split(line, "\t")
	case strings.Contains(line, "|"):
		cells = strings.Split(line, "|")
	default:
		fields := visionMultiSpaceRe.Split(line, -1)
		if len(fields) >= 2 {
			cells = fields
		}
	}
	if cells == nil {
		return nil
	}
	// 保位裁剪：只去空白与 Markdown 残留（反引号/星号），不丢弃空单元格以免错位。
	for i, c := range cells {
		cells[i] = strings.TrimSpace(strings.Trim(strings.TrimSpace(c), "`*"))
	}
	return cells
}

// visionIsHeader 判断一行单元格是否为表头：至少 2 个非空列，且含「名称类」
// 关键词与「单价/价格类」或「单位」关键词。
func visionIsHeader(cells []string) bool {
	nonEmpty := 0
	hasTitle, hasPrice, hasUnit := false, false, false
	for _, c := range cells {
		if c == "" {
			continue
		}
		nonEmpty++
		low := strings.ToLower(c)
		for _, kw := range visionFieldKeywords[visionFieldTitle].kws {
			if strings.Contains(low, strings.ToLower(kw)) {
				hasTitle = true
				break
			}
		}
		for _, kw := range visionFieldKeywords[visionFieldPrice].kws {
			if strings.Contains(low, strings.ToLower(kw)) {
				hasPrice = true
				break
			}
		}
		for _, kw := range visionFieldKeywords[visionFieldUnit].kws {
			if strings.Contains(low, strings.ToLower(kw)) {
				hasUnit = true
				break
			}
		}
	}
	if nonEmpty < 2 {
		return false
	}
	return hasTitle && (hasPrice || hasUnit)
}

// visionMapColumns 表头 → 列字段映射（长关键词优先，避免「单价」误吞「规格」）。
func visionMapColumns(header []string) map[int]int {
	out := map[int]int{}
	for c, h := range header {
		low := strings.ToLower(strings.TrimSpace(h))
		best := visionFieldNone
		bestLen := 0
		for _, fk := range visionFieldKeywords {
			for _, kw := range fk.kws {
				if strings.Contains(low, strings.ToLower(kw)) && len(kw) > bestLen {
					best = fk.field
					bestLen = len(kw)
				}
			}
		}
		if best != visionFieldNone {
			out[c] = best
		}
	}
	return out
}

// visionHasField 判断列映射里是否包含某成本字段。
func visionHasField(colMap map[int]int, field int) bool {
	for _, f := range colMap {
		if f == field {
			return true
		}
	}
	return false
}

// visionIsSeparatorRow 判断是否为 Markdown 表格分隔行（---/===）。
func visionIsSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if strings.Trim(c, "-= \t") != "" {
			return false
		}
	}
	return true
}

// visionParseTable 表格线解析：找含名称/单价/单位的表头行，其后数据行按列
// 映射为候选。无表头或无单价列时返回 ok=false（调用方回退整行解析）。
func visionParseTable(text string) (columns []string, rows []costimport.Row, ok bool) {
	lines := strings.Split(text, "\n")
	headerIdx := -1
	var colMap map[int]int
	for i, ln := range lines {
		cells := visionSplitCells(ln)
		if len(cells) < 2 || !visionIsHeader(cells) {
			continue
		}
		headerIdx = i
		columns = cells
		colMap = visionMapColumns(cells)
		break
	}
	if headerIdx < 0 || !visionHasField(colMap, visionFieldPrice) {
		return nil, nil, false
	}
	for _, ln := range lines[headerIdx+1:] {
		cells := visionSplitCells(ln)
		if len(cells) < 2 || visionIsSeparatorRow(cells) {
			continue
		}
		row := visionBuildRow(cells, colMap)
		if strings.TrimSpace(row.Title) == "" {
			continue
		}
		row.Raw = strings.Join(cells, " | ")
		rows = append(rows, row)
	}
	return columns, rows, len(rows) > 0
}

// visionBuildRow 按列映射组装一条候选。
func visionBuildRow(cells []string, colMap map[int]int) costimport.Row {
	row := costimport.Row{}
	for c, f := range colMap {
		if c >= len(cells) {
			continue
		}
		v := strings.TrimSpace(cells[c])
		switch f {
		case visionFieldTitle:
			if row.Title == "" && v != "" {
				row.Title = visionStripIndex(v)
			}
		case visionFieldSpec:
			if row.Spec == "" && v != "" {
				row.Spec = v
			}
		case visionFieldUnit:
			if row.Unit == "" && v != "" {
				row.Unit = v
			}
		case visionFieldPrice:
			if p, found := visionParsePrice(v); found && row.Price <= 0 {
				row.Price = p
			}
		case visionFieldSource:
			if row.Source == "" && v != "" {
				row.Source = v
			}
		case visionFieldCategory:
			if row.Category == "" && v != "" {
				row.Category = v
			}
		}
	}
	return row
}

// visionParseFallback 整行文本回退：每行 → 一条候选，名称=行首片段，
// 价格=行内首个可解析数字；无价格的标题/说明/页码行跳过。
func visionParseFallback(text string) []costimport.Row {
	var rows []costimport.Row
	for _, ln := range strings.Split(text, "\n") {
		line := strings.TrimSpace(ln)
		if visionIsNoiseLine(line) {
			continue
		}
		name, price, found := visionLineNamePrice(line)
		if !found {
			continue
		}
		rows = append(rows, costimport.Row{
			Title: visionStripIndex(name),
			Price: price,
			Raw:   line,
		})
	}
	return rows
}

// visionIsNoiseLine 跳过报价单常见噪音行：纯分隔线、页码行、无数字行。
func visionIsNoiseLine(line string) bool {
	if line == "" {
		return true
	}
	if strings.Trim(line, "-–—=*_·•| \t") == "" {
		return true // 纯分隔线
	}
	if strings.Contains(line, "第") && strings.Contains(line, "页") {
		return true // 页码行
	}
	return !strings.ContainsAny(line, "0123456789")
}

// visionLineNamePrice 从一行文本中提取（名称=行首片段, 价格=行内首个数字）。
// 价格在行首时取其后第一个非价格词作为名称片段。
func visionLineNamePrice(line string) (name string, price float64, ok bool) {
	tokens := strings.Fields(line)
	if len(tokens) == 0 {
		return "", 0, false
	}
	for i, tok := range tokens {
		if visionIsOrdinal(tok) {
			continue // 行首序号（"1." "2、"）不是价格
		}
		p, found := visionParsePrice(tok)
		if !found {
			continue
		}
		left := strings.TrimRight(strings.Join(tokens[:i], " "), "：:，,、;；|")
		if strings.TrimSpace(left) == "" {
			for j := i + 1; j < len(tokens); j++ {
				if _, num := visionParsePrice(tokens[j]); !num {
					return visionStripIndex(tokens[j]), p, true
				}
			}
			continue
		}
		return visionStripIndex(left), p, true
	}
	return "", 0, false
}

// visionStripIndex 剥离行首序号：数字 + 序号分隔符（.、:：空格）或括号包裹
// （（1）形态）。非序号前缀原样返回（如 "300kW 发电机"）。
func visionStripIndex(s string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	if r[0] == '（' || r[0] == '(' {
		for i := 1; i < len(r); i++ {
			if r[i] == '）' || r[i] == ')' {
				if visionAllDigits(r[1:i]) {
					return strings.TrimSpace(string(r[i+1:]))
				}
				break
			}
		}
	}
	i := 0
	for i < len(r) && r[i] >= '0' && r[i] <= '9' {
		i++
	}
	if i > 0 && i < len(r) {
		switch r[i] {
		case '.', '．', '、', ':', '：', ' ', '\t':
			return strings.TrimSpace(string(r[i+1:]))
		}
	}
	return s
}

// visionIsOrdinal 判断是否为行首序号 token（如 "1." "2、" "3:"），避免被
// visionParsePrice 误当价格。
func visionIsOrdinal(tok string) bool {
	r := []rune(strings.TrimSpace(tok))
	if len(r) < 2 {
		return false
	}
	for i := 0; i < len(r)-1; i++ {
		if r[i] < '0' || r[i] > '9' {
			return false
		}
	}
	switch r[len(r)-1] {
	case '.', '．', '、', ':', '：':
		return true
	}
	return false
}

func visionAllDigits(rs []rune) bool {
	if len(rs) == 0 {
		return false
	}
	for _, r := range rs {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// visionParsePrice 归一化价格文本：去 ¥/￥/元/千分位/空格，支持尾随 /单位
// （如 "3200元/台班"）。非价格文本返回 false。
func visionParsePrice(s string) (float64, bool) {
	clean := strings.NewReplacer(",", "", "，", "", "¥", "", "￥", "", "元", "", " ", "", " ", "").Replace(strings.TrimSpace(s))
	if clean == "" {
		return 0, false
	}
	if i := strings.IndexByte(clean, '/'); i > 0 {
		clean = clean[:i]
	}
	if !strings.ContainsAny(clean, "0123456789") {
		return 0, false
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

// ─── AI 字段归一化（可选增强）───────────────────────────────────

// visionAINormalize 用本地模型（routeSensitiveLocal，敏感域本地化开启时）
// 对提取文本做成本条目归一化，返回归一化后的候选行。任何一步不可用（开关
// 关闭/本地引擎缺失/请求失败/输出无效）都返回 (fallback, false)，由调用方
// 降级为规则解析并在 message 注明「未做 AI 归一化」。
func (a *App) visionAINormalize(text string, fallback []costimport.Row) (rows []costimport.Row, used bool) {
	if a == nil || a.core == nil || a.cfg == nil || !a.cfg.GetSensitiveLocal() || a.engineMgr == nil {
		return fallback, false
	}
	featEng, featModel, _ := a.routeSensitiveLocal("office")
	if featEng == "" || featModel == "" {
		return fallback, false
	}
	prov, err := provider.NewLLM("", provider.Config{Name: "cost-import-vision", Model: featModel, Engine: featEng})
	if err != nil {
		return fallback, false
	}

	// 截断过长的文本，避免本地模型上下文/超时失控。
	src := text
	if rs := []rune(src); len(rs) > 6000 {
		src = string(rs[:6000]) + "\n…（已截断）"
	}

	const sysPrompt = "你是成本数据提取助手。把报价单文本的每一行/表格归一化为成本条目 JSON 数组，规则：\n" +
		"title=材料/设备/项目名称（去掉序号前缀与规格噪音）；spec=规格型号（无则空串）；unit=单位（台班/吨/m³/工日等，无则空串）；\n" +
		"price=数字单价（元，去掉货币符号与千分位，无法识别填 0）；source=来源（供应商/产地，无则空串）；\n" +
		"category 只能是 机械/材料/人工/运输/检测/综合单价/其他 之一。\n" +
		"只输出 JSON 数组，不要代码块标记，不要任何解释。"

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	ch, err := prov.Stream(ctx, provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: sysPrompt},
			{Role: provider.RoleUser, Content: "请从以下报价单文本中提取成本条目：\n\n" + src},
		},
		Temperature: 0,
	})
	if err != nil {
		return fallback, false
	}
	var out strings.Builder
	for {
		select {
		case <-ctx.Done():
			return fallback, false
		case chunk, ok := <-ch:
			if !ok {
				return a.visionFinishAINormalize(out.String(), fallback)
			}
			switch chunk.Type {
			case provider.ChunkText:
				out.WriteString(chunk.Text)
			case provider.ChunkError:
				return fallback, false
			}
		}
	}
}

// visionFinishAINormalize 解析 AI JSON 输出为候选行；输出无效/全为噪音时返回
// 规则解析结果（false）。
func (a *App) visionFinishAINormalize(raw string, fallback []costimport.Row) ([]costimport.Row, bool) {
	start := strings.IndexByte(raw, '[')
	end := strings.LastIndexByte(raw, ']')
	if start < 0 || end <= start {
		return fallback, false
	}
	var aiRows []struct {
		Title    string  `json:"title"`
		Spec     string  `json:"spec"`
		Unit     string  `json:"unit"`
		Price    float64 `json:"price"`
		Source   string  `json:"source"`
		Category string  `json:"category"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &aiRows); err != nil || len(aiRows) == 0 {
		return fallback, false
	}
	rows := make([]costimport.Row, 0, len(aiRows))
	for _, r := range aiRows {
		rows = append(rows, costimport.Row{
			Title:    strings.TrimSpace(r.Title),
			Spec:     strings.TrimSpace(r.Spec),
			Unit:     strings.TrimSpace(r.Unit),
			Price:    r.Price,
			Source:   strings.TrimSpace(r.Source),
			Category: normalizeAICategory(r.Category),
		})
	}
	// 过滤明显无效行（无名称且无价格）。
	valid := rows[:0]
	for _, r := range rows {
		if strings.TrimSpace(r.Title) != "" || r.Price > 0 {
			valid = append(valid, r)
		}
	}
	if len(valid) == 0 {
		return fallback, false
	}
	return valid, true
}

// ─── B 部分：供应商比价 ────────────────────────────────────────

// CostCompareRow 供应商比价一行：某来源在某期/时刻的价格及相对库内现价的跳幅。
type CostCompareRow struct {
	Source    string  `json:"source"`
	Period    string  `json:"period"`
	Price     float64 `json:"price"`
	DiffPct   float64 `json:"diffPct"`
	FetchedAt string  `json:"fetchedAt"`
	Kind      string  `json:"kind"` // current=库内现价 / fetch=价格源抓取候选 / history=价格历史快照
}

// 比价来源类型。
const (
	compareKindCurrent = "current"
	compareKindFetch   = "fetch"
	compareKindHistory = "history"
)

// GaeaCostCompare 供应商比价：聚合 cost_entries 现价（title/name 匹配）、
// price_fetch 抓取候选（candidates 中 title 匹配，含各期价格）与
// cost_price_history 历史快照（按 name 匹配）。diffPct 为相对库内现价的单期
// 跳幅（复用/参照 pricefeed.DetectAnomalies 算法）。按 fetchedAt 倒序；
// 匹配不到返回空数组（不报错）。
func (a *App) GaeaCostCompare(name string) ([]CostCompareRow, error) {
	query := strings.TrimSpace(name)
	if query == "" {
		return nil, fmt.Errorf("比价名称不能为空")
	}

	var rows []CostCompareRow
	current := 0.0
	hasCurrent := false

	// 1) 库内现价（title 或 name 匹配）→ kind=current（跳幅基准）。
	if costStore := a.hubCostStore(); costStore.Available() {
		for _, s := range costStore.List() {
			if visionTitleMatch(s.Title, query) || visionNameMatch(s.Name, query) {
				rows = append(rows, CostCompareRow{
					Source:    s.Source,
					Period:    "现行",
					Price:     s.Price,
					FetchedAt: visionFormatTime(s.UpdatedAt),
					Kind:      compareKindCurrent,
				})
				if !hasCurrent {
					current = s.Price
					hasCurrent = true
				}
				break // 现价取首个匹配（同名 UPSERT 保证唯一键）
			}
		}
	}

	// 2) 价格源抓取候选 → kind=fetch（每期价格）。
	if priceStore := a.hubPriceStore(); priceStore.Available() {
		for _, rec := range priceStore.ListFetches(50) {
			for _, c := range rec.Candidates {
				if visionTitleMatch(c.Title, query) {
					rows = append(rows, CostCompareRow{
						Source:    rec.SourceName,
						Period:    rec.Period,
						Price:     c.Price,
						FetchedAt: rec.FetchedAt,
						Kind:      compareKindFetch,
					})
				}
			}
		}
		// 3) 价格历史快照 → kind=history（按 name 匹配）。
		slug := cost.SlugName(query)
		for _, h := range priceStore.ListHistory(slug, 50) {
			if h.Name == slug || visionTitleMatch(h.Title, query) {
				rows = append(rows, CostCompareRow{
					Source:    h.Source,
					Period:    h.Period,
					Price:     h.Price,
					FetchedAt: h.FetchedAt,
					Kind:      compareKindHistory,
				})
			}
		}
	}

	if len(rows) == 0 {
		return []CostCompareRow{}, nil
	}

	// diffPct：相对现价的单期跳幅（复用 DetectAnomalies：±round2((p-prev)/prev*100)）。
	if hasCurrent && current > 0 {
		for i := range rows {
			if rows[i].Kind == compareKindCurrent {
				continue
			}
			rows[i].DiffPct = visionRound2((rows[i].Price - current) / current * 100)
		}
	}

	// 按 fetchedAt 倒序（无法解析的时间视为最早）；同时刻 current 优先展示。
	sort.SliceStable(rows, func(i, j int) bool {
		ti := visionParseTime(rows[i].FetchedAt)
		tj := visionParseTime(rows[j].FetchedAt)
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return visionCompareKindRank(rows[i].Kind) < visionCompareKindRank(rows[j].Kind)
	})
	return rows, nil
}

// visionTitleMatch 标题/名称匹配：相等或互相包含（大小写不敏感）。
func visionTitleMatch(title, query string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	q := strings.ToLower(strings.TrimSpace(query))
	if t == "" || q == "" {
		return false
	}
	return t == q || strings.Contains(t, q) || strings.Contains(q, t)
}

// visionNameMatch 唯一键匹配：查询的 SlugName 等于条目 name。
func visionNameMatch(name, query string) bool {
	return strings.TrimSpace(name) == cost.SlugName(query)
}

// visionFormatTime 时间 → RFC3339 展示（零值返回空串）。
func visionFormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// visionParseTime 解析 fetchedAt；非法/空返回零值。
func visionParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// visionCompareKindRank 同时间展示优先级：current > fetch > history。
func visionCompareKindRank(kind string) int {
	switch kind {
	case compareKindCurrent:
		return 0
	case compareKindFetch:
		return 1
	default:
		return 2
	}
}

// visionRound2 两位小数（与 pricefeed.round2 一致）。
func visionRound2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}