package analysis

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// ── 分析侧候选清单：已埋入伏笔三层渲染（t1-P2）────────────────
//
// 口径来源：docs/distill/01-foreshadow-spec.md §6.1 + docs/distill/01-foreshadow.md
// §5.6（MuMu plot_analyzer.py:196-268 逐字）。替代旧实现把 foreshadows.json
// 整包 JSON 塞进分析 prompt：模型读不动 JSON 清单，更记不住该回填哪个 ID。
//
// 三层（分析注入折叠上限 MuMu :245/:255）：L1 本章必须回收（逐条带
// 「⚠️ 回收时 reference_stable_id 填写: {id}」紧邻指令，这是分层注入里第 1 层
// 独有的关键）；L2 超期 ≤5；L3 其他 ≤10+溢出注记。分层判定走共享入口
// types.ClassifyResolve（D9：全仓不养第二套分层/时机逻辑）。

const (
	// 候选清单渲染上限（MuMu 分析注入折叠：超期 5 / 其他 10）。
	candidateOverdueMax  = 5
	candidateOthersMax   = 10
	candidateContentMax  = 200 // L1 内容截断（MuMu :230）
	candidateHintMax     = 100 // L1 暗示截断（MuMu :238）
	candidateTitleMax    = 20   // 无标题条目的展示名回退截断
)

// RenderForeshadowCandidates 渲染分析侧「已埋入伏笔列表」三层候选清单。
//
// 入选口径：存活（非回收/废弃）、非 pending（未埋入的规划条目不能被回收）、
// include_in_context != false（作者显式排除的条目不进 prompt 面；注意登记表
// 同步池不受此开关影响，见 foreshadow_sync.go）。auto_remind 只约束生成侧
// 近期提醒（D8），分析侧候选不消隐——分析器应当看到全部可回收候选。
func RenderForeshadowCandidates(items []types.Foreshadow, chapterNum int) string {
	type cand struct {
		fs      types.Foreshadow
		cls     types.ResolveStatus
		plantNo int
	}
	var l1, l2, l3 []cand
	for _, f := range items {
		if types.IsResolvedStatus(f.Status) || f.Status == types.ForeshadowAbandoned ||
			f.Status == types.ForeshadowPending {
			continue
		}
		if f.IncludeInContext != nil && !*f.IncludeInContext {
			continue
		}
		c := cand{fs: f, cls: types.ClassifyResolve(f, chapterNum), plantNo: types.ChapterNumOf(f.PlantedIn)}
		switch c.cls {
		case types.ResolveMustNow:
			l1 = append(l1, c)
		case types.ResolveOverdue:
			l2 = append(l2, c)
		default: // not_yet / no_plan → 其他已埋入
			l3 = append(l3, c)
		}
	}
	// 层内确定性排序：埋入章升序，同章按 ID（保证同输入同输出，prompt 缓存可命中）
	stable := func(s []cand) {
		sort.SliceStable(s, func(i, j int) bool {
			if s[i].plantNo != s[j].plantNo {
				return s[i].plantNo < s[j].plantNo
			}
			return s[i].fs.ID < s[j].fs.ID
		})
	}
	stable(l1)
	stable(l2)
	stable(l3)

	if len(l1)+len(l2)+len(l3) == 0 {
		return "（当前没有已埋入待回收的伏笔。如本章埋入新伏笔，用 type='planted' 登记，并给出 estimated_resolve_chapter。）"
	}

	var b strings.Builder
	b.WriteString("【已埋入伏笔列表 - 用于回收匹配】\n")
	b.WriteString("以下是本项目中已埋入但尚未回收的伏笔，分析时如发现章节内容回收了某个伏笔，请使用对应的ID：\n\n")

	// ── 第 1 层：本章必须回收（最详细，逐条紧邻回填指令）──
	if len(l1) > 0 {
		line := strings.Repeat("=", 40)
		b.WriteString(line + "\n【🎯 本章必须回收的伏笔】\n" + line + "\n")
		for i, c := range l1 {
			fs := c.fs
			b.WriteString(fmt.Sprintf("%d. 【ID: %s】%s\n", i+1, fs.ID, displayTitle(fs)))
			b.WriteString(fmt.Sprintf("   埋入章节：第%d章\n", c.plantNo))
			b.WriteString(fmt.Sprintf("   伏笔内容：%s\n", truncateRunes(fs.Description, candidateContentMax)))
			if hint := strings.TrimSpace(fs.HintText); hint != "" {
				b.WriteString(fmt.Sprintf("   埋入暗示：%s\n", truncateRunes(hint, candidateHintMax)))
			}
			b.WriteString(fmt.Sprintf("   ⚠️ 回收时 reference_stable_id 填写: %s\n", fs.ID))
		}
		b.WriteString("\n")
	}

	// ── 第 2 层：超期未回收（≤5）──
	if len(l2) > 0 {
		b.WriteString("【⚠️ 超期未回收伏笔 - 如章节内容回收了请标记】\n")
		for i, c := range l2 {
			if i >= candidateOverdueMax {
				break
			}
			fs := c.fs
			planNo := types.ChapterNumOf(fs.TargetResolveIn)
			b.WriteString(fmt.Sprintf("- 【ID: %s】%s（第%d章埋入，原计划第%d章回收）\n",
				fs.ID, displayTitle(fs), c.plantNo, planNo))
		}
		if len(l2) > candidateOverdueMax {
			b.WriteString(fmt.Sprintf("  ... 还有%d个超期伏笔未列出\n", len(l2)-candidateOverdueMax))
		}
		b.WriteString("\n")
	}

	// ── 第 3 层：其他已埋入（≤10 + 溢出注记）──
	if len(l3) > 0 {
		b.WriteString("【📋 其他已埋入伏笔 - 如章节内容自然回收了请标记】\n")
		for i, c := range l3 {
			if i >= candidateOthersMax {
				break
			}
			fs := c.fs
			b.WriteString(fmt.Sprintf("- 【ID: %s】%s（第%d章埋入）\n", fs.ID, displayTitle(fs), c.plantNo))
		}
		if len(l3) > candidateOthersMax {
			b.WriteString(fmt.Sprintf("  ... 还有%d个伏笔未列出\n", len(l3)-candidateOthersMax))
		}
		b.WriteString("\n")
	}

	// 操作指引（MuMu :265-266）
	b.WriteString("提示：如果章节内容回收了上述任一伏笔，请在 foreshadows 数组中")
	b.WriteString("添加 type='resolved' 的记录，并在 reference_stable_id 填写对应ID；")
	b.WriteString("无法确定对应哪条时不猜，reference_stable_id 留空。")
	return b.String()
}

// displayTitle 展示名：有 Title 用 Title；存量条目多无标题，回退描述前 20 字。
func displayTitle(f types.Foreshadow) string {
	if t := strings.TrimSpace(f.Title); t != "" {
		return t
	}
	return truncateRunes(f.Description, candidateTitleMax)
}
