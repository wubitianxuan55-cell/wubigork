package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// SubagentRunner runs a runAs=subagent skill: it spawns an isolated child loop
// with the skill body as system prompt and `task` as its only input, returning
// the final answer. boot wires this over the agent's sub-agent machinery; nil
// means subagent skills are unavailable in this session (they error rather than
// silently inlining, which would lose the isolation the author asked for).
type SubagentRunner func(ctx context.Context, sk Skill, task string) (string, error)

// InstalledHook fires after install_skill writes a new file, so a host can
// refresh UI (e.g. a skills sidebar) without a reload. nil is fine.
type InstalledHook func(name, path string, scope Scope)

// --- run_skill ---

// CommandEntry 是可经 run_skill 按名调用的项目斜杠命令（.gaea/commands 下的
// 提示词模板）。2026-11-25 办公工具梳理二批 B2：slash_command 并入 run_skill
// ——命令与技能同用名字解析，命令只展开提示词文本（不执行任何东西），也不计入
// 技能调用统计。技能名优先：同名时仍按技能跑（run_skill 72 次真实调用的既有
// 行为不变），命令只在没有同名技能时才被解析到。
type CommandEntry struct {
	Name        string
	Description string
	ArgHint     string
	Render      func(args []string) string
}

type runSkillTool struct {
	// resolve 按名解析完整 Skill（保 RunAs——run_skill 必须拿到完整 Skill，
	// 不是 read_skill 那样的 Body 字符串）。names 供 unknown 错误提示名单。
	resolve func(name string) (Skill, bool)
	names   func() string
	runner  SubagentRunner
	// commands 项目斜杠命令（B2 并入）；commandNames 排序后的名单，清单输出稳定。
	commands     map[string]CommandEntry
	commandNames []string
}

// NewRunSkillTool builds the general skill-invocation tool. runner may be nil
// (subagent skills then error). 解析走 store 活读（装配后 install 的新技能即
// 可用）——办公引擎（无 SkillFilter/SkillExtras）现状逐字节；板块引擎
// （filter/extras 非空）应改用 NewRunSkillToolFromSet 与 read_skill 同源。
func NewRunSkillTool(store *Store, runner SubagentRunner) tool.Tool {
	return &runSkillTool{
		resolve: store.Read,
		names:   func() string { return availableNames(store) },
		runner:  runner,
	}
}

// NewRunSkillToolFromSet 以「可见 Skill 集合」构造 run_skill（2026-11-15 众神殿
// 首刀，pan_1794754879290 裁决）：与 read_skill 共享 sysprompt 产出的同一过滤
// 合并表——extras 内嵌件可达、被 filter 拒绝的名字 fail-closed（unknown 提示
// 名单同样只列可见，不泄露隐藏名）。办公引擎不走此构造（保持 store 活读现状）。
func NewRunSkillToolFromSet(visible []Skill, runner SubagentRunner) tool.Tool {
	return &runSkillTool{
		resolve: func(name string) (Skill, bool) {
			for _, sk := range visible {
				if sk.Name == name {
					return sk, true
				}
			}
			return Skill{}, false
		},
		names: func() string {
			if len(visible) == 0 {
				return "(none — no skills defined)"
			}
			ns := make([]string, len(visible))
			for i, s := range visible {
				ns[i] = s.Name
			}
			return strings.Join(ns, ", ")
		},
		runner: runner,
	}
}

func (*runSkillTool) Name() string { return "run_skill" }

// ReadOnly is false: an invoked subagent skill could call writer tools, so
// classify conservatively to keep the parallel-dispatch path from racing two
// skill runs' writes (mirrors the `task` tool).
func (*runSkillTool) ReadOnly() bool { return false }

