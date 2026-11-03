package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/config"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/pantheon"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// pantheonState is the app-facing lifecycle and event bridge for the local
// council runtime. It is a substructure rather than another flat App field.
// The first council release is deliberately read-only and never grants an
// external agent workspace-write access.
type pantheonState struct {
	app *App

	mu      sync.Mutex
	manager *pantheon.Manager
}

// pantheonEventPayload 构造 SSE 事件面。round 必须在场：前端直播残影与定稿
// 按「席位:轮次」去重，SSE 丢 round 会让所有实时事件轮次归零，每个席位的
// 定稿+残影各渲染一遍（2026-10-31 实弹「上面一条下面一条」根因）。
func pantheonEventPayload(event pantheon.Event) map[string]interface{} {
	return map[string]interface{}{
		"seq":        event.Seq,
		"sessionId":  event.SessionID,
		"phase":      event.Phase,
		"round":      event.Round,
		"agentId":    event.AgentID,
		"kind":       event.Kind,
		"text":       event.Text,
		"toolId":     event.ToolID,
		"toolName":   event.ToolName,
		"toolStatus": event.ToolStatus,
		"toolInput":  event.ToolInput,
		"toolOutput": event.ToolOutput,
		"at":         event.At,
		"durationMs": event.DurationMs,
		"error":      event.Error,
		// 决策门与提问（O-E）：前端要能凭事件渲染问题卡与门，而不只靠
		// 重新拉一次快照——直播与定稿必须同源。
		"question": event.Question,
		"answer":   event.Answer,
		"gate":     event.Gate,
	}
}

func (s *pantheonState) managerRef() (*pantheon.Manager, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.manager != nil {
		return s.manager, nil
	}
	root := filepath.Join(config.DataRoot(), "pantheon")
	manager, err := pantheon.NewManager(root, nil, func(event pantheon.Event) {
		s.app.emit("pantheon-event", pantheonEventPayload(event))
	})
	if err != nil {
		return nil, err
	}
	if _, err := manager.RecoverInterrupted(); err != nil {
		manager.Close()
		return nil, fmt.Errorf("恢复被中断的讨论失败: %w", err)
	}
	manager.SetGrokBackend(&pantheonGrokSeat{app: s.app})          // Grok 模型席位（xai 引擎直调；主持默认优先）
	manager.SetDSHAPIKey(s.app.engineMgr.EngineAPIKey("deepseek")) // dsh 陈词凭据：DSH 凭据服务无存档时官方 env 通道兜底
	manager.SetWorkDir(gaeaCwd())
	s.manager = manager
	return s.manager, nil
}

func (s *pantheonState) closePantheon() {
	s.mu.Lock()
	manager := s.manager
	s.manager = nil
	s.mu.Unlock()
	if manager != nil {
		manager.Close()
	}
}

func (s *pantheonState) PantheonAgents() ([]pantheon.AgentStatus, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return manager.Agents(ctx), nil
}

func (s *pantheonState) PantheonWorkspaces() []pantheon.WorkspaceOption {
	current := filepath.Clean(gaeaCwd())
	seen := map[string]bool{}
	options := make([]pantheon.WorkspaceOption, 0, 12)
	add := func(path string, isCurrent, recent bool) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		path = filepath.Clean(path)
		key := path
		if filepath.Separator == '\\' {
			key = strings.ToLower(key)
		}
		if seen[key] {
			return
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return
		}
		seen[key] = true
		name := filepath.Base(path)
		if name == "." || name == string(filepath.Separator) {
			name = path
		}
		options = append(options, pantheon.WorkspaceOption{
			Path: path, Name: name, Current: isCurrent, Recent: recent,
		})
	}
	add(current, true, false)
	for _, path := range gaeaConfig.LoadRecentWorkspaces() {
		add(path, false, true)
	}
	return options
}

// PantheonPickWorkspace opens a directory chooser without switching Gaea's
// global workspace. The chosen directory is scoped to the next council session.
func (s *pantheonState) PantheonPickWorkspace() (string, error) {
	if s.app == nil || s.app.ctx == nil {
		return "", fmt.Errorf("应用尚未就绪，无法选择工作空间")
	}
	defaultDir := gaeaCwd()
	if info, err := os.Stat(defaultDir); err != nil || !info.IsDir() {
		defaultDir = ""
	}
	dir, err := runtime.OpenDirectoryDialog(s.app.ctx, runtime.OpenDialogOptions{
		Title:            "选择本次议事的工作空间",
		DefaultDirectory: defaultDir,
	})
	if err != nil {
		return "", fmt.Errorf("打开工作空间选择器失败: %w", err)
	}
	if strings.TrimSpace(dir) == "" {
		return "", nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("所选工作空间不可访问: %s", dir)
	}
	return filepath.Clean(abs), nil
}

func (s *pantheonState) PantheonList() ([]pantheon.SessionSummary, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	return manager.List()
}

