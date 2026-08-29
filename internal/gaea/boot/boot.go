// Package boot assembles a ready-to-drive control.Controller from configuration:
// it loads config, resolves the model, builds the tool registry (built-ins +
// plugins), wires the permission gate, and constructs the single-model executor.
// It is the one place that turns "what the
// user configured" into "a Controller a frontend can drive", so every frontend —
// the terminal TUI, the HTTP/SSE server, the desktop webview — shares the exact
// same assembly instead of each re-deriving it. Frontends pass only a sink and a
// couple of run knobs; everything else comes from config.
package boot

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/archive"
	"github.com/gaea/gaea/internal/gaea/cache"
	"github.com/gaea/gaea/internal/gaea/command"
	"github.com/gaea/gaea/internal/gaea/config"
	tiancontext "github.com/gaea/gaea/internal/gaea/context"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/hook"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/largefile"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/gaea/plugin"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/netclient"
)

// Options carries the per-run knobs a frontend chooses; everything else is read
// from configuration. Model "" falls back to the configured default_model;
// MaxSteps 0 uses the config/default. RequireKey forces the executor's API key to
// be present (run/serve pass true so a missing key fails fast; chat/desktop pass
// false so the UI is reachable before a key is set). Sink receives the agent's
// typed event stream.
type Options struct {
	Model      string
	MaxSteps   int
	RequireKey bool
	Sink       event.Sink
	// Stderr is the writer for diagnostic warnings and plugin subprocess
	// stderr output. When nil, defaults to os.Stderr. Set to io.Discard
	// during model switch inside a bubbletea session to prevent any output
	// from corrupting the TUI's terminal raw mode.
	Stderr io.Writer
	// SessionDir overrides the global session directory. When empty,
	// config.SessionDir() (user-global) is used. Desktop mode sets this to
	// config.WorkspaceSessionDir(cwd) so sessions stay with the project.
	// Cwd 是工作空间根目录（桌面端 = gaeaCwd()）。非空时基础工具
	// （read_file/write_file/bash/ls）的相对路径基于它，而非进程 cwd。
	// 空 = 进程当前目录（CLI 原行为）。
	Cwd        string
	SessionDir string
	// ExtraTools 是前端（桌面端）额外注入的工具（如 image_gen 这类需要
	// 应用服务/客户端的工具）。空 = 不注入；与内置工具同名会被覆盖。
	ExtraTools []tool.Tool
}

// Build loads config, resolves the model, and returns a Controller wrapping a
// single Agent (single-model architecture; the two-model Hermes wrapper was
// removed). The returned controller owns plugin subprocesses; call Close (via
// Controller.Close) to release them.
var (
	sandboxWarnOnce sync.Once
	bashWarnOnce    sync.Once
)