// Description 供能力页等非模型面展示；模型实见的是 CompactDescription。
func (*runSkillTool) Description() string {
	return "Invoke a playbook from the Skills index, or a project command from .gaea/commands, by bare name (e.g. 'explore', not the [🧬 subagent] tag). Subagent skills spawn isolated loop — only final answer returns, supply arguments as task. Inline skills and commands fold their text as tool result. Prefer dedicated top-level tools (explore/review/etc) when available."
}

func (*runSkillTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "name":{"type":"string","description":"Skill identifier as it appears in the pinned Skills index (e.g. 'explore', 'review'), or a project command name. Case-sensitive. Just the identifier, not the [🧬 subagent] tag; a leading / is tolerated. \"list\" lists skills and commands."},
  "arguments":{"type":"string","description":"Free-form arguments. For inline skills: appended as an 'Arguments:' line; the skill's own instructions decide how to use them. For commands: substituted into the template ($ARGUMENTS / $1..$N). For subagent skills: REQUIRED — becomes the entire task the subagent receives."}
},
"required":["name"]
}`)
}

func (*runSkillTool) CompactDescription() string {
	return "Invoke a skill or a project command by name (leading / and [..] tags tolerated; name=\"list\" lists them). " +
		"Subagent skills spawn isolated loop; inline skills and commands fold their text as tool result."
}
func (*runSkillTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"arguments":{"type":"string"}},"required":["name"]}`)
}

// WithRunSkillCommands 给本包构造的 run_skill 挂上项目斜杠命令（B2 并入）。
// 传入的不是 run_skill 时原样返回（不报错：装配方不必关心构造分支）。
func WithRunSkillCommands(t tool.Tool, cmds []CommandEntry) tool.Tool {
	rs, ok := t.(*runSkillTool)
	if !ok || len(cmds) == 0 {
		return t
	}
	if rs.commands == nil {
		rs.commands = map[string]CommandEntry{}
	}
	for _, c := range cmds {
		name := strings.TrimPrefix(strings.TrimSpace(c.Name), "/")
		if name == "" {
			continue
		}
		c.Name = name
		rs.commands[name] = c
	}
	rs.commandNames = make([]string, 0, len(rs.commands))
	for n := range rs.commands {
		rs.commandNames = append(rs.commandNames, n)
	}
	sort.Strings(rs.commandNames)
	return rs
}

// CountsAsSkillUse 报告一次调用是否该计入技能用量（调用统计装饰器据此过滤）：
// 项目命令的展开不是技能调用，name="list" 取清单也不是——两者都不记，判据②的
// 「技能零调用」统计不被并入进来的命令污染。
func (t *runSkillTool) CountsAsSkillUse(raw string) bool {
	name := runSkillNameOf(raw)
	if name == "" || strings.EqualFold(name, "list") {
		return false
	}
	_, isCmd := t.commands[name]
	return !isCmd
}

// available 渲染 unknown 报错里的可用名单（技能 + 项目命令）。
func (t *runSkillTool) available() string {
	s := "skills: " + t.names()
	if len(t.commandNames) > 0 {
		s += "; commands: " + strings.Join(t.commandNames, ", ")
	}
	return s
}

