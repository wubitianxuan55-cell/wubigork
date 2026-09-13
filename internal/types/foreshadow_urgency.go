package types

// ── 伏笔运行时调度（分层注入 / 紧急度）────────────────────────
//
// 口径来源：docs/distill/01-foreshadow-spec.md §3.2 / §4.1。
// 全部为纯函数，不落库（spec A5：urgency 运行时计算，不存储）。
//
// 对 MuMu 缺陷的修正（spec 决策记录）：
//   - D8：AutoRemind=false 只抑制「近期参考」（L3），不抑制必须回收（L1）
//     与超期（L2）——这两层是硬约束，不因提醒开关而消隐。
//   - D15：紧急度判定从 status==planted 放宽为 status ∉ {revealed/resolved,
//     abandoned}，使 partially_resolved（与 gaea 原生 hinted）也有回收压力。
//   - D7：阈值统一以后端为准（剩余 ≤2 章 = 急需回收），前端不自算。

// 伏笔调度常量（spec P0.5：全仓唯一声明点，禁止散落魔数）。
const (
	// ForeshadowLookaheadDefault 「近期待回收」前瞻章数（spec §4.1，MuMu foreshadow_service.py:562）。
	ForeshadowLookaheadDefault = 5
	// ForeshadowLookaheadMax 前瞻章数上限（spec §4.1，MuMu schemas/foreshadow.py:187）。
	ForeshadowLookaheadMax = 20
	// ForeshadowRemindBeforeDefault RemindBeforeChapters 的缺省语义（提前 5 章进入需关注）。
	ForeshadowRemindBeforeDefault = 5
	// ForeshadowUrgentRemaining 剩余章数 ≤ 该值即为「急需回收」（spec §3.2 统一阈值）。
	ForeshadowUrgentRemaining = 2
	// ForeshadowMaxNewPerChapter 分析同步每章新建伏笔上限（spec §3.3，只约束新建）。
	ForeshadowMaxNewPerChapter = 5
)

// UrgencyLevel 运行时紧急度：0=不紧急 1=需关注 2=急需回收 3=已超期。
//
// 只有「已埋入且有计划回收章」的条目才可能非 0；已回收（含 resolved 别名）
// 与已废弃恒为 0；无计划回收章不猜测（spec A6：缺失值优于猜值，避免假超期）。
func UrgencyLevel(f Foreshadow, currentChapterNum int) int {
	if IsResolvedStatus(f.Status) || f.Status == ForeshadowAbandoned {
		return 0
	}
	target := ChapterNumOf(f.TargetResolveIn)
	if target <= 0 {
		return 0
	}
	remind := f.RemindBeforeChapters
	if remind <= 0 {
		remind = ForeshadowRemindBeforeDefault
	}
	remaining := target - currentChapterNum
	switch {
	case remaining < 0:
		return 3
	case remaining <= ForeshadowUrgentRemaining:
		return 2
	case remaining <= remind:
		return 1
	default:
		return 0
	}
}

// ClassifyResolve 回收时机四值判定（ResolveStatus）。
//
// 已回收/已废弃的条目无「时机」可言，判 no_plan（调用方应先用分层/状态过滤，
// 本判定只对存活条目有意义）；计划章缺失 = no_plan，不猜值。
func ClassifyResolve(f Foreshadow, currentChapterNum int) ResolveStatus {
	target := ChapterNumOf(f.TargetResolveIn)
	if target <= 0 {
		return ResolveNoPlan
	}
	switch {
	case target == currentChapterNum:
		return ResolveMustNow
	case target < currentChapterNum:
		return ResolveOverdue
	default:
		return ResolveNotYet
	}
}

// ForeshadowLayer 生成侧注入分层（spec §4.1 四层 + gaea 扩展第五层）。
//
//	L1 Must   本章必须回收（planted 且计划章 == 当前章）
//	L2 Overdue 超期待回收（planted 且计划章 < 当前章）
//	L3 Near   近期待回收（仅参考，明确禁止本章提前回收）
//	L4 Plant  本章计划埋入（pending 且埋入章 == 当前章）
//	L5 NoPlan 已埋入未规划回收章（gaea 扩展：存量 foreshadows.json 大多没有
//	  target_resolve_in，若严格按 MuMu 四层会整批从注入中消失——回归。
//	  故无计划的存活条目降入兜底层，保留既有「背景约束」行为，优先级最低）
//
// 远期伏笔（计划章 > 当前章+lookahead）不注入任何层（防止干扰，spec §4.1）。
type ForeshadowLayer int

const (
	ForeshadowLayerNone    ForeshadowLayer = iota // 不注入
	ForeshadowLayerMust                           // L1 本章必须回收
	ForeshadowLayerOverdue                        // L2 超期
	ForeshadowLayerNear                           // L3 近期参考
	ForeshadowLayerPlant                          // L4 本章计划埋入
	ForeshadowLayerNoPlan                         // L5 无计划兜底（gaea 扩展）
)

// ForeshadowLayerPriority 预算消耗顺序（值小者优先）：硬约束 L1/L4 声明
// 优先于参考信息，但消耗顺序按 spec §4.3 的 L1→L2→L3→L4 执行，L5 兜底
// 在全部四层之后吃剩余预算。
func ForeshadowLayerPriority(l ForeshadowLayer) int {
	switch l {
	case ForeshadowLayerMust:
		return 0
	case ForeshadowLayerOverdue:
		return 1
	case ForeshadowLayerNear:
		return 2
	case ForeshadowLayerPlant:
		return 3
	case ForeshadowLayerNoPlan:
		return 4
	default:
		return 5
	}
}

// ForeshadowLayerOf 单条目分层判定。返回 None 表示不注入。
//
// 状态口径（D15 扩展）：hinted 与 partially_resolved 视同 planted 参与回收
// 三层（L1/L2/L3/L5）；pending 只参与 L4（本章计划埋入）。
// include_in_context=false 的条目不注入（作者显式排除）；auto_remind=false
// 只把 L3 抑制为 None（D8：L1/L2 硬约束不受提醒开关影响）。
func ForeshadowLayerOf(f Foreshadow, currentChapterNum, lookahead int) ForeshadowLayer {
	if f.IncludeInContext != nil && !*f.IncludeInContext {
		return ForeshadowLayerNone
	}
	if lookahead <= 0 || lookahead > ForeshadowLookaheadMax {
		lookahead = ForeshadowLookaheadDefault
	}
	if IsResolvedStatus(f.Status) || f.Status == ForeshadowAbandoned {
		return ForeshadowLayerNone
	}
	if f.Status == ForeshadowPending {
		// L4：本章计划埋入
		if ChapterNumOf(f.PlantedIn) == currentChapterNum && currentChapterNum > 0 {
			return ForeshadowLayerPlant
		}
		return ForeshadowLayerNone
	}
	// planted / hinted / partially_resolved：按计划回收章分流
	target := ChapterNumOf(f.TargetResolveIn)
	if target <= 0 {
		return ForeshadowLayerNoPlan
	}
	switch {
	case target == currentChapterNum:
		return ForeshadowLayerMust
	case target < currentChapterNum:
		return ForeshadowLayerOverdue
	case target <= currentChapterNum+lookahead:
		if f.AutoRemind != nil && !*f.AutoRemind {
			return ForeshadowLayerNone // D8：仅近期参考层受提醒开关抑制
		}
		return ForeshadowLayerNear
	default:
		return ForeshadowLayerNone // 远期不注入
	}
}
