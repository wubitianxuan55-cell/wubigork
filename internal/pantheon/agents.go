package pantheon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type agentAdapter struct {
	id   string
	name string
	kind string

	// runMode 声明本次调用实际施加的只读机制。会话级 ReadOnly 布尔在四席
	// 的落地方式并不同质（sandbox/plan/环境变量/工具白名单），必须逐席入档
	// 而不是写成同一个笼统的「已只读」；无硬限制的能力（如 codex 直连网络）
	// 在文案中明示降级（2026-10-31 裁决第三刀）。
	runMode string

	probe func(context.Context) AgentStatus
	build func(prompt, cwd string, images []string) (commandSpec, error)
	// buildResume 非空 = 该席位的引擎支持「同一会话续接」（DSH headless
	// --session-id）。resumeID 为空时必须等价于 build：首次调用不凭空造会话
	// ID——引擎对未知 ID 是硬错误，不是隐式新建。
	buildResume func(prompt, cwd string, images []string, resumeID string) (commandSpec, error)
	// invoke 非空 = Gaea 模型席位：不经本机 CLI。app 层用 xAI 模型驱动
	// Gaea 办公 AgentRunner；build 为空，输出按纯文本处理。
	invoke func(ctx context.Context, prompt string) (string, error)
	stream func(ctx context.Context, prompt string, emit func(string)) (string, error)
	// streamWithTools is used by Gaea's model seat when it is backed by the
	// office AgentRunner instead of a bare chat completion.
	streamWithTools func(
		ctx context.Context,
		prompt string,
		workspace string,
		readPaths []string,
		emit func(string),
		onTool func(ToolTrace),
	) (string, error)
	// streamWithToolsSession 非空 = 模型席位实现 GrokHistoryBackend，可携带
	// 跨轮消息历史（A5③）。历史非空时宿主优先走这条路，并把提示词切换为
	// seeded 形态（往轮梯度省掉，只发当轮与指令）。
	streamWithToolsSession func(
		ctx context.Context,
		history []SeatMessage,
		prompt string,
		workspace string,
		readPaths []string,
		emit func(string),
		onTool func(ToolTrace),
	) (string, error)
}

// GrokBackend 是 Grok 模型席位的运行时后端。pantheon 不 import internal/app，
// 引擎解析、鉴权与调用记账全部留在注入方；协议测试用内存实现替换。
type GrokBackend interface {
	// ProbeGrok 返回席位状态；不可用时 Available=false 且 Reason 面向用户。
	ProbeGrok(ctx context.Context) AgentStatus
	// CallGrok 完成一次只读议事调用，返回纯文本。
	CallGrok(ctx context.Context, prompt string) (string, error)
}

// GrokStreamingBackend is optional. When provided, Gaea model seats expose
// provider deltas to the shared discussion stream instead of waiting for the
// complete answer.
type GrokStreamingBackend interface {
	StreamGrok(ctx context.Context, prompt string, emit func(string)) (string, error)
}

// GrokToolStreamingBackend is implemented by Gaea model seats that run through
// an AgentRunner with real tool schemas. The workspace bounds all tool access.
type GrokToolStreamingBackend interface {
	StreamGrokWithTools(
		ctx context.Context,
		prompt string,
		workspace string,
		readPaths []string,
		emit func(string),
		onTool func(ToolTrace),
	) (string, error)
}

// SeatMessage 是模型席位跨轮历史的最小消息形状（2026-11-07 A5③）。Role 只取
// user|assistant；system 提示词由后端自有，宿主不传。内容是全保真原文——
// 这条链的存在意义就是「往轮逐字保留」，任何截断都等于把失忆请回来。
type SeatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GrokHistoryBackend 由支持跨轮消息历史的模型席位实现。history 是宿主从贡献
// 记录确定性重建的往轮会议记录（跳过/重试/恢复/重启后按同一函数重建，永远
// 与存档一致，无可失效台账）；实现方必须把它 Seed 进引擎侧会话，使席位获得
// 全保真跨轮记忆，而不是依赖提示词里的立场卡/索引梯度。无法支持时不要实现
// 本接口：宿主自动退回单发提示词路径。
type GrokHistoryBackend interface {
	StreamGrokSession(
		ctx context.Context,
		history []SeatMessage,
		prompt string,
		workspace string,
		readPaths []string,
		emit func(string),
		onTool func(ToolTrace),
	) (string, error)
}

