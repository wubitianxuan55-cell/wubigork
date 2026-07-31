package app

import (
	"encoding/json"
	"fmt"
	"sync"

	gaeaBoot "github.com/wubigork/wubigork/internal/gaea/boot"
	gaeaConfig "github.com/wubigork/wubigork/internal/gaea/config"
	"github.com/wubigork/wubigork/internal/gaea/control"
	"github.com/wubigork/wubigork/internal/gaea/event"
	"github.com/wubigork/wubigork/internal/gaea/provider/bridge"
	"github.com/wubigork/wubigork/internal/gaea/tool"
	"github.com/wubigork/wubigork/internal/modelengine"
)

// ── gaea 工程办公板块（移植自 gaeaW）─────────────────────────────
// 独立板块：47 个工程工具 + 6 个技能 + Hermes/Hephaestus 双模型 agent。
// 模型走 wubigrok 模型中心（bridge provider），前端 UI + AI 双通道调用。

var ga = &gaeaRuntime{}

type gaeaRuntime struct {
	mu   sync.Mutex
	ctrl *control.Controller
}

// GaeaInit 初始化办公引擎（幂等）。用 wubigrok 模型中心的默认引擎驱动。
func (a *App) GaeaInit() error {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.ctrl != nil {
		return nil
	}

	// 1. 注入模型中心客户端（bridge provider 的底层）
	bridge.SetClient(a.client)

	// 2. 注入配置：bridge provider（kind=wubigrok 走 wubigrok 模型中心）
	//    Model 留空：每次请求由 ai.Client 按当前活跃引擎动态解析默认模型，
	//    实现办公板块自动跟随模型中心的引擎/模型切换。
	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "wubigrok"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name:          "wubigrok",
		Kind:          "wubigrok",
		Model:         "",
		ContextWindow: 1_000_000,
	}}
	// 全部 47 个工程工具注册（Enabled 为空 = 全部）
	cfg.Tools.Enabled = nil
	// 关闭写文件/网络类工具的沙箱限制，避免办公工具被无谓拦截
	cfg.Sandbox.Bash = "off"
	// 工具执行免审批（GUI 无审批 UI；gaeaW ask 模式在无 TTY 时也解析为 allow）
	cfg.Permissions.Mode = "allow"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })

	// 4. 事件转发：gaea 事件流 → 前端 gaea-event 回调
	sink := event.FuncSink(func(e event.Event) {
		a.emit("gaea-event", gaeaEventMap(e))
	})

	// 5. 构建 controller（Hermes 规划 + Hephaestus 执行）
	ctrl, err := gaeaBoot.Build(a.ctx, gaeaBoot.Options{
		Model:      "wubigrok",
		RequireKey: false,
		Sink:       sink,
		MaxSteps:   0,
	})
	if err != nil {
		return fmt.Errorf("gaea: 引擎初始化失败: %w", err)
	}
	ga.ctrl = ctrl
	return nil
}

// GaeaSend 提交对话（异步，事件经 gaea-event 回调）。未初始化时自动初始化。
func (a *App) GaeaSend(input string) {
	if err := a.GaeaInit(); err != nil {
		a.emit("gaea-event", map[string]interface{}{"kind": "error", "text": err.Error()})
		return
	}
	ga.mu.Lock()
	ctrl := ga.ctrl
	ga.mu.Unlock()
	if ctrl != nil {
		ctrl.Send(input)
	}
}

// GaeaCancel 取消当前回合。
func (a *App) GaeaCancel() {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.ctrl != nil {
		ga.ctrl.Cancel()
	}
}

// GaeaRunning 报告引擎是否正在运行。
func (a *App) GaeaRunning() bool {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	return ga.ctrl != nil && ga.ctrl.Running()
}

// GaeaNewSession 开启新会话（清空上下文，保留记忆/技能）。
func (a *App) GaeaNewSession() error {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.ctrl == nil {
		return nil
	}
	return ga.ctrl.NewSession()
}

// GaeaModel 实时返回模型中心当前活跃的引擎与模型（engine/model 格式）。
func (a *App) GaeaModel() string {
	engine := a.GetActiveEngine()
	model := a.GetActiveModel()
	if model == "" {
		return engine
	}
	return engine + "/" + model
}

// GaeaEngines 返回模型中心全部引擎（办公板块切换用）。
func (a *App) GaeaEngines() []modelengine.EngineConfig {
	if a.engineMgr == nil {
		return nil
	}
	return a.engineMgr.GetEngines()
}

// GaeaSetEngine 切换办公板块使用的模型中心引擎（与全局活跃引擎联动）。
func (a *App) GaeaSetEngine(engineID string) error {
	return a.SetActiveEngine(engineID)
}

// GaeaTools 列出全部内置工程工具（UI 面板用）。
func (a *App) GaeaTools() []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	for _, t := range tool.Builtins() {
		out = append(out, map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"schema":      string(t.Schema()),
		})
	}
	return out
}

// GaeaSkills 列出工程技能模块。
func (a *App) GaeaSkills() []map[string]interface{} {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	out := make([]map[string]interface{}, 0)
	if ga.ctrl == nil {
		return out
	}
	for _, s := range ga.ctrl.Skills() {
		out = append(out, map[string]interface{}{
			"name":        s.Name,
			"description": s.Description,
		})
	}
	return out
}

