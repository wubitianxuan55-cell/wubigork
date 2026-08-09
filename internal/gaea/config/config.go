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
	"github.com/gaea/gaea/internal/netclient"
)

// Config is Tianxuan's runtime configuration.
type Config struct {
	DefaultModel string            `toml:"default_model"`
	Language     string            `toml:"language"` // ui/model language tag (e.g. "zh"); empty = auto-detect from $LANG / $TIANXUAN_LANG
	Workspace    string            `toml:"workspace"` // 办公工作空间目录（空 = 进程启动目录）
	Agent        AgentConfig       `toml:"agent"`
	Providers    []ProviderEntry   `toml:"providers"`
	Tools        ToolsConfig       `toml:"tools"`
	Permissions  PermissionsConfig `toml:"permissions"`
	Sandbox      SandboxConfig     `toml:"sandbox"`
	Plugins      []PluginEntry     `toml:"plugins"`
	Skills       SkillsConfig      `toml:"skills"`
	Search       SearchConfig      `toml:"search"`
	Network      NetworkConfig     `toml:"network"`
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
	ProxyURL  string             `toml:"proxy_url"`   // http://host:port or socks5://host:port
	NoProxy   string             `toml:"no_proxy"`    // comma-separated host suffixes excluded from proxying
	Proxy     NetworkProxyConfig `toml:"proxy"`       // structured alternative to ProxyURL
}

// NetworkProxyConfig is the structured form of proxy configuration.
type NetworkProxyConfig struct {
	Type     string `toml:"type"`     // http | https | socks5 | socks5h
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
	// AutoPlan controls whether interactive turns that look multi-step start in
	// plan mode automatically: "off" disables it, "ask"/"on" enable the gate.
	AutoPlan string `toml:"auto_plan"`
	// AutoPlanClassifier optionally names a provider/model used to classify
	// borderline auto-plan decisions. Empty keeps the zero-cost heuristic path.
	AutoPlanClassifier string `toml:"auto_plan_classifier"`
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
	ContextWindow int               `toml:"context_window"`
	Price         *provider.Pricing `toml:"price"`
	// Prices holds per-model pricing when a single provider exposes multiple
	// models with different rates. The TOML key "prices" maps model names to
	// their Pricing. ModelList() includes these keys and ResolveModel picks
	// the matching Price when resolving "provider/model".
	Prices   map[string]*provider.Pricing `toml:"prices"`
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

// WorkspaceSessionDir returns the workspace-scoped session directory under
// cwd/.gaea/sessions/. Sessions are isolated per workspace so switching
// projects shows only that workspace's history.
func WorkspaceSessionDir(cwd string) string {
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			return SessionDir()
		}
	}
	return filepath.Join(cwd, ".gaea", "sessions")
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

// CommandDirs returns the directories scanned for custom slash commands, lowest
// priority first, so a later (more specific) directory overrides an earlier one
// on a name clash. Order: home-dir convention dirs (~/.claude/commands … ~/.gaea/commands),
// the legacy XDG user dir (~/.config/gaea/commands), then the project's
// convention dirs (.claude/commands … .gaea/commands). Scanning the .claude /
// .agents / .agent dirs lets commands authored for other agent tools (same .md +
// frontmatter format) work here unchanged.
func CommandDirs() []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, conventionSubdirsAsc(home, "commands")...)
	}
	if dir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(dir, "gaea", "commands")) // legacy XDG user dir
	}
	dirs = append(dirs, conventionSubdirsAsc(".", "commands")...)
	return dirs
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
