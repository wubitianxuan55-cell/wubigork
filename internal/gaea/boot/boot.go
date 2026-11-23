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
	"log/slog"
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
	"github.com/gaea/gaea/internal/gaea/guardian"
	"github.com/gaea/gaea/internal/gaea/hook"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/largefile"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/gaea/plugin"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/sessionquery"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/netclient"
	"github.com/gaea/gaea/internal/office/comhelper"
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
	// Guardian 是动作级自动评审器（codex 蒸馏 T11）：nil=关闭（默认）。
	// app 层按 [guardian] 配置构造（评审调用走办公本地优先路由）后注入。
	Guardian guardian.Reviewer
	// EmitSubagentText / OnFollowUpReady（v4.64 Side Chat 式追问）：前者把
	// 追问运行的文本增量发到前端专用通道；后者接收「追问执行器」——boot 用
	// taskTool 的 continue_from 管道组装，宿主（App）保存并在绑定里调用。
	EmitSubagentText func(ref, text string)
	OnFollowUpReady  func(runner agent.SubagentFollowUpRunner)
	// OnNodeRunnerReady（6.3 文件流水线）：接收「全新子代理运行」执行器
	// （taskTool.RunNew 管道，transcripts 落盘+headless 闸+过滤工具集），
	// 宿主（App）保存后供 DAG 节点执行器调用；emit 语义同 EmitSubagentText。
	OnNodeRunnerReady func(runner agent.SubagentRunRunner)
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
	// WriteRoots 显式接管写入根（2026-11-16 图文 write_file 越界 bug）：
	// 默认走 cfg.WriteRootsAt(cwd)，而它让 cfg.Sandbox.WorkspaceRoot（办公
	// 板块持久化的全局共享值）优先于 Cwd——图文引擎 Cwd=小说目录、配置里
	// workspace_root=办公项目时，write_file 被错误限定到办公目录。非空即
	// 整体接管（含 bash 沙箱与访问策略的写入根），不与 AllowWrite 合并——
	// 接管者自己负责边界。空 = 原行为逐字节（办公/CLI/TUI 不受影响）。
	WriteRoots []string
	// ReadRoots 显式接管额外读边界（2026-11-16 同族审计）：默认
	// cfg.ReadRootsAt(cwd) 含 cfg.Sandbox.WorkspaceRoot——办公持久化目录
	// 会变成图文引擎的额外只读边界（跨板块数据可读）。非空即整体接管
	// （workspace 本身恒可读，与该参数无关；拾取文件走 ReadPathAllowed
	// 回调）。空 = 原行为逐字节。
	ReadRoots []string
	// Feature 账目标签（2026-11-17 审查批5）：非空时注入 provider.Config.Extra
	// ["feature"]，bridge 工厂以此覆盖缺省 "gaea"/"office" 判定——图文引擎传
	// "wenxue"，用量记账 PerFeature 分桶随标签分列（此前 wx 主 agent 用量全落
	// "gaea" 桶）。空 = 现状逐字节（办公/CLI/TUI 不受影响）。
	Feature string
	// ExecBoard 是执行纪律板块缝（2026-11-18 众神殿 pan_1795007052552 v1.0
	// 冻结包）：非空 "wenxue" = 图文引擎——执行纪律换 agent.
	// WenxueSingleModelPrompt（独立常量，无空间渲染：wenxue_* 工具族恒 work
	// 锚，play 下 AllowsSpace 全滤，渲染函数是死代码），并跳过 officeLive 段
	// （COM 实时协同是办公域分工，图文无 office_ 工具）。中立交付段
	// （deliveryPromptSection）仍单源追加（用户拍板 q341：交付链接规则留装配
	// 层，不复制进两份正文）。空 = 现状逐字节（办公/CLI/TUI 用
	// SingleModelPromptFor 双态渲染 + officeLive 探测追加）。
	ExecBoard string
	// SpaceAnchor 是装配空间锚定（2026-11-18 众神殿余项·文学 play 装配洞
	// 根修）：非空时顶替 cfg.EffectiveSessionSpace() 作为本引擎的工具过滤/
	// 记忆读端/space_profiles/权限规则的生效空间。文学引擎传 "work"——
	// wenxue_* 工具族与基础读写工具的 SpaceTag 恒 work、SessionDir 平铺、
	// Journal 与账本在书内 .gaea/work，全局 session.space=play 时不锚定则
	// AllowsSpace 把它们全部滤掉（文学会话零写工具的装配洞；修复的前置
	// 是产品要求文学进 play 时 SpaceTag/SessionDir/账本三处同改，本锚是
	// 「文学恒 work」裁决的装配层兑现）。空 = 现状逐字节（办公/CLI/TUI
	// 跟随全局 session.space 双态）。
	SpaceAnchor string
	// ExtraTools 是前端（桌面端）额外注入的工具（如 image_gen 这类需要
	// 应用服务/客户端的工具）。空 = 不注入；与内置工具同名会被覆盖。
	ExtraTools []tool.Tool
	// MorningPreload 是晨报预载开关（v4.16 刀④）：work 空间会话装配时把
	// 高频工作记忆确定性聚合为「晨报预载」块预装配进系统提示词（零 LLM、
	// 预算受限、work 只读）。桌面端从 ~/.gaea_config.json 的 morning_preload
	// 键读取传入（默认开）；零值 false = 不注入（CLI/TUI 等不读该配置的
	// 前端维持原行为，无晨报预载）。
	MorningPreload bool
	// ProjectBrief 是项目本体注入开关（6.2）：work 空间会话装配时把固化/
	// 项目/反馈决策带 [MEM:] 引用键注入系统提示词（跨会话项目连续性）。
	// 桌面端从 ~/.gaea_config.json 的 project_brief 键读取传入（默认开）；
	// 零值 false = 不注入（CLI/TUI 维持原行为）。
	ProjectBrief bool
	// OnSkillUse 是技能调用计数回调（7.2-2 判据②）：read_skill 按需加载
	// 与 run_skill 管线执行各汇报一次，ok=工具级成功。空 = 不计数
	// （CLI/TUI 宿主不传不受影响——EmitSubagentText 同款先例）。
	OnSkillUse func(name string, ok bool)
	// ReadPathAllowed is the host's explicit-read allowlist callback (desktop
	// file picker selections). It extends reads only; writes always use config.
	ReadPathAllowed func(path string) bool
	// ToolFilter 是装配期工具过滤谓词（2026-11-15 刀1，图文板块专属工具集）：
	// 非空时对装配进注册表的每个工具先问一次，返回 false = 物理不装。
	// 作用点与 space 过滤同三处：addBuiltins、startPlugins（MCP）、ExtraTools；
	// 会话级元工具（task/ask/记忆族等 build-internal 直加面）不过滤——deny
	// 名单只收办公域工具，不含元工具。nil = 现状逐字节（办公引擎不传）。
	ToolFilter func(name string) bool
	// SkillFilter / SkillExtras（2026-11-15 刀2，图文板块技能子集）：filter
	// 非空时技能索引/slash 条目/run_skill/ctrl.Skills/read_skill 只留放行
	// 名单（deny-by-absence，未过闸名如实 not found）；extras 追加宿主内嵌
	// 技能（图文内嵌：wenxue-workbench）。
	// 双 nil = 现状逐字节（办公引擎不传）。
	SkillFilter func(name string) bool
	SkillExtras []skill.Skill
	// SkillFilterKeepsInstall（2026-11-21 办公工具面组开关，pan_1795224680124
	// 终案）：office 的 SkillFilter 是窄 deny 谓词（仅拒 schedule-edit/
	// cad-takeoff 两件，跟随组开关），与 wenxue 全量 allowlist 语义不同——
	// 摘 install_skill 会让精简面丢造技能元工具（真实回归）。true 时
	// install_skill 照常注册（新装技能进 run_skill 快照要等重建/GaeaReload，
	// 与 extras 路径既有语义一致）。零值 false = wenxue 现状逐字节。
	SkillFilterKeepsInstall bool
	// MemoryUserDir 是记忆脑目录覆盖（2026-11-16 记忆左右脑批）：非空时本引擎
	// 的全部记忆面——用户级 doc 记忆/画像文档发现、auto-memory SQLite 后端、
	// remember 画像表、会话检索 FTS、引用消毒 store——都锚定该目录（DB 路径
	// 随目录推导，db.GetDatabase 按目录分池），与默认脑物理隔离；控制器记忆
	// 热重载复用首载快照的 UserDir/DB，全链跟随。空值 = config.MemoryUserDir()
	// 现状逐字节（办公引擎不传）。
	MemoryUserDir string
	// SystemPrompt 是板块身份提示词覆写（2026-11-16 图文身份分板）：非空时
	// 顶替 cfg.ResolveSystemPrompt() 作为 L1 基底（部署级 prefix/suffix、语言
	// 策略、记忆、技能索引等装配块照常叠加）。板块引擎传自己的身份（图文走
	// cfg.ResolveWenxueSystemPrompt()），不再吃办公 system_prompt。空 = 现状
	// 逐字节（办公/CLI/TUI 不传）。
	SystemPrompt string
	// ToolHide 是装配完成后按名 Registry.Hide 的名单（2026-11-21 办公运行模式，
	// 众神殿 pan_1795202763988 终案）：schema 不可见但协议层按名可调（Registry
	// 既有 hidden 语义）。应用点在全部装配（builtins/plugins/extras）之后、
	// 首回合之前——主执行器每回合活读 reg.Schemas()，boot 期 Hide 即会话全程
	// 稳定（前缀缓存字节稳定）。名单里有未注册名是 caller 的对账责任
	// （app 层守卫测试钉死），boot 侧只对已注册名生效。空 = 现状逐字节。
	ToolHide []string
	// ModeLine 是 L1 运行模式行（同上终案）：非空时在技能索引之后追加固定
	// 文本「当前模式 + 本档未载入能力组 + 切档指引」。DSH-r5 裁决：无条件
	// Hide 与模式提示行必须同批上线，缺一即回到「能力无声消失」（boot.go
	// tools.deferred 同族自警）。空 = 零注入、现状逐字节。
	ModeLine string
}

