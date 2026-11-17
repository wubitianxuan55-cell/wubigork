package skill

import "strings"

// 技能选择（codex 蒸馏 T14）：两路（词法整数权重 + 字符 n-gram）各取 50 候选
// → RRF 融合 → 取 top N。不改现有加载/索引结构——本包此前没有检索器，
// 这层是纯新增，接线与否由调用方决定（未接线 = 零行为变化）。

// SelectResult 是一次选择的结果：IDs 是 skills 入参的下标（相关度降序）。
type SelectResult struct {
	IDs []int
	// QueryTermCount 是词法路切出的查询词数。
	QueryTermCount int
	// QueryTruncated / CandidatesTruncated：截断标志按 OR 合并——任何一路
	// 截断都必须让上层知道。
	QueryTruncated      bool
	CandidatesTruncated bool
}

// Select 从技能集中选与查询最相关的至多 limit 个。
// 空查询返回空结果（不猜）。
func Select(skills []Skill, query string, limit int) SelectResult {
	if strings.TrimSpace(query) == "" || len(skills) == 0 {
		return SelectResult{}
	}
	terms, phrase, qTrunc := tokenizeQuery(query)
	qBytes := len(query)

	type scored struct {
		id    int
		lex   uint32
		ngram uint32
	}
	all := make([]scored, len(skills))
	for i := range skills {
		lexScore, _ := scoreWeighted(terms, phrase, skills[i].Name, skills[i].Description, skills[i].Body)
		qg := charGrams(query)
		ngScore, _ := ngramScore(qg, qBytes, skills[i].Name, skills[i].Description, skills[i].Body)
		all[i] = scored{id: i, lex: lexScore, ngram: ngScore}
	}
	// 两路各自取前 50（分数同则 id 升序，稳定）。
	pickTop := func(key func(scored) uint32) []int {
		cands := append([]scored(nil), all...)
		for i := 1; i < len(cands); i++ {
			for j := i; j > 0 && (key(cands[j]) > key(cands[j-1]) ||
				(key(cands[j]) == key(cands[j-1]) && cands[j].id < cands[j-1].id)); j-- {
				cands[j], cands[j-1] = cands[j-1], cands[j]
			}
		}
		if len(cands) > fusionMaxCandidates {
			cands = cands[:fusionMaxCandidates]
		}
		out := make([]int, len(cands))
		for i, c := range cands {
			out[i] = c.id
		}
		return out
	}
	lexicalIDs := pickTop(func(c scored) uint32 { return c.lex })
	charIDs := pickTop(func(c scored) uint32 { return c.ngram })

	f := Fuse([][]int{lexicalIDs, charIDs}, limit)
	return SelectResult{
		IDs:                 f.IDs,
		QueryTermCount:      len(terms),
		QueryTruncated:      qTrunc,
		CandidatesTruncated: len(all) > fusionMaxCandidates,
	}
}
