package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	gconfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costimport"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledgeimport"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// CostImportRowView 是导入预览中的一条候选成本条目（前端可编辑后确认）。
type CostImportRowView struct {
	Name          string  `json:"name"`
	Title         string  `json:"title"`
	Category      string  `json:"category"`
	Unit          string  `json:"unit"`
	Price         float64 `json:"price"`
	Spec          string  `json:"spec"`
	Source        string  `json:"source"`
	Status        string  `json:"status"`
	ExistingName  string  `json:"existingName"`
	ExistingPrice float64 `json:"existingPrice"`
	MatchNote     string  `json:"matchNote"`
	Raw           string  `json:"raw"`
	Skip          bool    `json:"skip"`
	SkipReason    string  `json:"skipReason"`
}

// CostImportPreview 是导入解析结果视图（无确认不落库，写库走 ImportApply）。
type CostImportPreview struct {
	Path     string              `json:"path"`
	FileName string              `json:"fileName"`
	Columns  []string            `json:"columns"`
	Unmapped []string            `json:"unmapped"`
	Rows     []CostImportRowView `json:"rows"`
	Message  string              `json:"message"`
	AIUsed   bool                `json:"aiUsed"`
	// Source 标记识别来源（T5-5a 视觉识别入表）：pdf_text=文本 PDF /
	// pdf_scan=扫描件 PDF（OCR）/ image=图片（OCR）；xlsx/csv 导入为空。
	Source string `json:"source"`
}

// GaeaCostImportPreview 解析 xlsx/csv 报价单/测算表，返回可确认的候选成本条目。
// 复用 costimport 的列自动映射 + 价格归一化 + 既有条目匹配（覆盖更新提示）。
func (a *App) GaeaCostImportPreview(path string) (CostImportPreview, error) {
	abs, _ := resolvePreviewPath(path)
	if abs == "" {
		return CostImportPreview{}, fmt.Errorf("文件不存在: %s", path)
	}
	pv, err := costimport.Parse(abs, a.hubCostStore())
	if err != nil {
		return CostImportPreview{}, err
	}
	return toCostImportPreview(pv, false), nil
}

// extractImportText 可注入的 PDF/文档文本提取（T7-2：测试替换避免真实 PDF/OCR 链路）。
var extractImportText = knowledgeimport.ExtractText

// maxModelInputRunes 单次送入模型的文本上限（rune）。与视觉识别路径
// （visionAINormalize 的 6000 上限）对齐，避免本地模型上下文/超时失控。
const maxModelInputRunes = 6000

// truncateModelInput 把待送入模型的文本截断到 maxModelInputRunes rune，
// 超出部分以截断标注收尾（与 visionAINormalize 的截断口径一致）。
func truncateModelInput(s string) string {
	rs := []rune(s)
	if len(rs) <= maxModelInputRunes {
		return s
	}
	return string(rs[:maxModelInputRunes]) + "\n…（已截断）"
}

