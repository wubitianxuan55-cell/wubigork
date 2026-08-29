// Package config loads Tianxuan's runtime configuration from TOML. Resolution order:
// flag > project ./gaea.toml > user ~/.config/gaea/config.toml > built-in defaults.
// Secrets come from the environment via api_key_env and are never stored in
// config files.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/netclient"
)

// Config is Tianxuan's runtime configuration.
type Config struct {
	DefaultModel string            `toml:"default_model"`
	Language     string            `toml:"language"`  // ui/model language tag (e.g. "zh"); empty = auto-detect from $LANG / $TIANXUAN_LANG
	Workspace    string            `toml:"workspace"` // 办公工作空间目录（空 = 进程启动目录）
	Agent        AgentConfig       `toml:"agent"`
	Session      SessionConfig     `toml:"session"` // 3.0 Step 1: 会话持久化格式（事件日志回退开关）
	// Space 是双空间分区开关（S2）：[space] mode = "on"|"off"，默认 on。
	Space      SpaceConfig       `toml:"space"`
	// SpaceProfiles 是按空间装配 profile（S1.3-A/S1.5-A）：[space_profiles.<space>]。
	// 缺省（段缺失/空 map）= 零值 = 现状逐字节回退；space.mode=off 时整体不读。
	SpaceProfiles map[string]SpaceProfile `toml:"space_profiles"`
	Providers    []ProviderEntry   `toml:"providers"`
	Tools        ToolsConfig       `toml:"tools"`
	Permissions  PermissionsConfig `toml:"permissions"`
	Sandbox      SandboxConfig     `toml:"sandbox"`
	Plugins      []PluginEntry     `toml:"plugins"`
	Skills       SkillsConfig      `toml:"skills"`
	Search       SearchConfig      `toml:"search"`
	Network      NetworkConfig     `toml:"network"`
	Memory       MemoryConfig      `toml:"memory"`
	// 3.0 Step 3d Provider Seam：embed/rerank/vision/markdown_converter 后端选择。
	// 零值 = 全默认（本地 herdsman 兼容端点 + 各自默认模型），切换后端只改配置。
	Retrieval        RetrievalConfig        `toml:"retrieval"`
	Vision           VisionConfig           `toml:"vision"`
	MarkdownConverter MarkdownConverterConfig `toml:"markdown_converter"`
}

// SessionConfig 是会话持久化行为配置（3.0 Step 1 回退开关）。
// 缺省 legacy = 旧行为（整文件重写 JSONL），不迁移用户配置；显式配置
// log_format = "event" 才启用追加式事件日志（<id>.gaea-log.jsonl）。
type SessionConfig struct {
	// LogFormat 选择会话持久化格式："legacy"（默认，旧行为）| "event"（事件日志）。
	LogFormat string `toml:"log_format"`
	// Space 是新建会话的空间落点（S2 双空间）："work"（默认）| "play"。
	// 仅 space.mode=on 时生效；space.mode=off 时所有读写路径回退平铺目录。
	Space string `toml:"space"`
}

// SpaceConfig 是双空间分区开关（S2 回退开关，仿 session.log_format 三件套）。
type SpaceConfig struct {
	// Mode 控制会话空间分区："on"（默认）| "off"。off 时所有会话读写路径
	// 忽略空间（新会话回平铺目录、日志不写 space 字段），行为整体回退；
	// 旧分区数据仍可读（读端按目录归属降级）。
	Mode string `toml:"mode"`
}

// SpaceModeIsOn 报告会话空间分区是否启用（缺省 on；仅显式 "off" 关闭）。
func (c *Config) SpaceModeIsOn() bool {
	return c == nil || c.Space.Mode != "off"
}

// SessionSpace 解析 session.space 配置（trim + 小写；空/非法 → work）。
func (c *Config) SessionSpace() string {
	if c == nil {
		return spaces.SpaceWork
	}
	return spaces.Normalize(strings.ToLower(strings.TrimSpace(c.Session.Space)))
}

// EffectiveSessionSpace 返回写入侧生效空间："work" | "play"；
// space.mode=off 时返回 ""（调用方据此回退平铺目录、日志不写 space 字段）。
func (c *Config) EffectiveSessionSpace() string {
	if !c.SpaceModeIsOn() {
		return ""
	}
	return c.SessionSpace()
}

// LogFormatIsEvent 报告是否启用事件日志模式（大小写不敏感，仅精确匹配 "event"）。
func (c *Config) LogFormatIsEvent() bool {
	return c != nil && strings.EqualFold(c.Session.LogFormat, "event")
}

// ── 按空间装配（S1.3-A 模型 profile + S1.5-A 权限策略）────────────────

// SpacePermissionsConfig 是 [space_profiles.<space>.permissions] 子段（S1.5-A）。
// 字段语义对齐顶层 [permissions]；nil 指针 = 段未配置（逐字段回退现状）。
type SpacePermissionsConfig struct {
	// Mode 是该空间的 writer 回退决策（"ask"|"allow"|"deny"；空 = 回退顶层/空间缺省）。
	Mode string `toml:"mode"`
	// HardAsk 是逐条强制审批的工具名列表（覆盖 control 包默认集）。TOML 显式
	// 空数组 = 清空（play 不弹审批卡）；未写 = 按空间缺省（play 空集 / work 默认集）。
	HardAsk []string `toml:"hard_ask"`
	// ApprovalTimeoutSecs 是审批等待超时（0 = 回退 agent.approval_timeout_secs）。
	ApprovalTimeoutSecs int `toml:"approval_timeout_secs"`
	Allow               []string `toml:"allow"`
	Ask                 []string `toml:"ask"`
	Deny                []string `toml:"deny"`
}

