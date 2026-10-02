package app

// gaea_knowledge_migration.go — 旧知识库迁移失败的「可见化」接线（审计 GA3-13
// 的 app 侧欠账）。
//
// 批次十一已在 internal/gaea/knowledge 落地：迁移失败不再静默（MigrationState
// {Ran, Failed, Reason} 可查、不阻断 Store 可用性），但「失败可见」当时停在 API
// 层——app 启动路径与面板都没读它，用户侧表现为「库看起来是空的/条目少了」而
// 无从判断是迁移失败还是本来没条目。
//
// 本文件把结论经 **既有** gaea-event/notice 通道（与后台 panic 通知、
// reportCostReadError 同路，不新造通道）+ slog.Warn 上报。纪律：
//   - 不加新绑定方法（绑定面零变更）：notice 通道前端已消费，面板天然可见；
//   - 只在 Failed 时出声（成功/未跑完全静默），避免启动噪音；
//   - Reason 原样进文案（说人话、可定位），不吞错。

import (
	"fmt"
	"log/slog"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/knowledge"
)

// knowledgeMigrationStateFn 是知识库迁移状态读取缝（测试注入失败态/成功态；
// 生产恒读进程级 Service 的 MigrationState）。与 gaeaDreamRun/gaeaNoticeSink
// 同型：只做读取，不改 knowledge 包的语义。
var knowledgeMigrationStateFn = func() knowledge.MigrationState {
	return knowledge.Global().MigrationState()
}

// reportKnowledgeMigrationState 在启动路径读取旧知识库迁移状态：失败时经既有
// gaea-event notice + slog.Warn 上报（含 Reason）。迁移失败不阻断知识库可用性
// （部分迁移），所以这里是「可见」而不是「中止」。
func (a *App) reportKnowledgeMigrationState() {
	st := knowledgeMigrationStateFn()
	if !st.Ran || !st.Failed {
		return // 未跑（尚未打开库）/成功：不出声
	}
	slog.Warn("knowledge: 旧知识库迁移失败，库为部分迁移状态（知识库仍可用，条目可能不全）",
		"reason", st.Reason)
	payload := gaeaEventMap(event.Event{
		Kind:  event.Notice,
		Level: event.LevelWarn,
		Text: fmt.Sprintf("知识库旧数据迁移失败（%s）：库处于部分迁移状态，条目可能不全——"+
			"不影响继续使用，建议查看日志后重试迁移", st.Reason),
	})
	if gaeaNoticeSink != nil {
		gaeaNoticeSink("gaea-event", payload)
		return
	}
	a.emit("gaea-event", payload)
}
