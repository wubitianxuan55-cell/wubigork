package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/knowledgeimport"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// KnowledgeImportRowView 是知识导入预览中的一条候选条目。
type KnowledgeImportRowView struct {
	Name         string   `json:"name"`
	Title        string   `json:"title"`
	Category     string   `json:"category"`
	Phase        string   `json:"phase"`
	Discipline   string   `json:"discipline"`
	Tags         []string `json:"tags"`
	Status       string   `json:"status"`
	Source       string   `json:"source"`
	Body         string   `json:"body"`
	ExistingName string   `json:"existingName"`
	MatchNote    string   `json:"matchNote"`
	SimilarName  string   `json:"similarName"`
	SimilarNote  string   `json:"similarNote"`
	Raw          string   `json:"raw"`
	Skip         bool     `json:"skip"`
	SkipReason   string   `json:"skipReason"`
}

// KnowledgeImportPreview 是知识导入解析结果视图（无确认不落库）。
type KnowledgeImportPreview struct {
	Path     string                    `json:"path"`
	FileName string                    `json:"fileName"`
	Columns  []string                  `json:"columns"`
	Unmapped []string                  `json:"unmapped"`
	Rows     []KnowledgeImportRowView  `json:"rows"`
	Message  string                    `json:"message"`
	AIUsed   bool                      `json:"aiUsed"`
}

// hubKnowledgeStore 打开知识库（与面板/工具同一实例）。
func (a *App) hubKnowledgeStore() (*knowledge.Store, error) {
	return knowledge.Global().Store()
}

// GaeaKnowledgeImportPreview 解析 md/txt/docx/pdf/xlsx/csv 为候选知识条目。
func (a *App) GaeaKnowledgeImportPreview(path string) (KnowledgeImportPreview, error) {
	abs, _ := resolvePreviewPath(path)
	if abs == "" {
		return KnowledgeImportPreview{}, fmt.Errorf("文件不存在: %s", path)
	}
	store, err := a.hubKnowledgeStore()
	if err != nil {
		return KnowledgeImportPreview{}, err
	}
	pv, err := knowledgeimport.Parse(abs, store)
	if err != nil {
		return KnowledgeImportPreview{}, err
	}
	return toKnowledgeImportPreview(pv, false), nil
}

// GaeaKnowledgeImportAIParse 用办公功能模型把文档/表格结构化（AI 提取），
// 支持多主题文档拆分为多条知识条目。
func (a *App) GaeaKnowledgeImportAIParse(path string) (KnowledgeImportPreview, error) {
	abs, _ := resolvePreviewPath(path)
	if abs == "" {
		return KnowledgeImportPreview{}, fmt.Errorf("文件不存在: %s", path)
	}
	if a.client == nil {
		return KnowledgeImportPreview{}, fmt.Errorf("模型服务不可用，请先配置办公功能模型")
	}
	text, err := knowledgeimport.ExtractText(abs)
	if err != nil {
		return KnowledgeImportPreview{}, err
	}
	if r := []rune(strings.TrimSpace(text)); len(r) > 50000 {
		text = string(r[:50000]) + "\n…（已截断，AI 解析使用前 5 万字）"
	}
	if strings.TrimSpace(text) == "" {
		return KnowledgeImportPreview{}, fmt.Errorf("文件没有可提取的内容")
	}

	// 2026-08-28 本地优先强化：知识导入解析属办公功能级调用，优先本地 Herdsman。
	featEng, featModel, _ := a.routeOfficeLocal("office")
	prov, err := provider.NewLLM("", provider.Config{Name: "knowledge-import-ai", Model: featModel, Engine: featEng})
	if err != nil {
		return KnowledgeImportPreview{}, fmt.Errorf("AI 解析模型初始化失败: %w", err)
	}

	const sysPrompt = "你是工程知识库提取助手。把文档/表格内容拆分为知识条目 JSON 数组，规则：\n" +
		"title=条目标题（简短）；category 只能是 规范标准/工程案例/经验总结/材料工艺/法规政策/调查报告/设计方案/其他 之一；\n" +
		"phase 只能是 调查/设计/施工/验收/运维/全程 之一（不确定填空串）；discipline=专业领域（如 环境工程/岩土工程，无则空串）；\n" +
		"tags=标签数组；source=文档文件名；body=该条目的完整正文（保留关键数据、规范条文、要点，Markdown）。\n" +
		"同一文档含多个主题时拆成多条；只输出 JSON 数组，不要代码块标记，不要任何解释。"
	user := "请提取以下内容：\n\n" + text

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	ch, err := prov.Stream(ctx, provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: sysPrompt},
			{Role: provider.RoleUser, Content: user},
		},
		Temperature: 0,
	})
	if err != nil {
		return KnowledgeImportPreview{}, fmt.Errorf("AI 解析请求失败: %w", err)
	}
	var out strings.Builder
	for {
		select {
		case <-ctx.Done():
			return KnowledgeImportPreview{}, ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				return a.finishKnowledgeAIParse(abs, out.String())
			}
			switch chunk.Type {
			case provider.ChunkText:
				out.WriteString(chunk.Text)
			case provider.ChunkError:
				return KnowledgeImportPreview{}, chunk.Err
			}
		}
	}
}

