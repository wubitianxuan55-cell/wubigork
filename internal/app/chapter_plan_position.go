package app

// ── 章节计划的位置感知：每 10 章一阶段，阶段内按起承转合切换计划任务 ──
//
// v4.452.0 作者口径：「章节计划的提示词也要像剧情分支一样，不同的时间段用
// 不同的提示词；一个阶段的起承转合要做好」。本文件按 stagePhaseOf（分支同款
// 阶段模型，唯一真相）为章节计划生成【本章阶段位置与计划任务】段，经
// chapter-plan.json 的 story_position 槽位注入。
//
// 各段都带「不透支后段」纪律：起段不摊牌、承段不收束、转段不终极摊牌（留给
// 合）、合段不开新局（留给下一阶段开篇）——阶段节奏由 10 章一阶段保证。

import "fmt"

// planPositionContext 返回注入章节计划请求的【本章阶段位置与计划任务】段。
// 章号未知（<=0）返回空串（模板空槽位不渲染，零注入）。
func planPositionContext(chapterNum int) string {
	stage, pos, ph := stagePhaseOf(chapterNum)
	switch ph {
	case phaseOpening:
		return planTaskOpening()
	case phaseStageStart:
		return planTaskStageStart(chapterNum, stage)
	case phaseQi:
		return planTaskQi(chapterNum, stage, pos)
	case phaseCheng:
		return planTaskCheng(chapterNum, stage, pos)
	case phaseZhuan:
		return planTaskZhuan(chapterNum, stage, pos)
	case phaseHe:
		return planTaskHe(chapterNum, stage)
	default:
		return ""
	}
}

// planTaskOpening 全书开篇：立全书——引爆点前置、基调承诺、最小完备入场。
func planTaskOpening() string {
	return "【本章阶段位置与计划任务：全书开篇（第1章·第1阶段之起）】计划的头等任务是立住全书：①引爆点必须安排在本章——key_events 第一条就是触发引爆的事件；②以第一画面兑现基调承诺（让读者知道这本书是哪种爽）；③主要人物入场最小完备（只带本章必需的）；④章末大钩。不铺日常、不解释世界观——设定随事件带出。"
}

// planTaskStageStart 新阶段开篇：承接余波 + 立新驱动 + 钩子。
func planTaskStageStart(chapterNum, stage int) string {
	prevStart, prevEnd := stageBounds(stage - 1)
	return fmt.Sprintf("【本章阶段位置与计划任务：新阶段开篇（第%d章·第%d阶段第1章，「起」之首）】上一阶段（第%d~%d章）已收官。计划任务：①承接上一阶段的余波与代价——key_events 含消化余波的一笔，不假装无事发生；②章内把新阶段的核心驱动（新目标/新矛盾/新悬念）摆上台面；③遗留线头自然延续，不当不存在；④章末钩子指向新阶段第一波冲突。", chapterNum, stage, prevStart, prevEnd)
}

// planTaskQi 起（阶段内 2-3）：阶段目标落地 + 埋线。
func planTaskQi(chapterNum, stage, pos int) string {
	start, end := stageBounds(stage)
	return fmt.Sprintf("【本章阶段位置与计划任务：阶段之「起」（第%d章·第%d阶段第%d章，本阶段=第%d~%d章）】计划任务：把阶段核心目标具体化——①本章在阶段目标上落下第一块实土（推进了什么、拿到/失去什么）②埋入本阶段要用的线（人物/物件/信息）③登场配置最小完备。不透支后段：「起」段不摊牌、不兑现阶段悬念。", chapterNum, stage, pos, start, end)
}

// planTaskCheng 承（阶段内 4-6）：递进升级、支线交错。
func planTaskCheng(chapterNum, stage, pos int) string {
	start, end := stageBounds(stage)
	return fmt.Sprintf("【本章阶段位置与计划任务：阶段之「承」（第%d章·第%d阶段第%d章，本阶段=第%d~%d章）】计划任务：推进与升级——①与「前文摘要」形成可辨的递进（不得重复已完成剧情）②压力/赌注叠加一档③支线与主线交错推进，情绪与冲突逐章抬升。不透支后段：「承」段不提前收束阶段冲突。", chapterNum, stage, pos, start, end)
}

// planTaskZhuan 转（阶段内 7-9）：反转高压、代价兑付。
func planTaskZhuan(chapterNum, stage, pos int) string {
	start, end := stageBounds(stage)
	return fmt.Sprintf("【本章阶段位置与计划任务：阶段之「转」（第%d章·第%d阶段第%d章，本阶段=第%d~%d章，第%d章即将收官）】计划任务：反转与高压——①安排改写阶段走向的转折（揭底/倒转/爆雷择一）②转折必须付出真实代价③冲突推向本阶段峰值，情绪基调与「转」相称（紧张/压抑/震荡）。不透支后段：终极摊牌留给收官章。", chapterNum, stage, pos, start, end, end)
}

// planTaskHe 合（阶段内 10）：阶段收束 + 留线头 + 大钩。
func planTaskHe(chapterNum, stage int) string {
	start, end := stageBounds(stage)
	return fmt.Sprintf("【本章阶段位置与计划任务：阶段收官「合」（第%d章·第%d阶段末章，本阶段=第%d~%d章）】计划任务：阶段收束——①阶段核心问题的本章兑现（谁与谁、赌什么、怎么收）②谁付出什么真实代价（禁止无代价的胜利）③收掉哪些线头、刻意留哪个线头给下一阶段④章末钩子指向下一阶段更大格局。key_events 必须同时覆盖「兑现」与「代价」。", chapterNum, stage, start, end)
}
