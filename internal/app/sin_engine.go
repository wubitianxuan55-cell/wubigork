package app

// ── 原罪独立引擎（2026-11-15 升格刀）────────────────────────────────────
//
// 铁律（用户红线）：本文件族不修改任何办公共享件——boot 包私有装配、
// gaea_handler.go 的 ga.ctrl/事件转发器、spacetags/sysprompt 一律零触碰；
// 装配代码 fork 进 sin 命名空间，所需构造件全部为 internal/gaea 导出 API
// （control.New / session.New / session.NewEventLogSink / tool.NewRegistry /
// provider.NewLLM / memory.Load / jobs.NewManager / context.NewContextManager /
// boot.NewProvider〔导出，注释明言供自定义装配器复用〕）。
//
// 与办公引擎（ga.ctrl）的关系：完全平行的第二实例，独立锁/独立事件通道/
// 独立会话目录（书/.gaea/sessions/play/）。办公与原罪页面 keepAlive 下可同时
// 挂载，任何共享可变状态都会互踩——这里从构造上杜绝。
//
// 装配差异（相对 boot.Build 的裁剪）：无 task/子代理/技能/MCP 插件/大文件摘要
// （特化①：工具只服务图文创作）；builtin 只开 read_file/ls（只读面）；工具
// 注册表 = builtin 白名单 + shared 元工具（ask+记忆四件）+ sin 9 工具适配器
// （sin_engine_tools.go）；权限走 play 缺省（allow，deny 硬拒，无审批卡）。

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/cache"
	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	tiancontext "github.com/gaea/gaea/internal/gaea/context"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/hook"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/trajectory"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// sinEngineChannel 原罪引擎专用 wails 事件通道（与办公 gaea-event 完全隔离：
// 独立 seq 空间、独立断号补拉，前端 fork store 各订各的）。
const sinEngineChannel = "gaea-event-sin"

// sinEngineState 原罪引擎运行态（与办公 ga 结构平行的独立实例）。
type sinEngineState struct {
	mu       sync.Mutex
	ctrl     *control.Controller
	bookPath string
	bookSlug string
	label    string
	wire     *gaeaEventForwarder // 复用办公转发器类型（同包纯类型），seq 独立
	eventLog *session.EventLogSink
}

var sinEng = &sinEngineState{wire: newGaeaEventForwarder()}

// sinCtrlSnapshot 取当前控制器（快照读，调用方不持锁使用）。
func sinCtrlSnapshot() *control.Controller {
	sinEng.mu.Lock()
	defer sinEng.mu.Unlock()
	return sinEng.ctrl
}

// emitSinEngineEvent 事件转发：复用 gaeaEventMap/转发器类型（同包），发到
// sin 专用通道；不变量与办公一致——凡过 wire 的事件必已入 EventLogSink。
func (a *App) emitSinEngineEvent(e event.Event) {
	if m := sinEng.wire.payload(e); m != nil {
		a.emit(sinEngineChannel, m)
	}
}

// sinEngineSink 事件汇：转发到前端 + TurnDone 挂 sin 回合后钩子（刀3 章节
// 落盘扫描的挂点，当前为空实现占位）。
func (a *App) sinEngineSink() event.Sink {
	return event.FuncSink(func(e event.Event) {
		a.emitSinEngineEvent(e)
		if e.Kind == event.TurnDone && e.Err == nil {
			a.sinAfterTurn()
		}
	})
}

// sinAfterTurn 回合成功后的 sin 钩子（章节落盘扫描挂点；刀3 接线）。
func (a *App) sinAfterTurn() {}

// ── 装配 ────────────────────────────────────────────────────────────

// sinEngineOpenBook 打开/切换书工作区（幂等：同书已构建直接复用）。
// bookPath 必须通过书架校验（SinWorkspaceSet 已保证；直传路径这里再校验）。
func (a *App) SinEngineOpenBook(bookPath string) error {
	abs, _, err := a.sinValidateBookPath(bookPath)
	if err != nil {
		return err
	}
	sinEng.mu.Lock()
	defer sinEng.mu.Unlock()
	if sinEng.ctrl != nil && sinEng.bookPath == abs {
		return nil
	}
	if err := a.sinBuildController(abs); err != nil {
		return err
	}
	return nil
}