// Build loads config, resolves the model, and returns a Controller wrapping a
// single Agent (single-model architecture; the two-model Hermes wrapper was
// removed). The returned controller owns plugin subprocesses; call Close (via
// Controller.Close) to release them.
var (
	sandboxWarnOnce sync.Once
	bashWarnOnce    sync.Once
)

// resolveRoots 合成生效根（2026-11-16 图文 write_file 越界根修的纯函数面）：
// Options.WriteRoots/ReadRoots 非空即整体接管（不与 AllowWrite/AllowRead
// 合并——接管者自己负责边界），空走配置推导 WriteRootsAt/ReadRootsAt。
// 单测在 boot_roots_test.go 锁组合语义。
func resolveRoots(cfg *config.Config, cwd string, opts Options) (writeRoots, readRoots []string) {
	writeRoots = cfg.WriteRootsAt(cwd)
	if len(opts.WriteRoots) > 0 {
		writeRoots = opts.WriteRoots
	}
	readRoots = cfg.ReadRootsAt(cwd)
	if len(opts.ReadRoots) > 0 {
		readRoots = opts.ReadRoots
	}
	return writeRoots, readRoots
}

func Build(ctx context.Context, opts Options) (*control.Controller, error) {
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	// S1.3/S1.5 装配空间：session.space 的配置生效值（"work"/"play"）；
	// ""=space.mode=off（整体回退：不滤工具、模型与权限走现状路径）。
	// 与桌面端 gaeaBuildController 的 SessionDir 空间同源（同一 cfg），因此
	// GaeaSpaceActivate 的「重建生效」语义对工具/模型/权限装配一致成立。
	// SpaceAnchor 非空时顶替（板块引擎恒定空间，见 Options.SpaceAnchor）。
	space := cfg.EffectiveSessionSpace()
	if opts.SpaceAnchor != "" {
		space = opts.SpaceAnchor
	}
	// 记忆脑目录（2026-11-16 左右脑批）：effective 一处算好，四处消费点
	//（sysprompt 主 Load / sessionquery / remember ProfileStore / 引用消毒）
	// 统一取它——空值=现状 config.MemoryUserDir() 逐字节。
	memUserDir := orDefault(opts.MemoryUserDir, config.MemoryUserDir())
	modelName := opts.Model
	if modelName == "" {
		modelName = cfg.DefaultModel
	}
	entry, ok := cfg.ResolveModel(modelName)
	if !ok {
		return nil, fmt.Errorf("unknown model %q (configured: %s)", modelName, providerNames(cfg))
	}
	// S1.3-A：按空间取模型 profile（缺省零值=现状逐字节回退）。控制器按空间
	// 装配时，profile 的 gaea 键（办公 agent 功能域，对应既有 GetFeatureModel
	// 键体系）优先——经 ResolveModel 既有链解析为 provider entry，BalanceKind/
	// BalanceURL/usage 随所选 entry 自然分账（billing 不改）。桌面端 play 域
	// 模型走既有 bridge_feature 功能绑定（零新增绑定），此处只在 gaea.toml
	// 配置世界解析；引用不可解析时告警并回退现状，不阻断装配。
	if space != "" {
		if prof, perr := cfg.SpaceProfile(space); perr != nil {
			_, _ = fmt.Fprintf(stderr, "warning: space_profiles.%s: %v（现状模型继续生效）\n", space, perr)
		} else if ref := strings.TrimSpace(prof.Gaea); ref != "" {
			if e, ok := cfg.ResolveModel(ref); ok {
				entry = e
				modelName = ref
			} else {
				_, _ = fmt.Fprintf(stderr, "warning: space_profiles.%s.gaea = %q 无法解析（现状模型继续生效）\n", space, ref)
			}
		}
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

	// v4.61 子代理/本地模型工具 transcript 落盘：目录随控制器当前会话惰性解析
	// （与 eventLogSink 的 SetPathSource 同一时序问题——会话路径在控制构建完成
	// 后才可知）。闭包在真正写盘（回合内派发子代理/执行模型工具）时才执行，
	// ctrlSessionPath 彼时必已赋值。
	var ctrlSessionPath func() string
	subagentStore := agent.NewLazySubagentStore(func() string {
		if ctrlSessionPath == nil {
			return ""
		}
		sp := ctrlSessionPath()
		if sp == "" {
			return ""
		}
		return filepath.Join(filepath.Dir(filepath.Clean(sp)), "subagents")
	})
	// 会话隔离（2026-11-18 实弹缺陷刀）：subagents/ 目录按会话目录共享，清单
	// 不过滤会把「上一个会话的编队」串进来——运行按 meta.SessionPath 严格归
	// 属当前会话（盖章同源 ctrlSessionPath）。
	subagentStore.SetSessionPathFn(func() string {
		if ctrlSessionPath == nil {
			return ""
		}
		return ctrlSessionPath()
	})

	// effort 覆盖必须先于 NewProvider：NewProvider 构造即把 entry.Effort 逐值
	// 拷进 provider.Config.Extra，构造后再赋值是静默失效（GA4-01）。与下方子
	// 代理路径（先赋 SubagentEffort 再 NewProvider）同序。
	if cfg.Agent.Effort != "" {
		entry.Effort = cfg.Agent.Effort
	}
	execProv, err := newProviderWithFeature(entry, opts.Feature)
	if err != nil {
		return nil, err
	}

	cwd := opts.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// V10.22: system prompt + memory + skills assembled in sysprompt.go
	// 传入工作空间根（opts.Cwd），项目画像/技能索引基于真实工作区而非进程目录。
	// v4.5.1a 红线补课：记忆注入侧按装配空间收窄（space=""=mode=off 旧行为）。
	// sp, err := buildSystemPrompt(...) 参数链见该函数注释；ModeLine 是办公
	// 运行模式行（2026-11-21 众神殿终案，非空时技能索引后追加；空=零注入）。
	sp, err := buildSystemPrompt(cfg, cwd, space, opts.MorningPreload, opts.ProjectBrief, opts.Stderr, opts.OnSkillUse, opts.SkillFilter, opts.SkillExtras, memUserDir, opts.SystemPrompt, opts.ModeLine)
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
	writeRoots, readRoots := resolveRoots(cfg, cwd, opts)
	bashSpec := sandbox.Spec{Mode: cfg.BashMode(), WriteRoots: writeRoots, Network: cfg.Sandbox.Network, FailClosed: cfg.Sandbox.FailClosed}
	accessPolicy := tool.NewPathAccessPolicy(cwd, readRoots, writeRoots, opts.ReadPathAllowed)
	if bashSpec.Mode == "enforce" && !sandbox.Available() {
		sandboxWarnOnce.Do(func() {
			_, _ = fmt.Fprintln(stderr, "warning: bash sandbox requested but unavailable on this platform; running bash unconfined")
		})
	}
	if sandbox.ResolveShell().Kind == sandbox.ShellPowerShell {
		bashWarnOnce.Do(func() {
			_, _ = fmt.Fprintln(stderr, "warning: bash not found on PATH; the shell tool will run commands under Windows PowerShell. Install Git for Windows or WSL to use bash.")
		})
	}
	// 电脑操控（施工单 W3，2026-11-20）：[computer_use] enabled=true 才注册
	// 只读三件——默认关=Registry.Schemas() 与现状逐字节一致；输入注入族 W5。
	if cfg.ComputerUse.Enabled {
		builtin.RegisterComputerTools(cfg.ComputerUse.Apps, time.Duration(cfg.ComputerUse.IdleTTLSeconds)*time.Second)
	}
	addBuiltins(reg, cwd, cfg.Tools.Enabled, writeRoots, bashSpec, cfg.NetworkProxySpec(), stderr, space, opts.ToolFilter)
	// 2026-11-15 刀2：read_skill 实例化注册（替换内置全局同名件——Registry.Add
	// 同名覆盖语义）。resolver 走 sysprompt 产出的过滤闭包：双引擎（办公+图文）
	// 各持各的技能面，不再经全局 WireReadSkillResolver 串仓。
	addGated(reg, opts.ToolFilter, builtin.NewReadSkillWithResolver(sp.readSkill))
	// Always construct a host, even with no plugins configured, so the controller's
	// host pointer is stable for the session and `/mcp add` can hot-add into it.
	// V10.22: plugins + LSP assembled in plugins.go
	po := startPlugins(ctx, cfg, reg, sink, opts.Stderr, space, opts.ToolFilter)
	pluginHost := po.host
	cleanup := po.cleanup
	// 事件日志 sink 的写入器随控制器生命周期释放（Ctrl+Close / 重建换实例时
	// 关闭文件句柄；回合边界已关闭的写入器 Close 幂等）。缺省 event 模式下
	// 若漏关，Windows 上会话目录无法删除/迁移。
	ctrlCleanup := cleanup
	if eventLogSink != nil {
		ctrlCleanup = func() {
			if cleanup != nil {
				cleanup()
			}
			_ = eventLogSink.Close()
		}
	}
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
	// S1.5-A：权限策略按空间装配——space 为空（mode=off）或未配置段时
	// PermissionsForSpace 逐字段回退顶层 [permissions]（现状单 Policy）；play
	// 产品默认 mode=allow + hard_ask 空集（不弹审批卡），deny 硬拒绝仍生效。
	perm := cfg.PermissionsForSpace(space)
	policy := permission.New(perm.Mode, perm.Allow, perm.Ask, perm.Deny)
	headlessGate := permission.NewGate(policy, nil)
	// ⑤ deny-unapproved（2026-11-08 众神殿结论第五步）：默认 false=自治放行
	// （gaea run 既有语义不破坏）；配置 deny_unapproved=true 时，无人值守遇
	// ask/hard_ask 即拒绝——执行链不能靠「没人守门」获得写面。拒绝同样落账。
	headlessGate.DenyUnapproved = perm.DenyUnapproved
	// T4：审批模式归一（[permissions].approval_mode，"never"=Ask 硬拒给理由）。
	headlessGate.ApprovalMode = perm.ApprovalMode
	// 无人值守放行也要留痕：子代理、后台任务、headless 都没有 UI 可问，遇
	// 「本该问人」的工具时按自治放行——账上写一条 approval_auto，Source 记清
	// 是闸门的 ask 还是 hard_ask，Decided=headless。共用 sink 已串到会话事件
	// 日志（上面的 eventLogSink），所以它和交互路径的卡落在同一本账里。
	headlessGate.OnAutoAllow = func(toolName, subject, source string) {
		sink.Emit(event.Event{Kind: event.ApprovalAuto, Auto: event.AutoApproval{
			Tool:    toolName,
			Subject: subject,
			Source:  source,
			Decided: event.AutoDecidedHeadless,
		}})
	}
	headlessGate.OnUnapprovedDeny = func(toolName, subject, source string) {
		sink.Emit(event.Event{Kind: event.ApprovalAuto, Auto: event.AutoApproval{
			Tool:    toolName,
			Subject: subject,
			Source:  source,
			Decided: event.AutoDecidedUnapprovedDeny,
		}})
	}

	// P8 断网闸（codex 蒸馏 B6 折算）：[sandbox] network=false 时，网络能力
	// 工具（web/浏览器/mcp__ 动态工具）在闸门层拒绝——此前断网只关 bash
	// 子进程，web/浏览器照常出网是半开沙箱。network=true（默认）零介入。
	// 刀1：headless 无 UI，只挂只读台账（持久清单命中放行，其余照旧拒绝）。
	gatedHeadless := agent.WrapNetworkGateWith(headlessGate, cfg.Sandbox.Network,
		agent.NetworkLedgerOnly(cfg.Network.GateHostAllow))

	// Hooks: load the global settings.json plus the project's (only when trusted —
	// project hooks run arbitrary shell commands, so cloning a repo must not
	// silently execute them). Non-blocking hook output is surfaced to the user as
	// a Notice through the shared sink. The runner fires PreToolUse/PostToolUse in
	// the agent loop and UserPromptSubmit/Stop at the controller's turn boundary.
	hooksTrusted := hook.IsTrusted(cwd, "")
	// GA4-06：hook 命令沿用与 bash 工具同源的 bashSpec，装配到 Runner 上（唯一
	// 注入点在其 spawn 包装层，事件路径无需逐个填）。mode=enforce 而平台不可用时
	// hook 侧经 notify 发一次性诚实警告（自持 sync.Once，不抢本包 sandboxWarnOnce：
	// 本包那次讲的是 bash，hook 有自己的用户可见通道）。未 enforce 时 Runner 让
	// SpawnInput.Sandbox 保持 nil，与既有行为逐字节一致。
	hookRunner := hook.NewRunner(
		hook.Load(hook.LoadOptions{ProjectRoot: cwd, Trusted: hooksTrusted}),
		cwd, nil,
		func(msg string) { sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: msg}) },
	).WithSandbox(bashSpec)
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
	// v4.221 子代理证据落账：子代理写盘也进证据链（SessionID=sa_ ref，按会话
	// 分文件）——文件工作台版本时间线/回滚与 DAG 节点产物归因由此供给。与主
	// 执行器 JournalDir 同目录（下方 Options.JournalDir 同源路径）。
	taskTool.SetSubagentJournalDir(filepath.Join(cwd, ".gaea", "work", "journal"))
	// 出发快照的模型身份（B1 蒸馏）：缺省=父条目；下方显式子代理模型分支
	// 会用子条目覆盖。按**实际选中**的配置条目注入，不用 provider.Name()。
	taskTool.SetSubagentIdentity(entry.Model, entry.Effort)
	taskTool.WithHooksForSession(func(sessionID string) agent.ToolHooks {
		return hookRunner.WithSessionID(sessionID)
	})
	taskTool.WithAccessPolicy(accessPolicy)
	// v4.379 泄洪随父配置：子代理 runner 各持独立泄洪库（read_spill 同注册表可达）。
	taskTool.SetSpill(cfg.Agent.ToolSpillEnabled())
	// 有限并行闸（2026-10）：task/run_skill 前台派生共享的并发上限，超出的
	// 调用排队（queued 侧车可见）。≤0/超帽在 Set 内钳制，boot 一次装配。
	agent.SetSubagentMaxParallel(cfg.Agent.SubagentMaxParallel)
	// 嵌套委派深度上限（2026-11-18 dsh 完整智能体形态）：0/未配置=缺省 3
	// （主→子→孙）；1=关闭（平面工兵现行为）；硬帽 10。嵌套层槽满快拒不
	// 排队（防自锁，subagent_limit.go 头注）。
	taskTool.SetSubagentMaxDepth(cfg.Agent.SubagentMaxDepthVal())
	// 按调用路由授权清单（2026-11，dsh model-selection 蒸馏 P2）：空=路由
	// 关闭（task engine/model 参数一律拒绝）；非空=逐字核验+CompactDescription
	// 动态列出授权路由（发现面）。
	agent.SetSubagentRoutes(cfg.Agent.SubagentRoutes)
	// 路由 provider 构造器：wubigrok bridge 显式 Engine+Model（批5 真机验证
	// 同款构造式）。窗口/计价继承父（引擎无窗口元数据→K1 溢出自学习兜底）。
	taskTool.SetSubagentRouteFactory(func(engine, model string) (provider.LLMProvider, error) {
		return provider.NewLLM(provider.LLMKindWubigrok, provider.Config{
			Name:   "task-route",
			Engine: engine,
			Model:  model,
		})
	})

	// V10.22: resolve subagent model from config. When SubagentModel names a
	// configured provider, sub-agents use that provider (typically a cheaper
	// flash model) while the parent keeps its own provider — independent API
	// calls mean the parent's prefix cache is unaffected.
	if subRef := strings.TrimSpace(cfg.Agent.SubagentModel); subRef != "" {
		if subEntry, ok := cfg.ResolveModel(subRef); ok {
			if e := cfg.Agent.SubagentEffortVal(); e != "" {
				subEntry.Effort = e
			}
			if subProv, err := newProviderWithFeature(subEntry, opts.Feature); err == nil {
				taskTool.SetSubagentProvider(subProv, subEntry.Price, subEntry.ContextWindow)
				// 子代理模型显式命名时快照身份跟着换（实际选中的条目）。
				taskTool.SetSubagentIdentity(subEntry.Model, subEntry.Effort)
			}
		}
	}
	addGated(reg, opts.ToolFilter, taskTool)
	// interrupt_agent（2026-10 P0-A，dsh 控制面蒸馏）：模型面中断——在跑/排队
	// 子代理经 subRuns 句柄取消，后台任务转发 jobs.Kill。已入 subagentMetaTools
	// （子代理不见此工具，中断权只属父代理）。
	addGated(reg, opts.ToolFilter, agent.NewInterruptTool(subagentStore))
	// subagent_list（2026-10 P1-D）：模型面枚举当前会话子代理运行（lazy store
	// 跟随会话路径），为 interrupt_agent / continue_from 提供 ref 入口。
	addGated(reg, opts.ToolFilter, agent.NewSubagentListTool(subagentStore))
	// send_message（2026-11-18 刀2，DSH tool-subagent-control send_message 对
	// 齐）：统一传话通道——running steer / queued 排队 / 终态 followup 三态
	// 自动路由。followUp 复用 EmitSubagentText 闭包（FollowUpSink 专用流式
	// 通道，与用户追问同实现）；nil = 终态分支报可执行文案。
	// read_subagent_result 模型面同批退役（结果自动回投；内部 API 保留给 UI）。
	var msgFollowUp agent.SubagentFollowUpRunner
	if opts.EmitSubagentText != nil {
		emitText := opts.EmitSubagentText
		msgFollowUp = func(ctx context.Context, ref, prompt string) error {
			sink := agent.FollowUpSink(ref, func(text string) { emitText(ref, text) })
			return taskTool.RunFollowUp(ctx, ref, prompt, sink)
		}
	}
	addGated(reg, opts.ToolFilter, agent.NewSendMessageTool(subagentStore, msgFollowUp))
	// v4.384 会话检索（dsh ⑧ session-query 蒸馏首刀）：过往会话 user/assistant
	// 正文的 FTS5 索引（绑定装配空间会话目录——空间隔离由目录构造），模型经
	// session_search 检索跨会话记忆；搜索时惰性增量索引，零 boot 成本。
	if gdb := db.GetDatabase(memUserDir); gdb != nil {
		addGated(reg, opts.ToolFilter, sessionquery.NewSearchTool(gdb, orDefault(opts.SessionDir, config.SessionDir())))
	}
	// v4.61：真实引擎接线——task 工具派发的子代理从现在起真正落盘 transcript
	// （此前 WithTranscripts 只有测试调用，真机子代理 tab 无数据可显示）。
	taskTool.WithTranscripts(subagentStore)
	// P1-7（2026-11-21 沿江反馈）：wait 收 sa_/mt_ 引用——agent.WaitSubagents
	// 经注入缝进 builtin wait（builtin 不能反向 import agent，环约束）。
	builtin.SetSubagentWaiter(func(ctx context.Context, refs []string, timeoutSec int) []builtin.SubagentWaitRow {
		var out []builtin.SubagentWaitRow
		for _, r := range agent.WaitSubagents(ctx, refs, time.Duration(timeoutSec)*time.Second) {
			out = append(out, builtin.SubagentWaitRow{Ref: r.Ref, Status: r.Status, Output: r.Output})
		}
		return out
	})

	// v4.64 Side Chat 式追问：把「用户在子代理 tab 里追问」的执行器交给宿主
	// （App 绑定 GaeaSubagentFollowUp）。事件只走专用流式通道（文本增量），
	// 权威内容落子代理自身 transcript（~1s 快照），不进主对话账本避免刷屏；
	// 追问完成态由 meta 侧车承载（任务树/tab 状态点经轮询自校正）。
	if opts.EmitSubagentText != nil {
		emitText := opts.EmitSubagentText
		followUp := func(ctx context.Context, ref, prompt string) error {
			sink := agent.FollowUpSink(ref, func(text string) { emitText(ref, text) })
			return taskTool.RunFollowUp(ctx, ref, prompt, sink)
		}
		if opts.OnFollowUpReady != nil {
			opts.OnFollowUpReady(followUp)
		}
	}
	// 6.3 文件流水线：全新子代理执行器交给宿主（App 侧 DAG 节点执行器）。
	// 不依赖 EmitSubagentText——emit 由调用方逐次传入，nil 走 Discard。
	if opts.OnNodeRunnerReady != nil {
		opts.OnNodeRunnerReady(taskTool.RunNew)
	}

	// The `remember` tool lets the model persist durable facts to the project's
	// auto-memory store; `forget` prunes ones that turn out wrong. The saved index
	// loads into the prefix on the next session.
	addGated(reg, opts.ToolFilter, memory.NewRememberTool(mem.Store, memory.NewProfileStore(db.GetDatabase(memUserDir))))
	addGated(reg, opts.ToolFilter, memory.NewForgetTool(mem.Store))
	addGated(reg, opts.ToolFilter, memory.NewPromoteSessionFactsTool())
	addGated(reg, opts.ToolFilter, memory.NewMemoryGetTool(mem.Store))

	// The `ask` tool puts structured multiple-choice questions to the user. It
	// reaches them through the Asker on the call context, which interactive
	// frontends wire to the controller (EnableInteractiveApproval); a headless run
	// has none, so ask resolves to "decide for yourself".
	addGated(reg, opts.ToolFilter, agent.NewAskTool())

	// The `request_permission` tool lets the model explicitly request a
	// permission rule ("Tool" / "Tool(subject)") when it lacks one, surfacing a
	// permission-escalation card through the same approval channel (对齐 codex
	// request_permissions_for_environment 语义族). 与 ask 同路径直接注册：这是
	// 会话级元工具（shared 语义），不经 spacetags 装配过滤——work/play 两空间
	// 均可见；批准授予的是规则，真实调用仍走正常权限闸门。headless 运行无
	// PermissionRequester，工具返回非交互降级结果，不阻塞自治。
	addGated(reg, opts.ToolFilter, agent.NewRequestPermissionTool())

	// exit_plan_mode 计划审批门（v4.414，dsh plan-mode 蒸馏取道不取器）：恒
	// 注册（工具目录跨模式切换稳定），计划模式（/plan on）下把 markdown 计划
	// 提交 ask 通道审批；非计划模式调用如实报错。与 ask 同路径直接注册（会话
	// 级元工具，work/play 均可见）；子代理经 SubagentMetaTools 排除。headless
	// 无 Asker 时诚实拒绝（执行闸必须人来开，不仿 ask 的自选降级）。
	addGated(reg, opts.ToolFilter, agent.NewExitPlanModeTool())

	// Skill tools: run_skill / install_skill are the ONLY skill entry points —
	// 子代理入口收敛为 task + run_skill 两级（v4.61 用户拍板，对标 Codex 的
	// 「父代理发一条任务消息」）。此前为 format-convert/chart-builder/
	// doc-assemble 注册的顶层包装工具已删除；它们仍是 runAs=subagent 技能，
	// 经 run_skill（或 slash_command）按名调用。explore/research/review/
	// security_review 同样不作为独立工具面：仅保留 spawn 模板前缀（同类
	// run_skill 技能共享 L4 前缀缓存）与每技能模型覆盖。run_skill 子代理
	// 复用同一 runner —— 隔离循环 + 技能正文 + 按技能白名单的工具集（减去
	// task/skill 元工具防递归）+ 可选 per-skill 模型；工具活动嵌套在调用下。
	skillRunner := func(sctx context.Context, sk skill.Skill, task string) (string, error) {
		// S3 双空间：技能子代理与 task 子代理同规则继承父会话空间。正常链路
		// sctx 已由 agent.Run 注入空间（run_skill 工具的调用 ctx 原样透传，
		// 并行技能的 WithTimeout 派生也保留 value）；此处显式落定为防御点，
		// 保证 ctx 链被中间层重建（job 包裹等）时子代理空间仍确定（缺省 work），
		// 与 RunSubAgent 内部的 fail-closed 断言构成双重保险。
		sctx = agent.WithSpace(sctx, agent.SpaceFromContext(sctx))
		prov, price, ctxWin := execProv, entry.Price, entry.ContextWindow
		if modelRef := subagentModelRef(cfg, sk); modelRef != "" {
			if me, ok := cfg.ResolveModel(modelRef); ok {
				if p, err := newProviderWithFeature(me, opts.Feature); err == nil {
					prov, price, ctxWin = p, me.Price, me.ContextWindow
				}
			}
		}
		subReg := agent.FilterRegistry(reg, sk.AllowedTools, agent.SubagentMetaTools()...)
		// dsh toolFilter loud-unknown 纪律的技能面（2026-10 P0-B）：技能清单是
		// 人写的——错名不炸技能（部署差异下清单可以引用本机没有的工具），但
		// 响亮告警让维护者看得见，不再无声少工具。
		if unknown := agent.UnknownToolNames(reg, sk.AllowedTools, agent.SubagentMetaTools()...); len(unknown) > 0 {
			slog.Warn("skill requests unknown tools; skipped", "skill", sk.Name, "unknown", strings.Join(unknown, ","))
		}
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

		// v4.61：技能子代理与 task 子代理同等待遇——真实引擎落盘 transcript
		// （sa_ meta/jsonl），title 用任务本体而非 sk.Body+task 拼合串，UI 标题
		// 不以技能正文开头。
		return agent.RunPersistedSubAgent(sctx, prov, subReg, sysPrompt, sk.Body+"\n\n"+task, task, agent.Options{
			MaxSteps:    steps,
			Temperature: cfg.Agent.Temperature,
			Pricing:     price,
			Gate:        gatedHeadless,
			HooksForSession: func(sessionID string) agent.ToolHooks {
				return hookRunner.WithSessionID(sessionID)
			},
			AccessPolicy:  accessPolicy,
			ContextWindow: ctxWin,
			Compaction:    compactionConfig(cfg),
			RuntimePrompt: runtimeCtx.SystemPrompt(),
			// V5.30: 根据技能名查找子代理模板 — 同类子代理共享前缀缓存
			TemplatePrefix: lookupSubagentTemplatePrefix(sk.Name),
			// V10.36: 对齐父代理工具集以保证缓存命中
			ActiveSchemas: reg.Schemas(),
		}, agent.NestedSink(sctx, event.Discard), subagentStore, nil)
	}
	// run_skill 解析源（2026-11-15 众神殿首刀，pan_1794754879290 裁决）：板块
	// 引擎（SkillFilter/SkillExtras 任一非空）换用与 read_skill 同源的可见集合
	// ——extras 内嵌件可达、被 filter 拒名 fail-closed、unknown 提示名单只列
	// 可见；办公引擎（双 nil）保持 store 活读现状逐字节（红线）。
	var runSkill tool.Tool
	if opts.SkillFilter != nil || len(opts.SkillExtras) > 0 {
		runSkill = skill.NewRunSkillToolFromSet(sp.skills, skillRunner)
	} else {
		runSkill = skill.NewRunSkillTool(skillStore, skillRunner)
	}
	if opts.OnSkillUse != nil {
		// 7.2-2 判据②：run_skill 执行结果计入调用统计（args.name 解析失败
		// 不计——宁少勿扰）。装饰器只包 Execute，其余方法原样转发。
		runSkill = &skillUseCountTool{Tool: runSkill, onUse: opts.OnSkillUse}
	}
	addGated(reg, opts.ToolFilter, runSkill)
	// install_skill 靠活读 store 生效；静态 allowlist 引擎（SkillFilter!=nil）
	// 里新装技能永不可见不可 run（run_skill 走构建期快照集）——不装，免噪音
	// （2026-11-17 审查批3；办公双 nil 保持注册=现状逐字节红线）。
	// 例外：SkillFilterKeepsInstall（2026-11-21 办公组开关窄 deny 谓词）——
	// 办公面 filter 只拒两件领域技能，install_skill 照常注册。
	if opts.SkillFilter == nil || opts.SkillFilterKeepsInstall {
		addGated(reg, opts.ToolFilter, skill.NewInstallSkillTool(skillStore, nil))
	}
	// V5.30: 注册内置子代理模板，同类子代理共享 L4 前缀缓存
	for _, st := range cache.BuiltinSpawnTemplates() {
		cache.RegisterSpawnTemplate(st)
	}
	// 大文件分块摘要工具：需要注入会话 provider，故在 boot 处注册
	// （对标千问 500 页超长文 / WPS 整本书 / 豆包分段摘要）。
	addGated(reg, opts.ToolFilter, largefile.NewSummarizeToolAt(execProv, cwd))

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

	// T5（codex 蒸馏）：工具 schema 字节预算——默认关闭；开启后描述文本按
	// 共享预算截断（名字优先，收口在 Registry.FilteredSchemas 与 identity
	// 具名路径两处，同一函数）。Bytes<=0 走 tool 包内保守默认 12KiB。
	// 装配期一次性注入，会话中途改预算会打破前缀缓存稳定性。
	if cfg.Tools.SchemaBudget.Enabled {
		reg.SetSchemaBudget(cfg.Tools.SchemaBudget.Bytes)
	}

	// T8（codex 蒸馏）：检索展开——deferred 名单工具隐藏出首轮 schema（藏而
	// 可按名调用），tool_search 注册后模型按查询取回 schema 文本。只在
	// search.enabled=true 时隐藏（否则工具会无声消失）；两者默认关 = 现状。
	if cfg.Tools.Search.Enabled {
		for _, name := range cfg.Tools.Deferred {
			if _, ok := reg.Get(name); ok {
				reg.Hide(name)
			} else {
				slog.Warn("tools.deferred 名单含未注册工具", "name", name)
			}
		}
		reg.Add(tool.NewToolSearchTool(reg, cfg.Tools.Search.Limit))
	}

	// S4 双空间：执行纪律文本按会话空间渲染 exports/work 路径根段（设计 §5）。
	// work 缺省（含 space.mode=off 的 "" 生效值）输出与历史 const 逐字一致。
	// 板块缝（2026-11-18 众神殿 v1.0）：ExecBoard="wenxue" 的引擎换图文执行
	// 常量（无空间渲染），办公/CLI/TUI 维持 SingleModelPromptFor 双态。
	var execDiscipline string
	if opts.ExecBoard == "wenxue" {
		execDiscipline = agent.WenxueSingleModelPrompt
	} else {
		execDiscipline = agent.SingleModelPromptFor(space)
	}
	execPrompt := compiler.SystemPrompt() + "\n\n" + execDiscipline
	// 交付与收尾纪律（灵犀 delivery_format/response_style/completion_criteria
	// 蒸馏，docs/gaea-lingxi-advantages-2026-11.md #16/#19）：无条件追加，通用
	// 规则在前、专项分工在后。同样不动 SingleModelPrompt 常量。中立单源对两
	// 板块同追加（q341 拍板：不复制进两份专属稿正文）。
	execPrompt += "\n\n" + deliveryPromptSection
	// 波3.5 preferLive 引导回（dsh-office 蒸馏，docs/gaea-dsh-office-distill-
	// research-2026-11.md §6.3）：本机装了 Office/WPS（COM 档工具族已注册）
	// 时追加实时协同分工段——Word/PPT 文档工作优先 office_ 工具而非 docx/
	// pptx 技能造文件（上游 promptText 常驻段同形态；探测下沉 comhelper 供
	// 内核可达）。SingleModelPrompt 常量逐字节红线不动。图文引擎跳过（无
	// office_ 工具，段内分工全不可执行——板块缝同批）。
	if opts.ExecBoard != "wenxue" && comhelper.ProbeSuiteAvailable() {
		execPrompt += "\n\n" + officeLivePromptSection
	}
	execSess := agent.NewSession(execPrompt)
	executor := agent.New(execProv, reg, execSess, agent.Options{
		MaxSteps:    maxSteps,
		Temperature: cfg.Agent.Temperature,
		Pricing:     entry.Price,
		// B5 工具归因：runner 把模型 id 盖戳到工具/请求头事件（ResolveModel
		// 已归一出实际模型，含 provider/model 引用与 Models 列表缺省）。
		ModelName:     entry.Model,
		Gate:          gatedHeadless,
		Hooks:         hookRunner,
		AccessPolicy:  accessPolicy,
		Jobs:          jm,
		ContextWindow: entry.ContextWindow,
		Compaction:    compactionConfig(cfg),
		Dispatcher:    toolDispatcher,
		// v4.1 证据链 Journal 目录（work 空间；play 回合由 agent 红线过滤不落盘）。
		JournalDir: filepath.Join(cwd, ".gaea", "work", "journal"),
		// v4.379 泄洪库（dsh spill-policy 蒸馏）：大工具结果全文存会话库+
		// read_spill 取回；[agent] tool_spill 缺省开，false 回旧行为。
		Spill: cfg.Agent.ToolSpillEnabled(),
	}, sink)

	// v4.211 记忆 5.1 出口闸：最终回复定稿时剥离悬空 [MEM:] 引用键（真·发送前
	// ——改写后的全文随 Message 事件发出并进 session/摘要，模型幻觉键到不了
	// 持久层）。被剥离键落 dangling cite 事件（AppendCiteEvents）保住可观测性；
	// 命中键不在这里 Touch——触达仍是回合收尾 touchMemoryCitations 的职责。
	// 门控与装配同源：记忆总闸关闭 / 库不可用时不注入（nil = 不改写，子代理
	// 与 headless executor 均不受影响）。
	if cfg.Memory.Enabled {
		if gdb := db.GetDatabase(memUserDir); gdb != nil {
			sanitizeStore := memory.SQLiteStoreFor(gdb, memUserDir, cwd)
			sanitizeLog := &memory.EventLog{DB: gdb}
			sanitizeSpace := space
			executor.SetFinalizeText(func(text string) string {
				cleaned, stripped := sanitizeStore.StripDanglingCitations(text, sanitizeSpace)
				if len(stripped) > 0 {
					sanitizeLog.AppendCiteEvents(nil, stripped, sanitizeSpace)
				}
				return cleaned
			})
		}
	}

	// V7.0: session archive for cross-session Dream/Distill
	archiveDir := filepath.Join(cwd, ".gaea", "archive")
	if ar, err := archive.Open(archiveDir); err == nil && ar != nil {
		sid := filepath.Base(orDefault(opts.SessionDir, config.SessionDir()))
		if sid == "" || sid == "." {
			sid = fmt.Sprintf("session-%d", time.Now().Unix())
		}
		executor.SetArchive(ar, sid)
	}

	// D6 清理债务账本（dsh-ops 蒸馏 2026-11-10）：异步清扫过期 publish-quarantine
	// stamp（先记账后放句柄；只删账本背书的 30 天前 stamp，账本外名字整目录跳过，
	// 失败留账下次重试）。每进程每工作区一次；启动关键路径不做整树遍历。
	sweepQuarantineOnce(cwd)

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
	addGated(reg, opts.ToolFilter, command.NewSlashCommandTool(slashEntries))

	// 前端注入的工具（如桌面端 image_gen）追加到注册表。S1.3-B：按装配空间
	// 物理过滤（work=办公/编辑/检索域、play=生图/轻语/小说/角色域、shared
	// 两空间通用；space 为空 = mode=off 全注册现状），重建时随新空间生效。
	// 2026-11-15 刀1：ToolFilter 非空时叠加板块级过滤（图文 deny 名单）。
	for _, t := range opts.ExtraTools {
		if t != nil && builtin.AllowsSpace(t, space) && toolAllowed(opts.ToolFilter, t.Name()) {
			reg.Add(t)
		}
	}

	// ToolHide 装配终点收口（2026-11-21 办公运行模式，众神殿 pan_1795202763988
	// 终案）：按名 Registry.Hide——schema 不可见、协议层按名可调（hidden 既有
	// 语义）。与 tools.deferred（Search.Enabled 门控）不同：运行模式 Hide 是
	// 模式组合声明的一部分，缺省能力由 ModeLine 在 L1 明示（DSH-r5：两件必须
	// 同批，防「无声消失」）。未注册名跳过（warn 一次，名单漂移可观测）。
	for _, name := range opts.ToolHide {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if _, ok := reg.Get(name); !ok {
			slog.Warn("run mode ToolHide 名单含未注册工具", "name", name)
			continue
		}
		reg.Hide(name)
	}

	// v4.61：主执行器装配「本地模型工具 → 变相子代理」运行记录器——把注册表
	// 中声明 ModelBacked 的工具名集合交给执行器，调用时落 mt_ 记录（与子代理
	// 同一会话 UI）。MCP 热插工具不在此集合（目前无本地模型 MCP 工具；未来
	// 需要时在 AddMCPServer 装配点补同名接线）。
	modelBackedNames := map[string]bool{}
	for _, name := range reg.Names() {
		if t, ok := reg.Get(name); ok && tool.IsModelBacked(t) {
			modelBackedNames[name] = true
		}
	}
	executor.SetModelToolRecorder(modelBackedNames, subagentStore)

	// A2 动态工具目录（dsh-ops 蒸馏 2026-11-10）：注册面定稿后按
	// [agent] dynamic_tools 模式装配（缺省 auto=仅 MCP 重载面开启，标准会话
	// 零变化；见 dynamic_tools.go）。
	if wireDynamicTools(cfg.Agent.DynamicTools, reg, executor) {
		fmt.Fprintf(os.Stderr, "dynamic tools: 目录已开启（mode=%q，常驻外工具经 discover_tools 激活）\n", cfg.Agent.DynamicTools)
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
	_ = compiler.IdentityLayer().SaveHash(cacheDir) // best-effort

	// C4 TimedOut：审批等待超时贯通（0 = 不超时，交互场景默认等待）。
	// S1.5-A：空间权限段的 approval_timeout_secs 显式配置（>0）时优先于
	// agent 全局值；未配置回退现状。
	approvalTimeoutSecs := cfg.Agent.ApprovalTimeoutSecs
	if perm.ApprovalTimeoutSecs > 0 {
		approvalTimeoutSecs = perm.ApprovalTimeoutSecs
	}

	ctrlOpts := control.Options{
		Runner:       runner,
		Executor:     executor,
		Sink:         sink,
		Policy:       policy,
		ApprovalMode: perm.ApprovalMode,
		SectionDiff:  cfg.Agent.SectionDiff,
		Guardian:     opts.Guardian,
		Label:        label,
		SystemPrompt: sysPrompt,
		SessionDir:   orDefault(opts.SessionDir, config.SessionDir()),
		Host:         pluginHost,
		Commands:     cmds,
		Skills:       skills,
		// P8 断网闸（交互路径四个换级点共用 newGate 后统一包装）。
		DenyNetworkTools: !cfg.Sandbox.Network,
		// 刀1 断网闸逐主机修正案：开关+持久台账快照+配置回写回调（headless
		// 路径在 gatedHeadless 处只挂只读台账，不弹卡）。
		NetworkGateAmendment: cfg.Network.GateAmendment,
		NetworkHostAllow:     cfg.Network.GateHostAllow,
		PersistNetworkHost: func(entry string) error {
			pcfg, err := config.Load()
			if err != nil {
				return err
			}
			pcfg.Network.GateHostAllow = append(pcfg.Network.GateHostAllow, entry)
			return pcfg.Save()
		},
		Hooks:          hookRunner,
		Memory:         mem,
		MemoryDisabled: !cfg.Memory.Enabled,
		// C4 TimedOut：审批等待超时贯通（0 = 不超时，交互场景默认等待）。
		ApprovalTimeout: time.Duration(approvalTimeoutSecs) * time.Second,
		// S1.5-A：逐条强制审批工具集按空间策略生效（nil = control 包级默认集
		// 现状；play 产品默认空集 = 不弹审批卡）。
		HardAskTools: hardAskSetFromList(perm.HardAsk),
		// C4 persist_allow：审批「始终允许」回写 [permissions].allow 策略文件
		// （编辑器 AtomicWrite + RenderTOML→Load 往返保证；AddPermissionRule
		// 幂等去重）。加载失败/无配置文件时 Save() 落到 ./gaea.toml。
		// S1.5-A：按装配空间分段回写——该空间配置了 permissions 段写段内，
		// 否则顶层 [permissions]（现状路径；space 为空恒顶层）。
		PersistAllowRule: func(rule string) error {
			pcfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := pcfg.AddPermissionRuleForSpace(space, "allow", rule); err != nil {
				return err
			}
			return pcfg.Save()
		},
		Cleanup:    ctrlCleanup,
		BalanceURL: entry.BalanceURL,
		BalanceKey: entry.APIKey(),
		// 3.0 Wave 4：余额后端 kind 从 ProviderEntry 贯通（空 = controller 按
		// 历史默认 deepseek 形状；配置 balance_kind 即切换，代码零改动）。
		BalanceKind:   entry.BalanceKind,
		Jobs:          jm,
		Registry:      reg,
		PluginCtx:     ctx,
		CtxMgr:        ctxMgr,
		WorkspaceRoot: cwd,
	}
	ctrl := control.New(ctrlOpts)
	ctrl.WireJobWake() // Z6 蒸馏刀：后台任务完成主动唤醒（自激励预算在 controller 侧）
	// v4.61：惰性 subagents store 的会话路径源（写盘发生在回合内，此时必已
	// 有会话路径；与 eventLogSink 的 SetPathSource 同一晚绑定时序）。
	ctrlSessionPath = ctrl.SessionPath
	// 事件日志 sink 的会话路径只有控制构建完成后才可知（首次对话自动创建 /
	// Resume 注入），故在此注入路径解析器；事件发射发生在 Build 返回之后，
	// 闭包读取的是实时 SessionPath。S2：空间来源同风格懒解析——实时读取
	// 当前会话的写入侧空间（""=space.mode=off 平铺日志不写 space 字段）。
	if eventLogSink != nil {
		eventLogSink.SetPathSource(func() string { return ctrl.SessionPath() })
		eventLogSink.SetSpaceSource(func() string { return ctrl.SessionSpace() })
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

// hardAskSetFromList 把空间策略的 hard_ask 列表转成 control 的集合形态：
// nil（未按空间配置）原样透传（control 用包级默认集现状）；非 nil（含空集）
// 转集合，空条目丢弃。
func hardAskSetFromList(names []string) map[string]bool {
	if names == nil {
		return nil
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			m[n] = true
		}
	}
	return m
}

// NewProvider builds the LLM seam provider (provider.LLMProvider) from a
// configured entry. The kind comes from config (gaea.toml [[providers]] kind),
// so switching the chat backend is a config-only change: providers.Register(kind)
// selects the implementation, provider.NewLLM fails closed on an unknown kind,
// and an empty kind defaults to provider.DefaultLLMKind ("wubigrok" — today's
// bridge provider, keeping unconfigured behaviour unchanged). Exported so custom
// assemblers (e.g. the ACP per-session factory) can reuse it without going
// through the full Build. 账目 feature 标签不覆盖（bridge 工厂缺省 "gaea"/
// "office" 判定）。
func NewProvider(e *config.ProviderEntry) (provider.LLMProvider, error) {
	return newProviderWithFeature(e, "")
}

// newProviderWithFeature 同 NewProvider，非空 feature 注入 provider.Config
// .Extra["feature"]——bridge 工厂以此覆盖账目标签（2026-11-17 审查批5 刀1：
// 图文引擎 Build 传 opts.Feature="wenxue"，用量记账 PerFeature 分桶分列；
// 空=缺省判定，办公逐字节不变）。
func newProviderWithFeature(e *config.ProviderEntry, feature string) (provider.LLMProvider, error) {
	extra := map[string]any{
		"api_key_env": e.APIKeyEnv,
		"thinking":    e.Thinking,
		"effort":      e.Effort,
	}
	if f := strings.TrimSpace(feature); f != "" {
		extra["feature"] = f
	}
	return provider.NewLLM(e.Kind, provider.Config{
		Name:    e.Name,
		BaseURL: e.BaseURL,
		Model:   e.Model,
		APIKey:  e.APIKey(),
		// Pass the key's env var so auth failures can name where to fix it, plus
		// provider-kind-specific knobs (the anthropic provider reads thinking/effort;
		// the openai one ignores them).
		Extra: extra,
	})
}

// toolAllowed 是 ToolFilter 的 nil 安全包装（nil = 恒允许，现状逐字节）。
func toolAllowed(filter func(name string) bool, name string) bool {
	return filter == nil || filter(name)
}

// addGated 按板块装配闸注册模型面工具（2026-11-16 图文瘦身刀）。
// ToolFilter 此前只过 addBuiltins/plugins/ExtraTools 三处，boot 直注册的会话
// 级元工具（task/ask/记忆族/子代理管理族/request_permission/exit_plan_mode/
// summarize_file/slash_command 等）绕过板块名单——本 helper 把闸扩到全部模型
// 面注册点：filter 空=全放行（办公逐字节现状），非空时名单内物理不装。
func addGated(reg *tool.Registry, filter func(name string) bool, t tool.Tool) {
	if t != nil && toolAllowed(filter, t.Name()) {
		reg.Add(t)
	}
}

// addBuiltins adds enabled built-in tools to reg. An empty list means all of
// them. writeRoots confines the file-writing built-ins to the workspace: after
// the (unconfined) defaults are added, each enabled writer is replaced by an
// instance bound to writeRoots (preserving registry order).
// S1.3-B：space 为装配空间（"work"/"play"；""=space.mode=off 全注册现状），
// 按 builtin 分类表做装配期物理过滤（play 不含 edit 系/bash，work 不含
// image_gen，shared 两空间通用）——不改运行时行为，ActiveSchemas 对齐不受影响。
// 2026-11-15 刀1：toolFilter 非空时叠加板块级过滤（图文 deny 名单；被滤名
// 的 confined 替换块因 reg.Get 缺席自然跳过，零额外分支）。
func addBuiltins(reg *tool.Registry, dir string, enabled, writeRoots []string, bashSpec sandbox.Spec, proxySpec netclient.ProxySpec, stderr io.Writer, space string, toolFilter func(name string) bool) {
	if len(enabled) == 0 {
		for _, t := range tool.Builtins() {
			if builtin.AllowsSpace(t, space) && toolAllowed(toolFilter, t.Name()) {
				reg.Add(t)
			}
		}
	} else {
		for _, name := range enabled {
			t, ok := tool.LookupBuiltin(name)
			if !ok {
				_, _ = fmt.Fprintf(stderr, "warning: unknown built-in tool %q\n", name)
				continue
			}
			if builtin.AllowsSpace(t, space) && toolAllowed(toolFilter, t.Name()) {
				reg.Add(t)
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
		if v := e.StartupTimeout(); v > 0 {
			specs[i].StartupTimeout = time.Duration(v * float64(time.Second))
		}
		if v := e.ToolTimeout(); v > 0 {
			specs[i].ToolTimeout = time.Duration(v * float64(time.Second))
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
	// Batch editing: multi_edit is redundant with multiple edit_file calls
	reg.HideUnlessOnly([]string{"multi_edit"}, []string{"edit_file"})

	// Background job management: merge kill_shell + wait into bash/bgjobs
	reg.HideUnlessOnly([]string{"kill_shell", "wait"}, []string{"bash", "bash_output"})
}

// compactionConfig 压缩配置装配（阶段九 9.1「双阈值可配置」）：三层水位经
// [agent] compaction_soft_ratio/compaction_ratio/compaction_force_ratio 暴露，
// 0 值/非法值回落内置缺省（cfg.Agent.CompactionTiers 钳制）。归档目录沿用
// 全局 ArchiveDir。
func compactionConfig(cfg *config.Config) agent.CompactionConfig {
	soft, ratio, force := cfg.Agent.CompactionTiers()
	return agent.CompactionConfig{
		ArchiveDir: config.ArchiveDir(),
		SoftRatio:  soft,
		Ratio:      ratio,
		ForceRatio: force,
		// T9：token-budget 压缩形态开关（默认关=只有摘要式压缩）。
		NewWindow: cfg.Agent.CompactionNewWindow,
	}
}
