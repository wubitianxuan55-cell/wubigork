// Package memoryeval 三脑记忆检索质量受控测评（阶段七 7.1-1）。
// 与 internal/app/gaea_retrieval_eval.go（书斋四库·运行态·需 Herdsman
// embedding 引擎）分层：本包零外部依赖，用内置种子语料在本地跑出三域基线——
//
//	story     = 项目 StoryMemory（internal/memory，纯 Go BM25）
//	work      = 办公事实/知识（internal/gaea/bm25，零 token 零网络）
//	persona   = 轻语人格记忆（internal/whisper SQLite FTS + 中文 2-gram LIKE 降级）
//	experience= 经验习得题集（对齐 LongMemEval-V2「习得未来任务所需经验」方向，
//	            ≥20 题；跑在 work 同款 bm25 上，为阶段七 7.2 技能结晶的
//	            复用成功率度量预置评测基建）。
//
// 题集单一事实源 = docs/memory-eval-set.md 的 ```json 代码块；种子语料 = 本包
// seeds.go（题集 expected ID 与种子 ID 漂移由测试硬断言兜住）。
// 指标：recall@10（门槛 0.8，对齐既有 T5-6 口径）+ precision（命中列表中
// 相关占比）。CI 口径：质量低于门槛只告警不阻断（结构性错误才硬失败）。
package memoryeval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// TopK 每条查询参与评估的命中条数（recall@10）。
	TopK = 10
	// Threshold recall@10 门槛（沿 T5-6 ≥ 0.8 口径；软告警不阻断）。
	Threshold = 0.8
	// MinExperienceItems 经验习得题集下限（阶段七 7.1-1 出口判据 ≥ 20）。
	MinExperienceItems = 20
)

// Item 一条评测查询（expected 为种子语料中的稳定 ID，精确相等记命中）。
type Item struct {
	Query    string   `json:"query"`
	Expected []string `json:"expected"`
}

// Set 四域题集（docs/memory-eval-set.md JSON 块）。
type Set struct {
	Story      []Item `json:"story"`
	Work       []Item `json:"work"`
	Persona    []Item `json:"persona"`
	Experience []Item `json:"experience"`
}

// QueryReport 单条查询明细。
type QueryReport struct {
	Query     string   `json:"query"`
	Expected  []string `json:"expected"`
	TopHits   []string `json:"topHits"`
	Recall    float64  `json:"recall"`
	Precision float64  `json:"precision"`
}

// DomainReport 单域汇总报告。
type DomainReport struct {
	Domain       string        `json:"domain"`
	Total        int           `json:"total"`
	RecallAt10   float64       `json:"recallAt10"`
	PrecisionAvg float64       `json:"precisionAvg"`
	Passed       bool          `json:"passed"`
	PerQuery     []QueryReport `json:"perQuery"`
}

// Evaluate 通用评测核心：retrieve 返回按相关度排序的命中 ID（至多 TopK 参与计分），
// 逐条算 recall（命中预期数/预期总数）与 precision（命中中相关占比），汇总均值。
// 预期为空的条目不计入 recall 分母；无命中的条目不计入 precision 分母。
func Evaluate(domain string, retrieve func(query string) []string, items []Item) DomainReport {
	report := DomainReport{Domain: domain, Total: len(items), PerQuery: make([]QueryReport, 0, len(items))}
	recallSum, recallN := 0.0, 0
	precSum, precN := 0.0, 0
	for _, it := range items {
		q := strings.TrimSpace(it.Query)
		expected := make([]string, 0, len(it.Expected))
		for _, e := range it.Expected {
			if e = strings.TrimSpace(e); e != "" {
				expected = append(expected, e)
			}
		}
		row := QueryReport{Query: q, Expected: expected}
		var hits []string
		if q != "" && retrieve != nil {
			for _, h := range retrieve(q) {
				if len(hits) >= TopK {
					break
				}
				if h != "" {
					hits = append(hits, h)
				}
			}
		}
		row.TopHits = hits
		matched := 0
		for _, exp := range expected {
			for _, h := range hits {
				if h == exp {
					matched++
					break
				}
			}
		}
		if len(expected) > 0 {
			row.Recall = float64(matched) / float64(len(expected))
			recallSum += row.Recall
			recallN++
		}
		if len(hits) > 0 {
			row.Precision = float64(matched) / float64(len(hits))
			precSum += row.Precision
			precN++
		}
		report.PerQuery = append(report.PerQuery, row)
	}
	if recallN > 0 {
		report.RecallAt10 = recallSum / float64(recallN)
	}
	if precN > 0 {
		report.PrecisionAvg = precSum / float64(precN)
	}
	report.Passed = report.RecallAt10 >= Threshold
	return report
}

// ParseSet 解析题集文档：取 ```json ... ``` 代码块反序列化为 Set；
// 缺代码块或 JSON 非法返回错误（与 parseRetrievalEvalSet 同款约定）。
func ParseSet(data []byte) (Set, error) {
	var set Set
	text := strings.TrimPrefix(string(data), "\uFEFF")
	var buf []string
	in := false
	for _, ln := range strings.Split(text, "\n") {
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
		return set, fmt.Errorf("题集文档缺少 ```json 代码块")
	}
	if err := json.Unmarshal([]byte(strings.Join(buf, "\n")), &set); err != nil {
		return set, fmt.Errorf("题集 JSON 解析失败: %w", err)
	}
	return set, nil
}

// ResolveSetPath 定位题集文件：环境变量 GAEA_MEMORY_EVAL_SET >
// 自工作目录向上逐级查找 docs/memory-eval-set.md（6 层，覆盖包目录场景）。
func ResolveSetPath() (string, error) {
	const name = "memory-eval-set.md"
	if p := strings.TrimSpace(os.Getenv("GAEA_MEMORY_EVAL_SET")); p != "" {
		if fileExists(p) {
			return p, nil
		}
		return "", fmt.Errorf("GAEA_MEMORY_EVAL_SET 指向的文件不存在: %s", p)
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
	return "", fmt.Errorf("找不到题集 docs/%s（可用环境变量 GAEA_MEMORY_EVAL_SET 指定绝对路径）", name)
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