// sinProviderFor 构造原罪 provider：sin 功能绑定（FuncSinEngine/FuncSinModel）
// 显式 Engine/Model 走 wubigrok bridge（与 task-route 同构造式，不碰
// bridge.SetFeature 全局）；无绑定时回退办公 gaea 功能绑定；条目用于
// 计价/窗口/余额元数据。
func (a *App) sinProviderFor(cfg *gaeaConfig.Config) (provider.LLMProvider, *gaeaConfig.ProviderEntry, error) {
	eng, model := "", ""
	if a.cfg != nil {
		eng, model = a.cfg.GetFeatureModel("sin")
	}
	if model == "" && a.cfg != nil {
		// 无 sin 绑定：回退办公 gaea 功能绑定（app 配置读端，只读）。
		eng, model = a.cfg.GetFeatureModel("gaea")
	}
	if model == "" {
		model = cfg.DefaultModel
	}
	entry, ok := cfg.ResolveModel(model)
	if !ok {
		entry, ok = cfg.ResolveModel("gaea")
	}
	if !ok {
		entry = &gaeaConfig.ProviderEntry{Name: "sin", Kind: provider.LLMKindWubigrok, Model: model, ContextWindow: 1_000_000}
	}
	if entry.ContextWindow == 0 {
		entry.ContextWindow = 1_000_000
	}
	var prov provider.LLMProvider
	var err error
	if eng != "" && entry.Kind == provider.LLMKindWubigrok {
		// bridge provider：显式 Engine/Model 直传（不动全局 feature 缝）。
		prov, err = provider.NewLLM(provider.LLMKindWubigrok, provider.Config{
			Name: "sin", Engine: eng, Model: model,
		})
	} else {
		// 云端条目（deepseek 等）：按条目直连（boot.NewProvider 为导出装配缝）。
		prov, err = gaeaBoot.NewProvider(entry)
	}
	if err != nil {
		return nil, nil, err
	}
	return prov, entry, nil
}