// PlayGuardrails 是 [space_profiles.<space>.guardrails] 子段（S1.5-B）：
// play 域直连生成点（轻语人格对话/章节/支线/角色卡/生图）的参数钳制配置。
// 护栏不走 permission 引擎（这些点本来没有闸），只钳参数/加安全段，不改
// 生成逻辑语义。零值字段 = 不钳制（现状逐字节）。
type PlayGuardrails struct {
	// Enabled 是护栏总开关：false（含段未配置）= 全部钳制不生效。
	Enabled bool `toml:"enabled"`
	// TemperatureMax 是生成温度上限（0 = 不钳制）：生成温度超过它时钳到它，
	// 只降不升（该点未显式传温度时不注入新参数）。
	TemperatureMax float64 `toml:"temperature_max"`
	// MaxOutputTokens 是单次生成 max_tokens 上限（0 = 不钳制）：超过它的点
	// 钳到它；对未设置 max_tokens 的直连 ChatRequest 点显式下发该上限。
	MaxOutputTokens int `toml:"max_output_tokens"`
	// ImageSafeMode 是绘梦/生图安全模式：提交前为提示词注入安全段。
	ImageSafeMode bool `toml:"image_safe_mode"`
	// PersonaLock 是轻语人格锁：人格一致性参数（dims/voiceGuide）注入时
	// 追加人格锁定段（防系统层覆盖人格）并锁温度上限（上限取
	// TemperatureMax，0 = 不设温度上限）。
	PersonaLock bool `toml:"persona_lock"`
}

// SpaceProfile 是单个空间的装配 profile。模型字段引用现有模型选择键体系
// （feature_model_handler 功能域：chat/whisper/novel/office/gaea/characterlib/
// routine），值经 ResolveModel 既有链解析为 provider entry；空 = 该域维持现状
// （零值 = 现状逐字节回退）。桌面端 play 域模型走既有 bridge_feature 功能绑定
// （零新增绑定），boot 装配层消费 gaea 键（办公 agent 功能域）。
type SpaceProfile struct {
	Chat         string `toml:"chat"`
	Whisper      string `toml:"whisper"`
	Novel        string `toml:"novel"`
	Office       string `toml:"office"`
	Gaea         string `toml:"gaea"`
	CharacterLib string `toml:"characterlib"`
	Routine      string `toml:"routine"`
	// Permissions 是该空间的权限策略段（S1.5-A）；nil = 未配置。
	Permissions *SpacePermissionsConfig `toml:"permissions"`
	// Guardrails 是该空间的内容护栏段（S1.5-B，仅 play 域生成点消费）；
	// nil = 未配置（零钳制 = 现状逐字节）。
	Guardrails *PlayGuardrails `toml:"guardrails"`
}

// SpacePermissions 是空间生效的权限装配值（S1.5-A，PermissionsForSpace 返回）。
type SpacePermissions struct {
	// Mode 是 writer 回退决策（permission.ParseDecision 解析，缺省 ask）。
	Mode string
	// Allow/Ask/Deny 是生效规则列表（顶层 + 空间段叠加，precedence 引擎天然处理）。
	Allow []string
	Ask   []string
	Deny  []string
	// HardAsk 是逐条强制审批工具集：nil = 未按空间配置（control 用包级默认集，
	// 现状）；非 nil（可为空集）= 按空间策略生效（play 产品默认 = 空集，不弹审批卡）。
	HardAsk []string
	// ApprovalTimeoutSecs 是审批等待超时（0 = 未配置，boot 回退 agent.approval_timeout_secs）。
	ApprovalTimeoutSecs int
}

// SpaceProfile 返回空间的装配 profile（S1.3-A）。space 为 "work"/"play"
// （大小写不敏感）；空串（space.mode=off 回退形态）或该空间未配置段 → 零值
// profile + nil error（现状回退）；其他值 → 错误。
func (c *Config) SpaceProfile(space string) (*SpaceProfile, error) {
	space = strings.ToLower(strings.TrimSpace(space))
	if space == "" {
		return &SpaceProfile{}, nil
	}
	if !spaces.Valid(space) {
		return nil, fmt.Errorf("space profile: 非法空间 %q（仅 work|play）", space)
	}
	if c == nil {
		return &SpaceProfile{}, nil
	}
	for k, p := range c.SpaceProfiles {
		if strings.ToLower(strings.TrimSpace(k)) == space {
			cp := p
			return &cp, nil
		}
	}
	return &SpaceProfile{}, nil
}

// PermissionsForSpace 返回空间生效的权限装配值（S1.5-A）。缺省 = 现状单 Policy：
//   - space 为空（space.mode=off）或非法 → 顶层 [permissions] 原样（HardAsk=nil、
//     Timeout=0，boot 据此走现状路径）；
//   - work 未配置段 → 现状；
//   - play 未配置段 → 产品默认：mode="allow" + hard_ask 空集（不弹审批卡，与
//     S1.2 play 记忆隔离配套，产品已确认）；规则列表继承顶层（deny 硬拒绝仍生效）；
//   - 配置了 [space_profiles.<space>.permissions] 段 → mode 非空覆盖（否则按
//     空间缺省）、规则列表 = 顶层 + 段叠加、hard_ask 已配置（含空数组）用段值、
//     approval_timeout_secs > 0 覆盖。
func (c *Config) PermissionsForSpace(space string) SpacePermissions {
	flat := SpacePermissions{
		Mode:  c.Permissions.Mode,
		Allow: c.Permissions.Allow,
		Ask:   c.Permissions.Ask,
		Deny:  c.Permissions.Deny,
	}
	space = strings.ToLower(strings.TrimSpace(space))
	if c == nil || space == "" || !spaces.Valid(space) {
		return flat // mode=off / 异常值：现状逐字节回退
	}
	prof, err := c.SpaceProfile(space)
	if err != nil {
		return flat
	}
	out := flat
	if space == spaces.SpacePlay {
		out.Mode = "allow"       // play 产品默认（显式段 mode 仍可覆盖）
		out.HardAsk = []string{} // 显式空集：不弹审批卡（remember 等不再确认）
	}
	if perm := prof.Permissions; perm != nil {
		if perm.Mode != "" {
			out.Mode = perm.Mode
		}
		out.Allow = concatRuleLists(flat.Allow, perm.Allow)
		out.Ask = concatRuleLists(flat.Ask, perm.Ask)
		out.Deny = concatRuleLists(flat.Deny, perm.Deny)
		if perm.HardAsk != nil {
			out.HardAsk = perm.HardAsk
		}
		if perm.ApprovalTimeoutSecs > 0 {
			out.ApprovalTimeoutSecs = perm.ApprovalTimeoutSecs
		}
	}
	return out
}

