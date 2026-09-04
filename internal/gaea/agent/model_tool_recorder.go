package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// modelToolRecorder 把「调用本地模型的工具」（ModelBacked 标记，如 vision /
// summarize_file）的一次主线程调用落成与子代理同构的 mt_ 运行记录：
//   - 工具调用参数流出（pre-exec 起点 / batch 最终 dispatch）→ NewModelToolRun
//     （meta running + transcript 首行 user 任务标签）；
//   - 最终 ToolResult → FinishModelTool（结果入 transcript，meta completed/failed）。
//
// 只有主执行器装配 recorder（boot 把注册表中 ModelBacked 工具名集合 + 惰性
// store 注入）；子代理的 AgentRunner 不装配——其内部工具活动已经属于该子代理
// 的 sa_ transcript，重复开 mt_ 只会制造噪音。记录失败（目录未解析/写盘错误）
// 静默忽略：这是「尽力而为的可见性」增强，绝不能阻断工具执行主链路。

// isModelBacked reports whether name is in the recorded set.
func (a *AgentRunner) isModelBacked(name string) bool {
	a.mtMu.Lock()
	defer a.mtMu.Unlock()
	return a.modelBacked[name]
}

// startModelToolRun opens an mt_ record for a tool call. Idempotent per tool
// call ID; a no-op when recording is disabled, the tool is not model-backed,
// or the store cannot write (best-effort).
func (a *AgentRunner) startModelToolRun(ctx context.Context, id, name, args string) {
	a.mtMu.Lock()
	defer a.mtMu.Unlock()
	if a.mtStore == nil || len(a.modelBacked) == 0 || id == "" || !a.modelBacked[name] {
		return
	}
	if _, exists := a.mtRuns[id]; exists {
		return // already open (pre-exec start followed by batch start)
	}
	run, err := a.mtStore.NewModelToolRun(
		modelToolCallLabel(name, args), name, SpaceFromContext(ctx),
	)
	if err != nil || run == nil || run.Ref == "" {
		return
	}
	if a.mtRuns == nil {
		a.mtRuns = make(map[string]string, 8)
	}
	a.mtRuns[id] = run.Ref
}

// updateModelToolLabel refreshes an open mt_ record's title once the full tool
// arguments are known (the record opens at pre-exec where only the name is
// guaranteed).
func (a *AgentRunner) updateModelToolLabel(id, name, args string) {
	a.mtMu.Lock()
	ref, ok := a.mtRuns[id]
	a.mtMu.Unlock()
	if !ok || !a.isModelBacked(name) || a.mtStore == nil {
		return
	}
	_ = a.mtStore.UpdateModelToolTitle(ref, modelToolCallLabel(name, args))
}

// finishModelTool closes an open mt_ record with the tool's final output.
// Missing/empty records are ignored; write failures never surface to the
// agent loop.
func (a *AgentRunner) finishModelTool(id string, output string, toolErr error, note string) {
	a.mtMu.Lock()
	ref, ok := a.mtRuns[id]
	if ok {
		delete(a.mtRuns, id)
	}
	a.mtMu.Unlock()
	if !ok || a.mtStore == nil {
		return
	}
	if toolErr == nil && strings.TrimSpace(note) != "" {
		toolErr = fmt.Errorf("%s", note)
	}
	_ = a.mtStore.FinishModelTool(ref, unwrapModelToolOutput(output), toolErr)
}

