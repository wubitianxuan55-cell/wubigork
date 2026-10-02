package app

// ── 位置感知注入：分支与章节计划共用 10 章一阶段的起承转合节奏 ──
//
// v4.452.0 作者口径：每 10 章一个阶段，阶段内按 起承转合 推进。本文件把
// 【本章位置与XX任务】语境段按位置段切换——分支的分歧轴与必带项、章节计划的
// 任务纪律都随阶段位置走，不再全位置共用一个问法。
//
// 阶段模型唯一真相在 stagePhaseOf（create_chapter_stage.go）：
//   第1章=全书开篇；11/21/…=新阶段开篇；阶段内 2-3=起、4-6=承、7-9=转、10=合（阶段收官）。
//
// 与开篇/长跑教义的一致性：开篇三分支的「事发之前」门被要求第一章内抵达引爆
// 点（对齐 create-chapter-first 的钩子 mandate 与「迟进早出」）；阶段收官/开篇
// 的余波纪律与 stage-recap 的合账口径同源（合账已随 stageLength 收紧为每 10 章）。
//
// 位置未知（chapterNum<=0，如旧轻量路径）：回落常规推进——不作阶段特判。

import (
	"fmt"
	"strings"
)

// branchPositionContext 返回注入分支请求的【本章位置与分支任务】段。
// chapterNum>0 按章号定位置；chapterNum<=0（无节点轻量路径）按是否有前文
// 二分：无前文=开篇，有前文=常规（阶段边界章号未知，不作阶段特判）。
func branchPositionContext(chapterNum int, prevSummary string) string {
	_, _, ph := stagePhaseOf(chapterNum)
	switch ph {
	case phaseOpening:
		return branchTaskOpening()
	case phaseStageStart:
		return branchTaskStageStart(chapterNum)
	case phaseQi:
		return branchTaskQi(chapterNum)
	case phaseCheng:
		return branchTaskCheng(chapterNum)
	case phaseZhuan:
		return branchTaskZhuan(chapterNum)
	case phaseHe:
		return branchTaskStageEnd(chapterNum)
	default:
		if strings.TrimSpace(prevSummary) == "" {
			return branchTaskOpening()
		}
		return branchTaskMid()
	}
}

// branchStagePrefix 阶段位置段的公共头部：本阶段区间 + 阶段内位置。
func branchStagePrefix(chapterNum int, pos int, name string) string {
	stage := (chapterNum-1)/stageLength + 1
	start, end := stageBounds(stage)
	return fmt.Sprintf("本阶段=第%d~%d章（每10章一阶段），本章是阶段内第%d章——「%s」段。", start, end, pos, name)
}

// branchTaskOpening 开篇：三分支=三扇门（切入点分歧轴）。
func branchTaskOpening() string {
	return "【本章位置与分支任务：开篇】这是全书的第一章。开篇分支的分歧轴不是「发生什么事」，而是「从哪扇门进同一个故事」——三个分支必须是三种不同的切入点：①事发瞬间硬切（直接落进引爆事件）②事发之后倒推（从后果与谜团开场往回揭）③事发之前建立（先立人物与日常再引爆，第一章内必须抵达引爆点）。每支的 summary 按顺序写明：切入点＋第一画面（谁·在哪·正做什么）＋基调承诺（这本书是哪种爽）＋预支的代价；三支的基调承诺必须可辨地不同。"
}

// branchTaskStageStart 新阶段开篇：三分支=新阶段的三种可能（新驱动性质分歧轴）。
func branchTaskStageStart(chapterNum int) string {
	return fmt.Sprintf("【本章位置与分支任务：新阶段开篇】这是第%d章——新阶段的第一章（上一阶段已收官，本章是本阶段「起」之首）。三个分支＝新阶段的三种可能，分歧轴是新驱动的性质，三支各走一种：换目标（旧目标了结后要什么）/换对手（矛盾换了主体）/换问题类型（外部冲突转内部或反向）。每支的 summary 按顺序写明：①如何承接上一阶段的余波与代价（不假装无事发生）②新阶段的核心驱动③旧线头如何自然延续④本章末的新阶段钩子。", chapterNum)
}

// branchTaskQi 起（阶段内 2-3）：三分支=阶段目标的三种开局推进路线。
func branchTaskQi(chapterNum int) string {
	_, pos, _ := stagePhaseOf(chapterNum)
	return "【本章位置与分支任务：阶段之「起」】" + branchStagePrefix(chapterNum, pos, "起") +
		"新驱动已立，目标待落地。三个分支＝阶段核心目标的三种开局推进路线，分歧轴是路线性质：①正面主攻（直取阶段核心目标）②侧翼支线（先经营支线与资源再图主线）③暗线布局（埋子设局，本章只露冰山一角）。每支的 summary 按顺序写明：阶段目标的落地路径＋第一波实质进展＋初始代价；三支的路线必须可辨地不同。"
}

// branchTaskCheng 承（阶段内 4-6）：三分支=三种升级形态。
func branchTaskCheng(chapterNum int) string {
	_, pos, _ := stagePhaseOf(chapterNum)
	return "【本章位置与分支任务：阶段之「承」】" + branchStagePrefix(chapterNum, pos, "承") +
		"张力逐章叠加。三个分支＝三种升级形态，分歧轴是升级来源：①赌注加大（利害抬升一档）②对手升级（敌人更强或更近一步）③关系生变（盟友动摇/阵营位移）。每支的 summary 按顺序写明：升级事件＋压力如何叠加＋真实代价；且不得与前文摘要已完成的剧情重复。"
}

// branchTaskZhuan 转（阶段内 7-9）：三分支=三种反转形态。
func branchTaskZhuan(chapterNum int) string {
	_, pos, _ := stagePhaseOf(chapterNum)
	return "【本章位置与分支任务：阶段之「转」】" + branchStagePrefix(chapterNum, pos, "转") +
		"临近收官，允许把局面推向极端。三个分支＝三种反转形态，分歧轴是反转来源：①揭底（身份/立场/真相翻转）②倒转（攻守易位、局势翻盘）③爆雷（隐藏代价在此刻兑现）。每支的 summary 按顺序写明：反转是什么＋它如何改写本阶段走向＋谁付出什么代价。"
}

// branchTaskStageEnd 合（阶段内 10）：三分支=三种兑现方式（摊牌与代价分歧轴）。
func branchTaskStageEnd(chapterNum int) string {
	start, end := stageBounds((chapterNum-1)/stageLength + 1)
	return fmt.Sprintf("【本章位置与分支任务：阶段收官（合）】这是第%d章——本阶段（第%d~%d章）的最后一章，读者等着看这一阶段的答案。三个分支＝三种兑现方式，分歧轴是摊牌形态与代价归属。每支的 summary 按顺序写明：①阶段核心问题的摊牌（谁与谁、赌什么、怎么收）②谁付出什么真实代价（禁止无代价的胜利）③收掉哪些线头、刻意留哪个线头给下一阶段④章末如何同时完成本阶段收束与下一阶段更大格局的钩子。", chapterNum, start, end)
}

// branchTaskMid 常规推进（位置未知的兜底）：三分支=事件层分歧（补不重复前文）。
func branchTaskMid() string {
	return "【本章位置与分支任务：常规推进】三个分支＝三种明显不同的下一章走向，分歧轴是冲突与代价：不同冲突类型、不同代价归属，且不与「前文摘要」已完成的剧情重复。每支以具体场景开头（谁·动作·地点），写清触发事件→冲突→代价。"
}