// sinBuildController 装配原罪引擎控制器（fork boot.Build，裁剪面见文件头）。
// 调用方必须持有 sinEng.mu。
func (a *App) sinBuildController(bookPath string) error {
	cfg, err := gaeaLoadConfig()
	if err != nil {
		return fmt.Errorf("sin: 配置加载失败: %w", err)
	}
	// bridge provider 底层客户端注入（与办公 GaeaInit 同一客户端，幂等）。
	bridge.SetClient(a.clientRef())

	prov, entry, err := a.sinProviderFor(cfg)
	if err != nil {
		return fmt.Errorf("sin: provider 构造失败: %w", err)
	}

	bookSlug := sinBookSlug(bookPath)

	// 会话目录（书工作区 play 分区）+ 事件日志链（「模型可见必入日志」单点）。
	sessionDir := sinEngineSessionDir(bookPath)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("sin: 会话目录创建失败: %w", err)
	}
	baseSink := a.sinEngineSink()
	sink := event.Sync(baseSink)
	var eventLogSink *session.EventLogSink
	if cfg.LogFormatIsEvent() {
		eventLogSink = session.NewEventLogSink(sessionDir, sink)
		sink = event.Sync(eventLogSink)
	}
	jm := jobs.NewManager(sink)

	// 提示词 + 书级记忆（sin_engine_prompt.go；space 恒 play，slug=书）。
	sp, err := sinBuildEnginePrompt(cfg, bookPath)
	if err != nil {
		return fmt.Errorf("sin: 提示词装配失败: %w", err)
	}

	// 权限：play 空间策略（config 只读复用；allow 缺省不弹审批卡，deny 硬拒）。
	perm := cfg.PermissionsForSpace("play")
	policy := permission.New(perm.Mode, perm.Allow, perm.Ask, perm.Deny)
	gate := permission.NewGate(policy, nil)
	gate.OnAutoAllow = func(toolName, subject, source string) {
		sink.Emit(event.Event{Kind: event.ApprovalAuto, Auto: event.AutoApproval{
			Tool: toolName, Subject: subject, Source: source, Decided: event.AutoDecidedHeadless,
		}})
	}
	gate.OnUnapprovedDeny = func(toolName, subject, source string) {
		sink.Emit(event.Event{Kind: event.ApprovalAuto, Auto: event.AutoApproval{
			Tool: toolName, Subject: subject, Source: source, Decided: event.AutoDecidedUnapprovedDeny,
		}})
	}

	// 钩子 runner（sin 项目钩子缺省不信任=只装配不执行；保类型缝完整）。
	bashSpec := sandbox.Spec{WriteRoots: cfg.WriteRootsAt(bookPath), Network: cfg.Sandbox.Network, FailClosed: cfg.Sandbox.FailClosed}
	hookRunner := hook.NewRunner(
		hook.Load(hook.LoadOptions{ProjectRoot: bookPath, Trusted: false}),
		bookPath, nil,
		func(msg string) { sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: msg}) },
	).WithSandbox(bashSpec)
	accessPolicy := tool.NewPathAccessPolicy(bookPath, cfg.ReadRootsAt(bookPath), cfg.WriteRootsAt(bookPath), nil)

	// 工具注册表（sin_engine_tools.go：builtin 白名单+元工具+sin 适配器）。
	reg := sinEngineRegistry(a, cfg, bookPath, bookSlug, sp.mem)
	sp.compiler.SetRegistry(reg)

	// 中央工具派发器（执行前检查，与办公同语义）。
	dispatcher := agent.NewToolDispatcher(gate, hookRunner)

	// 单模型执行器（温度/参数走 sin 基线，不读办公 agent 配置）。
	execSess := agent.NewSession(sp.prompt)
	executor := agent.New(prov, reg, execSess, agent.Options{
		MaxSteps:      0,
		Temperature:   sinTemperature,
		Pricing:       entry.Price,
		Gate:          gate,
		Hooks:         hookRunner,
		AccessPolicy:  accessPolicy,
		Jobs:          jm,
		ContextWindow: entry.ContextWindow,
		Compaction:    sinCompactionConfig(cfg),
		Dispatcher:    dispatcher,
		JournalDir:    filepath.Join(bookPath, ".gaea", "work", "journal"),
		Spill:         cfg.Agent.ToolSpillEnabled(),
	}, sink)

	// 记忆 5.1 出口闸：定稿剥离悬空 [MEM:] 引用（与办公同语义，play 视图）。
	if cfg.Memory.Enabled {
		if gdb := db.GetDatabase(gaeaConfig.MemoryUserDir()); gdb != nil {
			sanitizeStore := memory.SQLiteStoreFor(gdb, gaeaConfig.MemoryUserDir(), bookPath)
			sanitizeLog := &memory.EventLog{DB: gdb}
			executor.SetFinalizeText(func(text string) string {
				cleaned, stripped := sanitizeStore.StripDanglingCitations(text, "play")
				if len(stripped) > 0 {
					sanitizeLog.AppendCiteEvents(nil, stripped, "play")
				}
				return cleaned
			})
		}
	}

	// 四层缓存内核（L1 身份/L2 运行时/技能占位/流式压缩层）。
	ctxMgr := tiancontext.NewContextManager(
		sp.compiler.IdentityLayer(),
		sp.runtimeCtx,
		cache.NewSkillLayer(),
		tiancontext.NewFlowLayer(tiancontext.CompactPolicy{
			Window:     entry.ContextWindow,
			TailTokens: 16384,
		}),
	)
	executor.SetCtxMgr(ctxMgr)

	ctrlOpts := control.Options{
		Runner:          executor,
		Executor:        executor,
		Sink:            sink,
		Policy:          policy,
		Label:           "sin/" + entry.Name,
		SystemPrompt:    sp.prompt,
		SessionDir:      sessionDir,
		LogFormat:       cfg.EffectiveLogFormat(),
		Space:           "play",
		Hooks:           hookRunner,
		Memory:          sp.mem,
		MemoryDisabled:  !cfg.Memory.Enabled,
		ApprovalTimeout: time.Duration(perm.ApprovalTimeoutSecs) * time.Second,
		HardAskTools:    map[string]bool{}, // play 缺省：不弹审批卡（deny 规则仍硬拒）
		Cleanup: func() {
			if eventLogSink != nil {
				_ = eventLogSink.Close()
			}
		},
		BalanceURL:    entry.BalanceURL,
		BalanceKey:    entry.APIKey(),
		BalanceKind:   entry.BalanceKind,
		Jobs:          jm,
		Registry:      reg,
		PluginCtx:     a.ctx,
		CtxMgr:        ctxMgr,
		WorkspaceRoot: bookPath,
	}
	ctrl := control.New(ctrlOpts)
	ctrl.WireJobWake()
	ctrl.EnableInteractiveApproval()
	if eventLogSink != nil {
		eventLogSink.SetPathSource(func() string { return ctrl.SessionPath() })
		eventLogSink.SetSpaceSource(func() string { return ctrl.SessionSpace() })
	}

	// 换实例：旧控制器 Close（与办公 gaeaRebuildLocked 同纪律）。
	old := sinEng.ctrl
	sinEng.ctrl = ctrl
	sinEng.bookPath = bookPath
	sinEng.bookSlug = bookSlug
	sinEng.label = ctrlOpts.Label
	sinEng.eventLog = eventLogSink
	sinEng.wire.reset()
	if old != nil {
		old.Close()
	}
	slog.Info("sin: 原罪引擎已装配", "book", filepath.ToSlash(bookPath), "model", entry.Model)
	return nil
}