// list 列出可按名调用的技能与项目命令（原 slash_command 的 list 动作；清单
// 按需取，不再把命令名单常驻在工具描述里）。
func (t *runSkillTool) list() string {
	var b strings.Builder
	b.WriteString("Callable by name via run_skill — skills: " + t.names() + "\n")
	if len(t.commandNames) == 0 {
		b.WriteString("Project commands: (none in this workspace)\n")
		return b.String()
	}
	b.WriteString("Project commands (prompt templates from .gaea/commands; arguments go into $ARGUMENTS / $1..$N):\n")
	for _, n := range t.commandNames {
		e := t.commands[n]
		line := "- /" + n
		if e.ArgHint != "" {
			line += " " + e.ArgHint
		}
		if e.Description != "" {
			line += " — " + e.Description
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// renderCommand 展开项目命令并把结果框成「现在照做的指令」（原 slash_command
// 的回执口径）：命令是提示词模板，工具只返回文本，执行仍由模型在本回合完成。
func renderCommand(c CommandEntry, args []string) string {
	body := ""
	if c.Render != nil {
		body = c.Render(args)
	}
	return fmt.Sprintf("Expanded /%s — follow these instructions now:\n\n%s", c.Name, body)
}

// runSkillNameOf 把模型给的 name 归一成查找键：去掉首尾空白、容错首部斜杠
// （模型常照用户输入的 "/name" 抄）与 [..] 标记，取首个以字母数字开头的词。
func runSkillNameOf(raw string) string {
	return cleanSkillName(strings.TrimPrefix(strings.TrimSpace(raw), "/"))
}

func (t *runSkillTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	name := runSkillNameOf(p.Name)
	if name == "" {
		return "", fmt.Errorf("run_skill requires a 'name' argument (got %q, which is just a marker/tag); name=\"list\" lists skills and project commands", p.Name)
	}
	rawArgs := strings.TrimSpace(p.Arguments)

	// 技能优先：同名技能照旧按技能跑（并入前 run_skill 的既有行为）；没有同名
	// 技能时才轮到项目命令，最后是 list 清单（除非真有叫 list 的技能/命令）。
	sk, ok := t.resolve(name)
	if !ok {
		if cmd, isCmd := t.commands[name]; isCmd {
			return renderCommand(cmd, strings.Fields(rawArgs)), nil
		}
		if strings.EqualFold(name, "list") {
			return t.list(), nil
		}
		return "", fmt.Errorf("unknown skill %q (and no project command by that name) — available: %s", name, t.available())
	}

	if sk.RunAs == RunSubagent {
		if t.runner == nil {
			return "", fmt.Errorf("run_skill: skill %q is runAs=subagent but no subagent runner is configured in this session", name)
		}
		if rawArgs == "" {
			return "", fmt.Errorf("run_skill: skill %q is a subagent and requires 'arguments' — the subagent has no other context, so describe the concrete task", name)
		}
		return t.runner(ctx, sk, rawArgs)
	}
	if sk.RunAs == RunPipeline {
		if t.runner == nil {
			return "", fmt.Errorf("run_skill: skill %q is runAs=pipeline but no subagent runner is configured in this session", name)
		}
		return t.runPipeline(ctx, sk, rawArgs)
	}
	return renderInline(sk, rawArgs), nil
}

// runPipeline 执行管道类型技能：解析 body 中的 JSON 管道定义，用参数填充后通过 RunDAG 执行。
func (t *runSkillTool) runPipeline(ctx context.Context, sk Skill, rawArgs string) (string, error) {
	pl, err := LoadPipelineJSON(strings.NewReader(sk.Body))
	if err != nil {
		return "", fmt.Errorf("run_skill: skill %q pipeline body is invalid: %w", sk.Name, err)
	}

	// 解析参数：key=value 格式，空格或换行分隔
	params := parsePipelineArgs(rawArgs)
	resolved := pl.Resolve(params)
	tasks := resolved.ToTasks()

	results, err := RunDAG(ctx, tasks, t.runner)
	if err != nil {
		return "", fmt.Errorf("run_skill: pipeline %q: %w", sk.Name, err)
	}

	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("run_skill: marshal pipeline results: %w", err)
	}
	return string(out), nil
}

// parsePipelineArgs 解析 key=value 格式的参数字符串。
func parsePipelineArgs(raw string) map[string]string {
	params := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return params
	}
	// 按空白字符分割
	parts := strings.Fields(raw)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			params[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return params
}

// --- install_skill ---

type installSkillTool struct {
	store       *Store
	onInstalled InstalledHook
}

// NewInstallSkillTool builds the skill-authoring tool. onInstalled may be nil.
func NewInstallSkillTool(store *Store, onInstalled InstalledHook) tool.Tool {
	return &installSkillTool{store: store, onInstalled: onInstalled}
}

func (*installSkillTool) Name() string       { return "install_skill" }
func (*installSkillTool) ReadOnly() bool     { return false }
func (*installSkillTool) PersistWrite() bool { return true }

func (t *installSkillTool) Description() string {
	scope := "'global' (only option — no project workspace) writes to ~/.gaea/skills/."
	if t.store.HasProjectScope() {
		scope = "'project' (default) writes to <repo>/.gaea/skills/ (this workspace only); 'global' writes to ~/.gaea/skills/ (every project)."
	}
	return "Author and save a new skill — reusable playbook invoked via run_skill or /<name>. Runnable immediately; appears in Skills index on next launch. " +
		"Updating an existing skill: call this tool again with the SAME name — it overwrites in place and keeps any state. Never try to 'uninstall then reinstall' to refresh: there is no uninstall tool, and delete-then-recreate loses skill state for nothing. " + scope
}

func (*installSkillTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "name":{"type":"string","description":"Identifier — letters/digits/_/-/., 1-64 chars, starts alphanumeric. Becomes the filename."},
  "description":{"type":"string","description":"≤120-char one-liner shown in the pinned Skills index — future agents read it to decide whether to invoke."},
  "body":{"type":"string","description":"Markdown playbook. For subagent skills, write the subagent's persona/rules — it gets no context besides 'arguments' at runtime."},
  "scope":{"type":"string","enum":["project","global"],"description":"Where to write. Defaults to project when a workspace exists, else global."},
  "runAs":{"type":"string","enum":["inline","subagent","pipeline"],"description":"inline (default) folds the body into the parent turn; subagent spawns an isolated child loop returning only its final answer; pipeline runs a sequence of subagent steps with dependency ordering."},
  "model":{"type":"string","description":"Optional model override for runAs=subagent (a configured provider/model name). Ignored otherwise."},
  "allowedTools":{"type":"array","items":{"type":"string"},"description":"Optional tool allowlist for runAs=subagent (e.g. ['read_file','grep'])."}
},
"required":["name","description","body"]
}`)
}

func (*installSkillTool) CompactDescription() string {
	return "Create a new skill from name + description + markdown body. Saves to project or global skills directory."
}
func (*installSkillTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"description":{"type":"string"},"body":{"type":"string"},"scope":{"type":"string","enum":["project","global"]},"runAs":{"type":"string","enum":["inline","subagent","pipeline"]}},"required":["name","description","body"]}`)
}