func buildAdapters(overrides map[string]string, grok GrokBackend, deepseekKey string) map[string]agentAdapter {
	adapters := []agentAdapter{
		newCodexAdapter(overrides["codex"]),
		newZCodeAdapter(overrides["zcode"]),
		newDSHAdapter(overrides["dsh"], deepseekKey),
	}
	if grok != nil {
		// Grok 席位列首位：Gaea 办公 AgentRunner、秒级探测，适合作协作汇总人。
		adapters = append(adapters, newGrokAdapter(grok))
	}
	out := make(map[string]agentAdapter, len(adapters))
	for _, adapter := range adapters {
		out[adapter.id] = adapter
	}
	return out
}

func ensureCapability(capabilities []string, capability string) []string {
	for _, existing := range capabilities {
		if existing == capability {
			return capabilities
		}
	}
	return append(capabilities, capability)
}

func (a agentAdapter) commandSpec(prompt, cwd string, images []string, resumeID string) (commandSpec, error) {
	if a.buildResume != nil {
		return a.buildResume(prompt, cwd, images, resumeID)
	}
	if a.build == nil {
		return commandSpec{}, fmt.Errorf("Agent %s 不支持进程调用", a.id)
	}
	return a.build(prompt, cwd, images)
}

// annotateSeatError 把引擎的凭据错误翻成可执行的修复路径。
//
// 2026-11-06 实弹：DSH headless 的 DeepSeek 凭据只来自「启动目录的 .env」或
// 环境变量（凭据库里没有 key）。众神殿允许把议事工作区切到任意项目目录——
// 那个目录没有 .env、Gaea 又没配 deepseek 引擎时，每一轮都会立刻失败，而
// 用户只看到一串 MISSING_CREDENTIAL。这里补一句去哪儿配，不改变错误本身。
func annotateSeatError(agentID string, err error) error {
	if err == nil || !strings.EqualFold(agentID, "dsh") {
		return err
	}
	if !strings.Contains(err.Error(), "MISSING_CREDENTIAL") {
		return err
	}
	return fmt.Errorf("%w（DSH 没拿到 DeepSeek 凭据：在 Gaea 模型中心配好 deepseek 引擎的 API Key，或让议事工作区根目录的 .env 带 DEEPSEEK_API_KEY）", err)
}

func newGrokAdapter(backend GrokBackend) agentAdapter {
	_, hasToolStreaming := backend.(GrokToolStreamingBackend)
	adapter := agentAdapter{
		id:   "grok",
		name: "Grok",
		kind: "Gaea 办公席位 · xAI",
		// 无写工具注册；文件读取以 workspace 与附件 readPaths 为边界。
		runMode: "Gaea 只读工具白名单（未注册写工具，读取限工作区与附件）",
		probe: func(ctx context.Context) AgentStatus {
			status := backend.ProbeGrok(ctx)
			status.Capabilities = ensureCapability(status.Capabilities, "shared-image-summary")
			if hasToolStreaming {
				status.Capabilities = ensureCapability(status.Capabilities, "image-path-tools")
			}
			return status
		},
		invoke: backend.CallGrok,
	}
	if streaming, ok := backend.(GrokStreamingBackend); ok {
		adapter.stream = streaming.StreamGrok
	}
	if streaming, ok := backend.(GrokToolStreamingBackend); ok {
		adapter.streamWithTools = func(
			ctx context.Context,
			prompt string,
			workspace string,
			readPaths []string,
			emit func(string),
			onTool func(ToolTrace),
		) (string, error) {
			return streaming.StreamGrokWithTools(ctx, prompt, workspace, readPaths, emit, onTool)
		}
	}
	if sessioned, ok := backend.(GrokHistoryBackend); ok {
		adapter.streamWithToolsSession = sessioned.StreamGrokSession
	}
	return adapter
}

