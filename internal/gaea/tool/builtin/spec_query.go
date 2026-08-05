package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/specdata"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(specQuery{}) }

// specQuery 提供土壤修复领域核心规范的智能索引与问答
type specQuery struct{}

func (specQuery) Name() string { return "spec_query" }

func (specQuery) Description() string {
	return "土壤修复规范智能查询：输入问题（如「砷的超标限值」「详调布点密度」「风评暴露参数」），返回相关规范条文编号+原文+中文解释。内置 HJ 25.1~6、GB 36600、GB 15618、HJ 682 等 15+ 核心规范。"
}

func (specQuery) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "question":{"type":"string","description":"查询问题，如「砷的管控限值」「布点密度要求」「风评暴露参数默认值」"}
},
"required":["question"]
}`)
}

func (specQuery) ReadOnly() bool { return true }

func (specQuery) CompactDescription() string     { return compactDesc["spec_query"] }
func (specQuery) CompactSchema() json.RawMessage { return compactSchema["spec_query"] }

// (EXW) 暴露参数后续可追加
func (specQuery) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Question == "" {
		return "", fmt.Errorf("question 不能为空")
	}

	// 优先从记忆中枢知识库检索（已入库的规范条文）
	if st, err := knowledge.Global().Store(); err == nil && st != nil {
		if entries := knowledge.Search(st, p.Question, knowledge.Filter{Category: knowledge.CatStandard}); len(entries) > 0 {
			var b strings.Builder
			fmt.Fprintf(&b, "🔍 查询「%s」找到 %d 条相关规范条文（知识库）：\n\n", p.Question, len(entries))
			for i, e := range entries {
				if i >= 5 {
					break
				}
				fmt.Fprintf(&b, "━━━ [%d/%d] %s ━━━\n", i+1, len(entries), e.Title)
				fmt.Fprintf(&b, "📋 原文：%s\n", e.Body)
				if i < 4 && i < len(entries)-1 {
					fmt.Fprintf(&b, "\n")
				}
			}
			return tool.WrapText(b.String()), nil
		}
	}

	q := strings.ToLower(strings.TrimSpace(p.Question))

	// 尝试匹配最相关的规范条文
	var matches []specdata.SpecEntry
	bestScore := 0

	// 关键词权重映射（用于匹配评分）
	keywordWeight := map[string]int{
		// 污染物
		"砷": 3, "as": 3, "镉": 3, "cd": 3, "铬": 3, "cr": 3, "铜": 3, "cu": 3,
		"铅": 3, "pb": 3, "汞": 3, "hg": 3, "镍": 3, "ni": 3, "锌": 3, "zn": 3,
		"氰化物": 3, "石油烃": 3, "六价铬": 3, "voc": 3, "svoc": 3,
		// 规范/标准概念
		"筛选值": 4, "管控值": 4, "限值": 3, "标准": 2, "达标": 2,
		// 布点
		"布点": 5, "采样": 4, "网格": 3, "密度": 4, "深度": 3,
		// 调查阶段
		"初调": 4, "详调": 4, "初步调查": 4, "详细调查": 4,
		// 风评
		"暴露": 4, "风评": 4, "风险评估": 4, "致癌": 3, "风险": 2,
		// 修复
		"修复": 3, "技术": 2, "方案": 2, "稳定化": 3, "氧化": 3, "sve": 3,
		// 分类过滤
		"农用地": 5, "建设用地": 4, "一类用地": 4, "二类用地": 4,
		// 流程
		"报告": 2, "效果评估": 3, "监测": 2, "验收": 2,
	}

	// 对每条条文进行评分
	for _, entry := range specdata.SpecLibrary() {
		score := 0
		text := strings.ToLower(entry.Code + " " + entry.Clause + " " + entry.Title + " " + entry.Content + " " + entry.Explanation)

		// 关键词匹配评分
		for kw, w := range keywordWeight {
			if strings.Contains(q, kw) {
				if strings.Contains(text, kw) {
					score += w * 2 // 匹配上的加权
				}
			}
		}

		// 问题中的每个字命中加分
		for _, r := range q {
			if r > 127 && strings.ContainsRune(text, r) {
				score++
			}
		}

		// 精确短语匹配加分（连续2字以上）
		runes := []rune(q)
		for i := 0; i < len(runes)-1; i++ {
			bigram := string(runes[i : i+2])
			if strings.Contains(text, bigram) {
				score += 2
			}
		}

		if score > bestScore {
			bestScore = score
		}
		if score > 3 {
			matches = append(matches, entry)
		}
	}

	// 如果最佳匹配分数太低，提供分类导航
	if bestScore < 5 || len(matches) == 0 {
		// 尝试按分类匹配
		categories := map[string]string{
			"标准": "GB 36600、GB 15618 中的筛选值和管控值（限值）",
			"调査": "HJ 25.1 三阶段调查流程",
			"布点": "HJ 25.1 初调和详调的采样布点要求",
			"风评": "HJ 25.3 风险评估模型与参数",
			"修复": "HJ 25.2、HJ 25.4 修复技术与方案编制",
			"监测": "HJ 25.5、HJ 1185 监测技术规范",
			"评估": "HJ 25.6 修复效果评估",
			"管控": "HJ 25.2 制度控制措施",
			"术语": "HJ 682 术语定义",
			"勘察": "CJJ/T 89 市政勘察",
		}
		var hints []string
		for cat, desc := range categories {
			if strings.Contains(q, cat) {
				hints = append(hints, fmt.Sprintf("%s → %s", cat, desc))
			}
		}

		if len(hints) > 0 {
			out := fmt.Sprintf("未找到精确匹配的条文。根据问题类别「%s」，可查询以下分类信息：\n", p.Question)
			for _, h := range hints {
				out += "  " + h + "\n"
			}
			return tool.WrapText(out), nil
		}

		return tool.WrapText(fmt.Sprintf(
			"未找到与「%s」直接匹配的规范条文。以下为内置规范全表：\n\n%s",
			p.Question, formatSpecIndex())), nil
	}

	// 按相关性排序（分数降序）
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			scoreI := scoreEntry(q, matches[i])
			scoreJ := scoreEntry(q, matches[j])
			if scoreJ > scoreI {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// 最多返回5条
	if len(matches) > 5 {
		matches = matches[:5]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "🔍 查询「%s」找到 %d 条相关规范条文：\n\n", p.Question, len(matches))
	for i, m := range matches {
		fmt.Fprintf(&b, "━━━ [%d/%d] %s %s %s ━━━\n", i+1, len(matches), m.Code, m.Clause, m.Title)
		fmt.Fprintf(&b, "📋 原文：%s\n", m.Content)
		fmt.Fprintf(&b, "💡 解释：%s\n", m.Explanation)
		if i < len(matches)-1 {
			fmt.Fprintf(&b, "\n")
		}
	}
	return tool.WrapText(b.String()), nil
}

// scoreEntry 计算单条条文与查询的相关度分数
func scoreEntry(q string, e specdata.SpecEntry) int {
	score := 0
	text := strings.ToLower(e.Code + " " + e.Clause + " " + e.Title + " " + e.Content + " " + e.Explanation)
	ql := strings.ToLower(q)

	// 整词匹配
	for _, word := range strings.Fields(ql) {
		if strings.Contains(text, word) {
			score += 5
		}
	}

	// 中文字匹配
	for _, r := range ql {
		if r > 127 && strings.ContainsRune(text, r) {
			score++
		}
	}
	return score
}

// formatSpecIndex 列出所有已索引的规范
func formatSpecIndex() string {
	seen := make(map[string]bool)
	var codes []string
	for _, e := range specdata.SpecLibrary() {
		if !seen[e.Code] {
			seen[e.Code] = true
			codes = append(codes, fmt.Sprintf("  • %s %s", e.Code, e.Title))
		}
	}
	return strings.Join(codes, "\n")
}
