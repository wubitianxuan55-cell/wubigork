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
	// 而不是写成同一个笼统的「已只读」；无硬限制的能力（如 codex 联网搜索）
	// 在文案中明示降级（2026-10-31 裁决第三刀）。
	runMode string

	probe func(context.Context) AgentStatus
	build func(prompt, cwd string, images []string) (commandSpec, error)
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

func newGrokAdapter(backend GrokBackend) agentAdapter {
	adapter := agentAdapter{
		id:   "grok",
		name: "Grok",
		kind: "Gaea 办公席位 · xAI",
		// 无写工具注册；文件读取以 workspace 与附件 readPaths 为边界。
		runMode: "Gaea 只读工具白名单（未注册写工具，读取限工作区与附件）",
		probe: func(ctx context.Context) AgentStatus {
			return backend.ProbeGrok(ctx)
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
		// 联网搜索显式开放（web_search=live），属无硬限制能力，必须降级表述。
		runMode: "codex sandbox read-only（文件写被沙箱拦截；联网搜索开放，结果按不可信资料处理）",
		probe: func(ctx context.Context) AgentStatus {
			status := AgentStatus{
				ID: "codex", Name: "Codex", Kind: "OpenAI Codex CLI",
				Capabilities: []string{
					"multi-round-discussion", "read-only-sandbox", "jsonl", "streaming", "tool-trajectory",
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
			path, err := find()
			if err != nil {
				return commandSpec{}, err
			}
			args := []string{
				"exec", "--json", "--ephemeral", "--sandbox", "read-only",
				"-c", `web_search="live"`, "--skip-git-repo-check",
			}
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
		},
	}
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
		runMode: "plan 模式（提案不落盘，依赖 harness 自身执行；provider 凭据经环境变量注入）",
		probe: func(ctx context.Context) AgentStatus {
			status := AgentStatus{
				ID: "zcode", Name: "ZCode", Kind: "ZCode CLI",
				Capabilities: []string{
					"multi-round-discussion", "plan-mode", "stream-json", "streaming", "tool-trajectory",
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
			node, script, err := resolve()
			if err != nil {
				return commandSpec{}, err
			}
			args := zcodePromptArgs(prompt, cwd, images)
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
			}, nil
		},
	}
}

func zcodePromptArgs(prompt, cwd string, images []string) []string {
	args := make([]string, 0, 10+len(images)*2)
	for _, image := range images {
		args = append(args, "--attach", image)
	}
	return append(args,
		"--prompt", clampPrompt(prompt), "--cwd", cwd,
		"--mode", "plan", "--output-format", "stream-json", "--no-color",
	)
}

func newDSHAdapter(override, deepseekKey string) agentAdapter {
	type dshPaths struct {
		exe  string
		cli  string
		mode string
	}
	resolve := func() (dshPaths, error) {
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
	baseArgs := func(paths dshPaths) []string {
		if paths.cli != "" {
			return []string{"--expose-internals", paths.cli}
		}
		return nil
	}
	build := func(paths dshPaths, args []string) commandSpec {
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
	return agentAdapter{
		id: "dsh", name: "DeepSeek Harness", kind: "DSH headless",
		runMode: "DSH_PERMISSION_MODE=read-only 环境变量（依赖 DSH 自身执行）",
		probe: func(ctx context.Context) AgentStatus {
			status := AgentStatus{
				ID: "dsh", Name: "DeepSeek Harness", Kind: "DSH headless",
				Capabilities: []string{
					"multi-round-discussion", "read-only-permission", "json-events", "streaming", "tool-trajectory",
				},
			}
			paths, err := resolve()
			if err != nil {
				status.Reason = err.Error()
				return status
			}
			status.Available = true
			status.Executable = paths.exe
			args := append(baseArgs(paths), "--version")
			spec := build(paths, args)
			status.Version = probeVersion(ctx, spec)
			return status
		},
		build: func(prompt, cwd string, _ []string) (commandSpec, error) {
			paths, err := resolve()
			if err != nil {
				return commandSpec{}, err
			}
			args := append(baseArgs(paths), "--profile", "headless", "--json", "-")
			spec := build(paths, args)
			spec.Dir = cwd
			spec.Stdin = clampPrompt(prompt)
			return spec, nil
		},
	}
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