func newCodexAdapter(override string) agentAdapter {
	find := func() (string, error) {
		if override != "" {
			if !executableExists(override) {
				return "", fmt.Errorf("自定义 Codex 路径不存在")
			}
			return override, nil
		}
		return exec.LookPath("codex")
	}
	return agentAdapter{
		id: "codex", name: "Codex", kind: "OpenAI Codex CLI",
		// 2026-11-03 用户拍板「都能上网查资料」：read-only 沙箱默认禁网（直连
		// GitHub API 实测被拦，内建 web_search 只是特权通道），放宽为
		// workspace-write+network——写面限工作区且由陈词纪律约束不写盘，
		// runMode 如实降级表述（裁决第三刀：能力不对称必须逐席入档）。
		runMode: "codex sandbox workspace-write+network（直连网络开放，结果按不可信资料处理；文件写沙箱限工作区，陈词纪律约束不写盘）",
		probe: func(ctx context.Context) AgentStatus {
			status := AgentStatus{
				ID: "codex", Name: "Codex", Kind: "OpenAI Codex CLI",
				Capabilities: []string{
					"multi-round-discussion", "read-only-sandbox", "jsonl", "streaming",
					"tool-trajectory", "image-input", "shared-image-summary",
				},
			}
			path, err := find()
			if err != nil {
				status.Reason = "未找到 codex 命令"
				return status
			}
			status.Available = true
			status.Executable = path
			status.Version = probeVersion(ctx, commandSpec{Executable: path, Args: []string{"--version"}})
			return status
		},
		build: func(prompt, cwd string, images []string) (commandSpec, error) {
			return codexExecSpec(find, "", prompt, cwd, images)
		},
		// buildResume（2026-11-07 A5②）：codex exec 原生支持
		// `exec resume <SESSION_ID>`（探针实锤续接：新会话能原样复述上一会话
		// 的回复）。三条硬约束同样探针实锤：①resume 子命令拒收 `-s`，沙箱
		// 须走 `-c sandbox_mode=…`；②--ephemeral 会话不落 rollout，resume 报
		// 「no rollout found」——续接档必须摘掉它（代价：codex 议事会话落
		// ~/.codex/sessions，与 zcode/dsh 持久化会话同款取舍）；③resumeID
		// 为空必须等价 build（会话丢失后回落新建，不硬错）。
		buildResume: func(prompt, cwd string, images []string, resumeID string) (commandSpec, error) {
			return codexExecSpec(find, strings.TrimSpace(resumeID), prompt, cwd, images)
		},
	}
}

// codexExecSpec 构造 codex exec 调用；resumeID 非空走 `exec resume` 续接，
// 为空走全新 `exec`（非 ephemeral——thread 必须落 rollout，下一轮才有的续）。
func codexExecSpec(find func() (string, error), resumeID, prompt, cwd string, images []string) (commandSpec, error) {
	path, err := find()
	if err != nil {
		return commandSpec{}, err
	}
	var args []string
	if id := strings.TrimSpace(resumeID); id != "" {
		args = append(args, "exec", "resume", id, "--json")
	} else {
		args = append(args, "exec", "--json")
	}
	args = append(args,
		"-c", `sandbox_mode="workspace-write"`,
		"-c", `sandbox_workspace_write.network_access=true`,
		"-c", `web_search="live"`,
		"--skip-git-repo-check",
	)
	for _, image := range images {
		args = append(args, "--image", image)
	}
	args = append(args, "-C", cwd, "-")
	return commandSpec{
		Executable: path,
		Args:       args,
		Dir:        cwd,
		Stdin:      clampPrompt(prompt),
	}, nil
}