func Build(ctx context.Context, opts Options) (*control.Controller, error) {
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	modelName := opts.Model
	if modelName == "" {
		modelName = cfg.DefaultModel
	}
	entry, ok := cfg.ResolveModel(modelName)
	if !ok {
		return nil, fmt.Errorf("unknown model %q (configured: %s)", modelName, providerNames(cfg))
	}
	// Providers without explicit context_window (e.g. user's "deepseek" with
	// per-model pricing) default to 0, which hides the status-bar gauge.
	if entry.ContextWindow == 0 {
		entry.ContextWindow = 1_000_000
	}
	if opts.RequireKey {
		if err := cfg.Validate(modelName); err != nil {
			return nil, err
		}
	}

	// Serialize the frontend's sink once: background jobs (below) emit from their
	// own goroutines, which can overlap a running turn's emission, so every emitter
	// shares this synchronized sink. The job manager is session-scoped — its jobs
	// outlive a turn and are cancelled by Controller.Close.
	sink := event.Sync(opts.Sink)
	// 3.0 Step 1: 会话事件日志（回退开关 session.log_format="event"；缺省 legacy
	// 保持旧行为，sink 链与改造前完全一致）。事件先写 <id>.gaea-log.jsonl 再转发
	// 前端——「模型可见必入日志」由该单点保证；会话路径在控制构建完成后注入。
	var eventLogSink *session.EventLogSink
	if cfg.LogFormatIsEvent() {
		eventLogSink = session.NewEventLogSink(orDefault(opts.SessionDir, config.SessionDir()), sink)
		sink = event.Sync(eventLogSink)
	}
	jm := jobs.NewManager(sink)

	execProv, err := NewProvider(entry)
	if cfg.Agent.Effort != "" {
		entry.Effort = cfg.Agent.Effort
	}
	if err != nil {
		return nil, err
	}

	cwd := opts.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// V10.22: system prompt + memory + skills assembled in sysprompt.go
	// 传入工作空间根（opts.Cwd），项目画像/技能索引基于真实工作区而非进程目录。
	sp, err := buildSystemPrompt(cfg, cwd, opts.Stderr)
	if err != nil {
		return nil, err
	}
	sysPrompt := sp.prompt
	mem := sp.mem
	skills := sp.skills
	compiler := sp.compiler
	runtimeCtx := sp.runtimeCtx
	skillStore := sp.store

	reg := tool.NewRegistry()
	bashSpec := sandbox.Spec{Mode: cfg.BashMode(), WriteRoots: cfg.WriteRoots(), Network: cfg.Sandbox.Network}
	if bashSpec.Mode == "enforce" && !sandbox.Available() {
		sandboxWarnOnce.Do(func() {
			fmt.Fprintln(stderr, "warning: bash sandbox requested but unavailable on this platform; running bash unconfined")
		})
	}
	if sandbox.ResolveShell().Kind == sandbox.ShellPowerShell {
		bashWarnOnce.Do(func() {
			fmt.Fprintln(stderr, "warning: bash not found on PATH; the shell tool will run commands under Windows PowerShell. Install Git for Windows or WSL to use bash.")
		})
	}
	addBuiltins(reg, opts.Cwd, cfg.Tools.Enabled, cfg.WriteRoots(), bashSpec, cfg.NetworkProxySpec(), stderr)
	// Always construct a host, even with no plugins configured, so the controller's
	// host pointer is stable for the session and `/mcp add` can hot-add into it.
	// V10.22: plugins + LSP assembled in plugins.go
	po := startPlugins(ctx, cfg, reg, sink, opts.Stderr)
	pluginHost := po.host
	cleanup := po.cleanup
	maxSteps := cfg.Agent.MaxSteps
	if opts.MaxSteps > 0 {
		maxSteps = opts.MaxSteps
	}

	// Permission policy gates every tool call. The headless gate (no Approver)
	// resolves "ask" to allow — preserving `gaea run` autonomy — while deny
	// rules hard-block in every mode. Interactive frontends (chat, desktop) swap
	// in an interactive gate later via Controller.EnableInteractiveApproval.
	// Sub-agents always run headless: they have no UI to answer a prompt, so they
	// inherit this same gate.
	policy := permission.New(cfg.Permissions.Mode, cfg.Permissions.Allow, cfg.Permissions.Ask, cfg.Permissions.Deny)
	headlessGate := permission.NewGate(policy, nil)

	// Hooks: load the global settings.json plus the project's (only when trusted —
	// project hooks run arbitrary shell commands, so cloning a repo must not
	// silently execute them). Non-blocking hook output is surfaced to the user as
	// a Notice through the shared sink. The runner fires PreToolUse/PostToolUse in
	// the agent loop and UserPromptSubmit/Stop at the controller's turn boundary.
	hooksTrusted := hook.IsTrusted(cwd, "")
	hookRunner := hook.NewRunner(
		hook.Load(hook.LoadOptions{ProjectRoot: cwd, Trusted: hooksTrusted}),
		cwd, nil,
		func(msg string) { sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: msg}) },
	)
	if hook.ProjectDefinesHooks(cwd) && !hooksTrusted {
		sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo,
			Text: "this project defines hooks but they are not trusted — run /hooks trust to enable them"})
	}

	// The `task` tool spawns sub-agents that reuse the parent's provider and
	// tool registry. Wired here after the built-ins / plugins are loaded so
	// sub-agents inherit the full tool set (minus `task` itself, to keep
	// nesting out of the picture). It registers into the same reg the
	// executor uses, so the model surfaces it like any other tool.
	taskTool := agent.NewTaskTool(execProv, entry.Price, reg, maxSteps,
		entry.ContextWindow, cfg.Agent.SubagentTemp(), config.ArchiveDir(), "", headlessGate)

	// V10.22: resolve subagent model from config. When SubagentModel names a
	// configured provider, sub-agents use that provider (typically a cheaper
	// flash model) while the parent keeps its own provider — independent API
	// calls mean the parent's prefix cache is unaffected.
	if subRef := strings.TrimSpace(cfg.Agent.SubagentModel); subRef != "" {
		if subEntry, ok := cfg.ResolveModel(subRef); ok {
			if e := cfg.Agent.SubagentEffortVal(); e != "" {
				subEntry.Effort = e
			}
			if subProv, err := NewProvider(subEntry); err == nil {
				taskTool.SetSubagentProvider(subProv, subEntry.Price, subEntry.ContextWindow)
			}
		}
	}
	reg.Add(taskTool)

	// The `remember` tool lets the model persist durable facts to the project's
	// auto-memory store; `forget` prunes ones that turn out wrong. The saved index
	// loads into the prefix on the next session.
	reg.Add(memory.NewRememberTool(mem.Store, memory.NewProfileStore(db.GetDatabase(config.MemoryUserDir()))))
	reg.Add(memory.NewForgetTool(mem.Store))
	reg.Add(memory.NewPromoteSessionFactsTool())
	reg.Add(memory.NewMemoryGetTool(mem.Store))

	// The `ask` tool puts structured multiple-choice questions to the user. It
	// reaches them through the Asker on the call context, which interactive
	// frontends wire to the controller (EnableInteractiveApproval); a headless run
	// has none, so ask resolves to "decide for yourself".
	reg.Add(agent.NewAskTool())

	// Skill tools: run_skill / install_skill plus the dedicated subagent wrappers
	// (explore / research / review / security_review). A subagent skill reuses the
	// sub-agent machinery via this runner — an isolated loop with the skill body
	// as system prompt, a tool set scoped to the skill's allowed-tools (minus the
	// task/skill meta-tools, to bar recursion), and an optional per-skill model.
	// Its tool activity nests under the invoking call, like `task`.
	skillRunner := func(sctx context.Context, sk skill.Skill, task string) (string, error) {
		prov, price, ctxWin := execProv, entry.Price, entry.ContextWindow
		if modelRef := subagentModelRef(cfg, sk); modelRef != "" {
			if me, ok := cfg.ResolveModel(modelRef); ok {
				if p, err := NewProvider(me); err == nil {
					prov, price, ctxWin = p, me.Price, me.ContextWindow
				}
			}
		}
		subReg := agent.FilterRegistry(reg, sk.AllowedTools, agent.SubagentMetaTools()...)
		steps := maxSteps
		if steps > 0 {
			if steps /= 2; steps < 5 {
				steps = 5
			}
		}
		// V5.25: 构建与父代理一致的 [L1][L2] 双 system 消息结构。
		// L1 来自 Fork 后的 compiler，L2 通过 opts.RuntimePrompt 注入。
		// skill body 放在 user task 前面，不混入 system 消息。
		childCompiler := compiler.Fork()
		sysPrompt := childCompiler.SystemPrompt()

		return agent.RunSubAgent(sctx, prov, subReg, sysPrompt, sk.Body+"\n\n"+task, agent.Options{
			MaxSteps:      steps,
			Temperature:   cfg.Agent.Temperature,
			Pricing:       price,
			Gate:          headlessGate,
			ContextWindow: ctxWin,
			Compaction:    agent.CompactionConfig{ArchiveDir: config.ArchiveDir()},
			RuntimePrompt: runtimeCtx.SystemPrompt(),
			// V5.30: 根据技能名查找子代理模板 — 同类子代理共享前缀缓存
			TemplatePrefix: lookupSubagentTemplatePrefix(sk.Name),
			// V10.36: 对齐父代理工具集以保证缓存命中
			ActiveSchemas: reg.Schemas(),
		}, agent.NestedSink(sctx, event.Discard), nil)
	}
	reg.Add(skill.NewRunSkillTool(skillStore, skillRunner))
	reg.Add(skill.NewInstallSkillTool(skillStore, nil))
	// V5.30: 注册内置子代理模板，同类子代理共享 L4 前缀缓存
	for _, st := range cache.BuiltinSpawnTemplates() {
		cache.RegisterSpawnTemplate(st)
	}
	for _, t := range skill.BuiltinSubagentTools(skillStore, skillRunner) {
		reg.Add(t)
	}
	// 大文件分块摘要工具：需要注入会话 provider，故在 boot 处注册
	// （对标千问 500 页超长文 / WPS 整本书 / 豆包分段摘要）。
	reg.Add(largefile.NewSummarizeTool(execProv))

	compiler.SetRegistry(reg)

	// Wire the task tool into the compiler so sub-agents inherit the parent's
	// Identity+Context domains via Fork — DeepSeek serves the shared prefix
	// from its server-side cache at near-zero token cost.
	taskTool.SetCompiler(&taskCompilerAdapter{c: compiler})
	// V5.25: 注入 L2 运行时上下文，子代理共享父代理的项目/工作区/目标
	taskTool.SetRuntimePrompt(runtimeCtx.SystemPrompt())

	// V2.4: centralised ToolDispatcher for pre-execution checks.
	toolDispatcher := agent.NewToolDispatcher(headlessGate, hookRunner)

	// V6.0 P8: compact toolset — hide redundant tools from model schema
	if cfg.Tools.Compact {
		applyCompactToolset(reg)
	}

	execSess := agent.NewSession(compiler.SystemPrompt() + "\n\n" + agent.SingleModelPrompt)
	executor := agent.New(execProv, reg, execSess, agent.Options{
		MaxSteps:      maxSteps,
		Temperature:   cfg.Agent.Temperature,
		Pricing:       entry.Price,
		Gate:          headlessGate,
		Hooks:         hookRunner,
		Jobs:          jm,
		ContextWindow: entry.ContextWindow,
		Compaction:    agent.CompactionConfig{ArchiveDir: config.ArchiveDir()},
		Dispatcher:    toolDispatcher,
	}, sink)

	// V7.0: session archive for cross-session Dream/Distill
	archiveDir := filepath.Join(cwd, ".gaea", "archive")
	if ar, err := archive.Open(archiveDir); err == nil && ar != nil {
		sid := filepath.Base(orDefault(opts.SessionDir, config.SessionDir()))
		if sid == "" || sid == "." {
			sid = fmt.Sprintf("session-%d", time.Now().Unix())
		}
		executor.SetArchive(ar, sid)
	}

	// Custom slash commands (.gaea/commands + user dir). Best-effort: a malformed
	// file is skipped, and a load error never blocks the session.
	// 基于工作区根（opts.Cwd）发现命令，与技能/记忆保持一致：桌面端切换
	// 工作空间后，模板与自定义命令跟随新工作区（进程目录不再影响发现）。
	cmds, _ := command.Load(config.CommandDirsAt(cwd)...)

	// Expose the loaded slash commands (skills + custom commands) to the model via
	// the slash_command tool, so it can invoke a project playbook by name the way a
	// user types "/name". Skills are added first, then commands, so a command wins
	// a name clash — matching the prompt's command-over-skill precedence.
	var slashEntries []command.SlashEntry
	for _, sk := range skills {
		sk := sk
		slashEntries = append(slashEntries, command.SlashEntry{
			Name:        sk.Name,
			Description: sk.Description,
		})
	}

	for _, cmd := range cmds {
		cmd := cmd
		slashEntries = append(slashEntries, command.SlashEntry{
			Name:        cmd.Name,
			Description: cmd.Description,
			ArgHint:     cmd.ArgHint,
			Render:      func(args []string) string { return cmd.Render(args) },
		})
	}
	reg.Add(command.NewSlashCommandTool(slashEntries))

	// 前端注入的工具（如桌面端 image_gen）追加到注册表。
	for _, t := range opts.ExtraTools {
		if t != nil {
			reg.Add(t)
		}
	}

	// V10.32: use provider name as label so users can distinguish models from
	// different providers (e.g. "flash" vs "pro") even when they share the same
	// underlying model name (e.g. both "deepseek-chat").
	// 单模型架构：runner 即 executor，不再有 Hermes/Hephaestus 双模型包装。
	label := entry.Name
	runner := executor

	skillLayer := cache.NewSkillLayer()

	// V3.0 Phase 5: ContextManager wraps the four-layer cache kernel.
	ctxMgr := tiancontext.NewContextManager(
		compiler.IdentityLayer(),
		runtimeCtx,
		skillLayer,
		tiancontext.NewFlowLayer(tiancontext.CompactPolicy{
			Window:     entry.ContextWindow,
			TailTokens: 16384,
		}),
	)

	// Wire ContextManager into AgentRunner and ToolDispatcher.
	executor.SetCtxMgr(ctxMgr)

	// V3.4: cache warmup — save L1 hash and check cross-session validity
	cacheDir := filepath.Join(cwd, ".gaea", "cache")
	if warm := compiler.IdentityLayer().LoadAndCompareHash(cacheDir); warm {
		sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo,
			Text: "cache warm: L1 identity matches previous session"})
	}
	compiler.IdentityLayer().SaveHash(cacheDir) // best-effort

	ctrlOpts := control.Options{
		Runner:         runner,
		Executor:       executor,
		Sink:           sink,
		Policy:         policy,
		Label:          label,
		SystemPrompt:   sysPrompt,
		SessionDir:     orDefault(opts.SessionDir, config.SessionDir()),
		Host:           pluginHost,
		Commands:       cmds,
		Skills:         skills,
		Hooks:          hookRunner,
		Memory:         mem,
		MemoryDisabled: !cfg.Memory.Enabled,
		// C4 TimedOut：审批等待超时贯通（0 = 不超时，交互场景默认等待）。
		ApprovalTimeout: time.Duration(cfg.Agent.ApprovalTimeoutSecs) * time.Second,
		Cleanup:        cleanup,
		BalanceURL:     entry.BalanceURL,
		BalanceKey:     entry.APIKey(),
		// 3.0 Wave 4：余额后端 kind 从 ProviderEntry 贯通（空 = controller 按
		// 历史默认 deepseek 形状；配置 balance_kind 即切换，代码零改动）。
		BalanceKind: entry.BalanceKind,
		Jobs:           jm,
		Registry:       reg,
		PluginCtx:      ctx,
		CtxMgr:         ctxMgr,
		WorkspaceRoot:  cwd,
	}
	ctrl := control.New(ctrlOpts)
	// 事件日志 sink 的会话路径只有控制构建完成后才可知（首次对话自动创建 /
	// Resume 注入），故在此注入路径解析器；事件发射发生在 Build 返回之后，
	// 闭包读取的是实时 SessionPath。
	if eventLogSink != nil {
		eventLogSink.SetPathSource(func() string { return ctrl.SessionPath() })
	}
	return ctrl, nil
}

