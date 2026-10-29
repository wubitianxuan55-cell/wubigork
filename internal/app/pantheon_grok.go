package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	gaeaProvider "github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/pantheon"
)

// pantheon_grok.go：众神殿 Grok 模型席位后端（2026-10-29 用户拍板「Grok 当
// 主持」+「主席从参与者中选，加入 Grok」）。Grok 不是裸 xAI 补全：它以 Gaea
// 已配置的 xAI 模型驱动办公 AgentRunner，复用工作区只读文件、搜索、造价、
// 知识与日程工具；鉴权仍走模型中心，pantheon 包只接收文本增量与工具轨迹。

// pantheonGrokSeat 是独立小类型而非 pantheonState 方法：App 按指针嵌入子结构，
// 挂在 pantheonState 上的导出方法会被提升进 App 方法集（绑定面守卫立红），
// 这两个后端方法不是 Wails 绑定。
type pantheonGrokSeat struct {
	app *App
}

var _ pantheon.GrokToolStreamingBackend = (*pantheonGrokSeat)(nil)

// ProbeGrok 返回 Grok 席位可用性（只读内存：引擎表 + 离线开关，无网络 IO）。
// 可用条件：xai 引擎存在、启用、非离线模式、有 Key，且能解析出一个 grok
// 对话模型。Version 槽位放解析出的模型名，前端席位卡直接可见。
func (s *pantheonGrokSeat) ProbeGrok(_ context.Context) pantheon.AgentStatus {
	status := pantheon.AgentStatus{
		ID: "grok", Name: "Grok", Kind: "Gaea 办公席位 · xAI",
		Capabilities: []string{
			"multi-round-discussion",
			"collaborative-synthesis",
			"gaea-model",
			"gaea-office-tools",
			"read-only-tools",
			"tool-trajectory",
			"streaming",
		},
	}
	a := s.app
	if a == nil || a.core == nil || a.engineMgr == nil {
		status.Reason = "引擎管理器未就绪"
		return status
	}
	if a.cfg != nil && a.cfg.GetOfflineMode() {
		status.Reason = "离线模式下云端模型席位不可用"
		return status
	}
	eng, ok := a.engineMgr.GetEngine("xai")
	if !ok {
		status.Reason = "未配置 xAI 引擎"
		return status
	}
	if !eng.Enabled {
		status.Reason = "xAI 引擎已停用"
		return status
	}
	if strings.TrimSpace(a.engineMgr.EngineAPIKey("xai")) == "" {
		status.Reason = "xAI 引擎未配置 API Key"
		return status
	}
	model, ok := grokChatModel(eng)
	if !ok {
		status.Reason = "xAI 引擎下未找到 grok 对话模型"
		return status
	}
	status.Available = true
	status.Version = model
	return status
}

// CallGrok 完成一次只读议事调用。它保留给旧适配器路径，实际同样经过
// Gaea 办公 AgentRunner，不再直接裸调模型。
func (s *pantheonGrokSeat) CallGrok(ctx context.Context, prompt string) (string, error) {
	return s.StreamGrokWithTools(ctx, prompt, gaeaCwd(), nil, nil)
}

func (s *pantheonGrokSeat) StreamGrok(ctx context.Context, prompt string, emit func(string)) (string, error) {
	return s.StreamGrokWithTools(ctx, prompt, gaeaCwd(), emit, nil)
}

// StreamGrokWithTools runs Grok through a fresh Gaea office AgentRunner with a
// workspace-bound, explicitly read-only tool registry. This keeps Pantheon
// isolated from the interactive office session while giving Grok the same
// document, search and domain tools available in the office board.
func (s *pantheonGrokSeat) StreamGrokWithTools(
	ctx context.Context,
	prompt string,
	workspace string,
	emit func(string),
	onTool func(pantheon.ToolTrace),
) (string, error) {
	a := s.app
	if a == nil || a.core == nil || a.engineMgr == nil {
		return "", errors.New("引擎管理器未就绪")
	}
	if a.cfg != nil && a.cfg.GetOfflineMode() {
		return "", errors.New("离线模式下云端模型席位不可用")
	}
	eng, ok := a.engineMgr.GetEngine("xai")
	if !ok || !eng.Enabled {
		return "", errors.New("xAI 引擎不可用")
	}
	model, ok := grokChatModel(eng)
	if !ok {
		return "", errors.New("xAI 引擎下未找到 grok 对话模型")
	}
	bridge.SetClient(a.clientRef())
	prov, err := gaeaProvider.NewLLM(gaeaProvider.LLMKindWubigrok, gaeaProvider.Config{
		Name:   "pantheon-grok",
		Model:  model,
		Engine: "xai",
		Extra:  map[string]any{"feature": "pantheon"},
	})
	if err != nil {
		return "", err
	}
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		workspace = gaeaCwd()
	}
	reg := s.officeToolRegistry(workspace)
	if reg.Len() == 0 {
		return "", fmt.Errorf("Grok 办公工具注册表为空")
	}
	return runPantheonGrokAgent(ctx, prov, reg, workspace, prompt, emit, onTool)
}

// grokChatModel 解析 xai 引擎上的对话模型：默认模型优先（用户在模型中心选
// 的对话主力），默认为空或是画图/语音/视频档时取模型表首个 Kind=llm 条目。
// 实弹（2026-10-29 本机 engines.json）：grok-imagine-video 被目录判成 llm，
// 故额外按名字排除 imagine/tts 档。xai 引擎的对话模型即 grok 家族，不硬编
// 码具体型号名。
func grokChatModel(eng *modelengine.EngineConfig) (string, bool) {
	if m := strings.TrimSpace(eng.DefaultModel); m != "" && isGrokChatModel(m) {
		return m, true
	}
	for _, m := range eng.Models {
		if strings.TrimSpace(m.ID) != "" && m.Kind == "llm" && isGrokChatModel(m.ID) {
			return m.ID, true
		}
	}
	return "", false
}

// isGrokChatModel 名字级防御：排除画图/语音/视频生成档（这些不该主持议事）。
func isGrokChatModel(modelID string) bool {
	if modelengine.ClassifyModelByName(modelID) != "llm" {
		return false
	}
	lower := strings.ToLower(modelID)
	return !strings.Contains(lower, "imagine") && !strings.Contains(lower, "tts")
}
