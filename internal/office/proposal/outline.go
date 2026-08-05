// Package proposal — 大纲策略与字数预算
package proposal

// outlineStrategyPrompt 返回目录生成策略的提示词片段
func outlineStrategyPrompt(strategy string) string {
	switch strategy {
	case OutlineStrategyScoring:
		return "严格按评标办法生成：一级目录以评标办法中技术标评分项为一级目录结构，评分标准拆分关键点为二级目录，可补充三级目录。"
	case OutlineStrategyFormat:
		return "严格按投标文件格式要求生成：以投标文件格式章节中的目录结构为一级/二级目录，可补充三级目录。"
	default:
		return "参考评标办法评分项与投标文件格式要求，根据项目理解自动补充扩展目录。"
	}
}

// AllocateWordBudget 把总字数递归分配到叶子章节。
// 章按同级平均分配，节按同级均分；最终叶子合计严格等于 total。
func AllocateWordBudget(sections []ProposalSection, total int) {
	if total <= 0 || len(sections) == 0 {
		return
	}
	allocateGroup(sections, total)
}

func allocateGroup(ss []ProposalSection, total int) {
	if len(ss) == 0 {
		return
	}
	base := total / len(ss)
	remainder := total % len(ss)
	for i := range ss {
		target := base
		if i == len(ss)-1 {
			target += remainder
		}
		ss[i].WordTarget = target
		ss[i].Words = countRunes(ss[i].Content)
		if len(ss[i].Children) > 0 {
			allocateGroup(ss[i].Children, target)
		}
	}
}

func countRunes(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// applyWordBudgetToProposal 对方案全部章节递归设置字数目标与字数
func applyWordBudgetToProposal(sections []ProposalSection, total int) {
	AllocateWordBudget(sections, total)
	var walk func(ss []ProposalSection)
	walk = func(ss []ProposalSection) {
		for i := range ss {
			ss[i].Words = countRunes(ss[i].Content)
			walk(ss[i].Children)
		}
	}
	walk(sections)
}
