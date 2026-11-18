package app

// GaeaSubagentSteer — 子代理插话/追问双语义绑定（2026-11-18，dsh 完整智能体
// 形态蒸馏 S4.4「Queue vs Steer」+ S6 右栏聊天页的写通道）：
//   - 在跑（running）：SteerSubagent 直穿插话——不打断工具执行，下一回合
//     生效（与主对话 GaeaSteer / DAG 节点改向直穿同机制）；
//   - 排队中（queued）：诚实拒绝——排队期插话无落点（起跑后可插话）；
//   - 终态（completed/failed/aborted）：走既有追问管道（RunFollowUp，Side
//     Chat 语义不变）。
//
// 与 GaeaSubagentFollowUp 的差异：**无主回合运行中闸**——插话目标是子代理，
// 与主回合并行是嵌套委派的常态（容量由子代理槽闸管）。终态腿的并发单飞/
// 受理清错/后台 goroutine 防线与 FollowUp 逐行同款（followUpClaims 共享）。

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
)

func (a *App) GaeaSubagentSteer(sessionPath, ref, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("插话内容不能为空")
	}
	if !gaeaAgent.ValidRunRef(ref) {
		return "", fmt.Errorf("无效的子代理引用: %s", ref)
	}

	// ① 在跑直穿：SteerSubagent 只对在跑 ref 成功（排队/终态/未知如实报错
	// ——绝不静默转续跑，DAG 改向同纪律）。
	if err := gaeaAgent.SteerSubagent(ref, prompt); err == nil {
		return "steered", nil
	}

	// ② 状态分诊（只读）：queued 拒；终态落追问管道；meta 不可达=未知 ref
	//（ephemeral/他会话）诚实报错。
	if dir := sessionDirForPath(sessionPath); dir != "" {
		store := gaeaAgent.NewSubagentStore(filepath.Join(dir, "subagents"))
		store.SetSessionPathFn(func() string { return sessionPath })
		status, perr := store.PeekStatus(ref)
		if perr != nil {
			return "", fmt.Errorf("子代理 %s 不在运行，且本会话无其记录（未知或已回收）", ref)
		}
		switch status {
		case gaeaAgent.SubagentQueued:
			return "", fmt.Errorf("该子代理排队中，起跑后再插话；或等其完成后追问")
		case gaeaAgent.SubagentRunning:
			// Peek 与直穿之间的竞态窗（恰好起跑）：再试一次直穿。
			if err := gaeaAgent.SteerSubagent(ref, prompt); err == nil {
				return "steered", nil
			}
		default:
			// 终态：落追问管道（下方）。
		}
	} else {
		return "", fmt.Errorf("子代理 %s 不在运行，且会话路径不可达（无法续跑）", ref)
	}

	// ③ 终态腿：追问管道（GaeaSubagentFollowUp 同款防线）。
	ga.mu.Lock()
	runner := ga.followUp
	ga.mu.Unlock()
	if runner == nil {
		return "", fmt.Errorf("子代理追问未接线（引擎尚未构建完成）")
	}
	if _, loaded := followUpClaims.LoadOrStore(ref, true); loaded {
		return "", fmt.Errorf("该子代理已有追问正在运行")
	}
	// 受理即同步清上一次追问失败摘要（FollowUp 同款：返回前清盘，前端首轮
	// 轮询不会把旧失败误记到这一次）。
	if dir := sessionDirForPath(sessionPath); dir != "" {
		store := gaeaAgent.NewSubagentStore(filepath.Join(dir, "subagents"))
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
		ctx := gaeaAgent.WithSpace(context.Background(), gaeaSessionSpace())
		if err := runner(ctx, ref, prompt); err != nil {
			slog.Warn("子代理插话续跑失败", "ref", ref, "error", err)
		}
	}()
	return "follow-up started", nil
}
