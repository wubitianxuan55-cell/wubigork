package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/pantheon"
)

// 2026-11-16 用户拍板（工具清单瘦身）：众神殿是通用议事板块，造价/日程是
// 办公板块域工具，对非域议题只是 system prompt 里的选择噪音；域知识改走
// read_skill 按需加载（规范知识技能化红线）。genui_validate 是聊天板块 GenUI
// 渲染校验，与议事无关。删：cost_search/cost_indicators/cost_compose/
// schedule_get/schedule_analyze/genui_validate（漂移锁测试同批改）。
// cad_ask 保留：图纸评审类议题的唯一 DWG 读图通道，与 vision 同属读材料面。
var pantheonGrokWorkspaceTools = []string{
	"read_file",
	"grep",
	"ls",
	// doc_render 已移出议事 T0（2026-11-08 众神殿结论 pan_1794098356822 第一
	// 步）：它带 ReadOnly 标签却会删除/重建工作区产物、写系统临时目录并启动
	// LibreOffice 子进程——元数据声明不构成隔离。移出即漂移锁：回归测试
	// TestPantheonGrokT0ExcludesDocRender 钉死，不得未经 effect 隔离审计复入。
	"cad_ask",
	"vision",
}

var pantheonGrokOfficeReadTools = map[string]bool{
	"memory_search":    true,
	"fact_list":        true,
	"web_search":       true,
	"web_fetch":        true,
	"read_skill":       true,
	"ocr":              true,
}

// pantheonGrokOfficeRegistry builds a fresh registry for one council turn. Core
// filesystem tools are rebound to the selected council workspace; additional
// office tools are copied from the live Gaea office registry only when they
// explicitly declare ReadOnly. No task/skill/write/approval tools are exposed.
func (s *pantheonGrokSeat) officeToolRegistry(workspace string) *tool.Registry {
	reg := tool.NewRegistry()
	ws := builtin.Workspace{Dir: workspace}
	for _, t := range ws.Tools(pantheonGrokWorkspaceTools...) {
		reg.Add(t)
	}

	// GaeaInit configures the same office registries and App-backed specialist
	// tools used by the office board. Failure is non-fatal: the workspace-bound
	// read/search core remains available.
	if s.app != nil {
		_ = s.app.GaeaInit()
		ga.mu.Lock()
		var officeTools []tool.Tool
		if ga.ctrl != nil {
			officeTools = ga.ctrl.Tools()
		}
		ga.mu.Unlock()
		for _, t := range officeTools {
			if !pantheonGrokOfficeReadTools[t.Name()] || !t.ReadOnly() {
				continue
			}
			if _, exists := reg.Get(t.Name()); exists {
				continue
			}
			reg.Add(t)
		}
	}

	// Global read-only fallbacks cover direct agent tests and first-run
	// environments where the office controller could not be assembled.
	for name := range pantheonGrokOfficeReadTools {
		if _, exists := reg.Get(name); exists {
			continue
		}
		if t, ok := tool.LookupBuiltin(name); ok && t.ReadOnly() {
			reg.Add(t)
		}
	}
	return reg
}

func pantheonGrokSystemPrompt(toolNames []string) string {
	names := append([]string(nil), toolNames...)
	sort.Strings(names)
	return `你是 Gaea 众神殿的 Grok 席位，由 Gaea 办公 AgentRunner 驱动，不是裸模型对话。
你可以使用当前注册表中的 Gaea 只读工具核验工作空间、项目文档、方案库、记忆与本地事实，需要领域规范或数据口径时用 read_skill 按需加载。只在与当前议题直接相关时调用工具，并说明证据来自哪个工具或结果。
禁止写文件、改配置、安装软件、启动副作用服务或读取凭证；不要声称执行了实际未发生的工具调用。模型内部推理不属于工具轨迹。

当前可用工具：` + strings.Join(names, "、")
}

func runPantheonGrokAgent(
	ctx context.Context,
	prov provider.LLMProvider,
	reg *tool.Registry,
	workspace string,
	prompt string,
	readPaths []string,
	emitText func(string),
	onTool func(pantheon.ToolTrace),
) (string, *pantheon.SeatUsage, error) {
	if prov == nil || reg == nil {
		return "", nil, fmt.Errorf("Grok 办公 AgentRunner 未完成装配")
	}
	// 会话装配：单发提示词形态（A5③ Seed 实弹证伪后回退——往轮记忆由宿主
	// 全量窗口内联进 prompt，见 pantheon.historyFullWindow；引擎侧会话对 grok
	// 的行为增益未证实、风险实测过）。
	return runPantheonAgentWithPrompt(ctx, prov, reg, workspace,
		pantheonGrokSystemPrompt(reg.Names()), prompt, readPaths, emitText, onTool,
		pantheonAgentRunOptions{})
}

// pantheonAgentRunOptions 是角色级配置层的执行预算覆盖；零值保持模型席位
// 历史默认（12 步 / 0.2 / 200k）。
type pantheonAgentRunOptions struct {
	MaxSteps      *int
	Temperature   *float64
	ContextWindow *int
}