// sinCompactionConfig 压缩配置（fork boot.compactionConfig 的 sin 子集）。
func sinCompactionConfig(cfg *gaeaConfig.Config) agent.CompactionConfig {
	soft, ratio, force := cfg.Agent.CompactionTiers()
	return agent.CompactionConfig{
		ArchiveDir: gaeaConfig.ArchiveDir(),
		SoftRatio:  soft,
		Ratio:      ratio,
		ForceRatio: force,
	}
}

// ── 绑定族（与办公 Gaea* 平行，全部落 sinEng）────────────────────────

// sinEngineEnsure 未初始化时先装配（书路径缺失报错，不猜默认书）。
func (a *App) sinEngineEnsure() error {
	sinEng.mu.Lock()
	defer sinEng.mu.Unlock()
	if sinEng.ctrl != nil {
		return nil
	}
	return errors.New("原罪引擎未初始化：先在书架选择一本书")
}

// SinEngineSend 提交一轮创作（异步，事件经 gaea-event-sin 回调）。
// 输入富化：角色块+底稿块按书 slug 注入（历史由引擎会话自带）。
func (a *App) SinEngineSend(input string) error {
	sinEng.mu.Lock()
	ctrl, slug := sinEng.ctrl, sinEng.bookSlug
	sinEng.mu.Unlock()
	if ctrl == nil {
		a.emitSinEngineEvent(event.Event{Kind: event.Phase, Text: "正在启动原罪引擎"})
		return errors.New("原罪引擎未初始化：先在书架选择一本书")
	}
	ctrl.Send(a.sinEngineAssembleInput(slug, input))
	return nil
}

// SinEngineSteer 运行中插话调整（不打断工具执行）。
func (a *App) SinEngineSteer(input string) error {
	c := sinCtrlSnapshot()
	if c == nil {
		return errors.New("原罪引擎未初始化")
	}
	c.Steer(input)
	return nil
}

// SinEngineCancel 取消当前回合。
func (a *App) SinEngineCancel() {
	if c := sinCtrlSnapshot(); c != nil {
		c.Cancel()
	}
}

