package memory

import (
	"fmt"
	"regexp"
	"strings"
)

// ── 记忆注入质量评测（市场调研候选2，docs/gaea-memory-injection-eval-design-2026-09.md）──
//
// 与检索评测（GaeaRetrievalEvalRun，Recall@10 统计门槛）分立：本核测**注入面**——
// work 会话装配进系统提示词的记忆索引的四条结构不变量：预算合规 / 归档零泄漏 /
// 跨空间零泄漏 / 条目零悬空。输入的索引必须来自真实库的渲染产物
// （IndexForInjection）——本核只做断言，不重建（重建即同义反复）。确定性：
// 不读时钟、不 IO，同输入同报告。
//
// 2026-11-25 办公工具梳理批：晨报预载与项目本体两块已停注（与索引逐条重复），
// 体检对象随之收成索引一块。

// EvalIndexBudget 是索引块的注入预算（与 memoryIndexBudgetBytes 同源，字节口径）。
const EvalIndexBudget = memoryIndexBudgetBytes

// InjectionEvalInput 是评测输入（app 层组装：真实索引 + 各名单集）。
type InjectionEvalInput struct {
	// Work 是 work 视图的活跃记忆（装配点同口径：SpaceModeIsOn → ListInSpace("work")，
	// mode=off → List() 全量——mode=off 时无跨空间概念）。
	Work []Memory
	// OtherNames 是非 work 空间的活跃记忆名（mode=off 时为空=跳过泄漏检查）。
	OtherNames map[string]bool
	// ArchivedNames 是归档记忆名（归档不泄：索引不得出现）。
	ArchivedNames []string
	// IndexBlock 是装配点对真实库渲染并截预算后的记忆索引（IndexForInjection）。
	IndexBlock string
}

// InjectionEvalReport 是注入体检报告（JSON 直出前端）。
type InjectionEvalReport struct {
	IndexPresent bool `json:"indexPresent"`
	IndexBytes   int  `json:"indexBytes"`
	IndexBudget  int  `json:"indexBudget"`
	// EntryCount 是索引条目数；Truncated=索引被预算截断（其余条目靠检索可达）。
	EntryCount int  `json:"entryCount"`
	Truncated  bool `json:"truncated"`
	// 固化覆盖（信息面，不计 Passed）：未进索引的固化条目清单。
	PinnedTotal   int      `json:"pinnedTotal"`
	PinnedInIndex int      `json:"pinnedInIndex"`
	MissingPinned []string `json:"missingPinned,omitempty"`
	Violations    []string `json:"violations,omitempty"`
	Passed        bool     `json:"passed"`
}

// indexEntryRe 匹配索引行的链接名：`- [E] [标题](name.md)〔模式〕 — 摘要`。
var indexEntryRe = regexp.MustCompile(`\]\(([^()]+)\.md\)`)

// indexTruncated 报告索引是否带预算截断提示（capMemoryIndex 的后缀）。
func indexTruncated(block string) bool {
	return strings.Contains(block, "记忆索引已截断")
}

// EvalIndexInjection 对真实索引块断言四条不变量。确定性：同输入同报告；
// 索引为空=未注入（空库/记忆关闭），只统计不判违。
func EvalIndexInjection(in InjectionEvalInput) InjectionEvalReport {
	rep := InjectionEvalReport{
		IndexPresent: strings.TrimSpace(in.IndexBlock) != "",
		IndexBudget:  EvalIndexBudget,
	}
	workNames := map[string]bool{}
	pinned := map[string]bool{}
	for _, m := range in.Work {
		workNames[m.Name] = true
		if m.Pinned {
			pinned[m.Name] = true
			rep.PinnedTotal++
		}
	}
	archived := map[string]bool{}
	for _, n := range in.ArchivedNames {
		archived[n] = true
	}

	// 预算合规：截断提示后缀计入容差（与 capMemoryIndex 的口径一致）。
	rep.IndexBytes = len(in.IndexBlock)
	rep.Truncated = indexTruncated(in.IndexBlock)
	tolerance := 0
	if rep.Truncated {
		tolerance = len(memoryIndexHint)
	}
	if rep.IndexBytes > rep.IndexBudget+tolerance {
		rep.Violations = append(rep.Violations,
			fmt.Sprintf("索引块 %d 字节超预算 %d", rep.IndexBytes, rep.IndexBudget))
	}

	seen := map[string]bool{}
	for _, m := range indexEntryRe.FindAllStringSubmatch(in.IndexBlock, -1) {
		name := m[1]
		if seen[name] {
			rep.Violations = append(rep.Violations, fmt.Sprintf("索引条目 %s 重复出现", name))
			continue
		}
		seen[name] = true
		rep.EntryCount++
		checkNameMentioned(&rep, name, workNames, archived, in.OtherNames, "索引")
		if pinned[name] {
			rep.PinnedInIndex++
		}
	}

	// 固化覆盖（信息面）：库内可渲染的固化条目没进索引的，透出清单不判死
	// （预算截断是设计口径；未截断时出现缺项才是真漏）。
	if !rep.Truncated {
		for _, m := range in.Work {
			if !m.Pinned || strings.TrimSpace(renderableContent(m)) == "" || seen[m.Name] {
				continue
			}
			rep.MissingPinned = append(rep.MissingPinned, m.Name)
		}
	}

	// 归档/跨空间的精确泄漏路径=条目链接名（checkNameMentioned 已覆盖）。不做
	// 全文子串兜底：条目摘要合法提及一个归档/他空间名不是注入泄漏。
	rep.Passed = len(rep.Violations) == 0
	return rep
}

// checkNameMentioned 断言一个被注入提名的名字在 work 视图内且不是归档/他空间名。
func checkNameMentioned(rep *InjectionEvalReport, name string, work, archived map[string]bool, other map[string]bool, what string) {
	// 先判归档/他空间再判「不在 work」：泄漏进来的名字必然也不在 work 视图，
	// 倒过来判会把真正的泄漏原因（归档/他空间）笼统报成「不在 work 视图」。
	switch {
	case archived[name]:
		rep.Violations = append(rep.Violations, fmt.Sprintf("%s条目 %s 是归档记忆", what, name))
	case other[name]:
		rep.Violations = append(rep.Violations, fmt.Sprintf("%s条目 %s 属他空间", what, name))
	case !work[name]:
		rep.Violations = append(rep.Violations, fmt.Sprintf("%s条目 %s 不在 work 视图", what, name))
	}
}

// renderableContent 判断一条记忆能否渲染出索引行（描述优先，空则标题；全空=
// 渲染不出=不计入固化覆盖的应进集）。
func renderableContent(m Memory) string {
	if d := strings.TrimSpace(m.Description); d != "" {
		return d
	}
	return strings.TrimSpace(m.Title)
}
