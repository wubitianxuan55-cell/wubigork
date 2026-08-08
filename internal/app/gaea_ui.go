package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// ── gaeaW 原生 UI 绑定（前端 gaea/lib/bridge.ts 适配层映射短名 → Gaea*）──
// 类型与实现对齐 gaeaW desktop/app_*.go，保证办公板块 UI 无改动可用。

// HistoryMessage 是一轮对话消息（历史面板/恢复会话用）。
type HistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SessionMeta 是一个已保存会话（历史面板列表项）。
type SessionMeta struct {
	Path    string `json:"path"`
	Preview string `json:"preview"`
	Title   string `json:"title,omitempty"`
	Turns   int    `json:"turns"`
	ModTime int64  `json:"modTime"`
	Current bool   `json:"current"`
}

// CheckpointMeta 是一个可回退点（用户回合）。
type CheckpointMeta struct {
	Turn   int      `json:"turn"`
	Prompt string   `json:"prompt"`
	Files  []string `json:"files"`
	Time   int64    `json:"time"`
}

var errActiveSession = errors.New("can't delete the session you're in — start a new one first")

// gaeaCtrl 返回当前办公控制器（未初始化返回 nil）。
func gaeaCtrl() *control.Controller {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	return ga.ctrl
}

// ── 会话 ─────────────────────────────────────────────────────────

// GaeaApprove 审批/拒绝一个待批工具调用。
func (a *App) GaeaApprove(id string, allow, session bool) {
	if c := gaeaCtrl(); c != nil {
		c.Approve(id, allow, session)
	}
}

// GaeaAnswer 回答结构化提问。
func (a *App) GaeaAnswer(id string, answers []event.AskAnswer) {
	if c := gaeaCtrl(); c != nil {
		c.AnswerQuestion(id, answers)
	}
}

// GaeaHistory 返回当前会话的对话历史。
func (a *App) GaeaHistory() []HistoryMessage {
	out := []HistoryMessage{}
	c := gaeaCtrl()
	if c == nil {
		return out
	}
	for _, m := range c.History() {
		out = append(out, HistoryMessage{Role: string(m.Role), Content: m.Content})
	}
	return out
}

// GaeaListSessions 返回已保存会话（新→旧），标记当前会话。
func (a *App) GaeaListSessions() []SessionMeta {
	// 会话写入统一在 gaeaCwd()/.gaea/sessions（见 gaeaBuildController），
	// 这里必须用同一路径读取，否则从不同目录启动时历史会"消失"。
	dir := gaeaConfig.WorkspaceSessionDir(gaeaCwd())
	infos, err := agent.ListSessions(dir)
	if err != nil {
		return []SessionMeta{}
	}
	titles := loadSessionTitles(dir)
	cur := ""
	if c := gaeaCtrl(); c != nil {
		cur = c.SessionPath()
	}
	out := make([]SessionMeta, 0, len(infos)+1)
	curFound := false
	for _, s := range infos {
		if s.Path == cur {
			curFound = true
		}
		out = append(out, SessionMeta{
			Path:    s.Path,
			Preview: s.Preview,
			Title:   titles[filepath.Base(s.Path)],
			Turns:   s.Turns,
			ModTime: s.ModTime.UnixMilli(),
			Current: s.Path == cur,
		})
	}
	if cur != "" && !curFound {
		out = append(out, SessionMeta{Path: cur, Preview: "(新会话)", ModTime: time.Now().UnixMilli(), Current: true})
	}
	return out
}

// GaeaDeleteSession 删除已保存会话（拒绝删除当前会话）。
func (a *App) GaeaDeleteSession(path string) error {
	if c := gaeaCtrl(); c != nil && c.SessionPath() == path {
		return errActiveSession
	}
	return deleteSessionFile(gaeaConfig.WorkspaceSessionDir(gaeaCwd()), path)
}

// GaeaRenameSession 设置会话自定义名称（空清除）。
func (a *App) GaeaRenameSession(path, title string) error {
	return setSessionTitle(gaeaConfig.WorkspaceSessionDir(gaeaCwd()), path, title)
}

