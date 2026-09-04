package app

// GaeaSubagentFollowUp — Side Chat 式追问（v4.64）：用户在子代理会话 tab 里
// 对一个已完结的 sa_ 运行追加提问。后端复用 task 工具的 continue_from 管道
// （PrepareContinue 拒绝 running/mt_/跨空间），文本增量走 gaea-subagent-text
// 专用通道，权威内容落子代理自身 transcript（~1s 快照）；完成态由 meta 侧车
// 承载，任务树/tab 状态点经轮询自校正。调用即刻返回「已受理」，运行在后台
// goroutine——前端凭追问本地态 + 轮询驱动界面，长跑不占用绑定调用。

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
)

// followUpClaims 同 ref 并发去重：双击/重复发送时第二个请求直接被拒。
var followUpClaims sync.Map

// GaeaSubagentFollowUp 对 ref 指定的子代理会话发起一次追问，立即返回。
// 主回合运行中拒绝（避免与主代理并发争用引擎与事件流语义）。
func (a *App) GaeaSubagentFollowUp(sessionPath, ref, prompt string) (string, error) {
	_ = sessionPath // transcript 目录随 controller 解析；ref 全局唯一无需路由
	ga.mu.Lock()
	runner := ga.followUp
	running := ga.ctrl != nil && ga.ctrl.Running()
	ga.mu.Unlock()
	if runner == nil {
		return "", fmt.Errorf("子代理追问未接线（引擎尚未构建完成）")
	}
	if running {
		return "", fmt.Errorf("主对话回合正在运行，请等回合结束再追问")
	}
	claimKey := ref
	if _, loaded := followUpClaims.LoadOrStore(claimKey, true); loaded {
		return "", fmt.Errorf("该子代理已有追问正在运行")
	}
	defer followUpClaims.Delete(claimKey)

	// 受理即同步清掉上一次追问的失败摘要（v4.66）：清盘发生在返回「已受理」
	// 之前——前端派发后的首轮轮询不会把上一枪的 followUpError 误记到这一次
	// 头上（若留到后台 goroutine 里清，存在先返回后清的竞态窗）。临时 store
	// 实例与 boot 的惰性 store 解析同一目录（<会话目录>/subagents）；此刻同
	// ref 无并发 meta 写（claims 单飞 + PrepareContinue 拒 running），读改写
	// 安全。best-effort：无会话目录/meta 时跳过，RunFollowUp 开跑清兜底。
	if dir := sessionDirForPath(sessionPath); dir != "" && gaeaAgent.ValidRunRef(ref) {
		store := gaeaAgent.NewSubagentStore(filepath.Join(dir, "subagents"))
		_ = store.RecordFollowUpError(ref, "")
	}

	ctx := gaeaAgent.WithSpace(context.Background(), gaeaSessionSpace())
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("子代理追问 panic", "ref", ref, "panic", r)
			}
		}()
		if err := runner(ctx, ref, prompt); err != nil {
			slog.Warn("子代理追问失败", "ref", ref, "error", err)
		}
	}()
	return "follow-up started", nil
}
