package app

// wenxue_subagent.go — 图文会话子代理的插话绑定（2026-11-26 文学同步批：办公
// 06b2b3fd GaeaSubagentSteer 的 Wenxue* 同构面，供右栏子会话窗与子代理会话 tab
// 的输入框使用）。
//
// 此前文学前端的 SubagentSteer 映射到办公绑定：在跑腿 SteerSubagent 按 ref
// 全局寻址、引擎无关，可用；**终态腿读的是办公 runner（ga.followUp）**，图文
// 子代理的追问会被办公控制器接走——事件落办公通道、图文界面看不到。本文件把
// 终态腿锚到图文引擎自己的 runner（wx.followUp）。
//
// 独立 Stop 不设 Wenxue* 换锚（审查批6 留档的决定不变）：中断面按 ref 全局
// 寻址（agent 包 subRuns 同进程双引擎共用），无会话路径锚，文学前端直用
// GaeaSubagentStop。

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	agent "github.com/gaea/gaea/internal/gaea/agent"
)

// WenxueSubagentSteer 子代理插话/追问双语义（与办公 GaeaSubagentSteer 逐段同构）：
//   - 在跑（running）：SteerSubagent 直穿插话——不打断工具执行，下一回合生效
//     （与主对话 GaeaSteer / DAG 节点改向直穿同机制，ref 全局寻址）；
//   - 排队中（queued）：诚实拒绝——排队期插话无落点（起跑后可插话）；
//   - 终态（completed/failed/aborted）：走图文引擎的追问管道（wx.followUp）。
//
// 与 WenxueSubagentFollowUp 的差异：**无主回合运行中闸**——插话目标是子代理，
// 与主回合并行是嵌套委派的常态（容量由子代理槽闸管）。终态腿的并发单飞/
// 受理清错/后台 goroutine 防线与 FollowUp 逐行同款（followUpClaims 共享：ref
// 全局唯一）。
func (a *App) WenxueSubagentSteer(sessionPath, ref, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("插话内容不能为空")
	}
	if !agent.ValidRunRef(ref) {
		return "", fmt.Errorf("无效的子代理引用: %s", ref)
	}

	// ① 在跑直穿：SteerSubagent 只对在跑 ref 成功（排队/终态/未知如实报错
	// ——绝不静默转续跑，DAG 改向同纪律）。
	if err := agent.SteerSubagent(ref, prompt); err == nil {
		return "steered", nil
	}

	// ② 状态分诊（只读）：queued 拒；终态落追问管道；meta 不可达=未知 ref
	// （ephemeral/他会话）诚实报错。
	if dir := sessionDirForPath(sessionPath); dir != "" {
		store := agent.NewSubagentStore(filepath.Join(dir, "subagents"))
		store.SetSessionPathFn(func() string { return sessionPath })
		status, perr := store.PeekStatus(ref)
		if perr != nil {
			return "", fmt.Errorf("子代理 %s 不在运行，且本会话无其记录（未知或已回收）", ref)
		}
		switch status {
		case agent.SubagentQueued:
			return "", fmt.Errorf("该子代理排队中，起跑后再插话；或等其完成后追问")
		case agent.SubagentRunning:
			// Peek 与直穿之间的竞态窗（恰好起跑）：再试一次直穿。
			if err := agent.SteerSubagent(ref, prompt); err == nil {
				return "steered", nil
			}
		default:
			// 终态：落追问管道（下方）。
		}
	} else {
		return "", fmt.Errorf("子代理 %s 不在运行，且会话路径不可达（无法续跑）", ref)
	}

	// ③ 终态腿：图文引擎的追问管道（WenxueSubagentFollowUp 同款防线；
	// runner/运行态读 wx，各持独立锁，不得与 wx.mu 交叉持有）。
	wx.followUpMu.Lock()
	runner := wx.followUp
	wx.followUpMu.Unlock()
	if runner == nil {
		return "", fmt.Errorf("子代理追问未接线（图文引擎尚未构建完成）")
	}
	if _, loaded := followUpClaims.LoadOrStore(ref, true); loaded {
		return "", fmt.Errorf("该子代理已有追问正在运行")
	}
	// 受理即同步清上一次追问失败摘要（FollowUp 同款：返回前清盘，前端首轮
	// 轮询不会把旧失败误记到这一次）。
	if dir := sessionDirForPath(sessionPath); dir != "" {
		store := agent.NewSubagentStore(filepath.Join(dir, "subagents"))
		store.SetSessionPathFn(func() string { return sessionPath })
		_ = store.RecordFollowUpError(ref, "")
	}
	go func() {
		defer func() {
			followUpClaims.Delete(ref)
			if r := recover(); r != nil {
				a.gaeaBackgroundPanicNotice("子代理插话续跑", r, map[string]interface{}{"ref": ref})
			}
		}()
		ctx := agent.WithSpace(context.Background(), gaeaSessionSpace())
		if err := runner(ctx, ref, prompt); err != nil {
			slog.Warn("子代理插话续跑失败", "ref", ref, "error", err)
		}
	}()
	return "follow-up started", nil
}
