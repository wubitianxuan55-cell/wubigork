package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/wubigork/wubigork/internal/gaea/knowledge"
	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(knowledgeAdd{}) }

// knowledgeAdd adds a new entry to the knowledge base.
type knowledgeAdd struct{}

func (knowledgeAdd) Name() string { return "knowledge_add" }
func (knowledgeAdd) Description() string {
	return "向工程知识库添加条目：输入标题、分类和正文，自动生成文件名。支持标签和其他元数据。"
}
func (knowledgeAdd) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "title":{"type":"string","description":"条目标题"},
  "category":{"type":"string","description":"分类：规范标准/工程案例/经验总结/材料工艺/法规政策/调查报告/设计方案/其他"},
  "body":{"type":"string","description":"正文内容（Markdown格式）"},
  "tags":{"type":"string","description":"标签，逗号分隔，如'修复,化学氧化'"},
  "phase":{"type":"string","description":"工程阶段：调查/风险评估/修复/管控/监测"},
  "discipline":{"type":"string","description":"专业领域：环境工程/岩土工程/水文地质等"},
  "source":{"type":"string","description":"来源：如'生态环境部'、'实践总结'"}
},
"required":["title","category","body"]
}`)
}
func (knowledgeAdd) ReadOnly() bool                 { return false }
func (knowledgeAdd) CompactDescription() string     { return compactDesc["knowledge_add"] }
func (knowledgeAdd) CompactSchema() json.RawMessage { return compactSchema["knowledge_add"] }

func (knowledgeAdd) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Title      string `json:"title"`
		Category   string `json:"category"`
		Body       string `json:"body"`
		Tags       string `json:"tags,omitempty"`
		Phase      string `json:"phase,omitempty"`
		Discipline string `json:"discipline,omitempty"`
		Source     string `json:"source,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Title == "" || p.Category == "" || p.Body == "" {
		return "", fmt.Errorf("title、category 和 body 为必填项")
	}

	store, err := openKnowledgeStore()
	if err != nil {
		return "", fmt.Errorf("打开知识库失败: %w", err)
	}

	// Parse tags.
	var tags []string
	if p.Tags != "" {
		for _, t := range strings.Split(p.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	now := time.Now()
	e := knowledge.Entry{
		Name:       generateName(p.Title),
		Title:      p.Title,
		Category:   p.Category,
		Phase:      p.Phase,
		Discipline: p.Discipline,
		Tags:       tags,
		Status:     "草稿",
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
		Source:     p.Source,
		Body:       p.Body,
	}

	if err := store.Save(e); err != nil {
		return "", fmt.Errorf("保存失败: %w", err)
	}

	filePath := filepath.Join(store.Dir, knowledge.FileName(e))

	var b strings.Builder
	fmt.Fprintf(&b, "✅ 已保存知识条目\n\n")
	fmt.Fprintf(&b, "**标题**: %s\n", e.Title)
	fmt.Fprintf(&b, "**分类**: %s\n", e.Category)
	fmt.Fprintf(&b, "**文件名**: `%s`\n", filePath)
	if len(tags) > 0 {
		fmt.Fprintf(&b, "**标签**: %s\n", strings.Join(tags, ", "))
	}
	fmt.Fprintf(&b, "**状态**: 草稿\n")
	return b.String(), nil
}

// generateName creates a safe filename from a title by taking the first 20
// characters (letters, digits, CJK characters) and appending a nanosecond
// timestamp to ensure uniqueness.
func generateName(title string) string {
	var b strings.Builder
	for _, r := range title {
		if len([]rune(b.String())) >= 20 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Han, r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "entry"
	}
	return fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
}