func subagentModelRef(cfg *config.Config, sk skill.Skill) string {
	if cfg != nil {
		for _, key := range subagentModelKeys(sk.Name) {
			if m := strings.TrimSpace(cfg.Agent.SubagentModels[key]); m != "" {
				return m
			}
		}
	}
	if m := strings.TrimSpace(sk.Model); m != "" {
		return m
	}
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Agent.SubagentModel)
}

func subagentModelKeys(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	keys := []string{name}
	for _, alias := range []string{
		strings.ReplaceAll(name, "-", "_"),
		strings.ReplaceAll(name, "_", "-"),
	} {
		if alias == "" {
			continue
		}
		seen := false
		for _, key := range keys {
			if key == alias {
				seen = true
				break
			}
		}
		if !seen {
			keys = append(keys, alias)
		}
	}
	return keys
}

// NewProvider builds the LLM seam provider (provider.LLMProvider) from a
// configured entry. The kind comes from config (gaea.toml [[providers]] kind),
// so switching the chat backend is a config-only change: providers.Register(kind)
// selects the implementation, provider.NewLLM fails closed on an unknown kind,
// and an empty kind defaults to provider.DefaultLLMKind ("wubigrok" — today's
// bridge provider, keeping unconfigured behaviour unchanged). Exported so custom
// assemblers (e.g. the ACP per-session factory) can reuse it without going
// through the full Build.
func NewProvider(e *config.ProviderEntry) (provider.LLMProvider, error) {
	return provider.NewLLM(e.Kind, provider.Config{
		Name:    e.Name,
		BaseURL: e.BaseURL,
		Model:   e.Model,
		APIKey:  e.APIKey(),
		// Pass the key's env var so auth failures can name where to fix it, plus
		// provider-kind-specific knobs (the anthropic provider reads thinking/effort;
		// the openai one ignores them).
		Extra: map[string]any{
			"api_key_env": e.APIKeyEnv,
			"thinking":    e.Thinking,
			"effort":      e.Effort,
		},
	})
}