func newZCodeAdapter(override string) agentAdapter {
	resolve := func() (node, script string, err error) {
		script = override
		if script == "" {
			script = windowsPath("Programs", "ZCode", "resources", "glm", "zcode.cjs")
		}
		if !executableExists(script) {
			if path, lookErr := exec.LookPath("zcode"); lookErr == nil {
				return path, "", nil
			}
			return "", "", fmt.Errorf("未找到 ZCode CLI 或 resources/glm/zcode.cjs")
		}
		if strings.EqualFold(filepath.Ext(script), ".cjs") || strings.EqualFold(filepath.Ext(script), ".js") {
			node, lookErr := exec.LookPath("node")
			if lookErr != nil {
				return "", "", fmt.Errorf("找到 ZCode CLI，但未找到 Node.js")
			}
			return node, script, nil
		}
		return script, "", nil
	}
	return agentAdapter{
		id: "zcode", name: "ZCode", kind: "ZCode CLI",
		// 2026-11-06 实弹：ZCode 每步模型往返 16–26s（远高于 DSH 的 4–7s），
		// 而每轮工具调用默认新开会话——13–25 步就是 220–430s，用户看到的就是
		// 「ZCode 总是卡死」。续接同一会话后实测 1 次模型请求、10.7s、
		// cacheRead 33856/34133（99.2%）。所以 runMode 把续接也写进去。
		runMode: "plan 模式（提案不落盘；禁用 Agent/Workflow；长任务书经临时文件下发；provider 凭据经环境变量注入；同桌续接同一引擎会话，前缀缓存可证）",
		probe: func(ctx context.Context) AgentStatus {
			status := AgentStatus{
				ID: "zcode", Name: "ZCode", Kind: "ZCode CLI",
				Capabilities: []string{
					"multi-round-discussion", "plan-mode", "stream-json", "streaming",
					"tool-trajectory", "image-input", "shared-image-summary",
					"session-resume", "usage-reporting",
				},
			}
			node, script, err := resolve()
			if err != nil {
				status.Reason = err.Error()
				return status
			}
			status.Available = true
			status.Executable = node
			if script != "" {
				status.Executable += " " + script
			}
			args := []string{"--version"}
			if script != "" {
				args = []string{script, "--version"}
			}
			status.Version = probeVersion(ctx, commandSpec{Executable: node, Args: args})
			return status
		},
		build: func(prompt, cwd string, images []string) (commandSpec, error) {
			return zcodeCommandSpec(resolve, prompt, cwd, images, "")
		},
		buildResume: func(prompt, cwd string, images []string, resumeID string) (commandSpec, error) {
			return zcodeCommandSpec(resolve, prompt, cwd, images, resumeID)
		},
	}
}

func zcodeCommandSpec(
	resolve func() (string, string, error),
	prompt, cwd string,
	images []string,
	resumeID string,
) (commandSpec, error) {
	node, script, err := resolve()
	if err != nil {
		return commandSpec{}, err
	}
	promptArg, _, cleanup, err := prepareZCodePrompt(prompt)
	if err != nil {
		return commandSpec{}, err
	}
	args := zcodePromptArgs(promptArg, cwd, images)
	if id := strings.TrimSpace(resumeID); id != "" {
		// ZCode 的会话 ID 形如 sess_…，由 turn.completed/result 事件上报；
		// 只认引擎给过的 ID，未知 ID 是硬错误（与 DSH 同一纪律）。
		args = append(args, "--resume", id)
	}
	if script != "" {
		args = append([]string{script}, args...)
	}
	env := []string{"NO_COLOR=1"}
	builtin := windowsPath("Programs", "ZCode", "resources", "config", "provider", "zcode-builtin.json")
	personal := ""
	if home, err := os.UserHomeDir(); err == nil {
		personal = filepath.Join(home, ".zcode", "v2", "provider_config.json")
	}
	if executableExists(builtin) {
		env = append(env, "ZCODE_BUILTIN_PROVIDER_CONFIG_FILE="+builtin)
	}
	if executableExists(personal) {
		env = append(env, "ZCODE_PERSONAL_PROVIDER_CONFIG_FILE="+personal)
	}
	return commandSpec{
		Executable: node,
		Args:       args,
		Dir:        cwd,
		Env:        env,
		Cleanup:    cleanup,
	}, nil
}

