package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/semantic"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(knowledgeSearch{}) }

// knowledgeSearch searches the knowledge base.
type knowledgeSearch struct{}

func (knowledgeSearch) Name() string { return "knowledge_search" }
func (knowledgeSearch) Description() string {
	return "搜索工程知识库：输入关键词，按标题/标签/正文匹配，支持分类和标签过滤。"
}
func (knowledgeSearch) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"搜索关键词（可选，不填返回全部分类概览）"},
  "category":{"type":"string","description":"分类过滤（可选）：规范标准/工程案例/经验总结/材料工艺/法规政策/调查报告/设计方案/其他"},
  "tag":{"type":"string","description":"标签过滤（可选）"}
}
}`)
}
func (knowledgeSearch) ReadOnly() bool                 { return true }
func (knowledgeSearch) CompactDescription() string     { return compactDesc["knowledge_search"] }
func (knowledgeSearch) CompactSchema() json.RawMessage { return compactSchema["knowledge_search"] }

func (knowledgeSearch) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query    string `json:"query,omitempty"`
		Category string `json:"category,omitempty"`
		Tag      string `json:"tag,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}

	store, err := knowledge.Global().Store()
	if err != nil {
		return "", fmt.Errorf("打开知识库失败: %w", err)
	}

	filter := knowledge.Filter{
		Category: p.Category,
		Tag:      p.Tag,
	}

	if p.Query == "" && p.Category == "" && p.Tag == "" {
		return knowledgeOverview(store)
	}

	results := knowledge.Search(store, p.Query, filter)
	// 语义召回：关键词召回不足（<3）时用本地 bge-m3 补召回（别名/口语表达）。
	if len(results) < 3 && strings.TrimSpace(p.Query) != "" {
		if sem := semanticKnowledgeRecall(store, p.Query, results, 10); len(sem) > 0 {
			results = sem
		}
	}
	if len(results) == 0 {
		return "未找到匹配的知识条目。", nil
	}
	// 本地语义精排（bge-reranker-v2-m3）：候选多时提升排序精度，失败回退。
	if reranked := rerankKnowledgeResults(p.Query, results, 20); len(reranked) > 0 {
		results = reranked
	} else if len(results) > 20 {
		results = results[:20]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## 知识库搜索结果\n\n")
	for _, e := range results {
		dateStr := ""
		if !e.UpdatedAt.IsZero() {
			dateStr = e.UpdatedAt.Format("2006-01-02")
		}
		tags := strings.Join(e.Tags, ", ")
		snippet := bodySnippet(e.Body)

		fmt.Fprintf(&b, "### %s\n\n", e.Title)
		fmt.Fprintf(&b, "**分类**: %s", e.Category)
		if tags != "" {
			fmt.Fprintf(&b, " | **标签**: %s", tags)
		}
		if dateStr != "" {
			fmt.Fprintf(&b, " | **更新**: %s", dateStr)
		}
		b.WriteString("\n\n")
		b.WriteString(snippet)
		b.WriteString("\n\n---\n\n")
	}

	return tool.WrapText(b.String()), nil
}

// semanticKnowledgeRecall 知识库语义召回：持久化向量索引（kind=knowledge）。
func semanticKnowledgeRecall(store *knowledge.Store, query string, have []knowledge.Entry, topN int) []knowledge.Entry {
	e := costEmbedder()
	if e == nil {
		return nil
	}
	st := openSemanticStore()
	if st == nil || !st.Available() {
		return nil
	}
	all := store.ReadAll()
	if len(all) == 0 {
		return nil
	}
	docs := make([]semantic.Doc, len(all))
	keep := make(map[string]bool, len(all))
	for i, e2 := range all {
		docs[i] = semantic.Doc{ID: e2.Name, Text: knowledgeDocText(e2)}
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

// rerankKnowledgeResults 知识条目本地精排（失败回退原顺序）。
func rerankKnowledgeResults(query string, list []knowledge.Entry, topN int) []knowledge.Entry {
	if len(list) <= 8 || strings.TrimSpace(query) == "" || topN <= 0 {
		return nil
	}
	r := costReranker()
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
		docs[i] = knowledgeDocText(e)
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

func knowledgeDocText(e knowledge.Entry) string {
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

func knowledgeOverview(store *knowledge.Store) (string, error) {
	list := store.List()
	if len(list) == 0 {
		return "知识库为空。通过对话让 AI 记录：'帮我把这段经验保存到知识库'", nil
	}

	// Count by category.
	catCount := make(map[string]int)
	for _, s := range list {
		catCount[s.Category]++
	}

	var b strings.Builder
	b.WriteString("## 知识库概览\n\n")
	b.WriteString("| 分类 | 条目数 |\n|------|--------|\n")
	total := 0
	for _, cat := range []string{
		knowledge.CatStandard, knowledge.CatCase, knowledge.CatExperience,
		knowledge.CatMaterial, knowledge.CatRegulation, knowledge.CatSurvey,
		knowledge.CatDesign, knowledge.CatOther,
	} {
		if count, ok := catCount[cat]; ok {
			fmt.Fprintf(&b, "| %s | %d |\n", cat, count)
			total += count
		}
	}
	fmt.Fprintf(&b, "| **合计** | **%d** |\n\n", total)
	b.WriteString("使用 `knowledge_search` 搜索或 `knowledge_add` 添加条目。")
	return b.String(), nil
}

func bodySnippet(body string) string {
	// Use the existing truncate from websearch.go.
	return truncate(body, 200)
}
