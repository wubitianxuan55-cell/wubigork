package app

// 检索质量受控测评（阶段 5 T5-6）：Recall@10 回归门槛。
// 受控测评（v2.18/2.19）只覆盖模型速度，检索质量（语义召回）此前无任何量化；
// 本文件用 docs/retrieval-eval-set.md 查询集（真实业务查询 + 预期命中标注）驱动
// 现有跨库统一语义检索（GaeaSemanticSearch：cost/knowledge/office/file 四库），
// 逐条算召回率、汇总 recall@10，与门槛 0.8（T5-6：Recall@10 ≥ 0.8）比较得出
// passed，供前端「检索质量受控测评」面板展示（风格参照模型中心测评区）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// retrievalEvalThreshold Recall@10 回归门槛（T5-6：≥ 0.8）。
const retrievalEvalThreshold = 0.8

// retrievalEvalTopK 每条查询参与评估的命中条数（Recall@10）。
const retrievalEvalTopK = 10

// RetrievalEvalReport 检索质量受控测评汇总报告。
type RetrievalEvalReport struct {
	Total      int                  `json:"total"`      // 参与测评的查询数
	Threshold  float64              `json:"threshold"`  // Recall@10 门槛（0.8）
	RecallAt10 float64              `json:"recallAt10"` // 平均召回率（各条 recall 的均值）
	Passed     bool                 `json:"passed"`     // recallAt10 >= threshold
	PerQuery   []RetrievalEvalQuery `json:"perQuery"`   // 逐条明细
	// Note 命中判定规则说明（name 子串匹配可算命中，在报告注明）。
	Note string `json:"note,omitempty"`
}

// RetrievalEvalQuery 单条查询的测评明细（expected/topHits 为 "kind:name" 形式，
// 便于前端直接对比）。
type RetrievalEvalQuery struct {
	Query    string   `json:"query"`
	Expected []string `json:"expected"`
	TopHits  []string `json:"topHits"`
	Recall   float64  `json:"recall"`
}

// RetrievalEvalItem 查询集里的一条查询（docs/retrieval-eval-set.md JSON 块）。
type RetrievalEvalItem struct {
	Query    string             `json:"query"`
	Expected []RetrievalEvalHit `json:"expected"`
}

