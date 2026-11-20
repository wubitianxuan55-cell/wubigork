package agent

import (
	"context"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// RunPersistedSubAgent runs a sub-agent (run_skill / 专用子代理技能派生路径)
// and persists its transcript live + terminally into the session store. It is
// the TaskTool-equivalent entry point for skillRunner: opening a fresh sa_
// run, writing the running sidecar + periodic snapshots while the sub-agent
// executes, and closing with SaveCompleted/SaveFailed.
//
// title 是任务摘要（技能调用方传 task 本体而非 sk.Body+task 拼合串，UI 标题
// 不会以技能正文开头）；存 meta.Title。成功结果附 "Subagent reference" 尾巴
//（与 TaskTool 回执同款）：它不只是 continue_from 的引用——对话流前端靠这行
// 把 run_skill 调用升级成委派卡（inline 技能无此行，按普通工具渲染，
// 2026-11-20 委派家族收窄），缺行会让技能子代理卡既无徽标也不可点。
func RunPersistedSubAgent(
	ctx context.Context,
	prov provider.LLMProvider,
	reg *tool.Registry,
	sysPrompt, prompt, title string,
	opts Options,
	sink event.Sink,
	store *SubagentStore,
	subUsage *provider.Usage,
) (string, error) {
	if store == nil {
		// ephemeral 路径同样吃并发闸（skill 派生的 LLM 循环是同一类负载）；
		// 无 store 即无 queued 可见面，直接等槽。
		if err := acquireSubSlot(ctx); err != nil {
			return "", fmt.Errorf("sub-agent queued: %w", err)
		}
		defer releaseSubSlot()
		opts = bindHooksToSession(opts, newEphemeralSubagentID())
		ctx = withSubagentDepth(ctx, subagentDepthFromContext(ctx)+1)
		return RunSubAgent(ctx, prov, reg, sysPrompt, prompt, opts, sink, subUsage)
	}
	run, err := store.PrepareFreshWithTitle(sysPrompt, SpaceFromContext(ctx), title)
	if err != nil {
		return "", fmt.Errorf("prepare subagent transcript: %w", err)
	}
	defer run.Release()
	// 出发快照（B1）：技能派发的执行面=调用方传入的注册表全体（技能路径无
	// 父白名单过滤，注册表即边界）。
	run.ToolScope = sortedToolNames(reg.Names())
	// D1 编队拓扑会话化：派生血缘盖章（先读调用方 ctx 再盖本运行 ID——顺序
	// 反了会把孩子记成自己的父亲）。技能子代理此前不盖 subagentSessionID，
	// 嵌套在技能子代理里的 task 调用读不到父 ref，这里一并补齐。
	run.ParentRef = PersistedParentRef(subagentSessionIDFromContext(ctx))
	if id, _, _, _ := CallContext(ctx); id != "" {
		run.SpawnCallID = id
	}
	// 深度记账（2026-11-18 嵌套委派）：技能子代理级别=调用方级别+1；注册表
	// 由调用方整体传入（技能装配面），深度闸对该面不作用——但记账要真，
	// 编队树与后续续跑的级别都从这里读。
	run.Depth = subagentDepthFromContext(ctx) + 1
	ctx = withSubagentSessionID(ctx, run.Ref)
	ctx = withSubagentDepth(ctx, run.Depth)
	opts = bindHooksToSession(opts, run.Ref)
	// 有限并行：排队先落 queued 侧车（分工面板如实显示「排队中」），起跑时
	// MarkRunning 覆写；等槽期间 ctx 取消则补 failed 收口，不留永久排队态。
	_ = store.MarkQueued(run)
	if err := acquireSubSlot(ctx); err != nil {
		_ = store.SaveFailed(run)
		return "", fmt.Errorf("sub-agent queued: %w", err)
	}
	defer releaseSubSlot()
	if err := store.MarkRunning(run); err != nil {
		return "", fmt.Errorf("mark subagent running: %w", err)
	}
	stop := store.TrackProgress(run, 0)
	defer stop()

	// P1 逐 token 流式：run（含 ref）在本函数内创建，而 sink（NestedSink 的
	// subSinkFor 包裹）来自调用方、拿不到 ref——这里在 sink 外层把内层 Text
	// 增量转标 SubagentText（打 ref），subSinkFor 见该 Kind 只补父调用 ID 透传。
	// 与 task 路径（subSinkFor 的 refSrc 直接闭包 run）殊途同归。
	if run.Ref != "" {
		sink = refTextSink(run.Ref, sink)
	}

	result, err := RunSubAgentWithSession(ctx, prov, reg, run.Session, prompt, opts, sink, subUsage)
	if err != nil {
		// D4 结算词汇：用户中止（ctx 取消）落 aborted，不冒充失败。
		if abortedErr(err) {
			_ = store.SaveAborted(run)
		} else {
			_ = store.SaveFailed(run)
		}
		return result, err
	}
	if err := store.SaveCompleted(run); err != nil {
		return "", fmt.Errorf("save subagent transcript: %w", err)
	}
	// 实测反馈④：计量行先算后拼，口径与 task 工具 finalizeRun 单源。
	return result + FormatReceiptMetrics(result) + FormatSubagentReference(run), nil
}

func bindHooksToSession(opts Options, sessionID string) Options {
	if opts.HooksForSession != nil {
		opts.Hooks = opts.HooksForSession(sessionID)
	}
	return opts
}

// refTextSink 把子代理 LLM 文本增量转标 SubagentText（打 ref）后转发 inner——
// 技能子代理（run_skill 派生）路径的 ref 注入点。其余事件原样透传。
func refTextSink(ref string, inner event.Sink) event.Sink {
	return event.FuncSink(func(e event.Event) {
		if e.Kind == event.Text {
			e.Kind = event.SubagentText
			e.SubagentRef = ref
		}
		inner.Emit(e)
	})
}
