package memory

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ── 记忆注入质量评测（市场调研候选2，docs/gaea-memory-injection-eval-design-2026-09.md）──
//
// 与检索评测（GaeaRetrievalEvalRun，Recall@10 统计门槛）分立：本核测**注入面**——
// work 会话装配进系统提示词的两个记忆块（晨报预载 + 项目本体）的五条结构不变量：
// 预算合规×2 / 归档零泄漏 / 跨空间零泄漏 / 引用零悬空。输入的块必须来自真实
// 构建器（BuildMorningPreloadBlock / BuildProjectBrief）对真实库的产物——本核
// 只做断言，不重建块（重建即同义反复）。确定性：不读时钟、不 IO，同输入同报告。

// EvalPreloadBudget / EvalBriefBudget 是两块的生效预算（与构建器同源：晨报预载
// DefaultMorningPreloadBudget=600；项目本体 projectBriefBudgetRunes=600）。
const EvalPreloadBudget = DefaultMorningPreloadBudget
const EvalBriefBudget = projectBriefBudgetRunes

// InjectionEvalInput 是评测输入（app 层组装：真实块 + 各名单集）。
type InjectionEvalInput struct {
	// Work 是 work 视图的活跃记忆（装配点同口径：SpaceModeIsOn → ListInSpace("work")，
	// mode=off → List() 全量——mode=off 时无跨空间概念）。
	Work []Memory
	// OtherNames 是非 work 空间的活跃记忆名（mode=off 时为空=跳过泄漏检查）。
	OtherNames map[string]bool
	// ArchivedNames 是归档记忆名（归档不泄：任何块都不得出现）。
	ArchivedNames []string
	// PreloadBlock / BriefBlock 是真实构建器对真实库的产物块（可为空=未注入）。
	PreloadBlock string
	BriefBlock   string
	// PreloadBudget / BriefBudget 是生效预算（<=0 时取 Eval*Budget 缺省）。
	PreloadBudget int
	BriefBudget   int
}

// InjectionEvalReport 是注入体检报告（JSON 直出前端）。
type InjectionEvalReport struct {
	PreloadPresent bool `json:"preloadPresent"`
	BriefPresent   bool `json:"briefPresent"`
	PreloadRunes   int  `json:"preloadRunes"`
	BriefRunes     int  `json:"briefRunes"`
	PreloadBudget  int  `json:"preloadBudget"`
	BriefBudget    int  `json:"briefBudget"`
	// EntryCount 是预载块条目数（"- 名称：" 行）；RefCount 是 brief 的 [MEM:] 引用数。
	EntryCount int `json:"entryCount"`
	RefCount   int `json:"refCount"`
	// 固化覆盖（信息面，不计 Passed）：预算内按行收录是设计口径，未全收只透出。
	PinnedTotal   int      `json:"pinnedTotal"`
	PinnedInBrief int      `json:"pinnedInBrief"`
	MissingPinned []string `json:"missingPinned,omitempty"`
	Violations    []string `json:"violations,omitempty"`
	Passed        bool     `json:"passed"`
}

// EvalInjectionBlocks 对真实注入块断言五条不变量（设计档 §口径）。确定性：
// 同输入同报告；块为空=未注入（门控关闭/空库），只统计不判违。
func EvalInjectionBlocks(in InjectionEvalInput) InjectionEvalReport {
	rep := InjectionEvalReport{
		PreloadPresent: strings.TrimSpace(in.PreloadBlock) != "",
		BriefPresent:   strings.TrimSpace(in.BriefBlock) != "",
		PreloadBudget:  in.PreloadBudget,
		BriefBudget:    in.BriefBudget,
	}
	if rep.PreloadBudget <= 0 {
		rep.PreloadBudget = EvalPreloadBudget
	}
	if rep.BriefBudget <= 0 {
		rep.BriefBudget = EvalBriefBudget
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

	// 预载块：预算合规 + 条目名必须在 work 视图 + 条目名不得是归档/他空间名。
	rep.PreloadRunes = utf8.RuneCountInString(in.PreloadBlock)
	if rep.PreloadRunes > rep.PreloadBudget {
		rep.Violations = append(rep.Violations,
			fmt.Sprintf("预载块 %d runes 超预算 %d", rep.PreloadRunes, rep.PreloadBudget))
	}
	for _, name := range preloadEntryNames(in.PreloadBlock) {
		rep.EntryCount++
		checkNameMentioned(&rep, name, workNames, archived, in.OtherNames, "预载")
	}

	// 本体块：预算合规 + [MEM:x] 引用必须在 work 视图 + 固化覆盖。
	rep.BriefRunes = utf8.RuneCountInString(in.BriefBlock)
	if rep.BriefRunes > rep.BriefBudget {
		rep.Violations = append(rep.Violations,
			fmt.Sprintf("本体块 %d runes 超预算 %d", rep.BriefRunes, rep.BriefBudget))
	}
	for _, ref := range ExtractRefNames(in.BriefBlock) {
		rep.RefCount++
		if !workNames[ref] {
			rep.Violations = append(rep.Violations,
				fmt.Sprintf("本体块引用 [MEM:%s] 不在 work 视图（悬空注入）", ref))
			continue
		}
		if pinned[ref] {
			rep.PinnedInBrief++
		}
	}

	// 固化覆盖（信息面）：库内固化名未出现在 brief 文本的，透出清单不判死。
	for _, m := range in.Work {
		if !m.Pinned || strings.TrimSpace(renderableContent(m)) == "" {
			continue
		}
		if !strings.Contains(in.BriefBlock, "[MEM:"+m.Name+"]") {
			rep.MissingPinned = append(rep.MissingPinned, m.Name)
		}
	}

	// 归档/跨空间的精确泄漏路径=条目名与引用键（checkNameMentioned 已覆盖）。
	// 不做块全文的子串兜底：work 条目正文合法提及一个归档/他空间名不是注入
	// 泄漏，子串匹配必然误报（slug 进入描述文本是正常形态）。

	rep.Passed = len(rep.Violations) == 0
	return rep
}

// checkNameMentioned 断言一个被注入提名的名字在 work 视图内且不是归档/他空间名。
func checkNameMentioned(rep *InjectionEvalReport, name string, work, archived map[string]bool, other map[string]bool, what string) {
	switch {
	case !work[name]:
		rep.Violations = append(rep.Violations, fmt.Sprintf("%s条目 %s 不在 work 视图", what, name))
	case archived[name]:
		rep.Violations = append(rep.Violations, fmt.Sprintf("%s条目 %s 是归档记忆", what, name))
	case other[name]:
		rep.Violations = append(rep.Violations, fmt.Sprintf("%s条目 %s 属他空间", what, name))
	}
}

// preloadEntryNames 从预载块解析条目名（渲染口径见 formatMorningPreloadLine：
// 每条一行「- 名称：摘要」）。
func preloadEntryNames(block string) []string {
	var out []string
	for _, ln := range strings.Split(block, "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "- ") {
			continue
		}
		body := strings.TrimPrefix(ln, "- ")
		if i := strings.Index(body, "："); i > 0 {
			out = append(out, strings.TrimSpace(body[:i]))
		}
	}
	return out
}

// renderableContent 判断一条记忆是否可被 brief 渲染出内容行（与 appendLine 的
// 早退口径同源：Description 优先，空则 Title；全空=渲染不出=不进 expected）。
func renderableContent(m Memory) string {
	if d := strings.TrimSpace(m.Description); d != "" {
		return d
	}
	return strings.TrimSpace(m.Title)
}