func zcodePromptArgs(prompt, cwd string, images []string) []string {
	args := make([]string, 0, 14+len(images)*2)
	for _, image := range images {
		args = append(args, "--attach", image)
	}
	return append(args,
		"--prompt", clampPrompt(prompt), "--cwd", cwd,
		"--mode", "plan",
		// 一席必须代表一个独立判断主体。ZCode 的 Agent/Workflow 会派生
		// 子代理或后台工作流，既违反圆桌席位契约，也会显著放大延迟。
		"--disallowed-tools", "Agent", "Workflow",
		"--output-format", "stream-json", "--no-color",
	)
}

const zcodeInlinePromptRunes = 12000

// prepareZCodePrompt keeps Windows argv comfortably below the ~32K
// CreateProcess limit. Longer discussion prompts are materialized as a
// short-lived UTF-8 file and ZCode receives only a short instruction to read
// it. The child is explicitly told this is a discussion turn, not a coding
// implementation plan, and may not fan out into helper agents.
func prepareZCodePrompt(prompt string) (promptArg, promptPath string, cleanup func(), err error) {
	clamped := clampPrompt(prompt)
	cleanup = func() {}
	if len([]rune(clamped)) <= zcodeInlinePromptRunes {
		return clamped, "", cleanup, nil
	}

	file, err := os.CreateTemp("", "gaea-zcode-prompt-*.md")
	if err != nil {
		return "", "", cleanup, fmt.Errorf("创建 ZCode 临时任务书: %w", err)
	}
	promptPath = file.Name()
	cleanup = func() {
		_ = os.Remove(promptPath)
	}
	if _, err := file.WriteString(clamped); err != nil {
		_ = file.Close()
		cleanup()
		return "", "", func() {}, fmt.Errorf("写入 ZCode 临时任务书: %w", err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", "", func() {}, fmt.Errorf("关闭 ZCode 临时任务书: %w", err)
	}

	promptArg = fmt.Sprintf(`你是众神殿圆桌讨论中的一个席位。请先用 Read 工具完整读取以下任务书，然后严格执行：

%s

这是议事讨论，不是代码实施计划，也不是等待批准的实施方案。不要派生 subagent、并行助手或 Workflow，不要写文件。只使用完成本轮陈词所必需的只读工具；材料足够后立即输出任务书要求的完整陈词，把未核验项写进置信度或遗留项。`, promptPath)
	return promptArg, promptPath, cleanup, nil
}

// dshPaths 是 DSH CLI 的解析结果。mode=desktop 表示命中的是随附的 .cmd 包装
// （经 cmd.exe /d /c call 启动），mode=path 表示 PATH 或用户指定的可执行文件。
type dshPaths struct {
	exe  string
	cli  string
	mode string
}

func resolveDSHPaths(override string) (dshPaths, error) {
	if override != "" {
		if !executableExists(override) {
			return dshPaths{}, fmt.Errorf("自定义 DSH 路径不存在")
		}
		mode := "path"
		switch strings.ToLower(filepath.Ext(override)) {
		case ".cmd", ".bat":
			mode = "desktop"
		}
		return dshPaths{exe: override, mode: mode}, nil
	}
	if path, err := exec.LookPath("dsh"); err == nil {
		return dshPaths{exe: path, mode: "path"}, nil
	}
	cmd := windowsPath("Programs", "DeepSeek Harness", "resources", "runtime", "cli", "bin", "dsh.cmd")
	if executableExists(cmd) {
		return dshPaths{exe: cmd, mode: "desktop"}, nil
	}
	return dshPaths{}, fmt.Errorf("未找到 dsh CLI 或 DeepSeek Harness 随附 CLI")
}

func dshBuildSpec(paths dshPaths, args []string, deepseekKey string) commandSpec {
	env := []string{"DSH_PERMISSION_MODE=read-only"}
	if deepseekKey != "" {
		env = append(env, "DEEPSEEK_API_KEY="+deepseekKey)
	}
	if paths.mode == "desktop" {
		// .cmd is an official Windows wrapper. Keep the prompt on stdin so
		// topic/material text never enters shell metacharacter parsing.
		// `call` accepts a quoted batch path as one argument after /c.
		return commandSpec{
			Executable: "cmd.exe",
			Args:       append([]string{"/d", "/c", "call", paths.exe}, args...),
			Env:        env,
		}
	}
	return commandSpec{Executable: paths.exe, Args: args, Env: env}
}

func newDSHAdapter(override, deepseekKey string) agentAdapter {
	resolve := func() (dshPaths, error) { return resolveDSHPaths(override) }
	spec := func(prompt, cwd string, resumeID string) (commandSpec, error) {
		paths, err := resolve()
		if err != nil {
			return commandSpec{}, err
		}
		args := append(dshBaseArgs(paths), dshSeatArgs(resumeID)...)
		out := dshBuildSpec(paths, args, deepseekKey)
		out.Dir = cwd
		out.Stdin = clampPrompt(prompt)
		return out, nil
	}
	return agentAdapter{
		id: "dsh", name: "DeepSeek Harness", kind: "DSH headless",
		// 2026-11-06 实弹：只靠 DSH_PERMISSION_MODE 环境变量是「依赖 DSH 自身
		// 执行」的软约束，且没有任何显式审批策略——命令通道被拦时模型会照
		// 「重试并申请更高权限」的提示反复消耗步数。改为把两个官方旗标显式
		// 写进命令行（实弹验证：写入被 sandbox 拒绝、审批不弹卡），runMode 也
		// 按实际命令行改写，不再申报一个只有环境变量的说法。
		runMode: "DSH headless --permission-mode read-only --approval-policy never（只读沙箱由 DSH 执行；同桌续接同一引擎会话，前缀缓存可证）",
		probe: func(ctx context.Context) AgentStatus {
			status := AgentStatus{
				ID: "dsh", Name: "DeepSeek Harness", Kind: "DSH headless",
				Capabilities: []string{
					"multi-round-discussion", "read-only-permission", "json-events", "streaming",
					"tool-trajectory", "shared-image-summary", "session-resume", "usage-reporting",
				},
			}
			paths, err := resolve()
			if err != nil {
				status.Reason = err.Error()
				return status
			}
			status.Available = true
			status.Executable = paths.exe
			args := append(dshBaseArgs(paths), "--version")
			status.Version = probeVersion(ctx, dshBuildSpec(paths, args, deepseekKey))
			return status
		},
		build: func(prompt, cwd string, _ []string) (commandSpec, error) {
			return spec(prompt, cwd, "")
		},
		buildResume: func(prompt, cwd string, _ []string, resumeID string) (commandSpec, error) {
			return spec(prompt, cwd, resumeID)
		},
	}
}

func dshBaseArgs(paths dshPaths) []string {
	if paths.cli != "" {
		return []string{"--expose-internals", paths.cli}
	}
	return nil
}

// dshSeatArgs 是圆桌席位固定使用的命令行参数。
//
// 1) 权限与审批显式化：无人值守的子进程没有审批面，ask 只会让模型反复撞同一
// 道门（每一个撞门的回合都是白烧的分钟级等待）。实弹验证：read-only 下写入
// 被 sandbox 拒绝，approval-policy never 不弹卡。
// 2) resumeID 非空时用 --session-id 续接同一引擎会话：引擎前缀（system+工具
// 表+历史步）逐字节不变，后续轮次只追加本轮议题与陈词——DeepSeek 前缀缓存
// 因此命中。2026-11-06 实测同一会话第二轮 cacheRead 6528/6757（96.6%），而
// 每轮新建会话时只有约 6k 固定前缀可命中，正文与工具输出全价重算。
func dshSeatArgs(resumeID string) []string {
	args := []string{
		"--profile", "headless", "--json",
		"--permission-mode", "read-only",
		"--approval-policy", "never",
	}
	if id := strings.TrimSpace(resumeID); id != "" {
		args = append(args, "--session-id", id)
	}
	return append(args, "-")
}

func probeVersion(parent context.Context, spec commandSpec) string {
	ctx, cancel := context.WithTimeout(parent, 6*time.Second)
	defer cancel()
	runner := osCommandRunner{}
	out, err := runner.Run(ctx, spec)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