// GaeaResumeSession 快照当前会话并加载目标会话继续，返回其消息。
func (a *App) GaeaResumeSession(path string) ([]HistoryMessage, error) {
	// 引擎未初始化时先初始化（幂等），避免重启后首次点击会话报"引擎未初始化"
	if err := a.GaeaInit(); err != nil {
		return nil, err
	}
	c := gaeaCtrl()
	if c == nil {
		return nil, errors.New("办公引擎未初始化")
	}
	loaded, err := agent.LoadSession(path)
	if err != nil {
		return nil, err
	}
	_ = c.Snapshot() // 切换前持久化当前会话
	c.Resume(loaded, path)
	c.SeedContextUsage() // 恢复后上下文读数立即反映该会话，而非 0
	return a.GaeaHistory(), nil
}

// resumeLastSession 自动恢复会话目录中最近一次有内容的会话。
// 仅在办公引擎首次初始化时调用；返回最近会话路径（无可恢复会话时为空）。
func (a *App) resumeLastSession(ctrl *control.Controller) string {
	dir := gaeaConfig.WorkspaceSessionDir(gaeaCwd())
	infos, err := agent.ListSessions(dir)
	if err != nil || len(infos) == 0 {
		return ""
	}
	loaded, err := agent.LoadSession(infos[0].Path)
	if err != nil {
		slog.Warn("gaea: 自动恢复最近会话失败", "path", infos[0].Path, "error", err)
		return ""
	}
	ctrl.Resume(loaded, infos[0].Path)
	ctrl.SeedContextUsage()
	slog.Info("gaea: 已自动恢复最近会话", "path", infos[0].Path, "messages", len(loaded.Messages))
	return infos[0].Path
}

// GaeaCheckpoints 列出会话回退点（办公板块暂不支持回退，返回空）。
func (a *App) GaeaCheckpoints() []CheckpointMeta { return []CheckpointMeta{} }

// GaeaRewind/GaeaFork/GaeaSummarizeFrom/GaeaSummarizeUpTo 暂不支持。
// 办公引擎无 checkpoint/分支系统，无法在会话中途回退到历史回合。
var errNoCheckpoint = errors.New("办公引擎不支持会话回退（无 checkpoint 系统）")

func (a *App) GaeaRewind(turn int, scope string) error { return errNoCheckpoint }
func (a *App) GaeaFork(turn int) error                 { return errNoCheckpoint }
func (a *App) GaeaSummarizeFrom(turn int) error        { return errNoCheckpoint }
func (a *App) GaeaSummarizeUpTo(turn int) error        { return errNoCheckpoint }

// ── 会话辅助（照搬 gaeaW desktop/sessions.go）──────────────────────

const sessionTitlesFile = ".titles.json"

func sessionTitlesPath(dir string) string { return filepath.Join(dir, sessionTitlesFile) }

func loadSessionTitles(dir string) map[string]string {
	m := map[string]string{}
	b, err := os.ReadFile(sessionTitlesPath(dir))
	if err != nil {
		return m
	}
	_ = json.Unmarshal(b, &m)
	return m
}

func saveAtomically(dir, pattern, path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.AtomicWrite(path, b, 0o644)
}

func setSessionTitle(dir, sessionPath, title string) error {
	m := loadSessionTitles(dir)
	key := filepath.Base(sessionPath)
	if strings.TrimSpace(title) == "" {
		delete(m, key)
	} else {
		m[key] = strings.TrimSpace(title)
	}
	return saveAtomically(dir, ".titles.*.tmp", sessionTitlesPath(dir), m)
}

func deleteSessionFile(dir, sessionPath string) error {
	// 防御：仅允许删除会话目录内的文件，防止路径逃逸。
	if filepath.Dir(filepath.Clean(sessionPath)) != filepath.Clean(dir) {
		return fmt.Errorf("session outside session dir: %s", sessionPath)
	}
	key := filepath.Base(sessionPath)
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	m := loadSessionTitles(dir)
	if _, ok := m[key]; ok {
		delete(m, key)
		return saveAtomically(dir, ".titles.*.tmp", sessionTitlesPath(dir), m)
	}
	return nil
}