func (t *installSkillTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		Body         string   `json:"body"`
		Scope        string   `json:"scope"`
		RunAs        string   `json:"runAs"`
		Model        string   `json:"model"`
		AllowedTools []string `json:"allowedTools"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	name := strings.TrimSpace(p.Name)
	desc := strings.TrimSpace(collapseSpaces(p.Description))
	if name == "" {
		return "", fmt.Errorf("install_skill requires a non-empty 'name'")
	}
	if desc == "" {
		return "", fmt.Errorf("install_skill requires a non-empty 'description' — it is what appears in the Skills index")
	}
	if strings.TrimSpace(p.Body) == "" {
		return "", fmt.Errorf("install_skill requires a non-empty 'body' — the playbook the skill executes")
	}

	scope := ScopeGlobal
	switch strings.TrimSpace(p.Scope) {
	case "global":
		scope = ScopeGlobal
	case "project":
		scope = ScopeProject
	default:
		if t.store.HasProjectScope() {
			scope = ScopeProject
		}
	}
	if scope == ScopeProject && !t.store.HasProjectScope() {
		return "", fmt.Errorf("install_skill: scope='project' requires a workspace — use scope='global'")
	}

	runAs := RunInline
	switch strings.TrimSpace(p.RunAs) {
	case "subagent":
		runAs = RunSubagent
	case "pipeline":
		runAs = RunPipeline
	}

	content := renderSkillFile(name, desc, p.Body, runAs, strings.TrimSpace(p.Model), p.AllowedTools)
	path, err := t.store.CreateWithContent(name, scope, content)
	if err != nil {
		return "", err
	}
	if t.onInstalled != nil {
		t.onInstalled(name, path, scope)
	}
	res, _ := json.Marshal(map[string]any{
		"ok":    true,
		"name":  name,
		"scope": string(scope),
		"path":  path,
		"runAs": string(runAs),
		"note":  "Callable now via run_skill({name}) or /" + name + ". Appears in the pinned Skills index on the next launch.",
	})
	return string(res), nil
}

// renderSkillFile assembles a skill file's frontmatter + body. Subagent-only
// fields (model, allowed-tools) are emitted only when relevant.
func renderSkillFile(name, desc, body string, runAs RunAs, model string, allowedTools []string) string {
	var fm strings.Builder
	fm.WriteString("---\nname: " + name + "\ndescription: " + desc + "\n")
	if runAs == RunSubagent || runAs == RunPipeline {
		fm.WriteString("runAs: " + string(runAs) + "\n")
		if runAs == RunSubagent {
			if model != "" {
				fm.WriteString("model: " + model + "\n")
			}
			var tools []string
			for _, t := range allowedTools {
				if t = strings.TrimSpace(t); t != "" {
					tools = append(tools, t)
				}
			}
			if len(tools) > 0 {
				fm.WriteString("allowed-tools: " + strings.Join(tools, ", ") + "\n")
			}
		}
	}
	fm.WriteString("---\n\n")
	return fm.String() + strings.TrimRight(body, " \t\r\n") + "\n"
}

// RenderSkillFile assembles a flat inline skill file (frontmatter + body) using
// the same canonical rendering as install_skill. Exported for host-side authors
// (e.g. the desktop "capture a turn as a skill" flow) that write skills outside
// the agent loop.
func RenderSkillFile(name, desc, body string) string {
	return renderSkillFile(name, desc, body, RunInline, "", nil)
}

// --- shared helpers ---

// Render builds a skill's invocation text: a header (name, description, source)
// followed by the body and any arguments. Used directly when a user invokes a
// skill via "/<name>" (sent as a turn); the run_skill tool wraps the same text
// in a skill-pin sentinel (see renderInline).
func Render(sk Skill, args string) string {
	var b strings.Builder
	b.WriteString("# Skill: " + sk.Name)
	if sk.Description != "" {
		b.WriteString("\n> " + sk.Description)
	}
	b.WriteString("\n(scope: " + string(sk.Scope) + " · " + sk.Path + ")\n\n")
	b.WriteString(sk.Body)
	if args != "" {
		b.WriteString("\n\nArguments: " + args)
	}
	return b.String()
}

// renderInline wraps Render's output in a skill-pin sentinel so context
// compaction preserves the body verbatim instead of paraphrasing it.
func renderInline(sk Skill, args string) string {
	return "<skill-pin name=" + strconv.Quote(sk.Name) + ">\n" + Render(sk, args) + "\n</skill-pin>"
}

var bracketTagRe = regexp.MustCompile(`\[[^\]]*\]`)

// cleanSkillName extracts the bare identifier from a possibly-decorated name:
// models sometimes copy the index's "explore [🧬 subagent]" verbatim into the
// `name` arg. Drop any [..] tag, then take the first token starting alphanumeric.
func cleanSkillName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	stripped := strings.TrimSpace(bracketTagRe.ReplaceAllString(raw, " "))
	for _, tok := range strings.Fields(stripped) {
		if c := tok[0]; (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return tok
		}
	}
	return ""
}

// collapseSpaces turns any run of whitespace (incl. newlines) into a single
// space, so a multi-line description stays a one-liner in the index.
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// availableNames lists the discoverable skill names for an error message.
func availableNames(store *Store) string {
	skills := store.List()
	if len(skills) == 0 {
		return "(none — no skills defined)"
	}
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}