// RetrievalEvalHit 预期命中标注：kind 为 cost/knowledge/office/file，
// name 为目标条目名或文件相对路径。
type RetrievalEvalHit struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// GaeaRetrievalEvalRun 运行检索质量受控测评：解析查询集 → 每条 query 调跨库
// 统一语义检索（取前 10）→ 单条 recall → 汇总 recall@10（门槛 0.8）。
// 引擎不可用（本地 embedding 未配置）时返回 error 提示先启用 Herdsman bge-m3。
func (a *App) GaeaRetrievalEvalRun() (RetrievalEvalReport, error) {
	path, err := resolveRetrievalEvalSet()
	if err != nil {
		return RetrievalEvalReport{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return RetrievalEvalReport{}, fmt.Errorf("读取查询集失败: %w", err)
	}
	items, err := parseRetrievalEvalSet(data)
	if err != nil {
		return RetrievalEvalReport{}, fmt.Errorf("解析查询集失败: %w", err)
	}
	return a.runRetrievalEval(items)
}

// runRetrievalEval 测评核心（独立可测：迷你查询集内嵌，不依赖 docs 文件）。
func (a *App) runRetrievalEval(items []RetrievalEvalItem) (RetrievalEvalReport, error) {
	e := a.localSearchEmbedder()
	if e == nil {
		return RetrievalEvalReport{}, fmt.Errorf("本地 embedding 未配置（Herdsman bge-m3），请先在模型中心启用 bge-m3 后重试")
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return RetrievalEvalReport{}, fmt.Errorf("向量索引存储不可用")
	}
	if len(items) == 0 {
		return RetrievalEvalReport{}, fmt.Errorf("查询集为空")
	}

	report := RetrievalEvalReport{
		Total:     len(items),
		Threshold: retrievalEvalThreshold,
		PerQuery:  make([]RetrievalEvalQuery, 0, len(items)),
		Note:      "匹配规则：expected 与 topHits 均为 kind:name；同 kind 且 name 精确相等或互为子串记命中",
	}
	recallSum := 0.0
	scored := 0
	for _, it := range items {
		q := strings.TrimSpace(it.Query)
		if q == "" {
			continue // 空查询不参与评分（不计入 recall@10 分母）
		}
		expected := make([]string, 0, len(it.Expected))
		for _, h := range it.Expected {
			if h.Kind == "" || h.Name == "" {
				continue
			}
			expected = append(expected, h.Kind+":"+h.Name)
		}
		row := RetrievalEvalQuery{Query: q, Expected: expected}
		hits, err := a.GaeaSemanticSearch(q)
		if err != nil {
			// 单条查询失败不中断整轮测评：记为空命中，便于整体跑完看趋势。
			report.PerQuery = append(report.PerQuery, row)
			continue
		}
		for i, h := range hits {
			if i >= retrievalEvalTopK {
				break
			}
			row.TopHits = append(row.TopHits, h.Kind+":"+h.Name)
		}
		row.Recall = evalRecall(expected, row.TopHits)
		report.PerQuery = append(report.PerQuery, row)
		if len(expected) > 0 {
			recallSum += row.Recall
			scored++
		}
	}
	if scored > 0 {
		report.RecallAt10 = recallSum / float64(scored)
	}
	report.Passed = report.RecallAt10 >= retrievalEvalThreshold
	return report, nil
}

// evalRecall 计算单条查询的召回率：命中预期数 / 预期总数（expected 为空记 0）。
func evalRecall(expected, topHits []string) float64 {
	if len(expected) == 0 {
		return 0
	}
	matched := 0
	for _, exp := range expected {
		ek, en, ok := splitEvalKey(exp)
		if !ok {
			continue
		}
		for _, hit := range topHits {
			hk, hn, ok2 := splitEvalKey(hit)
			if !ok2 {
				continue
			}
			if evalHitMatched(ek, en, hk, hn) {
				matched++
				break
			}
		}
	}
	return float64(matched) / float64(len(expected))
}

// splitEvalKey 拆分 "kind:name"；kind 或 name 为空视为非法键。
func splitEvalKey(key string) (kind, name string, ok bool) {
	i := strings.Index(key, ":")
	if i <= 0 || i == len(key)-1 {
		return "", "", false
	}
	return key[:i], key[i+1:], true
}

// evalHitMatched 命中判定：kind 相同且 name 精确相等，或互为子串（报告注明）。
func evalHitMatched(expectedKind, expectedName, hitKind, hitName string) bool {
	if expectedKind != hitKind {
		return false
	}
	if expectedName == "" || hitName == "" {
		return expectedName == hitName
	}
	return expectedName == hitName ||
		strings.Contains(hitName, expectedName) ||
		strings.Contains(expectedName, hitName)
}

// parseRetrievalEvalSet 解析查询集文档：取 ```json ... ``` 代码块反序列化为
// []RetrievalEvalItem；缺少代码块或 JSON 非法返回错误。
func parseRetrievalEvalSet(data []byte) ([]RetrievalEvalItem, error) {
	text := strings.TrimPrefix(string(data), "\uFEFF")
	lines := strings.Split(text, "\n")
	var buf []string
	in := false
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if !in {
			if strings.HasPrefix(t, "```") && strings.Trim(strings.TrimPrefix(t, "```"), " \t") == "json" {
				in = true
			}
			continue
		}
		if strings.HasPrefix(t, "```") {
			break
		}
		buf = append(buf, ln)
	}
	if !in || len(buf) == 0 {
		return nil, fmt.Errorf("查询集文档缺少 ```json 代码块")
	}
	var items []RetrievalEvalItem
	if err := json.Unmarshal([]byte(strings.Join(buf, "\n")), &items); err != nil {
		return nil, fmt.Errorf("查询集 JSON 解析失败: %w", err)
	}
	return items, nil
}

// resolveRetrievalEvalSet 定位查询集文件：环境变量 GAEA_RETRIEVAL_EVAL_SET >
// 工作区 docs/retrieval-eval-set.md > 自工作目录向上逐级查找（开发目录场景）。
func resolveRetrievalEvalSet() (string, error) {
	const name = "retrieval-eval-set.md"
	if p := strings.TrimSpace(os.Getenv("GAEA_RETRIEVAL_EVAL_SET")); p != "" {
		if fileExists(p) {
			return p, nil
		}
		return "", fmt.Errorf("GAEA_RETRIEVAL_EVAL_SET 指向的文件不存在: %s", p)
	}
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	for depth := 0; depth < 6; depth++ {
		cand := filepath.Join(dir, "docs", name)
		if fileExists(cand) {
			return cand, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("找不到查询集 docs/%s（可用环境变量 GAEA_RETRIEVAL_EVAL_SET 指定绝对路径）", name)
}
