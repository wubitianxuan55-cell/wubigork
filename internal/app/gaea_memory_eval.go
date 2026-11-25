package app

import (
	"github.com/gaea/gaea/internal/gaea/memory"
)

// ── 记忆注入质量评测（市场调研候选2，docs/gaea-memory-injection-eval-design-2026-09.md）──
//
// 与检索评测（GaeaRetrievalEvalRun，Recall@10≥0.8 统计门槛）分立：本绑定测
// **注入面**——对真实库的 work 视图跑与 boot/sysprompt.go 装配点同口径的记忆
// 索引（IndexForInjection），对产物断言四条结构不变量（预算合规/归档零泄漏/
// 跨空间零泄漏/条目零悬空）。纯读路径零 LLM 零 embedding，确定性可重复。
// 门控随报告透明下发（关态下结构体检照跑，Note 注明注入未发生）。
//
// 2026-11-25 办公工具梳理批：晨报预载与项目本体两块已停注（与索引逐条重复），
// 对应的两个门控开关（morning_preload / project_brief）随之撤下。

// MemoryEvalGates 注入门控。
type MemoryEvalGates struct {
	MemoryEnabled bool `json:"memoryEnabled"`
	SpaceModeOn   bool `json:"spaceModeOn"`
}

// MemoryEvalReport 注入体检总报告（门控 + 注入面四不变量 + 固化覆盖指标）。
type MemoryEvalReport struct {
	MemoryEvalGates
	memory.InjectionEvalReport
	Note string `json:"note,omitempty"`
}

// GaeaMemoryEvalRun 运行记忆注入体检。空间视图与装配点同口径：
// SpaceModeIsOn → ListInSpace("work")（双空间读谓词）；mode=off → List() 全量
// （无跨空间概念，泄漏检查自动跳过）。门控读取与装配点同源：gaeaConfig
// Memory.Enabled（gaeaCfgSnapshotOrLoad 兜底）。按运行模式的收窄不在此重现——
// 面板体检看全量，模式隔离由 boot 包装配测试钉住。
func (a *App) GaeaMemoryEvalRun() (MemoryEvalReport, error) {
	rep := MemoryEvalReport{}
	store := a.hubOfficeStore()

	engineCfg := gaeaCfgSnapshotOrLoad() // AP5-08：快照 nil 时读盘兜底（原内联同形）
	spaceModeOn := engineCfg != nil && engineCfg.SpaceModeIsOn()
	memoryEnabled := engineCfg != nil && engineCfg.Memory.Enabled
	gates := MemoryEvalGates{
		MemoryEnabled: memoryEnabled,
		SpaceModeOn:   spaceModeOn,
	}
	rep.MemoryEvalGates = gates

	// work 视图（装配点同口径）与他空间/归档名集。
	var view []memory.Memory
	var work memory.Store
	other := map[string]bool{}
	if gates.SpaceModeOn {
		view = store.ListInSpace("work")
		work = store.InSpace("work")
		for _, m := range store.ListInSpace("play") {
			other[m.Name] = true
		}
	} else {
		view = store.List()
		work = store
	}
	var archived []string
	for _, am := range store.ListArchived() {
		archived = append(archived, am.Name)
	}

	// 真实索引（与装配点同函数同预算；记忆关闭=装配不发生=空块）。
	indexBlock := ""
	if gates.MemoryEnabled {
		indexBlock = memory.IndexForInjection(work)
	}
	rep.InjectionEvalReport = memory.EvalIndexInjection(memory.InjectionEvalInput{
		Work:          view,
		OtherNames:    other,
		ArchivedNames: archived,
		IndexBlock:    indexBlock,
	})

	switch {
	case !gates.MemoryEnabled:
		rep.Note = "记忆总开关关闭，注入未发生；本报告仅覆盖结构不变量（空块=合规）"
	case rep.EntryCount == 0:
		rep.Note = "work 视图暂无记忆（或引擎未初始化为空库）：体检通过但没有注入内容可核"
	case rep.Truncated:
		rep.Note = "记忆索引超出注入预算已截断：未进索引的条目仍可经 memory_search 检索到"
	}
	return rep, nil
}