// addBuiltins adds enabled built-in tools to reg. An empty list means all of
// them. writeRoots confines the file-writing built-ins to the workspace: after
// the (unconfined) defaults are added, each enabled writer is replaced by an
// instance bound to writeRoots (preserving registry order).
func addBuiltins(reg *tool.Registry, dir string, enabled, writeRoots []string, bashSpec sandbox.Spec, proxySpec netclient.ProxySpec, stderr io.Writer) {
	if len(enabled) == 0 {
		for _, t := range tool.Builtins() {
			reg.Add(t)
		}
	} else {
		for _, name := range enabled {
			if t, ok := tool.LookupBuiltin(name); ok {
				reg.Add(t)
			} else {
				fmt.Fprintf(stderr, "warning: unknown built-in tool %q\n", name)
			}
		}
	}
	// Replace the unconfined defaults with confined instances (registry order is
	// preserved on replace): file-writers bound to the workspace, bash to the OS
	// sandbox. Only replace tools actually enabled/present.
	// 桌面端（dir 非空）：基础工具绑定工作空间目录，相对路径基于工作空间而非进程 cwd
	if dir != "" {
		ws := builtin.Workspace{Dir: dir, WriteRoots: writeRoots, Bash: bashSpec, ProxySpec: proxySpec}
		for _, t := range ws.Tools() {
			if _, ok := reg.Get(t.Name()); ok {
				reg.Add(t)
			}
		}
		return
	}
	confined := append(builtin.ConfineWriters(writeRoots), builtin.ConfineBash(bashSpec))
	for _, t := range confined {
		if _, ok := reg.Get(t.Name()); ok {
			reg.Add(t)
		}
	}
}