// PlayGuardrails 返回空间生效的 play 内容护栏值（S1.5-B，五个 play 域直连
// 生成点共用的取值点）。零值 = 零钳制 = 现状逐字节回退：
//   - space 非 "play"（含空串 = space.mode=off 回退形态）→ 零值；
//   - space.mode=off → 零值（空间维度整体关闭，恒等现状）；
//   - guardrails 段未配置或 enabled=false → 零值（总开关关 = 全部不钳制）；
//   - 段配置且 enabled=true → 段值原样（0/""/false 字段仍不钳制）。
func (c *Config) PlayGuardrails(space string) PlayGuardrails {
	if c == nil {
		return PlayGuardrails{}
	}
	space = strings.ToLower(strings.TrimSpace(space))
	if space != spaces.SpacePlay || !c.SpaceModeIsOn() {
		return PlayGuardrails{}
	}
	prof, err := c.SpaceProfile(space)
	if err != nil || prof == nil || prof.Guardrails == nil {
		return PlayGuardrails{}
	}
	g := *prof.Guardrails
	if !g.Enabled {
		return PlayGuardrails{}
	}
	return g
}

// concatRuleLists 拼接两个规则列表（base 在前、空间段在后），base 为空时直接
// 返回空间的切片（避免为纯空间配置多复制一层）。
func concatRuleLists(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	if len(base) == 0 {
		return extra
	}
	out := make([]string, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

// RetrievalConfig 配置本地检索后端（3.0 Step 3d #2/#3 Provider Seam）。
// 零值 = 全默认：kind=openai（OpenAI 兼容 /v1/embeddings、/v1/rerank），
// base_url 默认 http://localhost:8080，模型默认 bge-m3 / bge-reranker-v2-m3。
type RetrievalConfig struct {
	EmbedKind     string `toml:"embed_kind"`
	EmbedBaseURL  string `toml:"embed_base_url"`
	EmbedModel    string `toml:"embed_model"`
	RerankKind    string `toml:"rerank_kind"`
	RerankBaseURL string `toml:"rerank_base_url"`
	RerankModel   string `toml:"rerank_model"`
}

// VisionConfig 配置视觉识别后端（3.0 Step 3d #4 Provider Seam）。
// 零值 = 全默认：kind=openai，base_url 默认 http://127.0.0.1:8080/v1，
// model 默认本地 Qwen3.6 视觉模型（等价旧 GAEA_VISION_* 环境变量缺省行为）。
type VisionConfig struct {
	Kind    string `toml:"kind"`
	BaseURL string `toml:"base_url"`
	Model   string `toml:"model"`
}

// MarkdownConverterConfig 配置文档转换后端（3.0 Step 3d #7 Provider Seam）。
// kind 空 = 关闭转换（二进制文件走旧"提示安装 markitdown"错误路径）；
// kind="cli" = markitdown CLI→python -m markitdown 两级回退（默认）。
type MarkdownConverterConfig struct {
	Kind string `toml:"kind"`
}

// MemoryConfig 是办公记忆开关（记忆可控性：用户可一键关闭记忆注入）。
type MemoryConfig struct {
	// Enabled 控制自动记忆（画像/规则/事实）是否注入系统提示词与逐轮上下文。
	// 关闭后记忆不写入提示词（文档记忆文件仍保留在磁盘，重新开启即恢复）。
	Enabled bool `toml:"enabled"`
	// ArchivedRetentionDays 是归档事实的保留期（天）：归档超过该时长的事实由
	// GaeaMemoryCleanupArchived 硬删（0/缺省 = 90 天默认，钳制 [1,730]）。
	// 记忆统一层第二刀：保留期从常量改为可配置，误归档在保留期内可恢复。
	ArchivedRetentionDays int `toml:"archived_retention_days"`
}

// SearchConfig configures web search engines.

// SearchConfig configures web search engines. Resolution order: local SearXNG
// (fastest, private) → Tavily API → Brave Search API → public SearXNG instances.
// Each engine requires its own credentials; only configured engines are tried.
type SearchConfig struct {
	// LocalSearXNGURL is the base URL of a self-hosted SearXNG instance
	// (e.g. "http://localhost:8080"). Empty disables it.
	LocalSearXNGURL string `toml:"local_searxng_url"`
	// TavilyAPIKeyEnv names the environment variable holding a Tavily API key
	// (free tier: 1000 searches/month). Empty disables Tavily.
	TavilyAPIKeyEnv string `toml:"tavily_api_key_env"`
	// BraveAPIKeyEnv names the environment variable holding a Brave Search API key
	// (free tier: 2000 searches/month). Empty disables Brave.
	BraveAPIKeyEnv string `toml:"brave_api_key_env"`
	// TimeoutSeconds is the per-engine HTTP timeout in seconds (default 10).
	TimeoutSeconds int `toml:"timeout_seconds"`
	// AllowDomains restricts web_fetch to only these domains (supports *.example.com wildcards).
	// Empty means all non-blocked domains are allowed. Takes precedence after deny.
	AllowDomains []string `toml:"allow_domains"`
	// DenyDomains blocks web_fetch from accessing these domains (supports *.example.com wildcards).
	// Deny always takes precedence over allow.
	DenyDomains []string `toml:"deny_domains"`
	// EngineOrder overrides the fallback engine priority (3.0 Step 3d #1):
	// registry kinds like "local-searxng","tavily","brave","public-searxng","bing","duckduckgo-lite".
	// Empty = the built-in default order (local SearXNG → Tavily → Brave → public SearXNG → Bing → DDG).
	EngineOrder []string `toml:"engine_order"`
}

// TavilyKey resolves the Tavily API key from the configured environment variable.
func (c *SearchConfig) TavilyKey() string {
	if c.TavilyAPIKeyEnv == "" {
		return ""
	}
	return os.Getenv(c.TavilyAPIKeyEnv)
}

// BraveKey resolves the Brave Search API key.
func (c *SearchConfig) BraveKey() string {
	if c.BraveAPIKeyEnv == "" {
		return ""
	}
	return os.Getenv(c.BraveAPIKeyEnv)
}

// SearchTimeout returns the configured timeout with a safe floor of 5s.
func (c *SearchConfig) SearchTimeout() time.Duration {
	if c.TimeoutSeconds < 5 {
		return 10 * time.Second
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// ── Network proxy (V10.31) ──────────────────────────────────────────

// NetworkConfig controls how outgoing HTTP requests reach the internet.
// ProxyMode selects the strategy: "auto" (system proxy), "env" (HTTP_PROXY/
// HTTPS_PROXY/all_proxy env vars), "custom" (ProxyURL), or "off" (direct).
// Proxy is the structured alternative; when both ProxyURL and Proxy are set,
// Proxy takes precedence.
type NetworkConfig struct {
	ProxyMode string             `toml:"proxy_mode"` // auto | env | custom | off
	ProxyURL  string             `toml:"proxy_url"`  // http://host:port or socks5://host:port
	NoProxy   string             `toml:"no_proxy"`   // comma-separated host suffixes excluded from proxying
	Proxy     NetworkProxyConfig `toml:"proxy"`      // structured alternative to ProxyURL
}

// NetworkProxyConfig is the structured form of proxy configuration.
type NetworkProxyConfig struct {
	Type     string `toml:"type"` // http | https | socks5 | socks5h
	Server   string `toml:"server"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// NetworkProxySpec returns a netclient.ProxySpec suitable for configuring

// NetworkProxySpec returns a netclient.ProxySpec suitable for configuring
// HTTP clients throughout gaea. An empty spec means no proxy.
func (c *Config) NetworkProxySpec() netclient.ProxySpec {
	return netclient.ProxySpec{
		Mode:     c.Network.ProxyMode,
		URL:      c.Network.ProxyURL,
		NoProxy:  c.Network.NoProxy,
		Type:     c.Network.Proxy.Type,
		Server:   c.Network.Proxy.Server,
		Port:     c.Network.Proxy.Port,
		Username: c.Network.Proxy.Username,
		Password: c.Network.Proxy.Password,
	}
}

// SkillsConfig configures skill discovery. Paths adds extra "custom"-scope skill
// roots — each a directory of SKILL.md / <name>.md playbooks — scanned between
// the project roots (.gaea/.agents/.claude under the workspace) and the
// global roots (the same three under the home dir). ~ and relative paths and
// ${VAR} expansion are supported.
type SkillsConfig struct {
	Paths []string `toml:"paths"`
}

// SkillCustomPaths returns the configured custom skill roots with ${VAR}
// expanded; empty entries are dropped.
func (c *Config) SkillCustomPaths() []string {
	var out []string
	for _, p := range c.Skills.Paths {
		if p = ExpandVars(p); strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// SandboxConfig bounds the blast radius of tool calls (Phase 0: file-writer
// confinement). WorkspaceRoot is the directory the built-in file writers
// (write_file / edit_file / multi_edit) may modify; empty means the current
// working directory, so writes stay inside the project by default. AllowWrite
// lists extra directories writers may also touch (e.g. a sibling repo or a temp
// dir). Both support ${VAR} / ${VAR:-default} expansion. Reads are unrestricted;
// confining `bash` is Phase 1 (OS-level sandbox).
type SandboxConfig struct {
	WorkspaceRoot string   `toml:"workspace_root"`
	AllowWrite    []string `toml:"allow_write"`
	// Bash is the OS-sandbox mode for the bash tool: "enforce" (default) jails
	// each command, "off" runs it unconfined. Phase 1; macOS only for now, with
	// a graceful fallback elsewhere (see internal/sandbox).
	Bash string `toml:"bash"`
	// Network allows network egress from inside the bash sandbox. Defaults true
	// so module/package downloads keep working; the boundary is then writes.
	Network bool `toml:"network"`
}

// WriteRoots returns the directories file-writer tools may modify: the
// workspace root (defaulting to the current working directory when unset) plus
// any AllowWrite extras, with ${VAR} expanded. The roots are returned as given
// (relative or absolute); the confiner resolves them to absolute, symlink-free
// paths. The result is always non-empty, so confinement is on by default.
func (c *Config) WriteRoots() []string {
	root := ExpandVars(c.Sandbox.WorkspaceRoot)
	if root == "" {
		if wd, err := os.Getwd(); err == nil {
			root = wd
		} else {
			root = "."
		}
	}
	roots := []string{root}
	for _, d := range c.Sandbox.AllowWrite {
		if d = ExpandVars(d); d != "" {
			roots = append(roots, d)
		}
	}
	return roots
}

// BashMode normalises the bash-sandbox mode: only an explicit "off" disables
// it; empty or any other value resolves to "enforce", so the sandbox is on by
// default and fails safe.
func (c *Config) BashMode() string {
	if c.Sandbox.Bash == "off" {
		return "off"
	}
	return "enforce"
}

// AgentConfig configures the harness loop. SubagentModel is the optional default
// for runAs=subagent skills; SubagentModels overrides it per skill name.
type AgentConfig struct {
	SystemPrompt     string            `toml:"system_prompt"`
	SystemPromptFile string            `toml:"system_prompt_file"`
	MaxSteps         int               `toml:"max_steps"` // tool-call rounds per turn; 0 = unlimited
	Temperature      float64           `toml:"temperature"`
	SubagentModel    string            `toml:"subagent_model"`
	SubagentModels   map[string]string `toml:"subagent_models"`
	// SubagentTemperature overrides Temperature for task-tool sub-agents.
	// 0 means "use Temperature". Negative means "use Temperature".
	SubagentTemperature float64 `toml:"subagent_temperature"`
	// Effort overrides the reasoning effort for the executor.
	// "" means provider default. For DeepSeek: "high" (default) or "max".
	Effort string `toml:"effort"`
	// SubagentEffort overrides Effort for task-tool sub-agents.
	// "" means "use Effort" (or provider default). For DeepSeek: "high" or "max".
	SubagentEffort string `toml:"subagent_effort"`
	// startup (a built-in like "explanatory"/"learning"/"concise", or a custom
	// .gaea/output-styles/<name>.md). Empty = the unmodified prompt.
	OutputStyle string `toml:"output_style"`
	// ApprovalTimeoutSecs 是工具审批等待超时（C4 TimedOut，蒸馏 codex
	// ReviewDecision::TimedOut）：无人值守场景下审批请求等待超过该秒数按拒绝
	// 处理并发 Notice（回合继续，不静默放行）。0 = 不超时（默认，交互等待）。
	ApprovalTimeoutSecs int `toml:"approval_timeout_secs"`
}

// SubagentTemp returns the effective temperature for task-tool sub-agents.
// Falls back to Temperature when SubagentTemperature is zero or negative.
func (a AgentConfig) SubagentTemp() float64 {
	if a.SubagentTemperature > 0 {
		return a.SubagentTemperature
	}
	return a.Temperature
}

// SubagentEffortVal returns the effective reasoning effort for sub-agents.
// Falls back to Effort when SubagentEffort is empty.
func (a AgentConfig) SubagentEffortVal() string {
	if a.SubagentEffort != "" {
		return a.SubagentEffort
	}
	return a.Effort
}

// ProviderEntry declares a model provider instance. ContextWindow is the model's
// token budget; the harness compacts older history as a turn's prompt approaches
// it (see agent compaction). 0 disables compaction for the instance.
type ProviderEntry struct {
	Name          string            `toml:"name"`
	Kind          string            `toml:"kind"`
	BaseURL       string            `toml:"base_url"`
	Model         string            `toml:"model"`   // a single model (back-compat)
	Models        []string          `toml:"models"`  // a vendor's model list (one base_url/key, many models)
	Default       string            `toml:"default"` // default model when Models is set (else Models[0])
	APIKeyEnv     string            `toml:"api_key_env"`
	BalanceURL    string            `toml:"balance_url"` // optional; a provider-specific wallet-balance endpoint (DeepSeek: https://api.deepseek.com/user/balance). Empty = no balance readout.
	// BalanceKind 是余额查询后端的注册 kind（billing 包按 kind 注册形状；
	// 3.0 Step 3d #8 + Wave 4 收官：从 ProviderEntry 贯通到 controller，不再
	// 硬编码 deepseek）。空 = 历史默认 "deepseek" 形状。未知 kind fail-closed。
	BalanceKind string `toml:"balance_kind"`
	ContextWindow int               `toml:"context_window"`
	Price         *provider.Pricing `toml:"price"`
	// Prices holds per-model pricing when a single provider exposes multiple
	// models with different rates. The TOML key "prices" maps model names to
	// their Pricing. ModelList() includes these keys and ResolveModel picks
	// the matching Price when resolving "provider/model".
	Prices map[string]*provider.Pricing `toml:"prices"`
	// Thinking / Effort are provider-kind-specific knobs forwarded to the provider
	// via Config.Extra. The anthropic provider reads Thinking="adaptive" to enable
	// extended thinking and Effort ("low".."max") to tune depth. The
	// openai-compatible provider forwards Effort as reasoning_effort for
	// thinking-capable models (e.g. MiMo) and ignores Thinking. Empty = provider default.
	Thinking string `toml:"thinking"`
	Effort   string `toml:"effort"`
}

// ModelList returns the models this provider exposes: the explicit `models` list,
// the single `model` (back-compat), or the keys from `prices` (per-model pricing
// map). Empty if none are set.
func (e *ProviderEntry) ModelList() []string {
	if len(e.Models) > 0 {
		return e.Models
	}
	if e.Model != "" {
		return []string{e.Model}
	}
	if len(e.Prices) > 0 {
		out := make([]string, 0, len(e.Prices))
		for k := range e.Prices {
			out = append(out, k)
		}
		return out
	}
	return nil
}

// DefaultModel returns the provider's default model: the explicit `default`, else
// the first of ModelList.
func (e *ProviderEntry) DefaultModel() string {
	if e.Default != "" {
		return e.Default
	}
	if l := e.ModelList(); len(l) > 0 {
		return l[0]
	}
	return ""
}

// HasModel reports whether m is one of the provider's models.
func (e *ProviderEntry) HasModel(m string) bool {
	for _, x := range e.ModelList() {
		if x == m {
			return true
		}
	}
	return false
}

// ToolsConfig selects which built-in tools are enabled. Empty means all of them.
type ToolsConfig struct {
	Enabled []string `toml:"enabled"`
	// Compact enables V6.0 P8 reduced toolset (hides redundant tools from model view).
	// Hidden tools remain callable by name but don't appear in the schema list,
	// reducing model cognitive load from ~41 to ~25 visible tools.
	Compact bool `toml:"compact"`
}

// PermissionsConfig declares the per-call permission policy (see
// internal/permission). Mode is the fallback decision for writer tools when no
// rule matches ("ask" | "allow" | "deny"; default "ask"); read-only tools always
// fall back to allow. Allow/Ask/Deny are rule lists of the form "ToolName" or
// "ToolName(glob)". Precedence: deny > ask > allow > fallback.
type PermissionsConfig struct {
	Mode  string   `toml:"mode"`
	Allow []string `toml:"allow"`
	Ask   []string `toml:"ask"`
	Deny  []string `toml:"deny"`
}

// PluginEntry declares an external MCP server. Type selects the transport:
// "stdio" (default) launches Command/Args/Env as a subprocess; "http"
// (a.k.a. streamable-http) and "sse" connect to a remote URL with optional
// static Headers. String fields support ${VAR} / ${VAR:-default} expansion so
// secrets (bearer tokens, keys) come from the environment, not the file. The
// fields mirror Claude Code's mcpServers spec, so entries can come from either
// gaea.toml's [[plugins]] or a project-root .mcp.json (see loadMCPJSON).
type PluginEntry struct {
	Name    string            `toml:"name"`
	Type    string            `toml:"type"` // "stdio" (default) | "http" | "sse"
	Command string            `toml:"command"`
	Args    []string          `toml:"args"`
	Env     map[string]string `toml:"env"`
	URL     string            `toml:"url"`
	Headers map[string]string `toml:"headers"`
	// AutoStart controls whether the server connects during session startup.
	// Nil preserves historical behavior: configured servers start automatically.
	AutoStart *bool `toml:"auto_start"`
}

func (e PluginEntry) ShouldAutoStart() bool {
	return e.AutoStart == nil || *e.AutoStart
}

func (c *Config) AutoStartPlugins() []PluginEntry {
	out := make([]PluginEntry, 0, len(c.Plugins))
	for _, p := range c.Plugins {
		if p.ShouldAutoStart() {
			out = append(out, p)
		}
	}
	return out
}

// DefaultSystemPrompt is used when config provides none.
const DefaultSystemPrompt = `你是 gaea（盖亚）——用户的通用办公 AI 助手，也是日常 AI 伙伴。你沉稳、清晰、可靠，温和而不说教。
坦诚：你是 AI，不冒充人类、不编造出身；但你认真对待每一次对话，把用户的事放在心上。
可靠：先理解再动手，条理清楚；答应的事会做到，做不到会直说。
有温度：说话自然亲切，不甜腻、不客套、不空夸；该提醒的风险会提醒，该追问的需求会追问。
知之为知之：不知道就明说，然后主动去查、去验证，绝不编造。
沟通：使用简洁的中文，结论先行；需要时用列表、表格让信息一目了然；赞美要具体、基于事实。
所有思考和输出必须使用中文。

你负责协助用户完成日常办公工作：文档撰写与编辑、格式转换、表格与数据处理、
图表制作、演示文稿、资料检索、方案与报告编写、任务跟踪等。
使用提供的工具读取和写入文件以及运行 shell 命令。

**原则：**
- 理解请求后再行动；用工具验证而非猜测；保持变更最小且正确；完成后简要总结。
- 执行后必验证：每完成一个步骤，在 complete_step 之前先用工具验证结果正确性（检查生成文件、核对数据、确认输出格式）。
- 遇到用户真正需要决策的问题时（方案选择、范围、影响重大的判断），使用 ask 工具列出 2-4 个具体选项，不要猜测或把问题埋在文字里。有明确默认值时直接选择，不要为了确认而提问。
- 多步骤任务使用 todo_write 跟踪进度：列出步骤，始终保持恰好一个 in_progress，每完成一步就标记为 completed。随时更新列表，不要等到最后。
- 所有独立操作必须在一个响应中完成：并行读取多个文件、编辑不同文件、运行 shell 命令。只有顺序操作（编辑+验证同一文件、任务子代理）才分开发送。工具系统支持非冲突工具的并行执行——积极利用。
- 输出风格强调结构化文档、表格和计算过程，保持清晰可追溯。

**办公规范：**
- 长文档/表格/PDF 的创建与编辑交给已安装的 docx / xlsx / pdf 技能（run_skill 调用），agent 不自造文档格式
- 不同来源的文档统一用 format_convert 转为可编辑 Markdown 后再处理；表格数据也可用 bash + python（openpyxl/pandas）提取
- 图表用 chart_gen 生成（bar/line/pie/scatter）
- 报告先列结构大纲，再逐步填充；多份文档拼装用 doc-assemble 子代理
- 需要最新资料时用 web_search / web_fetch 检索并注明来源

**本地工具：**
- vision：识别图片内容（布局/对象/图表含义）。本地视觉模型，通常几秒，冷启动（模型未加载）约 20 秒+
- ocr：提取图片/扫描件中的文字。本地 OCR 服务，通常 2-5 秒/页，冷启动可能更久
- semantic_search：在成本库/工程知识库/办公记忆中按语义检索；scope=file 时检索工作区已索引文件。本地向量检索，通常 1-3 秒
- format_convert：docx/xlsx/pptx/pdf → Markdown（扫描件走 OCR 回退）。按文档大小数秒到数十秒
- chart_gen：统计图表（bar/line/pie/scatter）；diagram：流程图/时序图/甘特图等 Mermaid 图
- screen_capture：截取屏幕；image_gen：生成图片
- routine_llm：通用文本处理（摘要、归一化、抽取、改写等），目标模型在模型中心「常规办公」绑定，默认本地，可绑定免费云端模型
以上工具均在本地/免费运行，不消耗主模型 token。是否使用、何时使用，由你自行判断。

**子代理：**
task 工具可派发隔离子代理。以下场景优先使用子代理：
- 需把 docx/xlsx/pdf 转成 Markdown：用 format-convert 子代理
- 需从数据生成统计图表：用 chart-builder 子代理
- 需把多份文档拼装成完整报告：用 doc-assemble 子代理
子代理在独立上下文中运行——其工具调用不会撑大你的上下文。犹豫时直接派发。内置子代理技能（format-convert/chart-builder/doc-assemble）见下方 Skills 索引，用 run_skill 按名称调用或直接用 task。

**记忆：**
用 remember/forget 跨会话持久化事实：
- 用户纠正偏好或事实：记住，避免后续重复犯错
- 发现非显而易见的项目事实（关键参数、约定、决策依据）：记住供后续参考
- 记忆被证明错误：用 forget 删除
不要记录瞬时状态或用户明确要求不保存的内容。记忆是持久的——只保存跨会话不变的事实。`

// LanguagePolicy is the forced language directive appended to the system prompt.
// Always Chinese — the user is a native Chinese speaker and cannot read English.
// Static text, so it stays part of the cache-stable prefix.
const LanguagePolicy = `所有思考过程和输出必须使用中文。不要使用英文——用户看不懂英文。` +
	`代码标识符（变量名、函数名、API 路由、数据库字段名）保持英文，但注释、` +
	`解释、分析、回复全部使用中文。即使收到的消息是英文，也始终用中文回复。`

// Default returns the built-in default configuration (DeepSeek + MiMo presets).
// loaderOverride 由 gaea 办公板块注入：优先于文件加载，用于把
// bridge provider（kind=gaea，走 gaea 模型中心）写入配置。
var loaderOverride func() (*Config, error)

// SetLoader 注入配置加载器。传入 nil 恢复默认文件加载。
func SetLoader(f func() (*Config, error)) { loaderOverride = f }

func Default() *Config {
	return &Config{
		DefaultModel: "deepseek-flash",
		Agent: AgentConfig{
			SystemPrompt: DefaultSystemPrompt,
			// 0 = no step cap: the agent loops until the model gives a final answer,
			// the user cancels, or the provider errors. Context stays bounded by
			// compaction, not by a round count. Set a positive agent.max_steps only
			// if you want a hard guard against runaway.
			MaxSteps: 0,
		},
		// Mode "ask" with no rules keeps `gaea run` autonomous (no TTY → ask
		// resolves to allow) while `gaea chat` prompts before writers. Users add
		// deny/allow rules to harden or quiet specific tools.
		Permissions: PermissionsConfig{Mode: "ask", Allow: []string{"run_skill"}},
		// 办公记忆默认开启；用户在记忆面板可一键关闭（记忆可控性）。
		// 归档保留期默认 90 天（0 = 走默认值，见 memoryRetentionDays）。
		Memory: MemoryConfig{Enabled: true, ArchivedRetentionDays: 90},
		// Sandbox on by default: bash is jailed (macOS), network allowed so
		// builds/downloads work. Set bash = "off" to disable. Network=true here
		// so an absent [sandbox] in a user's file keeps egress (zero value would
		// wrongly deny it).
		Sandbox: SandboxConfig{Bash: "enforce", Network: true},
		// CodeGraph code-intelligence on by default: when it resolves it is injected
		// as a built-in MCP server, and AutoInstall fetches it into the cache on
		// first use. Set enabled = false to opt out, or auto_install = false to
		// require an explicit `gaea codegraph install`.
		Tools: ToolsConfig{Enabled: []string{
			"read_file", "write_file", "edit_file", "edit_lines", "move_file",
			"ls", "grep", "bash",
			"web_fetch", "web_search",
			"todo_write", "complete_step",
			"memory_search",
		}},
		Providers: []ProviderEntry{
			{Name: "deepseek-flash", Kind: "openai", BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash", APIKeyEnv: "DEEPSEEK_API_KEY", BalanceURL: "https://api.deepseek.com/user/balance", ContextWindow: 1_000_000, Price: &provider.Pricing{CacheHit: 0.02, Input: 1, Output: 2, Currency: "¥"}},
			{Name: "deepseek-pro", Kind: "openai", BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-pro", APIKeyEnv: "DEEPSEEK_API_KEY", BalanceURL: "https://api.deepseek.com/user/balance", ContextWindow: 1_000_000, Price: &provider.Pricing{CacheHit: 0.025, Input: 3, Output: 6, Currency: "¥"}},
			{Name: "mimo-pro", Kind: "openai", BaseURL: "https://token-plan-cn.xiaomimimo.com/v1", Model: "mimo-v2.5-pro", APIKeyEnv: "MIMO_API_KEY", ContextWindow: 1_000_000, Price: &provider.Pricing{CacheHit: 0.025, Input: 3, Output: 6, Currency: "¥"}},
			{Name: "mimo-flash", Kind: "openai", BaseURL: "https://token-plan-cn.xiaomimimo.com/v1", Model: "mimo-v2.5", APIKeyEnv: "MIMO_API_KEY", ContextWindow: 1_000_000, Price: &provider.Pricing{CacheHit: 0.02, Input: 1, Output: 2, Currency: "¥"}},
			{Name: "xai-oauth", Kind: "xai", BaseURL: "https://api.x.ai/v1", Model: "grok-4.3", APIKeyEnv: "", ContextWindow: 1_000_000},
		},
	}
}

// Load builds the configuration: defaults, then user config, then project
// config, then any MCP servers from Claude Code's .mcp.json. A .env in the
// working directory is loaded first so api_key_env can resolve.
func Load() (*Config, error) {
	if loaderOverride != nil {
		return loaderOverride()
	}
	loadDotEnv()
	cfg := Default()

	if uc := userConfigPath(); uc != "" {
		if err := mergeFile(cfg, uc); err != nil {
			return nil, err
		}
	}
	if err := mergeFile(cfg, "gaea.toml"); err != nil {
		return nil, err
	}
	// Claude Code's .mcp.json (project root) is read last and merged into
	// [[plugins]], so a server configured for Claude works here unchanged.
	// gaea.toml wins on a name collision (see mergeMCPJSON).
	entries, err := loadMCPJSON(mcpJSONFile)
	if err != nil {
		return nil, err
	}
	cfg.mergeMCPJSON(entries)
	return cfg, nil
}

// LoadForEdit returns a config to seed the `gaea setup` wizard when reconfiguring:
// the built-in defaults with the file at path (if present) decoded on top, so a
// reconfigure preserves the user's existing providers and agent settings instead
// of resetting to defaults. .env is loaded so api_key_env resolution works while
// the wizard decides which keys are still missing.
func LoadForEdit(path string) *Config {
	loadDotEnv()
	cfg := Default()
	if err := mergeFile(cfg, path); err != nil {
		slog.Warn("config: load for edit failed, using defaults", "path", path, "err", err)
	}
	return cfg
}

// mergeFile decodes a TOML file onto cfg if it exists. An absent file is not an error.
func mergeFile(cfg *Config, path string) error {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return fmt.Errorf("config %s: %w", path, err)
	}
	return nil
}

func userConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea", "config.toml")
}

// UserConfigPath is the user-global config file (~/.config/gaea/config.toml),
// or "" when the user config dir can't be resolved.
func UserConfigPath() string { return userConfigPath() }

// ArchiveDir is where compacted conversation history is archived for
// traceability (one timestamped .jsonl per compaction). Empty if the user config
// directory cannot be resolved, in which case archiving is skipped.
func ArchiveDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea", "archive")
}

// SessionDir is where chat sessions are persisted (one .jsonl per session).
// Used by `gaea chat --continue` / `--resume` to find the recent ones. Empty
// if the user config dir can't be resolved — sessions then aren't saved.
func SessionDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea", "sessions")
}

// WorkspaceSessionDir returns the workspace-scoped session directory.
// Sessions are isolated per workspace so switching projects shows only that
// workspace's history.
//
// S2 双空间：space 为 "work"/"play" 时返回分区目录 <cwd>/.gaea/sessions/<space>/；
// space 为 ""（space.mode=off 的回退形态）返回平铺目录 <cwd>/.gaea/sessions/
// （旧行为）。旧平铺会话恒按 work 兼容可读（读端各空间目录 + 平铺兜底）。
func WorkspaceSessionDir(cwd, space string) string {
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			return SessionDir()
		}
	}
	base := filepath.Join(cwd, ".gaea", "sessions")
	switch space {
	case spaces.SpaceWork:
		return filepath.Join(base, spaces.SpaceWork)
	case spaces.SpacePlay:
		return filepath.Join(base, spaces.SpacePlay)
	default:
		return base // "" = 平铺（space.mode=off 回退形态）
	}
}

// MemoryUserDir returns the gaea user config root (…/gaea), under which
// the user-global TIANXUAN.md and the per-project auto-memory store live. Empty
// when the user config dir can't be resolved, which disables user-scoped memory.
func MemoryUserDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea")
}

// ConventionDirs are the parent directories scanned for agent assets (skills,
// commands), in canonical-first order. .gaea is ours; .agents / .agent /
// .claude let users drop in assets authored for other agent tools without moving
// files. Shared so skills (internal/skill) and commands (CommandDirs) discover
// the same set. Note: hooks are NOT scanned across these — a .claude/settings.json
// uses a different hook schema that can't be parsed as ours, so hooks stay in
// .gaea/settings.json (see internal/hook).
var ConventionDirs = []string{".gaea", ".agents", ".agent", ".claude"}

// conventionSubdirsAsc joins sub under each ConventionDir of base, in ascending
// priority (reverse of ConventionDirs) so the canonical .gaea ends up the
// highest-priority entry — command.Load lets a later directory win on a clash.
func conventionSubdirsAsc(base, sub string) []string {
	out := make([]string, 0, len(ConventionDirs))
	for i := len(ConventionDirs) - 1; i >= 0; i-- {
		out = append(out, filepath.Join(base, ConventionDirs[i], sub))
	}
	return out
}

// CommandDirsAt returns the directories scanned for custom slash commands,
// lowest priority first, so a later (more specific) directory overrides an
// earlier one on a name clash. Order: home-dir convention dirs
// (~/.claude/commands … ~/.gaea/commands), the legacy XDG user dir
// (~/.config/gaea/commands), then the project's convention dirs under cwd
// (.claude/commands … .gaea/commands). Scanning the .claude / .agents / .agent
// dirs lets commands authored for other agent tools (same .md + frontmatter
// format) work here unchanged.
func CommandDirsAt(cwd string) []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, conventionSubdirsAsc(home, "commands")...)
	}
	if dir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(dir, "gaea", "commands")) // legacy XDG user dir
	}
	dirs = append(dirs, conventionSubdirsAsc(cwd, "commands")...)
	return dirs
}

// CommandDirs 是基于进程工作目录的命令扫描目录（兼容 CLI 场景）。
func CommandDirs() []string {
	return CommandDirsAt(".")
}

// SourcePath returns the highest-priority config file that exists, or "" if none.
func SourcePath() string {
	if _, err := os.Stat("gaea.toml"); err == nil {
		return "gaea.toml"
	}
	if uc := userConfigPath(); uc != "" {
		if _, err := os.Stat(uc); err == nil {
			return uc
		}
	}
	return ""
}

// WriteFile writes the configuration to path as annotated TOML.
func (c *Config) WriteFile(path string) error {
	return os.WriteFile(path, []byte(RenderTOML(c)), 0o644)
}

// Provider returns the named provider entry.
func (c *Config) Provider(name string) (*ProviderEntry, bool) {
	for i := range c.Providers {
		if c.Providers[i].Name == name {
			return &c.Providers[i], true
		}
	}
	return nil, false
}

// ResolveModel resolves a model reference to a provider entry whose Model is the
// selected model string (a copy, so the config's lists stay intact). It accepts:
//   - "provider/model" — that exact model under that provider;
//   - a provider name   — the provider's default model;
//   - a bare model name — the (first) provider that lists it.
//
// The returned entry is ready to build a provider from (NewProvider reads .Model),
// so a single "vendor with many models" entry yields one instance per model
// without duplicating base_url/api_key_env. Single-`model` entries still resolve
// by provider name, keeping older configs working unchanged.
func (c *Config) ResolveModel(ref string) (*ProviderEntry, bool) {
	if ref == "" {
		return nil, false
	}
	// "provider/model"
	if prov, model, ok := strings.Cut(ref, "/"); ok {
		if e, found := c.Provider(prov); found && e.HasModel(model) {
			cp := *e
			cp.Model = model
			// If the provider uses per-model pricing (prices map), pick the
			// matching entry. Falls back to cp.Price when no entry or nil.
			if cp.Prices != nil {
				if p, ok := cp.Prices[model]; ok {
					cp.Price = p
				}
			}
			return &cp, true
		}
	}
	// a provider name → its default model
	if e, found := c.Provider(ref); found {
		cp := *e
		cp.Model = e.DefaultModel()
		return &cp, true
	}
	// a bare model name → the provider that lists it
	for i := range c.Providers {
		if c.Providers[i].HasModel(ref) {
			cp := c.Providers[i]
			cp.Model = ref
			return &cp, true
		}
	}
	return nil, false
}

// APIKey resolves the entry's API key from its api_key_env.
func (e *ProviderEntry) APIKey() string {
	if e.APIKeyEnv == "" {
		return ""
	}
	return os.Getenv(e.APIKeyEnv)
}

// Configured reports whether the provider's api_key_env is set — the same check
// Validate enforces, so pickers can filter on it.
func (e *ProviderEntry) Configured() bool {
	return e.APIKey() != ""
}

// ResolveSystemPrompt returns the system prompt, reading system_prompt_file if set.
func (c *Config) ResolveSystemPrompt() (string, error) {
	if c.Agent.SystemPromptFile != "" {
		b, err := os.ReadFile(c.Agent.SystemPromptFile)
		if err != nil {
			return "", fmt.Errorf("system_prompt_file: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	if strings.TrimSpace(c.Agent.SystemPrompt) == "" {
		return DefaultSystemPrompt, nil
	}
	return c.Agent.SystemPrompt, nil
}

// Validate checks that the selected model's provider is usable.
func (c *Config) Validate(model string) error {
	e, ok := c.ResolveModel(model)
	if !ok {
		return fmt.Errorf("unknown model %q (configured: %s)", model, c.providerNames())
	}
	if e.Kind == "" {
		return fmt.Errorf("provider %q: kind is required", model)
	}
	if e.BaseURL == "" {
		return fmt.Errorf("provider %q: base_url is required", model)
	}
	// OAuth providers (kind=xai) manage their own credentials — no api_key_env needed.
	if e.Kind == "xai" {
		return nil
	}
	if e.APIKey() == "" {
		return fmt.Errorf("provider %q: missing env %s", model, e.APIKeyEnv)
	}
	return nil
}

func (c *Config) providerNames() string {
	names := make([]string, len(c.Providers))
	for i, p := range c.Providers {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}