// SinEngineRunning 报告引擎是否在运行。
func (a *App) SinEngineRunning() bool {
	c := sinCtrlSnapshot()
	return c != nil && c.Running()
}

// SinEngineNewSession 开新会话（保留书级记忆；与办公同款运行闸）。
func (a *App) SinEngineNewSession() error {
	sinEng.mu.Lock()
	defer sinEng.mu.Unlock()
	if sinEng.ctrl == nil {
		return errors.New("原罪引擎未初始化：先在书架选择一本书")
	}
	if sinEng.ctrl.Running() {
		return errors.New("回合正在运行，请先停止当前回合再开启新会话")
	}
	sinEng.ctrl.SetGoal("")
	if err := sinEng.ctrl.NewSession(); err != nil {
		return err
	}
	sinEng.wire.reset()
	return nil
}

// SinEngineState 引擎状态视图（前端壳启动/书架联动消费）。
type SinEngineStateView struct {
	Ready    bool   `json:"ready"`
	BookPath string `json:"bookPath"`
	BookSlug string `json:"bookSlug"`
	Running  bool   `json:"running"`
	Label    string `json:"label"`
}

// SinEngineState 返回原罪引擎当前状态。
func (a *App) SinEngineState() SinEngineStateView {
	sinEng.mu.Lock()
	defer sinEng.mu.Unlock()
	st := SinEngineStateView{BookPath: sinEng.bookPath, BookSlug: sinEng.bookSlug}
	if sinEng.ctrl != nil {
		st.Ready = true
		st.Running = sinEng.ctrl.Running()
		st.Label = sinEng.label
	}
	return st
}

// SinEngineApprove 审批决策回传（决策语义与办公同族）。
func (a *App) SinEngineApprove(id string, decision string) error {
	c := sinCtrlSnapshot()
	if c == nil {
		return errors.New("原罪引擎未初始化")
	}
	c.Approve(id, decision)
	return nil
}

// SinEngineAnswerQuestion ask 结构化提问答复（与办公 GaeaAnswer 同参形）。
func (a *App) SinEngineAnswerQuestion(id string, answers []event.AskAnswer) error {
	c := sinCtrlSnapshot()
	if c == nil {
		return errors.New("原罪引擎未初始化")
	}
	c.AnswerQuestion(id, answers)
	return nil
}

// SinEngineResyncEvents 断号补拉（fork GaeaResyncEvents，独立 wire/会话）。
func (a *App) SinEngineResyncEvents(afterSeq int) GaeaResyncResult {
	res := GaeaResyncResult{Seq: sinEng.wire.last(), Items: []GaeaResyncItem{}}
	c := sinCtrlSnapshot()
	if c == nil {
		return res
	}
	entries, err := session.ReadEntriesFor(c.SessionPath())
	if err != nil {
		return res
	}
	res.Items = foldResyncItems(entriesAfter(entries, int64(afterSeq)))
	return res
}

// SinEngineListSessions 书工作区会话清单（新→旧；复用办公折叠器，读 sin 目录）。
func (a *App) SinEngineListSessions() []SessionMeta {
	cur := ""
	if c := sinCtrlSnapshot(); c != nil {
		cur = c.SessionPath()
	}
	sinEng.mu.Lock()
	book := sinEng.bookPath
	sinEng.mu.Unlock()
	if book == "" {
		return []SessionMeta{}
	}
	// 传平铺基目录（与办公 GaeaListSessions 同口径：listSessionsForDir 会同时
	// 枚举两个空间分区目录；sin 只写 play 分区）。
	return a.listSessionsForDir(gaeaConfig.WorkspaceSessionDir(book, ""), cur, true)
}

