package builtin

// websearch_coverage.go — 覆盖门 v0（众神殿 2026-11-05 刀2）。教训锚点：
// grok-r2 keyless 探针「2026年11月3日 人民币汇率」→ bing 返回百度百科「2026年」
// /节假日日历/万年历，非空全离题却被当「实时搜索结果」注入。门做两件事：
//   1. 剥日期/数字后再剥泛词（最新/消息/今天…），剩不出核心实体的查询直接判
//      不过（coverage-generic-query）——「最新 消息」类查询没有可核验实体；
//   2. 要求至少一个核心实体（整词或 bigram，search.Tokenize 同源）命中任一
//      结果的 title+snippet，否则 coverage-miss——结果集与查询词面不相干。
// 已知局限（会话在案）：「苹果 价格」类词面命中语义离题仍会过门；阈值与误杀率
// 由 30 集 E/F 类基线校准，误杀 >10% 退本机 reranker 二档（冷启 4.4s 不进聊天
// 热路径）。

import (
	"regexp"
	"strings"
)

// searchDateNumberRe 剥日期/数字（「2026年」「11月」「3日」「92」「6.71」）：
// 纯 bigram 覆盖率会被日期串灌水（dsh-r2/r3 复核在案）。
var searchDateNumberRe = regexp.MustCompile(`\d{4}\s*年|\d{1,2}\s*月|\d{1,2}\s*日|\d+(?:\.\d+)?`)

// coverageStopTokens 泛词/指代/疑问词：不算核心实体。
var coverageStopTokens = map[string]bool{
	"最新": true, "最近": true, "今天": true, "今日": true,
	"消息": true, "信息": true, "资料": true,
	"怎么样": true, "怎样": true, "多少钱": true, "多少": true,
	"在哪里": true, "在哪儿": true, "什么时候": true, "多久": true,
	"哪些": true, "有没有": true, "什么": true, "怎么": true, "请问": true, "帮我": true,
	"这个": true, "那个": true, "他们": true,
}

// coverageGate 覆盖门 v0（基线校准版）：剥日期/数字与泛词后得到核心实体
// （字段级），要求至少过半实体命中同一结果的 title+snippet（实体级命中=整词
// 或其任一 bigram）。单实体查询退化为「任一命中」；多实体查询拦截词面部分
// 命中的离题结果——基线实测误判案例「火星 房地产 均价 2026」3 实体仅命中
// 「火星」（火星百科/ NASA），任一命中口径误判 OK，过半口径正确判 miss。
// pass=false 时 reason ∈ {coverage-generic-query, coverage-miss}。
func coverageGate(query string, results []SearchResult) (pass bool, reason string) {
	entities := coverageCoreEntities(query)
	if len(entities) == 0 {
		return false, "coverage-generic-query"
	}
	need := (len(entities) + 1) / 2 // ceil(n/2)：至少过半实体命中
	for _, r := range results {
		hay := strings.ToLower(r.Title + "\n" + r.Snippet)
		hits := 0
		for _, e := range entities {
			if entityHits(e, hay) {
				hits++
			}
		}
		if hits >= need {
			return true, ""
		}
	}
	return false, "coverage-miss"
}

// coverageCoreEntities 剥日期/数字与泛词后的核心实体（字段级，不含 bigram 拆分）。
func coverageCoreEntities(query string) []string {
	stripped := searchDateNumberRe.ReplaceAllString(query, " ")
	var out []string
	for _, f := range strings.FieldsFunc(stripped, func(r rune) bool {
		return r == '，' || r == '。' || r == '！' || r == '？' || r == '、' ||
			r == '；' || r == '：' || r == ' ' || r == '\t' || r == '\n' ||
			r == '“' || r == '”' || r == '"' || r == '（' || r == '）' ||
			r == '【' || r == '】' || r == '.' || r == ',' || r == '!' || r == '?'
	}) {
		f = strings.TrimSpace(f)
		if f == "" || coverageStopTokens[f] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// entityHits 实体级命中：整词包含，或其任一相邻 bigram 包含（「中芯国际」类
// 未被结果分词的实体可经 bigram 命中）。
func entityHits(entity, hay string) bool {
	if strings.Contains(hay, strings.ToLower(entity)) {
		return true
	}
	runes := []rune(entity)
	for i := 0; i+1 < len(runes); i++ {
		if strings.Contains(hay, strings.ToLower(string(runes[i:i+2]))) {
			return true
		}
	}
	return false
}
