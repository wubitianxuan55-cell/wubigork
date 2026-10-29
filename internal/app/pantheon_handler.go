package app

import (
	"context"
	"fmt"
	"os"
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

func (s *pantheonState) managerRef() (*pantheon.Manager, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.manager != nil {
		return s.manager, nil
	}
	root := filepath.Join(config.DataRoot(), "pantheon")
	manager, err := pantheon.NewManager(root, nil, func(event pantheon.Event) {
		s.app.emit("pantheon-event", map[string]interface{}{
			"seq":        event.Seq,
			"sessionId":  event.SessionID,
			"phase":      event.Phase,
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
		})
	})
	if err != nil {
		return nil, err
	}
	if _, err := manager.RecoverInterrupted(); err != nil {
		manager.Close()
		return nil, fmt.Errorf("恢复被中断的讨论失败: %w", err)
	}
	manager.SetGrokBackend(&pantheonGrokSeat{app: s.app}) // Grok 模型席位（xai 引擎直调；主持默认优先）
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
