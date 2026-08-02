package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/BurntSushi/toml"

	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/modelengine"
)

// ── gaea 工程办公板块（移植自 gaeaW）─────────────────────────────
// 独立板块：47 个工程工具 + 6 个技能 + 单模型 agent（规划+执行一体）。
// 模型走 gaea 模型中心（bridge provider），前端 UI + AI 双通道调用。

var ga = &gaeaRuntime{}

type gaeaRuntime struct {
	mu   sync.Mutex
	ctrl *control.Controller
	// cfg 是当前生效的办公引擎配置。设置面板的写操作（Agent 参数/权限/沙箱）
	// 直接修改它并持久化到用户配置，随后重建 controller 使变更生效。
	cfg *gaeaConfig.Config
}

// gaeaLoadConfig 加载办公引擎配置：内置默认 + 用户持久化文件（若有），
// 再注入 bridge provider（kind=gaea 走 gaea 模型中心）。返回可直接修改的配置。
func gaeaLoadConfig() (*gaeaConfig.Config, error) {
	cfg := gaeaConfig.Default()
	if p := gaeaConfig.UserConfigPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			if _, err := toml.Decode(string(b), cfg); err != nil {
				return nil, fmt.Errorf("gaea: 解析持久化配置 %s: %w", p, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("gaea: 读取持久化配置 %s: %w", p, err)
		}
	}
	cfg.DefaultModel = "gaea"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name:          "gaea",
		Kind:          "wubigrok", // 内部 provider 注册名（bridge provider）
		Model:         "",
		ContextWindow: 1_000_000,
	}}
	// 全部 47 个工程工具注册（Enabled 为空 = 全部）
	cfg.Tools.Enabled = nil
	// 关闭写文件/网络类工具的沙箱限制，避免办公工具被无谓拦截
	cfg.Sandbox.Bash = "off"
	return cfg, nil
}

// gaeaBuildController 用当前配置构建 controller（不持有 ga.mu，调用方负责）。
func (a *App) gaeaBuildController() (*control.Controller, error) {
	// 事件转发：gaea 事件流 → 前端 gaea-event 回调
	sink := event.FuncSink(func(e event.Event) {
		a.emit("gaea-event", gaeaEventMap(e))
	})
	// 构建 controller（单模型 agent）
	//    SessionDir 必须指向工作区会话目录（cwd/.gaea/sessions），与
	//    GaeaListSessions/GaeaResumeSession 的读取路径一致，否则历史面板
	//    永远看不到当前会话（会落到用户级 AppData/Roaming/gaea/sessions）。
	ctrl, err := gaeaBoot.Build(a.ctx, gaeaBoot.Options{
		Model:      "gaea",
		RequireKey: false,
		Sink:       sink,
		MaxSteps:   0,
		SessionDir: gaeaConfig.WorkspaceSessionDir(""),
	})
	if err != nil {
		return nil, fmt.Errorf("gaea: 引擎初始化失败: %w", err)
	}
	// 启用交互式审批：工具调用放行/拒绝、ask 结构化提问经前端确认，
	// 否则全部工具（含写文件/网络）自动放行且审批弹窗永不出现。
	ctrl.EnableInteractiveApproval()
	return ctrl, nil
}

// gaeaRebuildLocked 用当前配置重建 controller（设置变更后生效），替换旧实例。
// 调用方必须已持有 ga.mu。
func (a *App) gaeaRebuildLocked() error {
	newCtrl, err := a.gaeaBuildController()
	if err != nil {
		return err
	}
	old := ga.ctrl
	ga.ctrl = newCtrl
	if old != nil {
		old.Close()
	}
	return nil
}

// GaeaInit 初始化办公引擎（幂等）。用 gaea 模型中心的默认引擎驱动。
func (a *App) GaeaInit() error {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.ctrl != nil {
		return nil
	}

	// 1. 注入模型中心客户端（bridge provider 的底层）
	bridge.SetClient(a.client)

	// 2. 注入配置：持久化文件（用户设置）+ bridge provider
	cfg, err := gaeaLoadConfig()
	if err != nil {
		return err
	}
	ga.cfg = cfg
	// loader 无锁读 ga.cfg：ga.cfg 指针的替换在持锁下进行，读取方只会拿到
	// 一个完整可用的配置（旧指针在替换后不再被修改），不会与重建死锁。
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) {
		if ga.cfg == nil {
			return gaeaConfig.Default(), nil
		}
		return ga.cfg, nil
	})

	// 3. 构建 controller
	ctrl, err := a.gaeaBuildController()
	if err != nil {
		return err
	}
	ga.ctrl = ctrl
	// 通知前端办公引擎就绪（对应 gaea/lib/bridge.ts 的 onReady 监听 gaea-ready）
	a.emit("gaea-ready", map[string]interface{}{"kind": "ready"})
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
