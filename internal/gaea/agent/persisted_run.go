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
// 不会以技能正文开头）；存 meta.Title。与 TaskTool 的差异：结果不带
// "Subagent reference" 尾巴——技能子代理没有 continue_from 入口，噪音无益。
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
		return RunSubAgent(ctx, prov, reg, sysPrompt, prompt, opts, sink, subUsage)
	}
	run, err := store.PrepareFreshWithTitle(sysPrompt, SpaceFromContext(ctx), title)
	if err != nil {
		return "", fmt.Errorf("prepare subagent transcript: %w", err)
	}
	defer run.Release()
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
		_ = store.SaveFailed(run)
		return result, err
	}
	if err := store.SaveCompleted(run); err != nil {
		return "", fmt.Errorf("save subagent transcript: %w", err)
	}
	return result, nil
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
