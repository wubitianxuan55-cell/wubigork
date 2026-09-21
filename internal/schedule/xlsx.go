package schedule

// xlsx.go — 进度计划 Excel 导入导出（v4.134.0 刀D3，差距清单 #2）。
//
// 上报口径（对标 MS Project 横道导出表）：序号/WBS/任务名称/工期(天)/开始/完成/前置。
// 导出：CPM fail-closed（坏计划不出表），日期按工作日历换算，分组行跨子孙
// 取开始/完成跨度，前置列 Project 式「2FS+3」引用；v4.138 #14 起 7 标准列后
// 按注册表序追加「有值」自定义字段槽列（列头=项目 customLabels 覆盖否则缺省
// 中文名，分组行该列留空）。
// 导入：表头行在前 10 行内按别名识别（任务名称/工作名称…、工期/持续时间…、
// 开始/开工日期…、前置/紧前工作…），标准列之外剩余列再按自定义槽名匹配
// （缺省 label 与项目覆盖名都认；number 槽非法值丢弃不报错），层级按 WBS
// （存在「X.」子编码的行=分组；无 WBS 列时空工期行=分组），前置引用
// 「2、2FS、2SS+3」按序号回链，日期仅取最早值作开工日——排程一律交回 CPM
// 引擎重算（与 mspdi 导入同律），开始/完成列不进模型。里程碑=0 工期叶任务；
// 分组行工期恒 0。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const xlsxSheetName = "进度计划"

// ── 导出 ─────────────────────────────────────────────────────────────────

