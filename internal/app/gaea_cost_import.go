package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costimport"
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
	Path     string             `json:"path"`
	FileName string             `json:"fileName"`
	Columns  []string           `json:"columns"`
	Unmapped []string           `json:"unmapped"`
	Rows     []CostImportRowView `json:"rows"`
	Message  string             `json:"message"`
	AIUsed   bool               `json:"aiUsed"`
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
		textFallback, err = knowledgeimport.ExtractText(abs)
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
	prov, err := provider.New("wubigrok", provider.Config{Name: "cost-import-ai", Model: featModel, Engine: featEng})
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

	const sysPrompt = "你是成本数据提取助手。把报价/成本表格的每一行归一化为成本条目 JSON 数组，规则：\n" +
		"title=材料/设备/项目名称（去掉序号前缀）；spec=规格型号（无则空串）；unit=单位（台班/吨/m³/工日等，无则空串）；\n" +
		"price=数字单价（元，去掉货币符号与千分位，无法识别填 0）；source=来源（取文件中的供应商/产地/备注，无则\"导入文件\"）；\n" +
		"category 只能是 机械/材料/人工/运输/检测/综合单价/其他 之一。\n" +
		"只输出 JSON 数组，不要代码块标记，不要任何解释。"
	user := "请提取以下表格（表头 + 样本行）：\n\n" + table.String()
	if textFallback != "" {
		user = "请从以下文件文本中提取成本条目（若为 Markdown 表格，按表头与行提取）：\n\n" + table.String()
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

// GaeaCostImportApply 批量写入成本条目（前端确认后的行）；返回成功写入条数。
func (a *App) GaeaCostImportApply(rows []CostEntry) (int, error) {
	store := a.hubCostStore()
	saved := 0
	for _, e := range rows {
		e.Name = strings.TrimSpace(e.Name)
		if e.Name == "" {
			e.Name = cost.SlugName(e.Title)
		}
		if strings.TrimSpace(e.Title) == "" {
			continue
		}
		if e.Price < 0 {
			continue
		}
		if e.Category == "" {
			e.Category = "其他"
		}
		if e.Status == "" {
			e.Status = "现行"
		}
		if err := store.Save(cost.Entry{
			Name: e.Name, Title: e.Title, Category: e.Category, Unit: e.Unit,
			Price: e.Price, Spec: e.Spec, Source: e.Source, Tags: e.Tags,
			Status: e.Status, Body: e.Body,
		}); err != nil {
			return saved, fmt.Errorf("第 %d 条保存失败: %w", saved+1, err)
		}
		saved++
	}
	return saved, nil
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
