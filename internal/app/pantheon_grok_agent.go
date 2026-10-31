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

var pantheonGrokWorkspaceTools = []string{
	"read_file",
	"grep",
	"ls",
	"doc_render",
	"cad_ask",
	"vision",
	"schedule_get",
	"schedule_analyze",
}

var pantheonGrokOfficeReadTools = map[string]bool{
	"cost_search":      true,
	"cost_indicators":  true,
	"cost_compose":     true,
	"knowledge_search": true,
	"memory_search":    true,
	"fact_list":        true,
	"web_search":       true,
	"web_fetch":        true,
	"read_skill":       true,
	"genui_validate":   true,
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
你可以使用当前注册表中的 Gaea 办公只读工具核验工作空间、项目文档、造价库、知识库、记忆与日程信息。只在与当前议题直接相关时调用工具，并说明证据来自哪个工具或结果。
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
) (string, error) {
	if prov == nil || reg == nil {
		return "", fmt.Errorf("Grok 办公 AgentRunner 未完成装配")
	}
	var streamed strings.Builder
	var finalMessage string
	var mu sync.Mutex
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

	session := gaeaAgent.NewSession(pantheonGrokSystemPrompt(reg.Names()))
	runner := gaeaAgent.New(prov, reg, session, gaeaAgent.Options{
		MaxSteps:      12,
		Temperature:   0.2,
		AccessPolicy:  tool.NewPathAccessPolicy(workspace, readPaths, nil, nil),
		ContextWindow: 200_000,
		DisableVerify: true,
	}, sink)
	result, err := runner.Run(ctx, prompt)
	if err != nil {
		return "", err
	}
	mu.Lock()
	final := finalMessage
	streamedText := streamed.String()
	mu.Unlock()
	if result != nil && strings.TrimSpace(result.Summary) != "" {
		final = result.Summary
	}
	if strings.TrimSpace(final) == "" {
		final = streamedText
	}
	final = strings.TrimSpace(final)
	if final == "" {
		return "", fmt.Errorf("Grok 办公席位返回空结果")
	}
	if strings.TrimSpace(streamedText) == "" && emitText != nil {
		emitText(final)
	}
	return final, nil
}

func boundedGrokPreview(text string) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= 1400 {
		return text
	}
	return string(runes[:1400]) + "…"
}
