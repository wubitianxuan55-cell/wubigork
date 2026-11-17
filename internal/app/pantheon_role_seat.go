package app

// 文件席位后端（codex 蒸馏 T15）：把 agents/<role>.md 定义的角色跑在 Gaea
// 办公 AgentRunner 上——persona 即席位 system prompt，模型走办公本地优先
// 路由（与 dream 同口径），工具面与 grok 席同一只读注册表。
//
// 与 grok 席的差异：persona 按角色文件（非 xAI 固定人设）、引擎不限定 xAI
// （本地引擎可用即本地，否则回落对话引擎配置），其余隔离纪律一致
// （workspace 只读注册表 + readPaths 边界 + MaxSteps 预算）。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	gaeaProvider "github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/pantheon"
)

type pantheonRoleSeat struct {
	app *App
}

var _ pantheon.RoleSeatBackend = (*pantheonRoleSeat)(nil)
var _ pantheon.RoleStreamingBackend = (*pantheonRoleSeat)(nil)

// ProbeRole 报告文件席位可用性：办公引擎路由可得即可用（只读内存检查，
// 无网络 IO）。
func (s *pantheonRoleSeat) ProbeRole(_ context.Context, role pantheon.RoleFile) pantheon.AgentStatus {
	status := pantheon.AgentStatus{
		ID:           role.Name,
		Name:         role.Name,
		Kind:         "文件席位",
		Description:  role.Description,
		Capabilities: []string{"multi-round-discussion", "gaea-model", "gaea-office-tools", "read-only-tools", "streaming"},
	}
	a := s.app
	if a == nil || a.core == nil || a.engineMgr == nil {
		status.Reason = "引擎管理器未就绪"
		return status
	}
	if a.cfg != nil && a.cfg.GetOfflineMode() {
		status.Reason = "离线模式下模型席位不可用"
		return status
	}
	// 审计补（2026-11-17）：可用路径必须显式置位——AgentStatus 零值
	// Available=false，漏设会让文件席位探测恒不可用。
	status.Available = true
	return status
}

// CallRole 一次只读议事调用（非流式形态由流式核承载）。
func (s *pantheonRoleSeat) CallRole(ctx context.Context, role pantheon.RoleFile, prompt string) (string, error) {
	return s.StreamRole(ctx, role, prompt, nil)
}

// StreamRole 把角色席位跑在办公 AgentRunner 上：persona=system prompt。
func (s *pantheonRoleSeat) StreamRole(
	ctx context.Context,
	role pantheon.RoleFile,
	prompt string,
	emit func(string),
) (string, error) {
	a := s.app
	if a == nil || a.core == nil || a.engineMgr == nil {
		return "", errors.New("引擎管理器未就绪")
	}
	if a.cfg != nil && a.cfg.GetOfflineMode() {
		return "", errors.New("离线模式下模型席位不可用")
	}
	// 办公本地优先路由（与 dream/识图同口径）：本地引擎可用走本地，否则
	// 回落办公对话引擎配置。
	eng, model, _ := a.routeOfficeLocal("office")
	if strings.TrimSpace(model) == "" {
		return "", errors.New("办公对话模型不可用")
	}
	bridge.SetClient(a.clientRef())
	prov, err := gaeaProvider.NewLLM(gaeaProvider.LLMKindWubigrok, gaeaProvider.Config{
		Name:   "pantheon-role-" + role.Name,
		Model:  model,
		Engine: eng,
		Extra:  map[string]any{"feature": "pantheon"},
	})
	if err != nil {
		return "", err
	}
	workspace := gaeaCwd()
	reg := (&pantheonGrokSeat{app: a}).officeToolRegistry(workspace)
	if reg.Len() == 0 {
		return "", fmt.Errorf("文件席位 %s 的办公工具注册表为空", role.Name)
	}
	// persona 即席位 system prompt；空 persona 回退办公工具基线提示。
	system := strings.TrimSpace(role.Persona)
	if system == "" {
		system = pantheonGrokSystemPrompt(reg.Names())
	}
	overrides, err := pantheon.LoadRoleOverrides(role)
	if err != nil {
		return "", fmt.Errorf("文件席位 %s 配置层无效: %w", role.Name, err)
	}
	return runPantheonAgentWithPrompt(
		ctx, prov, reg, workspace, system, prompt, nil, emit, nil,
		pantheonAgentRunOptions{
			MaxSteps:      overrides.MaxSteps,
			Temperature:   overrides.Temperature,
			ContextWindow: overrides.ContextWindow,
		})
}