// PluginSpecs maps configured plugin entries to plugin.Spec, expanding ${VAR}
// references. Exported so custom assemblers can connect the config's plugins
// alongside their own (e.g. ACP's per-session MCP servers).
func PluginSpecs(entries []config.PluginEntry) []plugin.Spec {
	specs := make([]plugin.Spec, len(entries))
	for i, e := range entries {
		e = e.ExpandedPlugin() // resolve ${VAR} / ${VAR:-default} from the environment
		specs[i] = plugin.Spec{
			Name:    e.Name,
			Type:    e.Type,
			Command: e.Command,
			Args:    e.Args,
			Env:     e.Env,
			URL:     e.URL,
			Headers: e.Headers,
		}
	}
	return specs
}

// MCPStartupNotice formats the warning shown when configured MCP servers failed
// to connect, naming the first few; ok is false when none failed.
func MCPStartupNotice(failures []plugin.Failure) (text string, ok bool) {
	if len(failures) == 0 {
		return "", false
	}
	names := make([]string, 0, min(len(failures), 3))
	for i, f := range failures {
		if i >= 3 {
			break
		}
		names = append(names, f.Name)
	}
	more := ""
	if len(failures) > len(names) {
		more = fmt.Sprintf(" (+%d more)", len(failures)-len(names))
	}
	return fmt.Sprintf("%d MCP server(s) failed to start: %s%s — run /mcp for details",
		len(failures), strings.Join(names, ", "), more), true
}

