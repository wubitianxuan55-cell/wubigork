package app

import (
	"time"

	"github.com/gaea/gaea/internal/gaea/memory"
)

// ── 记忆注入质量评测（市场调研候选2，docs/gaea-memory-injection-eval-design-2026-09.md）──
//
// 与检索评测（GaeaRetrievalEvalRun，Recall@10≥0.8 统计门槛）分立：本绑定测
// **注入面**——对真实库的 work 视图跑与 boot/sysprompt.go 装配点完全相同的
// 两个真实构建器（晨报预载 + 项目本体），对产物断言五条结构不变量
// （预算合规×2/归档零泄漏/跨空间零泄漏/引用零悬空）。纯读路径零 LLM 零
// embedding，确定性可重复。门控三开关随报告透明下发（关态下结构体检照跑，
// Note 注明注入未发生）。

// MemoryEvalGates 注入门控三开关（与 gaea_handler.go 喂给 boot opts 的同源读取）。
type MemoryEvalGates struct {
	MemoryEnabled  bool `json:"memoryEnabled"`
	MorningPreload bool `json:"morningPreload"`
	ProjectBrief   bool `json:"projectBrief"`
	SpaceModeOn    bool `json:"spaceModeOn"`
}

// MemoryEvalReport 注入体检总报告（门控 + 注入面五不变量 + 固化覆盖指标）。
type MemoryEvalReport struct {
	MemoryEvalGates
	memory.InjectionEvalReport
	Note string `json:"note,omitempty"`
}

// GaeaMemoryEvalRun 运行记忆注入体检。空间视图与装配点同口径：
// SpaceModeIsOn → ListInSpace("work")（双空间读谓词）；mode=off → List() 全量
// （无跨空间概念，泄漏检查自动跳过）。门控读取与装配点同源：
// morningPreloadEnabled/projectBriefEnabled（gaea_handler.go 喂 boot opts 的
// 同两函数）+ gaeaConfig Memory.Enabled（gaeaLoadConfig 兜底）。
func (a *App) GaeaMemoryEvalRun() (MemoryEvalReport, error) {
	rep := MemoryEvalReport{}
	store := a.hubOfficeStore()

	ga.mu.Lock()
	engineCfg := ga.cfg
	ga.mu.Unlock()
	if engineCfg == nil {
		if c, err := gaeaLoadConfig(); err == nil {
			engineCfg = c
		}
	}
	spaceModeOn := engineCfg != nil && engineCfg.SpaceModeIsOn()
	memoryEnabled := engineCfg != nil && engineCfg.Memory.Enabled
	gates := MemoryEvalGates{
		MemoryEnabled:  memoryEnabled,
		MorningPreload: morningPreloadEnabled(a),
		ProjectBrief:   projectBriefEnabled(a),
		SpaceModeOn:    spaceModeOn,
	}
	rep.MemoryEvalGates = gates

	// work 视图（装配点同口径）与他空间/归档名集。
	var view []memory.Memory
	other := map[string]bool{}
	if gates.SpaceModeOn {
		view = store.ListInSpace("work")
		for _, m := range store.ListInSpace("play") {
			other[m.Name] = true
		}
	} else {
		view = store.List()
	}
	var archived []string
	for _, am := range store.ListArchived() {
		archived = append(archived, am.Name)
	}

	// 真实构建器跑真实库（与装配点同函数同预算；门控关闭=装配不发生=空块）。
	now := time.Now()
	preload, brief := "", ""
	if gates.MemoryEnabled && gates.MorningPreload {
		preload = memory.BuildMorningPreloadBlock(view, now, 0)
	}
	if gates.MemoryEnabled && gates.ProjectBrief {
		brief = memory.BuildProjectBrief(view, now, 0)
	}

	rep.InjectionEvalReport = memory.EvalInjectionBlocks(memory.InjectionEvalInput{
		Work:          view,
		OtherNames:    other,
		ArchivedNames: archived,
		PreloadBlock:  preload,
		BriefBlock:    brief,
	})

	switch {
	case !gates.MemoryEnabled:
		rep.Note = "记忆总开关关闭，注入未发生；本报告仅覆盖结构不变量（空块=合规）"
	case !gates.MorningPreload && !gates.ProjectBrief:
		rep.Note = "晨报预载与项目本体两门控均关闭，注入未发生；本报告仅覆盖结构不变量"
	case !gates.MorningPreload:
		rep.Note = "晨报预载门控关闭，仅项目本体参与注入"
	case !gates.ProjectBrief:
		rep.Note = "项目本体门控关闭，仅晨报预载参与注入"
	case rep.EntryCount == 0 && rep.RefCount == 0:
		rep.Note = "work 视图暂无记忆（或引擎未初始化为空库）：体检通过但没有注入内容可核"
	}
	return rep, nil
}