// GaeaCostImportAIParse 用办公功能模型把表格行归一化为成本条目（AI 解析）。
// 表头 + 样本行进提示词，模型只输出 JSON 数组；失败时由前端保留自动映射预览。
func (a *App) GaeaCostImportAIParse(path string) (CostImportPreview, error) {
	abs, _ := resolvePreviewPath(path)
	if abs == "" {
		return CostImportPreview{}, fmt.Errorf("文件不存在: %s", path)
	}
	if a.client == nil {
		return CostImportPreview{}, fmt.Errorf("模型服务不可用，请先配置办公功能模型")
	}
	columns, rows, err := costimport.RawTable(abs)
	textFallback := ""
	if err != nil {
		// PDF 等无法直接读表的文件：先做格式转换（文本提取，扫描件自动 OCR），
		// 把转换后的文本交给 AI 归一化——绝不把原始 PDF 字节直接发给模型。
		textFallback, err = extractImportText(abs)
		if err != nil {
			return CostImportPreview{}, fmt.Errorf("解析文件失败: %w", err)
		}
		if !pdfTextUsable(textFallback) {
			return CostImportPreview{}, fmt.Errorf("PDF 转换/OCR 结果异常（提取内容过短或为乱码），请确认文件为可读文本或清晰扫描件后重试")
		}
	}
	if textFallback == "" && len(rows) == 0 {
		return CostImportPreview{}, fmt.Errorf("文件没有数据行")
	}

	// S2-4/D8：成本/报价属敏感数据，走 routeSensitiveLocal——
	// 「敏感域本地化」开关开启时强制本地 Herdsman，关闭时按常规路由（可回云端）。
	featEng, featModel, _ := a.routeSensitiveLocal("office")
	prov, err := provider.NewLLM("", provider.Config{Name: "cost-import-ai", Model: featModel, Engine: featEng})
	if err != nil {
		return CostImportPreview{}, fmt.Errorf("AI 解析模型初始化失败: %w", err)
	}

	var table strings.Builder
	if textFallback != "" {
		table.WriteString(textFallback)
	} else {
		if len(rows) > 50 {
			rows = rows[:50]
		}
		table.WriteString(strings.Join(columns, " | "))
		table.WriteString("\n")
		for _, r := range rows {
			table.WriteString(strings.Join(r, " | "))
			table.WriteString("\n")
		}
	}
	// T7-2：textFallback 与表格文本统一截断到 6000 rune（与 vision 识别对齐），
	// 超长文本不再整段塞给模型（超上下文/超时），截断标注让模型知道内容不完整。
	tableContent := truncateModelInput(table.String())

	const sysPrompt = "你是成本数据提取助手。把报价/成本表格的每一行归一化为成本条目 JSON 数组，规则：\n" +
		"title=材料/设备/项目名称（去掉序号前缀）；spec=规格型号（无则空串）；unit=单位（台班/吨/m³/工日等，无则空串）；\n" +
		"price=数字单价（元，去掉货币符号与千分位，无法识别填 0）；source=来源（取文件中的供应商/产地/备注，无则\"导入文件\"）；\n" +
		"category 只能是 机械/材料/人工/运输/检测/综合单价/其他 之一。\n" +
		"只输出 JSON 数组，不要代码块标记，不要任何解释。"
	user := "请提取以下表格（表头 + 样本行）：\n\n" + tableContent
	if textFallback != "" {
		user = "请从以下文件文本中提取成本条目（若为 Markdown 表格，按表头与行提取）：\n\n" + tableContent
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	ch, err := prov.Stream(ctx, provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: sysPrompt},
			{Role: provider.RoleUser, Content: user},
		},
		Temperature: 0,
	})
	if err != nil {
		return CostImportPreview{}, fmt.Errorf("AI 解析请求失败: %w", err)
	}
	var out strings.Builder
	for {
		select {
		case <-ctx.Done():
			return CostImportPreview{}, ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				return a.finishAIParse(abs, columns, out.String())
			}
			switch chunk.Type {
			case provider.ChunkText:
				out.WriteString(chunk.Text)
			case provider.ChunkError:
				return CostImportPreview{}, chunk.Err
			}
		}
	}
}

