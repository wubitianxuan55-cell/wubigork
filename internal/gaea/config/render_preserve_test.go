package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderTOMLPreservingKeepsUnknownSections P0-1 刀：RenderTOMLPreserving 必须
// 把「渲染器不负责的顶层内容」逐字留下——未建模/未渲染的段（[memory]/[dream]）、
// 顶层标量（workspace）以及用户自定义注释，一个都不许丢。渲染器负责的段
// （[agent]/[[providers]]/[tools]/[permissions]/[sandbox]/[[plugins]]）走新渲染结果，
// 不在保留范围内。
func TestRenderTOMLPreservingKeepsUnknownSections(t *testing.T) {
	existing := strings.Join([]string{
		"default_model = \"old-model\"",
		"language = \"zh\"",
		"workspace = \"D:/ws\"",
		"# 用户手写的顶层注释：渲染器不认识，必须原样保留",
		"",
		"[agent]",
		"system_prompt = \"\"",
		"max_steps = 0",
		"temperature = 0.7",
		"",
		"[memory]",
		"enabled = false",
		"",
		"[dream]",
		"mode = \"x\"",
		"",
	}, "\n")

	c := Default()
	c.Workspace = "D:/ws"

	out := RenderTOMLPreserving(c, existing)

	for _, want := range []string{
		"workspace = \"D:/ws\"",
		"# 用户手写的顶层注释：渲染器不认识，必须原样保留",
		"[memory]",
		"enabled = false",
		"[dream]",
		"mode = \"x\"",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("保留式渲染丢了字面内容 %q\n--- 输出 ---\n%s", want, out)
		}
	}
	// 渲染器负责的段用新渲染结果：旧值不得残留，default_model 取内存里的新值。
	if strings.Contains(out, "old-model") {
		t.Errorf("渲染器负责的 default_model 应来自内存，不该保留旧文件值\n%s", out)
	}
	if !strings.Contains(out, "default_model = \"deepseek-flash\"") {
		t.Errorf("新渲染的 default_model 缺失\n%s", out)
	}
	if strings.Count(out, "[agent]") != 1 {
		t.Errorf("[agent] 段应恰好出现一次，实际 %d 次\n%s", strings.Count(out, "[agent]"), out)
	}
	if strings.Count(out, "[memory]") != 1 {
		t.Errorf("[memory] 段应恰好出现一次，实际 %d 次\n%s", strings.Count(out, "[memory]"), out)
	}
	// existing 为空 = 与 RenderTOML 等价（零回归）。
	if got, want := RenderTOMLPreserving(c, ""), RenderTOML(c); got != want {
		t.Errorf("existing 为空时应逐字节等价 RenderTOML\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestSaveRoundTripKeepsAllSections 钱测试（P0-1）：真实形态的配置（渲染器不负责的
// 13 段全带非零值）经 Load → 改一处（AddPermissionRuleForSpace，等价「始终允许」
// 审批的 PersistAllowRule）→ Save/SaveTo → 再 Load，逐字段不许变。
// 这条在修复前必红：RenderTOML 只渲染 8 段，其余段在第一次 Save 就从磁盘上消失。
func TestSaveRoundTripKeepsAllSections(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", home) // Windows os.UserConfigDir
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir()) // SourcePath() 先找 ./gaea.toml —— 隔离到空目录

	path := filepath.Join(home, "gaea", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(roundTripConfigTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := Load()
	if err != nil {
		t.Fatalf("首次 Load: %v", err)
	}
	if first.Workspace != "D:\\ws" {
		t.Fatalf("前置条件失败：workspace = %q", first.Workspace)
	}
	// 改一处再落盘：与 boot.go PersistAllowRule 的「始终允许」审批同路径。
	if err := first.AddPermissionRuleForSpace("work", "allow", "write_file"); err != nil {
		t.Fatalf("AddPermissionRuleForSpace: %v", err)
	}
	if err := first.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("二次 Load: %v", err)
	}

	// ── 顶层标量 ──
	if got.Workspace != first.Workspace {
		t.Errorf("workspace 丢: got %q want %q", got.Workspace, first.Workspace)
	}
	if got.DefaultModel != first.DefaultModel {
		t.Errorf("default_model 变: got %q want %q", got.DefaultModel, first.DefaultModel)
	}
	if got.Language != first.Language {
		t.Errorf("language 变: got %q want %q", got.Language, first.Language)
	}

	// ── [session] / [space] ──
	if got.Session != first.Session {
		t.Errorf("[session] 丢字段: got %+v want %+v", got.Session, first.Session)
	}
	if got.Space != first.Space {
		t.Errorf("[space] 丢字段: got %+v want %+v", got.Space, first.Space)
	}

	// ── [memory] / [dream] ──
	if got.Memory != first.Memory {
		t.Errorf("[memory] 丢字段: got %+v want %+v", got.Memory, first.Memory)
	}
	if got.Dream != first.Dream {
		t.Errorf("[dream] 丢字段: got %+v want %+v", got.Dream, first.Dream)
	}

	// ── [network] + [network.proxy] ──
	if got.Network != first.Network {
		t.Errorf("[network]/[network.proxy] 丢字段: got %+v want %+v", got.Network, first.Network)
	}
	if got.Network.Proxy.Server != "127.0.0.1" || got.Network.Proxy.Port != 7890 || got.Network.Proxy.Username != "u" {
		t.Errorf("[network.proxy] 子段丢: %+v", got.Network.Proxy)
	}

	// ── [search] / [tasks] / [retrieval] / [vision] / [skills] ──
	if got.Search.TimeoutSeconds != 42 || got.Search.LocalSearXNGURL != "http://127.0.0.1:8888" {
		t.Errorf("[search] 丢字段: %+v", got.Search)
	}
	if len(got.Search.EngineOrder) != 2 || got.Search.EngineOrder[0] != "local-searxng" {
		t.Errorf("[search].engine_order 丢: %v", got.Search.EngineOrder)
	}
	if got.Tasks.MaxConcurrent != 3 {
		t.Errorf("[tasks] 丢字段: %+v", got.Tasks)
	}
	if got.Tasks.PerSpace["work"] != 2 || got.Tasks.PerSpace["play"] != 1 {
		t.Errorf("[tasks].per_space 丢: %v", got.Tasks.PerSpace)
	}
	if got.Tasks.Priority["price_fetch"] != 20 {
		t.Errorf("[tasks].priority 丢: %v", got.Tasks.Priority)
	}
	if got.Retrieval.EmbedKind != "openai" || got.Retrieval.EmbedModel != "bge-m3" || got.Retrieval.RerankModel != "bge-reranker-v2-m3" {
		t.Errorf("[retrieval] 丢字段: %+v", got.Retrieval)
	}
	if got.Vision.Kind != "openai" || got.Vision.Model != "qwen3-vl" {
		t.Errorf("[vision] 丢字段: %+v", got.Vision)
	}
	if got.MarkdownConverter.Kind != "cli" {
		t.Errorf("[markdown_converter] 丢字段: %+v", got.MarkdownConverter)
	}
	if len(got.Skills.Paths) != 2 || got.Skills.Paths[1] != "D:\\skills2" {
		t.Errorf("[skills] 丢字段: %v", got.Skills.Paths)
	}

	// ── 用户自定义注释也逐字留 ──
	if !strings.Contains(string(raw), "# 用户手写注释：不要删我") {
		t.Errorf("落盘文件丢了用户自定义注释\n%s", raw)
	}

	// ── 改的那一处必须真的生效（不能靠「什么都不写」骗过断言）──
	if !strings.Contains(string(raw), `"write_file"`) {
		t.Errorf("新加的 allow 规则没落盘\n%s", raw)
	}
	if got.SpaceProfiles["work"].Permissions == nil {
		t.Fatalf("[space_profiles.work.permissions] 丢段")
	}
	if !containsStr(got.SpaceProfiles["work"].Permissions.Allow, "write_file") {
		t.Errorf("新规则没进空间段: %v", got.SpaceProfiles["work"].Permissions.Allow)
	}

	// ── 段头逐个仍在（P0#2 收尾）：字段级断言只保「已建模字段」，这层保
	// 「原文的每个段，保存后表头仍逐字在场」——任何一段被静默整段删除即红。
	for _, ln := range strings.Split(roundTripConfigTOML, "\n") {
		if name, ok := tableHeaderName(ln); ok {
			if !strings.Contains(string(raw), ln) {
				t.Errorf("落盘文件丢了段头 %q（P0#2 段级保真）\n%s", name, raw)
			}
		}
	}
}

// TestSaveRoundTripKeepsProfileGuardrailsAndPlugins P0#2 收尾：审批链
// （Load → AddPermissionRuleForSpace → Save）落盘后，渲染器建模较浅的键级
// 合并面必须零漂移——space_profiles.<space>.permissions 的未建模键与
// hard_ask/approval_timeout_secs、space_profiles.<space>.guardrails、
// [[plugins]] 条目（含 headers 内联表与 auto_start）。
func TestSaveRoundTripKeepsProfileGuardrailsAndPlugins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir()) // SourcePath() 先找 ./gaea.toml —— 隔离到空目录

	path := filepath.Join(home, "gaea", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	src := `default_model = "deepseek-flash"

[space_profiles.work.permissions]
mode = "ask"
hard_ask = ["deploy_prod"]
approval_timeout_secs = 42
allow = ["read_file"]
future_perm_key = "keep-me"

[space_profiles.work.guardrails]
enabled = true
temperature_max = 0.8
max_output_tokens = 4096
image_safe_mode = true

[[plugins]]
name = "mcp-fs"
type = "stdio"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-fs"]
auto_start = false

[[plugins]]
name = "mcp-http"
type = "http"
url = "https://mcp.example/s"
headers = { X-Token = "t" }
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := Load()
	if err != nil {
		t.Fatalf("首次 Load: %v", err)
	}
	if err := c.AddPermissionRuleForSpace("work", "allow", "write_file"); err != nil {
		t.Fatalf("AddPermissionRuleForSpace: %v", err)
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("二次 Load: %v", err)
	}

	perm := got.SpaceProfiles["work"].Permissions
	if perm == nil {
		t.Fatalf("[space_profiles.work.permissions] 丢段\n%s", raw)
	}
	if perm.Mode != "ask" || perm.ApprovalTimeoutSecs != 42 || !containsStr(perm.HardAsk, "deploy_prod") {
		t.Errorf("permissions 建模键丢: %+v", perm)
	}
	if !containsStr(perm.Allow, "write_file") || !containsStr(perm.Allow, "read_file") {
		t.Errorf("allow 应新旧并存: %v", perm.Allow)
	}
	gr := got.SpaceProfiles["work"].Guardrails
	if gr == nil || !gr.Enabled || gr.TemperatureMax != 0.8 || gr.MaxOutputTokens != 4096 || !gr.ImageSafeMode {
		t.Errorf("guardrails 丢/漂移: %+v\n%s", gr, raw)
	}
	if len(got.Plugins) != 2 {
		t.Fatalf("[[plugins]] 应 2 条，实得 %d\n%s", len(got.Plugins), raw)
	}
	p0, p1 := got.Plugins[0], got.Plugins[1]
	if p0.Name != "mcp-fs" || p0.Command != "npx" || len(p0.Args) != 2 || p0.AutoStart == nil || *p0.AutoStart {
		t.Errorf("plugins[0] 漂移: %+v\n%s", p0, raw)
	}
	if p1.Name != "mcp-http" || p1.Type != "http" || p1.URL != "https://mcp.example/s" || p1.Headers["X-Token"] != "t" {
		t.Errorf("plugins[1] 漂移: %+v\n%s", p1, raw)
	}
	if !strings.Contains(string(raw), "future_perm_key") {
		t.Errorf("permissions 未建模键丢（键级合并面）\n%s", raw)
	}
}

// containsStr 报告 ss 是否含 s。
func containsStr(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// roundTripConfigTOML 是「用户真实配置」形态的样本：渲染器不负责的 13 段全带
// 非零值 + 用户自定义注释 + 顶层 workspace 标量。
const roundTripConfigTOML = `default_model = "deepseek-flash"
language = "zh"
workspace = "D:\\ws"
# 用户手写注释：不要删我

[agent]
system_prompt = "你是 gaea。\n第二行。"
max_steps = 7
temperature = 0.5

[[providers]]
name = "deepseek-flash"
kind = "openai"
base_url = "https://api.deepseek.com"
model = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
context_window = 1000000

[tools]
enabled = ["read_file", "write_file"]

[permissions]
mode = "ask"
allow = ["run_skill"]

[space_profiles.work.permissions]
allow = []

[sandbox]
bash = "off"
network = false

[session]
log_format = "legacy"
space = "work"

[space]
mode = "on"

[skills]
paths = ["C:\\skills1", "D:\\skills2"]

[search]
local_searxng_url = "http://127.0.0.1:8888"
tavily_api_key_env = "TAVILY_KEY"
brave_api_key_env = "BRAVE_KEY"
timeout_seconds = 42
engine_order = ["local-searxng", "tavily"]

[network]
proxy_mode = "custom"
proxy_url = "http://127.0.0.1:7890"
no_proxy = "localhost,127.0.0.1"
  [network.proxy]
  type = "socks5"
  server = "127.0.0.1"
  port = 7890
  username = "u"
  password = "p"

[memory]
enabled = false
archived_retention_days = 7

[tasks]
max_concurrent = 3
per_space = { work = 2, play = 1 }
priority = { price_fetch = 20, file_index = 10 }

[retrieval]
embed_kind = "openai"
embed_base_url = "http://localhost:8080"
embed_model = "bge-m3"
rerank_kind = "openai"
rerank_base_url = "http://localhost:8080"
rerank_model = "bge-reranker-v2-m3"

[vision]
kind = "openai"
base_url = "http://127.0.0.1:8080/v1"
model = "qwen3-vl"

[markdown_converter]
kind = "cli"

[dream]
mode = "auto"
`

// TestSavePreservesUnrenderedKeysInKnownSections P0-1 刀（键级兜底）：段级保留只堵了
// 一半的洞——渲染器负责的段（[agent]/[tools]/[permissions]/[[providers]]）内部，
// 它忘了渲染的键在整文件重写下同样会静默消失。这条测试用「渲染器已知不渲染的键」
// 逐个对账：Load → Save（SaveTo）→ 再 Load，键与落盘文本都必须保住。
// 修复前必红。
func TestSavePreservesUnrenderedKeysInKnownSections(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir())

	path := filepath.Join(home, "gaea", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(knownSectionKeysTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := Load()
	if err != nil {
		t.Fatalf("首次 Load: %v", err)
	}
	// 前置条件：这些键确实被 Load 读进来了（否则测试没意义）。
	if first.Agent.Effort != "high" || first.Agent.SubagentEffort != "low" || first.Agent.ApprovalTimeoutSecs != 42 {
		t.Fatalf("前置条件失败：[agent] 未渲染键没读进来 %+v", first.Agent)
	}
	if !first.Tools.Compact {
		t.Fatalf("前置条件失败：[tools] compact 没读进来")
	}
	if first.Agent.ToolSpill == nil || !*first.Agent.ToolSpill {
		t.Fatalf("前置条件失败：[agent] tool_spill 没读进来 %+v", first.Agent.ToolSpill)
	}
	if err := first.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("二次 Load: %v", err)
	}

	// ── 值级断言（崩在哪个键一目了然）──
	if got.Agent.Effort != "high" {
		t.Errorf("[agent] effort 丢: got %q want high", got.Agent.Effort)
	}
	if got.Agent.SubagentEffort != "low" {
		t.Errorf("[agent] subagent_effort 丢: got %q want low", got.Agent.SubagentEffort)
	}
	if got.Agent.ApprovalTimeoutSecs != 42 {
		t.Errorf("[agent] approval_timeout_secs 丢: got %d want 42", got.Agent.ApprovalTimeoutSecs)
	}
	if !got.Tools.Compact {
		t.Errorf("[tools] compact 丢: got false want true")
	}
	if got.Agent.ToolSpill == nil || !*got.Agent.ToolSpill {
		t.Errorf("[agent] tool_spill 丢: got %v want true", got.Agent.ToolSpill)
	}
	p1, ok := got.Provider("p1")
	if !ok {
		t.Fatalf("provider p1 丢")
	}
	if p1.Thinking != "adaptive" {
		t.Errorf("[[providers]] p1 thinking 丢: got %q want adaptive", p1.Thinking)
	}
	if p1.Effort != "high" {
		t.Errorf("[[providers]] p1 effort 丢: got %q want high", p1.Effort)
	}
	if sz := len(got.Providers); sz != 2 {
		t.Errorf("provider 条目数应保持 2（p1 name=p2 渲染条目），got %d", sz)
	}
	if _, ok := got.Provider("p2"); !ok {
		t.Errorf("原有的第二个 provider 条目丢了")
	}

	// ── 落盘文本级断言（键还在文件里，不是只留在内存）──
	for _, want := range []string{
		"effort = \"high\"",
		"subagent_effort = \"low\"",
		"approval_timeout_secs = 42",
		"tool_spill = true",
		"compact = true",
		"thinking = \"adaptive\"",
		// Config 里没有建模的键（模拟「将来新增字段」）：也必须逐字留在文件里。
		"future_top_key = \"keep-me\"",
	} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("落盘文件里丢了键 %q\n--- 文件 ---\n%s", want, raw)
		}
	}
	// [permissions] 段内的未建模键不能被塞进别的段：它必须仍出现在 [permissions] 头后。
	if i, j := strings.Index(string(raw), "[permissions]"), strings.Index(string(raw), "future_top_key"); i < 0 || j < i {
		t.Errorf("[permissions] 的未建模键位置不对（不应被挪进别的段）")
	}
}

// knownSectionKeysTOML 样本：known 段内部的未渲染键 + 一个「不匹配任何渲染条目」
// 的 provider 子段（渲染器按 name 配对，配得上的补键、配不上的整条保留）。
const knownSectionKeysTOML = `default_model = "deepseek-flash"

[agent]
system_prompt = ""
max_steps = 0
temperature = 0.5
effort = "high"
subagent_effort = "low"
approval_timeout_secs = 42
tool_spill = true

[[providers]]
name = "p1"
kind = "openai"
base_url = "https://api.deepseek.com"
model = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
thinking = "adaptive"
effort = "high"

[[providers]]
name = "p2"
kind = "openai"
base_url = "https://api.deepseek.com"
model = "deepseek-v4-pro"
api_key_env = "DEEPSEEK_API_KEY"

[tools]
enabled = ["read_file"]
compact = true

[permissions]
mode = "ask"
# 模拟「将来新增字段而渲染器忘了渲染」：Config 没建模、渲染器也不写，
# 键级兜底必须把它逐字留在 [permissions] 段里。
future_top_key = "keep-me"

# 用户注释：[sandbox] 段头注释也要活下来
[sandbox]
bash = "off"
network = false

[dream]
mode = "suggest"
`

// realShapeTOML 是用户真实配置 %APPDATA%\gaea\config.toml 的「段形状」复刻
// （键位/顺序/缩进照抄，值脱敏）。修复前实测这份配置经一次 Save 会从 16 段
// 掉到 5 段（丢 11 段）。这条测试把「16 段全在」钉死。
const realShapeTOML = `default_model = "gaea"
language = ""
workspace = "D:\\ws"

[agent]
system_prompt = "你是 gaea。\n第二行。"
system_prompt_file = ""
max_steps = 0
temperature = 0.42
subagent_model = ""
subagent_temperature = 0.0
effort = ""
subagent_effort = ""
output_style = ""
approval_timeout_secs = 0

[session]
log_format = ""
space = "work"

[space]
mode = ""

[[providers]]
name = "gaea"
kind = "wubigrok"
base_url = ""
model = ""
default = ""
api_key_env = ""
balance_url = ""
balance_kind = ""
context_window = 256000
thinking = ""
effort = ""

[tools]
compact = false

[permissions]
mode = "auto"
allow = ["run_skill"]

[sandbox]
workspace_root = "D:\\ws"
bash = "off"
network = false

[skills]

[search]
local_searxng_url = ""
tavily_api_key_env = ""
brave_api_key_env = ""
timeout_seconds = 0

[network]
proxy_mode = ""
proxy_url = ""
no_proxy = ""
  [network.proxy]
  type = ""
  server = ""
  port = 0
  username = ""
  password = ""

[memory]
enabled = true
archived_retention_days = 90

[tasks]
max_concurrent = 0

[retrieval]
embed_kind = ""
embed_base_url = ""
embed_model = ""
rerank_kind = ""
rerank_base_url = ""
rerank_model = ""

[vision]
kind = ""
base_url = ""
model = ""

[markdown_converter]
kind = ""

[dream]
mode = "suggest"
`

// TestSaveKeepsRealConfigShape 用用户真实配置的段形状复跑一遍 Load→改一处→Save→
// Load：16 个顶层段（含 workspace 标量与 [network.proxy] 子段）一个都不许丢。
func TestSaveKeepsRealConfigShape(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir())

	path := filepath.Join(home, "gaea", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(realShapeTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Workspace != "D:\\ws" {
		t.Fatalf("前置条件失败：workspace = %q", cfg.Workspace)
	}
	// 改一处（等价 boot.go PersistAllowRule 的「始终允许」审批）再落盘。
	if err := cfg.AddPermissionRuleForSpace("work", "allow", "bash(ls*)"); err != nil {
		t.Fatalf("AddPermissionRuleForSpace: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("二次 Load: %v", err)
	}

	// 16 个顶层段全部还在（段级）。
	for _, sec := range []string{
		"[agent]", "[session]", "[space]", "[[providers]]", "[tools]", "[permissions]",
		"[sandbox]", "[skills]", "[search]", "[network]", "[network.proxy]", "[memory]",
		"[tasks]", "[retrieval]", "[vision]", "[markdown_converter]", "[dream]",
	} {
		if !strings.Contains(string(raw), sec) {
			t.Errorf("落盘后丢了段 %s\n--- 文件 ---\n%s", sec, raw)
		}
	}
	if !strings.Contains(string(raw), `workspace = "D:\\ws"`) {
		t.Errorf("落盘后丢了顶层 workspace 标量")
	}
	// 字段级零漂移。
	if got.Workspace != "D:\\ws" {
		t.Errorf("workspace 漂移: %q", got.Workspace)
	}
	if got.Session.Space != "work" || got.Session.LogFormat != "" {
		t.Errorf("[session] 漂移: %+v", got.Session)
	}
	if got.Memory.Enabled != true || got.Memory.ArchivedRetentionDays != 90 {
		t.Errorf("[memory] 漂移: %+v", got.Memory)
	}
	if got.Dream.Mode != "suggest" {
		t.Errorf("[dream] 漂移: %q", got.Dream.Mode)
	}
	if got.Sandbox.Bash != "off" || got.Sandbox.WorkspaceRoot != "D:\\ws" {
		t.Errorf("[sandbox] 漂移: %+v", got.Sandbox)
	}
	if got.Network.ProxyMode != "" || got.Network.Proxy.Server != "" {
		t.Errorf("[network] 漂移: %+v", got.Network)
	}
	if p, ok := got.Provider("gaea"); !ok || p.ContextWindow != 256000 || p.Kind != "wubigrok" {
		t.Errorf("[[providers]] gaea 漂移: %+v", p)
	}

	// 连存多次必须收敛：保留区不堆叠、段不重复、字段不漂移。
	third := cfg
	for i := 0; i < 3; i++ {
		if err := third.Save(); err != nil {
			t.Fatalf("第 %d 次 Save: %v", i+2, err)
		}
		third, err = Load()
		if err != nil {
			t.Fatalf("第 %d 次 Load: %v", i+2, err)
		}
	}
	raw2, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(raw2), "[[providers]]"); n != 1 {
		t.Errorf("重复保存后 provider 条目数 = %d，应为 1（保留区堆叠了）", n)
	}
	if n := strings.Count(string(raw2), "[network.proxy]"); n != 1 {
		t.Errorf("重复保存后 [network.proxy] 出现 %d 次，应为 1", n)
	}
	if n := strings.Count(string(raw2), `workspace = "D:\\ws"`); n != 1 {
		t.Errorf("重复保存后 workspace 标量出现 %d 次，应为 1", n)
	}
	if third.Workspace != "D:\\ws" || third.Session.Space != "work" || third.Memory.ArchivedRetentionDays != 90 {
		t.Errorf("重复保存后字段漂移: ws=%q memory=%+v", third.Workspace, third.Memory)
	}
}