// SinEngineResumeSession 切换到指定会话继续（fork GaeaResumeSession）。
func (a *App) SinEngineResumeSession(path string) ([]HistoryMessage, error) {
	sinEng.mu.Lock()
	c := sinEng.ctrl
	sinEng.mu.Unlock()
	if c == nil {
		return nil, errors.New("原罪引擎未初始化")
	}
	if c.Running() {
		return nil, errors.New("回合正在运行，请先停止当前回合再切换会话")
	}
	_ = c.Snapshot()
	loaded, err := c.ResumeFromDisk(path)
	if err != nil {
		return nil, err
	}
	sinEng.wire.reset()
	injectInterruption(loaded, path)
	c.SeedContextUsage()
	return a.sinEngineHistory(), nil
}

// sinEngineHistory 当前会话对话历史（fork GaeaHistory 的 sin 子集：无子代理
// 锚点——sin 不装子代理工具）。
func (a *App) sinEngineHistory() []HistoryMessage {
	out := []HistoryMessage{}
	c := sinCtrlSnapshot()
	if c == nil {
		return out
	}
	return buildHistoryMessages(c.History())
}

// SinEngineHistory 当前会话对话历史（前端 store 装载/兜底消费）。
func (a *App) SinEngineHistory() []HistoryMessage {
	return a.sinEngineHistory()
}

// SinEngineContext 上下文用量快照（fork GaeaContext，读 sin 控制器）。
func (a *App) SinEngineContext() ContextInfo {
	c := sinCtrlSnapshot()
	if c == nil {
		return ContextInfo{}
	}
	used, window := c.ContextSnapshot()
	info := ContextInfo{Used: used, Window: window}
	if _, ratio, _ := c.CompactionTiers(); ratio > 0 && window > 0 {
		info.CompactRatio = ratio
		left := int(float64(window)*ratio) - used
		if left < 0 {
			left = 0
		}
		info.CompactLeft = left
	}
	return info
}

// SinEngineTrajectory 会话轨迹时间线（fork GaeaTrajectory，读 sin 会话日志；
// 折叠器 readEntriesForCached/FoldTrajectory 为同包复用）。
func (a *App) SinEngineTrajectory() (trajectory.Trajectory, error) {
	c := sinCtrlSnapshot()
	if c == nil {
		return trajectory.EmptyTrajectory(), nil
	}
	entries, err := readEntriesForCached(c.SessionPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trajectory.EmptyTrajectory(), nil
		}
		return trajectory.EmptyTrajectory(), err
	}
	return trajectory.FoldTrajectory(entries), nil
}

// SinEngineContextView 会话上下文构成快照（fork GaeaContextView）。
func (a *App) SinEngineContextView() (contextview.ContextTimeline, error) {
	c := sinCtrlSnapshot()
	if c == nil {
		return contextview.EmptyTimeline(), nil
	}
	entries, err := readEntriesForCached(c.SessionPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return contextview.EmptyTimeline(), nil
		}
		return contextview.EmptyTimeline(), err
	}
	_, window := c.ContextSnapshot()
	tl := contextview.FoldTimeline(entries, int64(window), 0)
	if soft, ratio, force := c.CompactionTiers(); ratio > 0 && soft > 0 && force > 0 {
		tl.Compact = &contextview.CompactTiers{Soft: soft, Ratio: ratio, Force: force}
	}
	return tl, nil
}

// SinEngineContextNodeDetail 上下文节点「完整调用」详情（fork GaeaContextNodeDetail）。
func (a *App) SinEngineContextNodeDetail(seq int64) (contextview.NodeDetail, error) {
	c := sinCtrlSnapshot()
	if c == nil {
		return contextview.NodeDetail{}, fmt.Errorf("原罪引擎未初始化")
	}
	entries, err := readEntriesForCached(c.SessionPath())
	if err != nil {
		return contextview.NodeDetail{}, err
	}
	d, ok := contextview.NodeDetailFor(entries, seq)
	if !ok {
		return contextview.NodeDetail{}, fmt.Errorf("未找到 seq=%d 的可展开节点", seq)
	}
	resolveNodeImages(&d, sinEng.bookPath)
	resolveNodeAttachments(&d, sinEng.bookPath)
	return d, nil
}