// GaeaCallTool 前端直接调用工具（UI 双通道）：name 工具名，argsJSON 参数 JSON。
func (a *App) GaeaCallTool(name, argsJSON string) (string, error) {
	t, ok := tool.LookupBuiltin(name)
	if !ok {
		return "", fmt.Errorf("gaea: 未知工具 %q", name)
	}
	return t.Execute(a.ctx, json.RawMessage(argsJSON))
}

// ── 事件转换 ────────────────────────────────────────────────────

// gaeaEventMap 把 gaea 事件流转换为 gaeaW WireEvent 兼容格式（前端 store 直接消费）。
func gaeaEventMap(e event.Event) map[string]interface{} {
	m := map[string]interface{}{"kind": gaeaKindName(e.Kind)}
	switch e.Kind {
	case event.Text, event.Reasoning:
		if e.Text != "" {
			m["text"] = e.Text
		}
		if e.Reasoning != "" {
			m["reasoning"] = e.Reasoning
		}
	case event.Message:
		m["text"] = e.Text
		if e.Reasoning != "" {
			m["reasoning"] = e.Reasoning
		}
	case event.ToolDispatch:
		m["tool"] = map[string]interface{}{
			"id": e.Tool.ID, "name": e.Tool.Name, "args": e.Tool.Args,
			"readOnly": e.Tool.ReadOnly, "partial": e.Tool.Partial,
			"parentId": e.Tool.ParentID,
		}
	case event.ToolResult:
		t := map[string]interface{}{
			"id": e.Tool.ID, "name": e.Tool.Name, "output": e.Tool.Output,
			"recoverable": e.Tool.Recoverable, "truncated": e.Tool.Truncated,
		}
		if e.Tool.Err != "" {
			t["err"] = e.Tool.Err
		}
		m["tool"] = t
	case event.Notice:
		m["text"] = e.Text
		m["level"] = gaeaLevelName(e.Level)
	case event.Phase:
		m["text"] = e.Text
	case event.TurnDone:
		if e.Err != nil {
			m["err"] = e.Err.Error()
		}
	case event.Usage:
		if e.Usage != nil {
			m["usage"] = map[string]interface{}{
				"promptTokens":           e.Usage.PromptTokens,
				"completionTokens":       e.Usage.CompletionTokens,
				"totalTokens":            e.Usage.TotalTokens,
				"cacheHitTokens":         e.Usage.CacheHitTokens,
				"cacheMissTokens":        e.Usage.CacheMissTokens,
				"reasoningTokens":        e.Usage.ReasoningTokens,
				"sessionCacheHitTokens":  e.SessionHit,
				"sessionCacheMissTokens": e.SessionMiss,
				"turn":                   e.Turn,
				"source":                 e.UsageSource,
			}
		}
	case event.ApprovalRequest:
		m["approval"] = map[string]interface{}{"id": e.Approval.ID, "tool": e.Approval.Tool, "subject": e.Approval.Subject}
	case event.AskRequest:
		qs := make([]map[string]interface{}, 0, len(e.Ask.Questions))
		for _, q := range e.Ask.Questions {
			opts := make([]map[string]interface{}, 0, len(q.Options))
			for _, o := range q.Options {
				opt := map[string]interface{}{"label": o.Label}
				if o.Description != "" {
					opt["description"] = o.Description
				}
				opts = append(opts, opt)
			}
			qq := map[string]interface{}{"id": q.ID, "prompt": q.Prompt, "options": opts, "multi": q.Multi}
			if q.Header != "" {
				qq["header"] = q.Header
			}
			if q.Plan != "" {
				qq["plan"] = q.Plan
			}
			qs = append(qs, qq)
		}
		m["ask"] = map[string]interface{}{"id": e.Ask.ID, "questions": qs}
	case event.CompactionStarted:
		m["compaction"] = map[string]interface{}{"trigger": e.Compaction.Trigger}
	case event.CompactionDone:
		m["compaction"] = map[string]interface{}{
			"trigger": e.Compaction.Trigger, "messages": e.Compaction.Messages,
			"summary": e.Compaction.Summary, "archive": e.Compaction.Archive,
		}
	}
	return m
}

// gaeaKindName 事件类型名映射（对齐 gaeaW WireEvent.EventKind）。
func gaeaKindName(k event.Kind) string {
	names := map[event.Kind]string{
		event.TurnStarted: "turn_started", event.Reasoning: "reasoning", event.Text: "text",
		event.Message: "message", event.ToolDispatch: "tool_dispatch", event.ToolResult: "tool_result",
		event.Usage: "usage", event.Notice: "notice", event.Phase: "phase",
		event.ApprovalRequest: "approval_request", event.AskRequest: "ask_request",
		event.TurnDone: "turn_done", event.CompactionStarted: "compaction_started",
		event.CompactionDone: "compaction_done",
	}
	if n, ok := names[k]; ok {
		return n
	}
	return "unknown"
}

// gaeaLevelName 通知级别名。

// gaeaLevelName 通知级别名。
func gaeaLevelName(l event.Level) string {
	switch l {
	case event.LevelWarn:
		return "warn"
	default:
		return "info"
	}
}