// unwrapModelToolOutput 把工具结果信封拆包出正文（v4.62.2）。
//
// Why: 本地模型工具（vision / summarize_file）的 ToolResult.Output 是 JSON
// 信封串（{"ok":true,...,"data":{"result":"正文"}}）。原样落 transcript 会让
// 会话 tab 渲染出一整面「字面 \n」的转义墙（信封编码吃掉真实换行），实机
// 报告为「输出没有渲染、一团乱」。这里识别标准信封并只取 data.result 正文；
// 非信封形态（自由文本/已拆包/解析失败）原样返回，绝不猜。
func unwrapModelToolOutput(output string) string {
	// v4.64.1 改递归拆包：实测信封存在双层嵌套——外层执行器信封的
	// data.result 里又装着工具自身的 {"ok":...,"message":"正文"} 信封，
	// 只拆一层仍会留下 JSON 转义墙。每轮从 data.result / data.message /
	// data.output / message / result / output 取第一个非空字符串字段作为
	// 新内容，直到不再是 JSON 或取不出新内容（上限 4 层）；非信封形态
	// 原样返回，绝不猜。
	cur := strings.TrimSpace(output)
	for i := 0; i < 4 && strings.HasPrefix(cur, "{"); i++ {
		var envelope struct {
			Data struct {
				Result  string `json:"result"`
				Message string `json:"message"`
				Output  string `json:"output"`
			} `json:"data"`
			Message string `json:"message"`
			Result  string `json:"result"`
			Output  string `json:"output"`
		}
		if json.Unmarshal([]byte(cur), &envelope) != nil {
			break
		}
		next := strings.TrimSpace(envelope.Data.Result)
		if next == "" {
			next = strings.TrimSpace(envelope.Data.Message)
		}
		if next == "" {
			next = strings.TrimSpace(envelope.Data.Output)
		}
		if next == "" {
			next = strings.TrimSpace(envelope.Message)
		}
		if next == "" {
			next = strings.TrimSpace(envelope.Result)
		}
		if next == "" {
			next = strings.TrimSpace(envelope.Output)
		}
		if next == "" {
			break
		}
		cur = next
	}
	return cur
}

// cleanupModelToolRunsOnTurnEnd fails any model-tool record that never saw a
// ToolResult (stream aborted mid-tool, batch suppressed before dispatch,
// context cancelled). Runs at every turn boundary via runDirect's defer.
func (a *AgentRunner) cleanupModelToolRunsOnTurnEnd(reason string) {
	a.mtMu.Lock()
	stale := a.mtRuns
	a.mtRuns = make(map[string]string, 8)
	store := a.mtStore
	a.mtMu.Unlock()
	if len(stale) == 0 || store == nil {
		return
	}
	for _, ref := range stale {
		_ = store.FinishModelTool(ref, "", fmt.Errorf("%s", reason))
	}
}

// modelToolCallLabel turns a model-backed tool call into a short UI title
// (kept < 160 runes; fallback = tool name).
func modelToolCallLabel(name, rawArgs string) string {
	if strings.TrimSpace(rawArgs) == "" {
		return name
	}
	base := name
	var extra string
	switch name {
	case "vision":
		var p struct {
			ImagePath string `json:"image_path"`
			Prompt    string `json:"prompt"`
		}
		if json.Unmarshal([]byte(rawArgs), &p) == nil {
			if p.ImagePath != "" {
				extra = "识别图片 " + p.ImagePath
				if p.Prompt != "" {
					extra += "（" + p.Prompt + "）"
				}
			}
		}
	case "summarize_file":
		var p struct {
			Path  string   `json:"path"`
			Paths []string `json:"paths"`
			Focus string   `json:"focus"`
		}
		if json.Unmarshal([]byte(rawArgs), &p) == nil {
			paths := p.Paths
			if len(paths) == 0 && p.Path != "" {
				paths = []string{p.Path}
			}
			if len(paths) == 1 {
				extra = "摘要 " + paths[0]
			} else if len(paths) > 1 {
				extra = fmt.Sprintf("摘要 %d 个大文件", len(paths))
			}
			if p.Focus != "" {
				extra += "（侧重：" + p.Focus + "）"
			}
		}
	default:
		if len(rawArgs) > 80 {
			extra = rawArgs[:80] + "…"
		} else {
			extra = rawArgs
		}
	}
	if extra == "" {
		return base
	}
	s := base + " · " + strings.TrimSpace(extra)
	r := []rune(s)
	if len(r) <= 160 {
		return s
	}
	return string(r[:159]) + "…"
}
