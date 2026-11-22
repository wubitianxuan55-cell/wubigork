package agent

// wait_subagents.go — wait 收 sa_/mt_ 引用（2026-11-21 反馈 P1-7：wait 传 sa_
// 返回 "No background jobs to wait for"，无法阻塞等待，主控只能 sleep 260~290
// 轮询，一次 sleep 420 撞上 bash 5 分钟硬超时被杀）。
//
// 跨包缝：wait 是 builtin（tool/builtin），子代理状态在 agent 包——builtin 不
// 能反向 import agent（环）。经 builtin.SetSubagentWaiter 注入（boot 装配点接
// 线，与 SetSubagentRouteFactory 同款注入方向）；未接线（单测/无 transcripts
// store）时 wait 保持旧行为（只认 bash job ids）。

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// SubagentWaitResult 一段子代理等待的结论（bgjobs wait 输出行消费）。
type SubagentWaitResult struct {
	Ref    string
	Status string // completed/failed/aborted/running（超时仍在跑）/unknown
	Output string // 模型面短句
}

// isAnyRunRef 报告 ref 是否子代理/模型工具运行引用（sa_/mt_ 前缀；包内已有
// 细分的 IsSubagentRef/IsModelToolRef，此处是 wait 分流用的并集判定）。
func isAnyRunRef(ref string) bool {
	return agentHasPrefix(ref, "sa_") || agentHasPrefix(ref, "mt_")
}

func agentHasPrefix(s, p string) bool { return strings.HasPrefix(s, p) }

// WaitSubagents 阻塞等一批子代理运行到终态（或超时）。store 缺席/引用未知→
// 立即回 unknown（如实，不冒充等过）；终态=completed/failed/aborted；超时仍在
// 跑→running 如实报。轮询 2s（meta 侧车落盘节奏匹配，TouchAction 每次工具
// 调用都在刷）。
func WaitSubagents(ctx context.Context, refs []string, timeout time.Duration) []SubagentWaitResult {
	out := make([]SubagentWaitResult, 0, len(refs))
	if subWaitStore == nil || len(refs) == 0 {
		for _, r := range refs {
			out = append(out, SubagentWaitResult{Ref: r, Status: "unknown",
				Output: "子代理等待面未接线（无 transcripts store）——用 subagent_list 查状态"})
		}
		return out
	}
	if timeout <= 0 {
		timeout = 300 // 与 wait 工具默认 5 分钟对齐
	}
	deadline := time.Now().Add(timeout)
	pending := map[string]bool{}
	for _, r := range refs {
		pending[r] = true
	}
	terminal := func(ref string) (SubagentWaitResult, bool) {
		st, err := subWaitStore.PeekStatus(ref)
		if err != nil {
			return SubagentWaitResult{Ref: ref, Status: "unknown",
				Output: fmt.Sprintf("引用未找到（%v）——用 subagent_list 核对 ref", err)}, true
		}
		switch st {
		case SubagentCompleted, SubagentFailed, SubagentAborted:
			return SubagentWaitResult{Ref: ref, Status: string(st),
				Output: subagentWaitLine(ref, string(st))}, true
		}
		return SubagentWaitResult{}, false
	}
	// 首轮先查一遍（多数已终态的立即返回）。
	for ref := range pending {
		if res, done := terminal(ref); done {
			out = append(out, res)
			delete(pending, ref)
		}
	}
	for len(pending) > 0 {
		if time.Now().After(deadline) {
			refs := make([]string, 0, len(pending))
			for ref := range pending {
				refs = append(refs, ref)
			}
			sort.Strings(refs)
			for _, ref := range refs {
				out = append(out, SubagentWaitResult{Ref: ref, Status: "running",
					Output: "等待超时仍在运行——用 subagent_list 看进度（last: 动作行）后再 wait 或直接收结果"})
			}
			return out
		}
		select {
		case <-ctx.Done():
			for ref := range pending {
				out = append(out, SubagentWaitResult{Ref: ref, Status: "running",
					Output: "等待被取消，仍在运行"})
			}
			return out
		case <-time.After(2 * time.Second):
		}
		for ref := range pending {
			if res, done := terminal(ref); done {
				out = append(out, res)
				delete(pending, ref)
			}
		}
	}
	return out
}

// subagentWaitLine 终态行（DSH wait 输出风格：结论+下一步一句）。
func subagentWaitLine(ref, status string) string {
	switch status {
	case string(SubagentCompleted):
		return "完成——结果已自动回投（前台 task 回执内；后台在完成事件/续聊可见）"
	case string(SubagentAborted):
		return "被中止（interrupt/回合停止）——需要结果可 send_message 续跑或重派"
	default:
		return "失败——重派前先 send_message " + ref + " 追问诊断或查其会话记录"
	}
}