// ExportXlsx 渲染上报口径 Excel 字节。
func ExportXlsx(p Project) ([]byte, error) {
	cpm, _ := p.Analyze()
	if !cpm.OK {
		return nil, fmt.Errorf("计划存在循环依赖：%s", cpm.Error)
	}
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	if err := f.SetSheetName(sheet, xlsxSheetName); err != nil {
		return nil, err
	}
	sheet = xlsxSheetName

	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	usedKeys := customUsedKeys(p.Tasks)
	header := []string{"序号", "WBS", "任务名称", "工期(天)", "开始", "完成", "前置"}
	for _, key := range usedKeys {
		header = append(header, customLabelOf(key, p.CustomLabels))
	}
	for c, h := range header {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}
	lastCol, _ := excelize.ColumnNumberToName(len(header))
	if err := f.SetCellStyle(sheet, "A1", lastCol+"1", bold); err != nil {
		return nil, err
	}

	refsOf := func(id string) string {
		parts := make([]string, 0, 2)
		for _, l := range p.Links {
			if l.To == id {
				parts = append(parts, refText(rowNoOf(p.Tasks, l.From), string(l.Type), l.Lag))
			}
		}
		return strings.Join(parts, ",")
	}
	for i, t := range p.Tasks {
		row := i + 2
		set := func(col int, v any) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
		set(1, i+1)
		set(2, wbsOf(p.Tasks, i))
		set(3, t.Name)
		if t.Level > 0 {
			dur := t.Duration
			if t.IsMilestone {
				dur = 0
			}
			set(4, dur)
			if r, ok := cpm.Rows[t.ID]; ok {
				set(5, fmtDate(WdToDate(p.StartDate, r.ES, p.Calendar)))
				set(6, fmtDate(WdToDate(p.StartDate, r.EF, p.Calendar)))
			}
			if refs := refsOf(t.ID); refs != "" {
				set(7, refs)
			}
		} else if es, ef, ok := groupSpanRows(p.Tasks, i, cpm); ok {
			set(5, fmtDate(WdToDate(p.StartDate, es, p.Calendar)))
			set(6, fmtDate(WdToDate(p.StartDate, ef, p.Calendar)))
		}
		// 自定义槽列（8=标准 7 列之后）：叶任务有值才写，分组行留空
		//（customUsedKeys 口径即只看叶任务，分组行必然取不到值）。
		for j, key := range usedKeys {
			if v, ok := customValueOf(t, key); ok {
				set(8+j, v)
			}
		}
	}
	widths := map[string]float64{"A": 8, "B": 10, "C": 40, "D": 10, "E": 13, "F": 13, "G": 18}
	for idx := range usedKeys {
		col, _ := excelize.ColumnNumberToName(8 + idx)
		widths[col] = 14
	}
	for col, w := range widths {
		if err := f.SetColWidth(sheet, col, col, w); err != nil {
			return nil, err
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// wbsOf WBS 编码：分组 n、叶 n.k（与前端 GanttView/mspdi 同构）。
func wbsOf(tasks []Task, idx int) string {
	g, k := 0, 0
	for i := 0; i <= idx && i < len(tasks); i++ {
		if tasks[i].Level == 0 {
			g++
			k = 0
		} else {
			k++
		}
	}
	if tasks[idx].Level == 0 {
		return strconv.Itoa(g)
	}
	return strconv.Itoa(g) + "." + strconv.Itoa(k)
}

// rowNoOf 任务 id → 1 基行号（Project 引用口径）。
func rowNoOf(tasks []Task, id string) int {
	for i, t := range tasks {
		if t.ID == id {
			return i + 1
		}
	}
	return 0
}

// refText Project 式前置引用：行号+类型+时距（lag=0 省略，负值 -n）。
func refText(no int, typ string, lag int) string {
	s := strconv.Itoa(no) + typ
	if lag > 0 {
		s += "+" + strconv.Itoa(lag)
	} else if lag < 0 {
		s += strconv.Itoa(lag)
	}
	return s
}

// groupSpanRows 分组行汇总跨度（两级大纲：分组后连续 Level>0 段的 min ES / max EF）。
func groupSpanRows(tasks []Task, idx int, cpm CpmResult) (es, ef int, ok bool) {
	es, ef = int(^uint(0)>>1), 0
	for j := idx + 1; j < len(tasks) && tasks[j].Level > 0; j++ {
		if r, found := cpm.Rows[tasks[j].ID]; found {
			if r.ES < es {
				es = r.ES
			}
			if r.EF > ef {
				ef = r.EF
			}
			ok = true
		}
	}
	return es, ef, ok
}

// fmtDate 日期 → YYYY/M/D 文本。
func fmtDate(d time.Time) string {
	return fmt.Sprintf("%d/%d/%d", d.Year(), int(d.Month()), d.Day())
}

// ── 自定义字段（v4.138 #14，镜像 frontend/src/schedule/customFields.ts）──

// customFieldDef 自定义槽定义（固定 5 槽：文本×3 + 数值×2，序=xlsx 导出列序）。
type customFieldDef struct {
	key   string
	label string // 缺省列名
	isNum bool   // true=数值槽
}

// customFieldRegistry 缺省注册表（key 稳定勿动——会破坏存量 xlsx 识别，
// 与 TS CUSTOM_FIELDS 逐项对齐）。
var customFieldRegistry = []customFieldDef{
	{"text1", "文本1", false},
	{"text2", "文本2", false},
	{"text3", "文本3", false},
	{"num1", "数值1", true},
	{"num2", "数值2", true},
}

// customLabelOf 列名：项目 customLabels 覆盖优先（非空白），否则缺省 label
// （镜像 TS customLabelOf 的空串回落口径）。
func customLabelOf(key string, labels map[string]string) string {
	if ov := strings.TrimSpace(labels[key]); ov != "" {
		return ov
	}
	for _, d := range customFieldRegistry {
		if d.key == key {
			return d.label
		}
	}
	return key
}

// customNumber 槽值 → float64。JSON 反序列化产出 float64；兼容 int/int64/
// float32/json.Number；NaN/Inf 与解析失败视为无值（对齐 TS Number.isFinite）。
func customNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		if !math.IsNaN(n) && !math.IsInf(n, 0) {
			return n, true
		}
	case float32:
		if f := float64(n); !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f, true
		}
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		if f, err := n.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// customValueOf 任务在槽上的值（镜像 TS customValueOf：text=非空 string、
// number=有限数；无值/类型不符返回 false）。
func customValueOf(t Task, key string) (any, bool) {
	raw, ok := t.Custom[key]
	if !ok {
		return nil, false
	}
	for _, d := range customFieldRegistry {
		if d.key != key {
			continue
		}
		if d.isNum {
			return customNumber(raw)
		}
		if s, isStr := raw.(string); isStr && strings.TrimSpace(s) != "" {
			return s, true
		}
		break
	}
	return nil, false
}

// customUsedKeys 有值槽 key 集（按注册表序；口径=任一叶任务的 custom 里该槽
// 有值，分组行不算——与 TS customUsedKeys(叶任务) 同律）。
func customUsedKeys(tasks []Task) []string {
	used := map[string]bool{}
	for _, t := range tasks {
		if t.Level == 0 {
			continue
		}
		for _, d := range customFieldRegistry {
			if _, ok := customValueOf(t, d.key); ok {
				used[d.key] = true
			}
		}
	}
	keys := make([]string, 0, len(used))
	for _, d := range customFieldRegistry {
		if used[d.key] {
			keys = append(keys, d.key)
		}
	}
	return keys
}

// ── 导入 ─────────────────────────────────────────────────────────────────

// xlsxColKind 表头列别名的语义类别。
type xlsxColKind int

const (
	colNone xlsxColKind = iota
	colSeq
	colWBS
	colName
	colDur
	colStart
	colFinish
	colPreds
)

// xlsxAliases 语义类别 → 表头别名集（单元格归一后精确匹配）。
var xlsxAliases = map[xlsxColKind][]string{
	colSeq:    {"序号", "标识号", "id"},
	colWBS:    {"wbs", "wbs编码", "大纲", "编码"},
	colName:   {"任务名称", "工作名称", "工作内容", "工作项目", "名称"},
	colDur:    {"工期", "持续时间", "工期天数"},
	colStart:  {"开始", "开始时间", "开始日期", "开工", "开工日期", "计划开始", "计划开工"},
	colFinish: {"完成", "完成时间", "完成日期", "竣工", "竣工日期", "计划完成", "计划竣工"},
	colPreds:  {"前置", "前置任务", "前置工作", "紧前", "紧前工作", "逻辑关系"},
}

// xlsxNorm 表头单元格归一：去空白/括号/冒号/「天」字、转小写（工期(天)→工期）。
func xlsxNorm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\t', '（', '）', '(', ')', '天', '：', ':':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// matchHeaderRow 识别表头行：须含任务名称列 + （工期/开始/完成/WBS 任一）。
func matchHeaderRow(row []string) map[xlsxColKind]int {
	cols := map[xlsxColKind]int{}
	for c, cell := range row {
		n := xlsxNorm(cell)
		if n == "" {
			continue
		}
		for kind, aliases := range xlsxAliases {
			for _, a := range aliases {
				if n == a {
					if _, dup := cols[kind]; !dup {
						cols[kind] = c
					}
					break
				}
			}
		}
	}
	if _, hasName := cols[colName]; !hasName {
		return nil
	}
	for _, kind := range []xlsxColKind{colDur, colStart, colFinish, colWBS} {
		if _, ok := cols[kind]; ok {
			return cols
		}
	}
	return nil
}

// matchCustomCols 表头行标准列之外的自定义槽列识别（v4.138 #14）：列头归一后
// 等于缺省 label 或项目覆盖名（同一槽两个名字都认，覆盖名后注册可顶替缺省名；
// 同名列先到先得）。已被标准列占用的列不再参与——标准列优先。
func matchCustomCols(row []string, std map[xlsxColKind]int, labels map[string]string) map[string]int {
	byName := map[string]string{}
	for _, d := range customFieldRegistry {
		byName[xlsxNorm(d.label)] = d.key
	}
	for _, d := range customFieldRegistry {
		if ov := strings.TrimSpace(labels[d.key]); ov != "" {
			byName[xlsxNorm(ov)] = d.key
		}
	}
	stdUsed := map[int]bool{}
	for _, c := range std {
		stdUsed[c] = true
	}
	cols := map[string]int{}
	for c, name := range row {
		if stdUsed[c] {
			continue
		}
		key, ok := byName[xlsxNorm(name)]
		if !ok {
			continue
		}
		if _, dup := cols[key]; !dup {
			cols[key] = c
		}
	}
	return cols
}

// xlsxRefRe 前置引用：「2」「2FS」「2SS+3」「2fs-1」。
var xlsxRefRe = regexp.MustCompile(`(?i)^\s*(\d+)\s*(FS|SS|FF|SF)?\s*([+-]\d+)?\s*$`)

// xlsxDurRe 工期文本提取整数：「5」「5 个工作日」「5d」「工期5天」。
var xlsxDurRe = regexp.MustCompile(`\d+`)

// xlsxDateLayouts 日期文本口径（GetRows 返回格式化文本；带时间尾巴时截断）。
var xlsxDateLayouts = []string{"2006-01-02", "2006/1/2", "2006/01/02", "2006-1-2", "2006年1月2日", "2006年01月02日"}

// parseXlsxDate 解析日期文本；失败返回零值。
func parseXlsxDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i > 0 {
		if t, ok := tryParse(s[:i]); ok {
			return t, true
		}
	}
	return tryParse(s)
}

func tryParse(s string) (time.Time, bool) {
	for _, layout := range xlsxDateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ImportXlsx 解析上报 Excel → 计划 Project。排程不进模型（开始/完成列仅取
// 最早开始作开工日），交回 CPM 引擎重算；解析失败给出可读错误。
// customLabels 可选传入项目自定义列名覆盖（v4.138 #14：改名槽按覆盖名识别，
// 缺省 label 恒识别）；不传即按缺省名。
func ImportXlsx(data []byte, customLabels ...map[string]string) (project Project, err error) {
	// v4.385 工程健康轮#2：excelize GO-2026-6452（负 shared-string 下标 panic，
	// 上游 Fixed in N/A）符号追踪 GetRows 可达面——入口 defer 兜底转可读错误
	// （对齐 v4.370 xlsxpreview 同款防线）。
	defer func() {
		if r := recover(); r != nil {
			project = Project{}
			err = fmt.Errorf("Excel 解析失败（文档结构异常）: %v", r)
		}
	}()
	labels := map[string]string{}
	if len(customLabels) > 0 && customLabels[0] != nil {
		labels = customLabels[0]
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return Project{}, fmt.Errorf("Excel 打开失败：%w", err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return Project{}, fmt.Errorf("Excel 无工作表")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return Project{}, fmt.Errorf("读取工作表失败：%w", err)
	}
	cols, headerRow := map[xlsxColKind]int{}, -1
	customCols := map[string]int{}
	for ri, row := range rows {
		if ri >= 10 {
			break
		}
		if c := matchHeaderRow(row); c != nil {
			cols, headerRow = c, ri
			customCols = matchCustomCols(row, c, labels)
			break
		}
	}
	if headerRow < 0 {
		return Project{}, fmt.Errorf("未识别表头：需含「任务名称/工作名称」列，且包含「工期/开始/完成/WBS」任一列")
	}

	type rawRow struct {
		seq    int
		wbs    string
		name   string
		dur    string
		start  string
		preds  string
		hasSeq bool
		// custom 自定义槽原始文本（列头→槽 key 命中列；空单元格不入）。
		custom map[string]string
	}
	var raws []rawRow
	minStart := time.Time{}
	cell := func(row []string, kind xlsxColKind) string {
		c, ok := cols[kind]
		if !ok || c >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[c])
	}
	for ri := headerRow + 1; ri < len(rows); ri++ {
		row := rows[ri]
		name := cell(row, colName)
		if name == "" {
			continue
		}
		r := rawRow{name: name, wbs: cell(row, colWBS), dur: cell(row, colDur), start: cell(row, colStart), preds: cell(row, colPreds), seq: len(raws) + 1}
		for key, c := range customCols {
			if c >= len(row) {
				continue
			}
			if s := strings.TrimSpace(row[c]); s != "" {
				if r.custom == nil {
					r.custom = map[string]string{}
				}
				r.custom[key] = s
			}
		}
		if s := cell(row, colSeq); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 {
				r.seq, r.hasSeq = n, true
			}
		}
		if t, ok := parseXlsxDate(r.start); ok {
			if minStart.IsZero() || t.Before(minStart) {
				minStart = t
			}
		}
		raws = append(raws, r)
	}
	if len(raws) == 0 {
		return Project{}, fmt.Errorf("表头下没有数据行")
	}

	// 层级：有 WBS 列 → 存在「X.」子编码的行=分组、其余=叶；无 WBS 列 →
	// 空工期行=分组、其余=叶（两级大纲口径）。
	isGroup := make([]bool, len(raws))
	if _, hasWbs := cols[colWBS]; hasWbs {
		for i, r := range raws {
			if r.wbs == "" {
				continue
			}
			for j, q := range raws {
				if i != j && q.wbs != "" && strings.HasPrefix(q.wbs, r.wbs+".") {
					isGroup[i] = true
					break
				}
			}
		}
	} else {
		for i, r := range raws {
			isGroup[i] = r.dur == ""
		}
	}

	p := Project{Name: sheetNameOr(sheets[0])}
	if minStart.IsZero() {
		p.StartDate = time.Now().Format("2006-01-02")
	} else {
		p.StartDate = minStart.Format("2006-01-02")
	}
	seqID := make(map[int]string, len(raws))
	for i, r := range raws {
		id := fmt.Sprintf("t%d", i+1)
		seqID[r.seq] = id
		dur := 0
		if !isGroup[i] {
			if m := xlsxDurRe.FindString(r.dur); m != "" {
				dur, _ = strconv.Atoi(m)
			}
		}
		// 自定义槽回填：number 槽 strconv.ParseFloat（去空白后整体解析，
		// 「3.5」「3」可解析、带单位/非法值丢弃不报错——与整体容错风格一致）；
		// text 槽原样字符串；空单元格已在采集时跳过。
		var custom map[string]interface{}
		for _, d := range customFieldRegistry {
			s, ok := r.custom[d.key]
			if !ok {
				continue
			}
			if d.isNum {
				if fv, err := strconv.ParseFloat(s, 64); err == nil {
					if custom == nil {
						custom = map[string]interface{}{}
					}
					custom[d.key] = fv
				}
				continue
			}
			if custom == nil {
				custom = map[string]interface{}{}
			}
			custom[d.key] = s
		}
		p.Tasks = append(p.Tasks, Task{
			ID:          id,
			Name:        r.name,
			Duration:    dur,
			Level:       map[bool]int{true: 0, false: 1}[isGroup[i]],
			IsMilestone: !isGroup[i] && dur == 0,
			Custom:      custom,
		})
	}
	// 前置引用 → 搭接：跳过未知序号与涉及分组行的引用，同键去重
	seen := map[string]bool{}
	for i, r := range raws {
		if isGroup[i] || r.preds == "" {
			continue
		}
		for _, part := range strings.FieldsFunc(r.preds, func(rr rune) bool {
			return rr == ',' || rr == '，' || rr == ';' || rr == '；' || rr == '/' || rr == ' '
		}) {
			m := xlsxRefRe.FindStringSubmatch(part)
			if m == nil {
				continue
			}
			no, _ := strconv.Atoi(m[1])
			fromID, ok := seqID[no]
			if !ok {
				continue
			}
			fromIdx := -1
			for j, q := range raws {
				if q.seq == no {
					fromIdx = j
					break
				}
			}
			if fromIdx < 0 || isGroup[fromIdx] || fromIdx == i {
				continue
			}
			typ := strings.ToUpper(m[2])
			if typ == "" {
				typ = "FS"
			}
			lag := 0
			if m[3] != "" {
				lag, _ = strconv.Atoi(m[3])
			}
			link := Link{From: fromID, To: p.Tasks[i].ID, Type: LinkType(typ), Lag: lag}
			k := fmt.Sprintf("%s>%s:%s+%d", link.From, link.To, link.Type, link.Lag)
			if seen[k] {
				continue
			}
			seen[k] = true
			p.Links = append(p.Links, link)
		}
	}
	if err := Validate(&p); err != nil {
		return Project{}, fmt.Errorf("导入内容未通过校验：%s", err)
	}
	return p, nil
}

// sheetNameOr 工作表名作工程名（默认占位名回落）。
func sheetNameOr(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "sheet1", "sheet", "工作表1", "worksheet":
		return "导入计划"
	}
	return s
}
