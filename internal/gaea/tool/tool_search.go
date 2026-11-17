package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// 检索展开的模型侧入口（codex 蒸馏 T8，tool_search 同构）：工具被 defer 进
// 隐藏集后（藏而可搜），模型经本工具按查询取回目标工具的 schema 文本——
// 隐藏工具保持可按名调用（Registry 语义），所以「schema 以文本出现在工具
// 结果里」即完成物化，agent 循环零改动。
//
// 纪律：tool_search 自身与 websearch 不进索引（BuildSearchIndex 已排除），
// 避免「搜索工具出现在搜索结果里」。

// toolSearchDefaultLimit 是检索默认返回数（上游 TOOL_SEARCH_DEFAULT_LIMIT 同值）。
const toolSearchDefaultLimit = 8

type toolSearchTool struct {
	reg      *Registry
	defLimit int
}

// NewToolSearchTool 构造检索工具（defaultLimit<=0 用内置 8；注册进 reg 时
// 跳过索引排除规则，见上）。
func NewToolSearchTool(reg *Registry, defaultLimit int) Tool {
	if defaultLimit <= 0 {
		defaultLimit = toolSearchDefaultLimit
	}
	return toolSearchTool{reg: reg, defLimit: defaultLimit}
}

func (toolSearchTool) Name() string { return "tool_search" }

func (toolSearchTool) Description() string {
	return "Search the deferred tool catalog by query and return matching tools with their full schemas. Call the returned tool by name afterwards — its schema in the result is authoritative."
}

func (toolSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"Search text (tool name, purpose, keywords; Chinese ok)"},
  "limit":{"type":"integer","description":"Max results (default 8)"}
},
"required":["query"]
}`)
}

func (toolSearchTool) ReadOnly() bool { return true }

func (t toolSearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("tool_search 参数解析失败: %w", err)
	}
	if strings.TrimSpace(p.Query) == "" {
		return "", fmt.Errorf("query is required")
	}
	limit := p.Limit
	if limit <= 0 {
		limit = t.defLimit
	}
	hits := BuildSearchIndex(t.reg).Search(p.Query, limit)
	if len(hits) == 0 {
		return "no matching tools in the deferred catalog — the tools already visible in your tool list cover the rest.", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "found %d tool(s), relevance order — call by name, schema below is authoritative:\n", len(hits))
	for _, h := range hits {
		fmt.Fprintf(&b, "\n## %s", h.Name)
		if h.Namespace != "" {
			fmt.Fprintf(&b, " [%s]", h.Namespace)
		}
		fmt.Fprintf(&b, "\n%s", strings.TrimSpace(h.SearchText))
		// schema 文本即物化：从注册表取同一份 canonical schema。
		if t, ok := t.reg.Get(h.Name); ok {
			fmt.Fprintf(&b, "\nschema: %s", string(t.Schema()))
		}
	}
	return b.String(), nil
}
