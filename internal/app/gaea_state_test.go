package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// noopStateRunner 是轻量控制器的占位 runner（测试只走 Resume/History，不跑回合）。
type noopStateRunner struct{}

func (noopStateRunner) Run(ctx context.Context, input string) (*agent.TurnResult, error) {
	return &agent.TurnResult{Success: true}, nil
}

// TestGaeaListSessionsInterrupted 验证 state 残留 running=true 的会话在列表
// 中标记 Interrupted；无 state / running=false 的会话不标记。
func TestGaeaListSessionsInterrupted(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	ga.cfg, ga.ctrl = nil, nil
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)
	writeProjectSession(t, sessionDir, "s1", "整理季度报告", time.Now().Add(-time.Hour))
	writeProjectSession(t, sessionDir, "s2", "写市场方案", time.Now().Add(-30*time.Minute))

	a := &App{}
	// 无 state 文件 → 不标记
	for _, m := range a.GaeaListSessions() {
		if m.Interrupted {
			t.Fatalf("无 state 的会话被标记 Interrupted: %+v", m)
		}
	}

	// s1 崩溃残留 running=true → 标记；s2 正常结束 running=false → 不标记
	s1 := filepath.Join(sessionDir, "s1.jsonl")
	if err := session.SaveState(session.StatePath(s1), session.SessionState{
		Running: true, Summary: "正在输出表格", UpdatedAt: time.Now().UnixMilli(),
	}); err != nil {
		t.Fatalf("SaveState s1: %v", err)
	}
	s2 := filepath.Join(sessionDir, "s2.jsonl")
	if err := session.SaveState(session.StatePath(s2), session.SessionState{
		Running: false, Summary: "正常完成", UpdatedAt: time.Now().UnixMilli(),
	}); err != nil {
		t.Fatalf("SaveState s2: %v", err)
	}
	for _, m := range a.GaeaListSessions() {
		switch filepath.Base(m.Path) {
		case "s1.jsonl":
			if !m.Interrupted {
				t.Fatalf("s1 应标记 Interrupted: %+v", m)
			}
		case "s2.jsonl":
			if m.Interrupted {
				t.Fatalf("s2 不应标记 Interrupted: %+v", m)
			}
		}
	}
}

// TestGaeaResumeSessionInjectsInterruption 验证恢复一个 state 残留 running=true
// 的会话时：注入中断摘要 system 消息、清除 state 文件；非中断会话不注入、
// state 文件保留。
func TestGaeaResumeSessionInjectsInterruption(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)

	// 构造会话：用户回合 + 助手回复
	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "帮我写季度总结"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "已完成数据收集"})
	path := filepath.Join(sessionDir, "s1.jsonl")
	if err := s.Save(path); err != nil {
		t.Fatalf("保存会话: %v", err)
	}
	stPath := session.StatePath(path)
	if err := session.SaveState(stPath, session.SessionState{
		Running: true, Summary: "正在生成表格", UpdatedAt: 1,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// 轻量控制器（不走真实引擎 boot）：GaeaResumeSession 只用到
	// Snapshot/Resume/SeedContextUsage/History，足够验证注入流程。
	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	ctrl := control.New(control.Options{Runner: noopStateRunner{}, Executor: exec, Sink: event.Discard})
	ga.ctrl = ctrl
	defer ctrl.Close()

	a := &App{}
	hist, err := a.GaeaResumeSession(path)
	if err != nil {
		t.Fatalf("GaeaResumeSession: %v", err)
	}
	// 注入的 system 中断摘要追加在消息末尾（Append 语义，带位置引号）
	if len(hist) == 0 || hist[len(hist)-1].Role != "system" || !strings.Contains(hist[len(hist)-1].Content, "「正在生成表格」") {
		t.Fatalf("恢复后缺少中断摘要: %+v", hist)
	}
	// state 文件应已清除
	if _, err := os.Stat(stPath); !os.IsNotExist(err) {
		t.Fatalf("恢复后 state 文件未清除")
	}
	// 控制器会话应已包含注入消息
	h := ctrl.History()
	if len(h) == 0 || h[len(h)-1].Role != provider.RoleSystem || !strings.Contains(h[len(h)-1].Content, "中断") {
		t.Fatalf("控制器会话未注入中断摘要: %+v", h)
	}
}

// TestGaeaResumeSessionNoInterruption 验证非中断会话（running=false）恢复时
// 不注入、state 文件保留。
func TestGaeaResumeSessionNoInterruption(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)

	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "帮我写季度总结"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "已完成数据收集"})
	path := filepath.Join(sessionDir, "s1.jsonl")
	if err := s.Save(path); err != nil {
		t.Fatalf("保存会话: %v", err)
	}
	stPath := session.StatePath(path)
	if err := session.SaveState(stPath, session.SessionState{
		Running: false, Summary: "正常结束", UpdatedAt: 2,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	ctrl := control.New(control.Options{Runner: noopStateRunner{}, Executor: exec, Sink: event.Discard})
	ga.ctrl = ctrl
	defer ctrl.Close()

	a := &App{}
	hist, err := a.GaeaResumeSession(path)
	if err != nil {
		t.Fatalf("GaeaResumeSession: %v", err)
	}
	for _, m := range hist {
		if m.Role == "system" && strings.Contains(m.Content, "中断") {
			t.Fatalf("非中断会话不应注入中断摘要: %+v", hist)
		}
	}
	if _, err := os.Stat(stPath); err != nil {
		t.Fatalf("非中断会话的 state 文件不应被清除")
	}
}

// TestInterruptionMessage 验证中断摘要文案：有位置带引号，空位置省略引号。
func TestInterruptionMessage(t *testing.T) {
	m := interruptionMessage("正在生成表格")
	if !strings.Contains(m, "「正在生成表格」") || !strings.Contains(m, "总结已完成进度") {
		t.Fatalf("interruptionMessage 文案异常: %q", m)
	}
	m2 := interruptionMessage("   ")
	if strings.Contains(m2, "「") || !strings.Contains(m2, "上次会话中断") {
		t.Fatalf("空位置不应带引号: %q", m2)
	}
}

// TestLastUserPreview 验证最后用户消息预览：取最后一条用户消息并截断。
func TestLastUserPreview(t *testing.T) {
	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "第一条"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "回复"})
	s.Add(provider.Message{Role: provider.RoleUser, Content: "最后一条"})
	if got := lastUserPreview(s); got != "最后一条" {
		t.Fatalf("lastUserPreview = %q, want 最后一条", got)
	}
	long := strings.Repeat("长", 100)
	s2 := agent.NewSession("you are gaea")
	s2.Add(provider.Message{Role: provider.RoleUser, Content: long})
	if got := lastUserPreview(s2); len([]rune(got)) > 80 {
		t.Fatalf("lastUserPreview 未截断: %d rune", len([]rune(got)))
	}
}