// PantheonEvaluation is a read-only R6 report over the durable council ledger.
func (s *pantheonState) PantheonEvaluation(minCases int) (pantheon.EvaluationReport, error) {
	manager, err := s.managerRef()
	if err != nil {
		return pantheon.EvaluationReport{}, err
	}
	return manager.EvaluateProtocols(minCases)
}

func (s *pantheonState) PantheonGet(id string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	return manager.Get(id)
}

func (s *pantheonState) PantheonStart(req pantheon.StartRequest) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	attachments, err := pantheon.NormalizeAttachments(req.Attachments)
	if err != nil {
		return nil, err
	}
	req.Attachments = attachments
	visualEvidence, err := pantheonAttachmentEvidence(attachments)
	if err != nil {
		return nil, err
	}
	baseEvidence := req.Evidence[:0]
	for _, item := range req.Evidence {
		if !strings.HasPrefix(item.Name, "共享识图 · ") {
			baseEvidence = append(baseEvidence, item)
		}
	}
	req.Evidence = append(baseEvidence, visualEvidence...)
	manager.SetWorkDir(gaeaCwd())
	session, err := manager.Start(req)
	if err == nil && session != nil && session.Request.Workspace != "" {
		gaeaConfig.TouchRecentWorkspace(session.Request.Workspace)
	}
	return session, err
}

func (s *pantheonState) PantheonCancel(id string) error {
	manager, err := s.managerRef()
	if err != nil {
		return err
	}
	return manager.Cancel(id)
}

func (s *pantheonState) PantheonSkipAgent(id, agentID string) error {
	manager, err := s.managerRef()
	if err != nil {
		return err
	}
	return manager.SkipAgent(id, agentID)
}

func (s *pantheonState) PantheonRetryAgent(id, agentID string) error {
	manager, err := s.managerRef()
	if err != nil {
		return err
	}
	return manager.RetryAgent(id, agentID)
}

func (s *pantheonState) PantheonRestart(id string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	manager.SetWorkDir(gaeaCwd())
	session, err := manager.Restart(id)
	if err == nil && session != nil && session.Request.Workspace != "" {
		gaeaConfig.TouchRecentWorkspace(session.Request.Workspace)
	}
	return session, err
}

// PantheonResume 恢复被中断的讨论：保留已有贡献，从中断处下一轮继续
// （2026-10-31 实弹反馈「中途保存与临时变故的保存，方便后续继续讨论」）。
func (s *pantheonState) PantheonResume(id string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	manager.SetWorkDir(gaeaCwd())
	return manager.Resume(id)
}

func (s *pantheonState) PantheonDelete(id string) error {
	manager, err := s.managerRef()
	if err != nil {
		return err
	}
	return manager.Delete(id)
}

func (s *pantheonState) PantheonExport(id string) (string, error) {
	manager, err := s.managerRef()
	if err != nil {
		return "", err
	}
	return manager.Export(id)
}

func (s *pantheonState) PantheonOpenResult(id string) (string, error) {
	manager, err := s.managerRef()
	if err != nil {
		return "", err
	}
	path, err := manager.ResultPath(id)
	if err != nil {
		return "", err
	}
	if err := exec.Command("explorer", path).Start(); err != nil {
		return path, fmt.Errorf("打开结果文档失败: %w", err)
	}
	return path, nil
}

func (s *pantheonState) PantheonSignOff(id string, decision string, note string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	return manager.SignOff(id, decision, note)
}

func (s *pantheonState) PantheonSetExecutable(id string, path string) ([]pantheon.AgentStatus, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	return manager.SetExecutable(id, path)
}

// PantheonAskSeat 让一席向用户提问：整场转 waiting_input，等人回答。
// 不自动续跑——续跑是一次独立、可见的动作（PantheonResume）。
func (s *pantheonState) PantheonAskSeat(id, agentID, question string, options []string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	return manager.AskSeat(id, agentID, question, options)
}

func (s *pantheonState) PantheonAnswerSeat(id, questionID, body string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	return manager.AnswerSeat(id, questionID, body)
}

// PantheonAddUserNote 记录用户在讨论进行中的评论：不打断当前发言，下一席
// 开始全文注入【用户插话】（2026-11-03 拍板）。
func (s *pantheonState) PantheonAddUserNote(id, content string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	return manager.AddUserNote(id, content)
}

// PantheonOpenGate 在轮次之间开一道决策门；PantheonResolveGate 放行或放弃。
func (s *pantheonState) PantheonOpenGate(id, reason string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	return manager.OpenGate(id, reason)
}

func (s *pantheonState) PantheonResolveGate(id, action, reason string) (*pantheon.Session, error) {
	manager, err := s.managerRef()
	if err != nil {
		return nil, err
	}
	manager.SetWorkDir(gaeaCwd())
	return manager.ResolveGate(id, action, reason)
}