func providerNames(cfg *config.Config) string {
	names := make([]string, len(cfg.Providers))
	for i, p := range cfg.Providers {
		names[i] = p.Name
	}
	return strings.Join(names, "/")
}

// taskCompilerAdapter wraps *cache.Compiler to satisfy agent.TaskCompiler,
// bridging the return-type mismatch between Fork() *cache.Compiler (concrete)
// and Fork() interface{SystemPrompt() string} (interface).
type taskCompilerAdapter struct {
	c *cache.Compiler
}

func (a *taskCompilerAdapter) Fork() interface{ SystemPrompt() string } { return a.c.Fork() }
func (a *taskCompilerAdapter) SystemPrompt() string                     { return a.c.SystemPrompt() }

func orDefault(val, def string) string {
	if val != "" {
		return val
	}
	return def
}

func subagentSkillToTemplateKind(skillName string) cache.TaskKind {
	switch skillName {
	case "explore":
		return "subagent_explore"
	case "research":
		return "subagent_research"
	case "review", "code-review":
		return "subagent_review"
	case "security-review", "security_review":
		return "subagent_security"
	default:
		return ""
	}
}

// lookupSubagentTemplatePrefix 根据技能名称查找子代理模板前缀（V5.30）。
// 同类子代理共享相同模板前缀 → DeepSeek 前缀缓存命中。
func lookupSubagentTemplatePrefix(skillName string) string {
	kind := subagentSkillToTemplateKind(skillName)
	if kind == "" {
		return ""
	}
	tmpl, ok := cache.LookupSpawnTemplate(kind)
	if !ok || tmpl.Prefix == "" {
		return ""
	}
	return tmpl.Prefix
}

