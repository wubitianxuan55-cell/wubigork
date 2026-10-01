package app

// ── 剧情分支的位置感知：开篇 / 阶段收官 / 新阶段开篇 / 常规推进 ──
//
// 分支浏览器此前对全位置共用一个问法（"三个明显不同的下一章走向"），分歧轴
// 恒为事件层。作者口径：开篇分支该分「从哪扇门进故事」、阶段收官该分「怎么
// 兑现」、新阶段开篇该分「新驱动的性质」。本文件按下一章的位置生成【本章位置
// 与分支任务】语境段注入请求——分歧轴与必带项随位置切换；分支 JSON 形状不变
// （解析/前端契约零改动），位置纪律落进各支的 summary 文本。
//
// 与开篇/长跑教义的一致性：开篇三分支的「事发之前」门被要求第一章内抵达引爆
// 点（对齐 create-chapter-first 的钩子 mandate 与「迟进早出」）；阶段收官/开篇
// 的余波纪律与 stage-recap 的合账口径同源。

import (
	"fmt"
	"strings"
)

// branchPositionContext 返回注入分支请求的【本章位置与分支任务】段。
// chapterNum>0 按章号定位置；chapterNum<=0（无节点轻量路径）按是否有前文
// 二分：无前文=开篇，有前文=常规（阶段边界章号未知，不作阶段特判）。
func branchPositionContext(chapterNum int, prevSummary string) string {
	switch {
	case chapterNum == 1:
		return branchTaskOpening()
	case chapterNum > 1 && chapterNum%stageLength == 0:
		return branchTaskStageEnd(chapterNum)
	case isStageOpener(chapterNum):
		return branchTaskStageStart(chapterNum)
	case chapterNum > 1:
		return branchTaskMid()
	default:
		if strings.TrimSpace(prevSummary) == "" {
			return branchTaskOpening()
		}
		return branchTaskMid()
	}
}

// branchTaskOpening 开篇：三分支=三扇门（切入点分歧轴）。
func branchTaskOpening() string {
	return "【本章位置与分支任务：开篇】这是全书的第一章。开篇分支的分歧轴不是「发生什么事」，而是「从哪扇门进同一个故事」——三个分支必须是三种不同的切入点：①事发瞬间硬切（直接落进引爆事件）②事发之后倒推（从后果与谜团开场往回揭）③事发之前建立（先立人物与日常再引爆，第一章内必须抵达引爆点）。每支的 summary 按顺序写明：切入点＋第一画面（谁·在哪·正做什么）＋基调承诺（这本书是哪种爽）＋预支的代价；三支的基调承诺必须可辨地不同。"
}

// branchTaskStageEnd 阶段收官：三分支=三种兑现方式（摊牌与代价分歧轴）。
func branchTaskStageEnd(chapterNum int) string {
	start := chapterNum - stageLength + 1
	if start < 1 {
		start = 1
	}
	return fmt.Sprintf("【本章位置与分支任务：阶段收官】这是第%d章——本阶段（第%d~%d章）的最后一章，读者等着看这一阶段的答案。三个分支＝三种兑现方式，分歧轴是摊牌形态与代价归属。每支的 summary 按顺序写明：①阶段核心问题的摊牌（谁与谁、赌什么、怎么收）②谁付出什么真实代价（禁止无代价的胜利）③收掉哪些线头、刻意留哪个线头给下一阶段④章末如何同时完成本阶段收束与下一阶段更大格局的钩子。", chapterNum, start, chapterNum)
}

// branchTaskStageStart 新阶段开篇：三分支=新阶段的三种可能（新驱动性质分歧轴）。
func branchTaskStageStart(chapterNum int) string {
	return fmt.Sprintf("【本章位置与分支任务：新阶段开篇】这是第%d章——新阶段的第一章（上一阶段已收官）。三个分支＝新阶段的三种可能，分歧轴是新驱动的性质，三支各走一种：换目标（旧目标了结后要什么）/换对手（矛盾换了主体）/换问题类型（外部冲突转内部或反向）。每支的 summary 按顺序写明：①如何承接上一阶段的余波与代价（不假装无事发生）②新阶段的核心驱动③旧线头如何自然延续④本章末的新阶段钩子。", chapterNum)
}

// branchTaskMid 常规推进：三分支=事件层分歧（既有口径，补不重复前文）。
func branchTaskMid() string {
	return "【本章位置与分支任务：常规推进】三个分支＝三种明显不同的下一章走向，分歧轴是冲突与代价：不同冲突类型、不同代价归属，且不与「前文摘要」已完成的剧情重复。每支以具体场景开头（谁·动作·地点），写清触发事件→冲突→代价。"
}