// finishAIParse 解析模型 JSON 输出为候选行并做既有匹配。
func (a *App) finishAIParse(abs string, columns []string, raw string) (CostImportPreview, error) {
	start := strings.IndexByte(raw, '[')
	end := strings.LastIndexByte(raw, ']')
	if start < 0 || end <= start {
		return CostImportPreview{}, fmt.Errorf("AI 解析输出不是 JSON 数组")
	}
	var aiRows []struct {
		Title    string  `json:"title"`
		Spec     string  `json:"spec"`
		Unit     string  `json:"unit"`
		Price    float64 `json:"price"`
		Source   string  `json:"source"`
		Category string  `json:"category"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &aiRows); err != nil {
		return CostImportPreview{}, fmt.Errorf("AI 解析输出无效: %w", err)
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
	pv := &costimport.Preview{
		Path:     abs,
		FileName: filepathBase(abs),
		Columns:  columns,
		Rows:     costimport.MatchRows(rows, a.hubCostStore()),
		Message:  "AI 智能解析完成，请核对后确认导入。",
	}
	return toCostImportPreview(pv, true), nil
}

// costEntryUpsertSQL 与 cost.Store.Save 同构的 UPSERT（整批事务内逐条执行，
// 保证事务内写入与常规 Save 的落盘形态完全一致）。
const costEntryUpsertSQL = `
INSERT INTO cost_entries(name, title, category, category_path, unit, price, spec, source, tags, status, body, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(name) DO UPDATE SET
  title=excluded.title, category=excluded.category, category_path=excluded.category_path, unit=excluded.unit,
  price=excluded.price, spec=excluded.spec, source=excluded.source,
  tags=excluded.tags, status=excluded.status, body=excluded.body,
  updated_at=excluded.updated_at`

// marshalCostTags 序列化标签 JSON（与 cost.Store.Save 口径一致）。
func marshalCostTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	if b, err := json.Marshal(tags); err == nil {
		return string(b)
	}
	return "[]"
}

// normalizeCostEntryForTx 校验并归一化一条导入行；无效行返回 error
// （整个批次拒绝，不做「跳过部分行」的半批写入）。
func normalizeCostEntryForTx(e CostEntry) (cost.Entry, error) {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" {
		e.Name = cost.SlugName(e.Title)
	}
	if strings.TrimSpace(e.Title) == "" {
		return cost.Entry{}, fmt.Errorf("标题为空")
	}
	if e.Price < 0 {
		return cost.Entry{}, fmt.Errorf("价格为负（%v）", e.Price)
	}
	if e.Category == "" {
		e.Category = "其他"
	}
	if e.Status == "" {
		e.Status = "现行"
	}
	if e.CategoryPath == "" {
		e.CategoryPath = e.Category
	}
	now := time.Now().UTC()
	return cost.Entry{
		Name: e.Name, Title: strings.TrimSpace(e.Title), Category: e.Category,
		CategoryPath: e.CategoryPath, Unit: e.Unit, Price: e.Price, Spec: e.Spec,
		Source: e.Source, Tags: e.Tags, Status: e.Status, Body: e.Body,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// GaeaCostImportApply 批量写入成本条目（前端确认后的行）；返回成功写入条数。
// T7-2 整批事务：先整体校验，再单事务写入——任一行无效或写库失败，整个批次
// 回滚，不再出现「前几条写入、后几条失败」的半批状态。
func (a *App) GaeaCostImportApply(rows []CostEntry) (int, error) {
	entries := make([]cost.Entry, 0, len(rows))
	for i, e := range rows {
		norm, err := normalizeCostEntryForTx(e)
		if err != nil {
			return 0, fmt.Errorf("第 %d 行无效: %w", i+1, err)
		}
		entries = append(entries, norm)
	}
	if len(entries) == 0 {
		return 0, nil
	}
	if !a.hubCostStore().Available() {
		return 0, fmt.Errorf("成本库不可用")
	}
	if err := db.WithTransaction(gconfig.MemoryUserDir(), func(tx *sql.Tx) error {
		for i, e := range entries {
			if _, err := tx.Exec(costEntryUpsertSQL,
				e.Name, e.Title, e.Category, e.CategoryPath, e.Unit, e.Price, e.Spec, e.Source,
				marshalCostTags(e.Tags), e.Status, e.Body,
				e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339)); err != nil {
				return fmt.Errorf("第 %d 条写入失败: %w", i+1, err)
			}
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return len(entries), nil
}

func toCostImportPreview(pv *costimport.Preview, aiUsed bool) CostImportPreview {
	out := CostImportPreview{
		Path:     pv.Path,
		FileName: pv.FileName,
		Columns:  pv.Columns,
		Unmapped: pv.Unmapped,
		Message:  pv.Message,
		AIUsed:   aiUsed,
		Rows:     make([]CostImportRowView, 0, len(pv.Rows)),
	}
	for _, r := range pv.Rows {
		out.Rows = append(out.Rows, CostImportRowView{
			Name: r.Name, Title: r.Title, Category: r.Category, Unit: r.Unit,
			Price: r.Price, Spec: r.Spec, Source: r.Source, Status: r.Status,
			ExistingName: r.ExistingName, ExistingPrice: r.ExistingPrice,
			MatchNote: r.MatchNote, Raw: r.Raw, Skip: r.Skip, SkipReason: r.SkipReason,
		})
	}
	return out
}

func normalizeAICategory(s string) string {
	switch strings.TrimSpace(s) {
	case "机械", "材料", "人工", "运输", "检测", "综合单价", "其他":
		return strings.TrimSpace(s)
	default:
		return "其他"
	}
}

// pdfTextUsable 判断 PDF 转换/OCR 出的文本是否可读：长度过短、乱码占比
// 过高（问号/替换符）或有效字符过少时判定不可用，避免把乱码喂给模型。
func pdfTextUsable(s string) bool {
	s = strings.TrimSpace(s)
	if len([]rune(s)) < 20 {
		return false
	}
	total := 0
	bad := 0
	meaningful := 0
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if r == '?' || r == '\ufffd' {
			bad++
			continue
		}
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsDigit(r) {
			meaningful++
		}
	}
	if total == 0 {
		return false
	}
	if bad*100/total > 40 {
		return false
	}
	return meaningful >= 10
}

func filepathBase(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}