// runPantheonAgentWithPrompt 是模型席位的通用执行核：system prompt 参数化
// （T15 文件席位传角色 persona，grok 席传 pantheonGrokSystemPrompt）。
func runPantheonAgentWithPrompt(
	ctx context.Context,
	prov provider.LLMProvider,
	reg *tool.Registry,
	workspace string,
	systemPrompt string,
	prompt string,
	readPaths []string,
	emitText func(string),
	onTool func(pantheon.ToolTrace),
	opts pantheonAgentRunOptions,
) (string, *pantheon.SeatUsage, error) {
	if prov == nil || reg == nil {
		return "", nil, fmt.Errorf("Grok 办公 AgentRunner 未完成装配")
	}
	var streamed strings.Builder
	var finalMessage string
	var mu sync.Mutex
	// 2026-11-18 上下文账本断链根修：AgentRunner 每次模型往返发一条 Usage
	// 事件（provider.Usage 单轮值）——这里累计成 SeatUsage（未命中/命中/
	// 输出三栏+步数=模型请求数），模型席（grok/T15）的 usage 从此有数
	// （此前模型席不经 CLI 解析管道，前端恒显「上下文未上报」）。
	var usage pantheon.SeatUsage
	sawUsage := false
	sink := event.FuncSink(func(e event.Event) {
		switch e.Kind {
		case event.Text:
			if e.Text == "" {
				return
			}
			mu.Lock()
			streamed.WriteString(e.Text)
			mu.Unlock()
			if emitText != nil {
				emitText(e.Text)
			}
		case event.Message:
			mu.Lock()
			finalMessage = e.Text
			mu.Unlock()
		case event.Usage:
			if e.Usage == nil {
				return
			}
			mu.Lock()
			sawUsage = true
			usage.Steps++
			// provider.Usage 口径：PromptTokens=该轮 prompt 总量，
			// CacheHit/CacheMiss 是其中命中/未命中分栏。归一到 SeatUsage
			// 的「未命中 / 命中」两栏（与 CLI 席 parseSeatRunStats 同口径）。
			miss := e.Usage.CacheMissTokens
			if miss <= 0 {
				miss = e.Usage.PromptTokens - e.Usage.CacheHitTokens
				if miss < 0 {
					miss = 0
				}
			}
			usage.InputTokens += int64(miss)
			usage.CacheReadTokens += int64(e.Usage.CacheHitTokens)
			usage.OutputTokens += int64(e.Usage.CompletionTokens)
			mu.Unlock()
		case event.ToolDispatch:
			if e.Tool.Partial || onTool == nil {
				return
			}
			onTool(pantheon.ToolTrace{
				ID: e.Tool.ID, Name: e.Tool.Name, Status: "started",
				Input: boundedGrokPreview(e.Tool.Args),
			})
		case event.ToolResult:
			if onTool == nil {
				return
			}
			status := "completed"
			if e.Tool.Err != "" {
				status = "failed"
			}
			onTool(pantheon.ToolTrace{
				ID: e.Tool.ID, Name: e.Tool.Name, Status: status,
				Output: boundedGrokPreview(e.Tool.Output), Error: e.Tool.Err,
			})
		}
	})

	maxSteps := opts.MaxSteps
	maxStepsValue := 12
	if maxSteps != nil {
		maxStepsValue = *maxSteps
	}
	temperatureValue := 0.2
	if opts.Temperature != nil {
		temperatureValue = *opts.Temperature
	}
	contextWindow := opts.ContextWindow
	contextWindowValue := 200_000
	if contextWindow != nil {
		contextWindowValue = *contextWindow
	}
	session := gaeaAgent.NewSession(systemPrompt)
	runner := gaeaAgent.New(prov, reg, session, gaeaAgent.Options{
		MaxSteps:      maxStepsValue,
		Temperature:   temperatureValue,
		AccessPolicy:  tool.NewPathAccessPolicy(workspace, readPaths, nil, nil),
		ContextWindow: contextWindowValue,
		DisableVerify: true,
	}, sink)
	result, err := runner.Run(ctx, prompt)
	if err != nil {
		return "", nil, err
	}
	mu.Lock()
	final := finalMessage
	streamedText := streamed.String()
	reported := sawUsage
	seatUsage := usage
	mu.Unlock()
	if result != nil && strings.TrimSpace(result.Summary) != "" {
		final = result.Summary
	}
	if strings.TrimSpace(final) == "" {
		final = streamedText
	}
	final = strings.TrimSpace(final)
	if final == "" {
		return "", nil, fmt.Errorf("Grok 办公席位返回空结果")
	}
	if strings.TrimSpace(streamedText) == "" && emitText != nil {
		emitText(final)
	}
	if !reported {
		return final, nil, nil
	}
	return final, &seatUsage, nil
}

func boundedGrokPreview(text string) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= 1400 {
		return text
	}
	return string(runes[:1400]) + "…"
}