func (a *App) finishKnowledgeAIParse(abs, raw string) (KnowledgeImportPreview, error) {
	start := strings.IndexByte(raw, '[')
	end := strings.LastIndexByte(raw, ']')
	if start < 0 || end <= start {
		return KnowledgeImportPreview{}, fmt.Errorf("AI 解析输出不是 JSON 数组")
	}
	var aiRows []struct {
		Title      string   `json:"title"`
		Category   string   `json:"category"`
		Phase      string   `json:"phase"`
		Discipline string   `json:"discipline"`
		Tags       []string `json:"tags"`
		Source     string   `json:"source"`
		Body       string   `json:"body"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &aiRows); err != nil {
		return KnowledgeImportPreview{}, fmt.Errorf("AI 解析输出无效: %w", err)
	}
	rows := make([]knowledgeimport.Row, 0, len(aiRows))
	for _, r := range aiRows {
		rows = append(rows, knowledgeimport.Row{
			Title:      strings.TrimSpace(r.Title),
			Category:   normalizeKnowledgeCategory(r.Category),
			Phase:      strings.TrimSpace(r.Phase),
			Discipline: strings.TrimSpace(r.Discipline),
			Tags:       r.Tags,
			Source:     strings.TrimSpace(r.Source),
			Body:       strings.TrimSpace(r.Body),
		})
	}
	store, err := a.hubKnowledgeStore()
	if err != nil {
		return KnowledgeImportPreview{}, err
	}
	pv := &knowledgeimport.Preview{
		Path:     abs,
		FileName: filepathBase(abs),
		Rows:     knowledgeimport.MatchRows(rows, store),
		Message:  "AI 智能解析完成，请核对后确认导入。",
	}
	return toKnowledgeImportPreview(pv, true), nil
}

// GaeaKnowledgeImportApply 批量写入确认后的知识条目，返回成功条数。
func (a *App) GaeaKnowledgeImportApply(rows []KnowledgeEntry) (int, error) {
	store, err := a.hubKnowledgeStore()
	if err != nil {
		return 0, err
	}
	saved := 0
	for _, e := range rows {
		if strings.TrimSpace(e.Title) == "" || strings.TrimSpace(e.Body) == "" {
			continue
		}
		if e.Name == "" {
			e.Name = slugFromTitle(e.Title)
		}
		if e.Category == "" {
			e.Category = knowledge.CatOther
		}
		if e.Status == "" {
			e.Status = "现行"
		}
		if err := saveKnowledgeVersioned(store, knowledge.Entry{
			Name: e.Name, Title: e.Title, Category: e.Category, Phase: e.Phase,
			Discipline: e.Discipline, Tags: e.Tags, Status: e.Status, Version: 1,
			Source: e.Source, Body: e.Body,
		}); err != nil {
			return saved, fmt.Errorf("第 %d 条保存失败: %w", saved+1, err)
		}
		saved++
	}
	return saved, nil
}

func toKnowledgeImportPreview(pv *knowledgeimport.Preview, aiUsed bool) KnowledgeImportPreview {
	out := KnowledgeImportPreview{
		Path: pv.Path, FileName: pv.FileName, Columns: pv.Columns,
		Unmapped: pv.Unmapped, Message: pv.Message, AIUsed: aiUsed,
		Rows: make([]KnowledgeImportRowView, 0, len(pv.Rows)),
	}
	for _, r := range pv.Rows {
		out.Rows = append(out.Rows, KnowledgeImportRowView{
			Name: r.Name, Title: r.Title, Category: r.Category, Phase: r.Phase,
			Discipline: r.Discipline, Tags: r.Tags, Status: r.Status, Source: r.Source,
			Body: r.Body, ExistingName: r.ExistingName, MatchNote: r.MatchNote,
			SimilarName: r.SimilarName, SimilarNote: r.SimilarNote,
			Raw: r.Raw, Skip: r.Skip, SkipReason: r.SkipReason,
		})
	}
	return out
}

func normalizeKnowledgeCategory(s string) string {
	switch strings.TrimSpace(s) {
	case "规范标准", "工程案例", "经验总结", "材料工艺", "法规政策", "调查报告", "设计方案", "其他":
		return strings.TrimSpace(s)
	default:
		return knowledge.CatOther
	}
}

func slugFromTitle(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r >= 0x4e00 {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "entry"
	}
	if runes := []rune(name); len(runes) > 60 {
		name = string(runes[:60])
	}
	return name
}

// semanticKnowledgeRecall App 层知识语义召回（持久化向量索引）。
func (a *App) semanticKnowledgeRecall(query string, have []knowledge.Entry, topN int) []knowledge.Entry {
	e := a.localSearchEmbedder()
	if e == nil {
		return nil
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return nil
	}
	store, err := a.hubKnowledgeStore()
	if err != nil {
		return nil
	}
	all := store.ReadAll()
	if len(all) == 0 {
		return nil
	}
	docs := make([]semantic.Doc, len(all))
	keep := make(map[string]bool, len(all))
	for i, e2 := range all {
		docs[i] = semantic.Doc{ID: e2.Name, Text: knowledgeDocString(e2)}
		keep[e2.Name] = true
	}
	_, _ = st.Stale("knowledge", keep)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	hits, err := st.Search(ctx, e, "knowledge", docs, query, topN)
	if err != nil || len(hits) == 0 {
		return nil
	}
	haveNames := make(map[string]bool, len(have))
	for _, h := range have {
		haveNames[h.Name] = true
	}
	byName := make(map[string]knowledge.Entry, len(all))
	for _, e2 := range all {
		byName[e2.Name] = e2
	}
	out := append([]knowledge.Entry{}, have...)
	for _, h := range hits {
		if e2, ok := byName[h.ID]; ok && !haveNames[h.ID] {
			out = append(out, e2)
		}
	}
	return out
}

// rerankKnowledgeResults App 层知识精排（bge-reranker-v2-m3，失败回退）。
func (a *App) rerankKnowledgeResults(query string, list []knowledge.Entry, topN int) []knowledge.Entry {
	if len(list) <= 8 || strings.TrimSpace(query) == "" || topN <= 0 {
		return nil
	}
	r := a.localSearchReranker()
	if r == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if !r.Available(ctx) {
		return nil
	}
	docs := make([]string, len(list))
	for i, e := range list {
		docs[i] = knowledgeDocString(e)
	}
	scored, err := r.Rerank(ctx, query, docs, topN)
	if err != nil || len(scored) == 0 {
		return nil
	}
	out := make([]knowledge.Entry, 0, len(scored))
	for _, s := range scored {
		if s.Index >= 0 && s.Index < len(list) {
			out = append(out, list[s.Index])
		}
	}
	return out
}

func knowledgeDocString(e knowledge.Entry) string {
	var b strings.Builder
	b.WriteString(e.Title)
	if e.Category != "" {
		b.WriteString(" 分类" + e.Category)
	}
	if e.Phase != "" {
		b.WriteString(" 阶段" + e.Phase)
	}
	if e.Discipline != "" {
		b.WriteString(" 专业" + e.Discipline)
	}
	if len(e.Tags) > 0 {
		b.WriteString(" 标签" + strings.Join(e.Tags, ","))
	}
	if e.Source != "" {
		b.WriteString(" 来源" + e.Source)
	}
	body := strings.TrimSpace(e.Body)
	if r := []rune(body); len(r) > 2000 {
		body = string(r[:2000])
	}
	if body != "" {
		b.WriteString("\n" + body)
	}
	return b.String()
}