// applyCompactToolset hides redundant tools from the model's tool schema list
// while keeping them callable by name. V6.0 P8: reduces visible tool count
// from ~41 to ~25, lowering model cognitive load.
func applyCompactToolset(reg *tool.Registry) {
	// File deletion: merge delete_range + delete_symbol into edit_file
	// edit_file already supports delete via mode parameter
	reg.HideUnlessOnly([]string{"delete_range", "delete_symbol"}, []string{"edit_file"})

	// Batch editing: multi_edit is redundant with multiple edit_file calls
	reg.HideUnlessOnly([]string{"multi_edit"}, []string{"edit_file"})

	// Background job management: merge kill_shell + wait into bash/bgjobs
	reg.HideUnlessOnly([]string{"kill_shell", "wait"}, []string{"bash", "bash_output"})

	// Specialized sub-agents: merge into task with kind parameter
	reg.HideUnlessOnly([]string{"explore", "research", "review", "security_review"}, []string{"task"})

	// Notebook editing: rarely used, hide unless explicitly enabled
	reg.HideUnlessOnly([]string{"notebook_edit"}, []string{"edit_file", "write_file"})

	// File listing: glob is redundant with ls (which supports patterns)
	reg.HideUnlessOnly([]string{"glob"}, []string{"ls"})
}
