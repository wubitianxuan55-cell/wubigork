package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/knowledge"
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

	store, err := openKnowledgeStore()
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
	if len(results) == 0 {
		return "未找到匹配的知识条目。", nil
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

func openKnowledgeStore() (*knowledge.Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".gaea", "knowledge")
	return knowledge.Open(dir)
}

func bodySnippet(body string) string {
	// Use the existing truncate from websearch.go.
	return truncate(body, 200)
}
