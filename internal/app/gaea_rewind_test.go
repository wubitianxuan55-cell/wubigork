package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// buildRewindSession 构造一个带真实事件日志的两轮会话（与运行期 sink 写入同构）：
//   轮 0: user「帮我写周报」→ write_file report.md → assistant「完成」→ turn_done
//   轮 1: user「改成英文」→ assistant「Done」→ turn_done
// 返回会话路径（日志已落盘；镜像由投影消息 Save 生成）。
func buildRewindSession(t *testing.T, sessionDir string) string {
	t.Helper()
	path := filepath.Join(sessionDir, "s1.jsonl")
	logPath := session.LogPathFor(path)
	w, err := session.OpenLog(logPath, "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	append := func(kind string, payload any) {
		t.Helper()
		if _, err := w.Append(kind, payload); err != nil {
			t.Fatalf("Append %s: %v", kind, err)
		}
	}
	append(session.KindUserMessage, map[string]any{"content": "帮我写周报"})
	append("turn_started", map[string]any{})
	append("tool_dispatch", map[string]any{"id": "c1", "name": "write_file", "args": `{"path":"report.md"}`})
	append("tool_result", map[string]any{"id": "c1", "name": "write_file", "output": "ok"})
	append("assistant_message", map[string]any{"text": "完成"})
	append("turn_done", map[string]any{})
	append(session.KindUserMessage, map[string]any{"content": "改成英文"})
	append("turn_started", map[string]any{})
	append("assistant_message", map[string]any{"text": "Done"})
	append("turn_done", map[string]any{})
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// 镜像：投影日志为消息写入 .jsonl（列表/历史读取用；日志是真相源）
	entries, err := session.ReadLog(logPath)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	s := agent.NewSession("you are gaea")
	s.Replace(session.ProjectMessages(entries))
	if err := s.Save(path); err != nil {
		t.Fatalf("Save 镜像: %v", err)
	}
	return path
}

// rewindController 装配轻量控制器（事件日志模式）并恢复指定会话。
func rewindController(t *testing.T, path string) *control.Controller {
	t.Helper()
	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	ctrl := control.New(control.Options{Runner: noopStateRunner{}, Executor: exec, Sink: event.Discard})
	ctrl.SetLogFormat("event")
	if _, err := ctrl.ResumeFromDisk(path); err != nil {
		t.Fatalf("ResumeFromDisk: %v", err)
	}
	return ctrl
}

func TestGaeaRewindConversation(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)
	path := buildRewindSession(t, sessionDir)

	ctrl := rewindController(t, path)
	ga.ctrl = ctrl
	defer ctrl.Close()

	a := &App{}
	if err := a.GaeaRewind(1, "conversation"); err != nil {
		t.Fatalf("GaeaRewind(1, conversation): %v", err)
	}
	// 会话只剩第 0 轮
	hist := ctrl.History()
	turns := 0
	for _, m := range hist {
		if m.Role == provider.RoleUser {
			turns++
		}
	}
	if turns != 1 {
		t.Fatalf("回退后回合数 = %d, want 1; hist=%+v", turns, hist)
	}
	if hist[len(hist)-1].Role != provider.RoleAssistant || hist[len(hist)-1].Content != "完成" {
		t.Errorf("最后消息 = %+v, want assistant「完成」（第 0 轮完整保留）", hist[len(hist)-1])
	}
	// 日志已截断：只剩第 0 轮条目
	entries, err := session.ReadLog(session.LogPathFor(path))
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 6 {
		t.Errorf("回退后日志条目 = %d, want 6（第 0 轮）", len(entries))
	}
	// 镜像同步：.jsonl 也只含第 0 轮（投影 = user + assistant(工具调用) + tool + assistant）
	msgs, err := agent.LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	snapshot := msgs.Snapshot()
	if len(snapshot) != 4 {
		t.Errorf("镜像消息数 = %d, want 4（第 0 轮投影）", len(snapshot))
	}
	if snapshot[0].Role != provider.RoleUser || snapshot[0].Content != "帮我写周报" {
		t.Errorf("镜像首条 = %+v, want user「帮我写周报」", snapshot[0])
	}
	if last := snapshot[len(snapshot)-1]; last.Role != provider.RoleAssistant || last.Content != "完成" {
		t.Errorf("镜像末条 = %+v, want assistant「完成」", last)
	}
}

func TestGaeaRewindTurn0(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	path := buildRewindSession(t, gaeaConfig.WorkspaceSessionDir(ws))

	ctrl := rewindController(t, path)
	ga.ctrl = ctrl
	defer ctrl.Close()

	a := &App{}
	if err := a.GaeaRewind(0, "both"); err != nil {
		t.Fatalf("GaeaRewind(0, both): %v", err)
	}
	hist := ctrl.History()
	users := 0
	for _, m := range hist {
		if m.Role == provider.RoleUser {
			users++
		}
	}
	if users != 0 {
		t.Fatalf("回退到第 0 轮之前应无用户消息, got %d", users)
	}
	entries, err := session.ReadLog(session.LogPathFor(path))
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("回退到初始后日志 = %d 条, want 0", len(entries))
	}
}

func TestGaeaRewindUnsupportedScopes(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	path := buildRewindSession(t, gaeaConfig.WorkspaceSessionDir(ws))

	ctrl := rewindController(t, path)
	ga.ctrl = ctrl
	defer ctrl.Close()

	a := &App{}
	for _, scope := range []string{"code", "summ-from", "summ-upto"} {
		if err := a.GaeaRewind(0, scope); err == nil {
			t.Errorf("scope %q 应报错（未支持）, got nil", scope)
		}
	}
	// 不存在的回合也报错
	if err := a.GaeaRewind(99, "conversation"); err == nil {
		t.Error("不存在的 turn 应报错")
	}
	// 引擎未初始化
	ga.ctrl = nil
	if err := a.GaeaRewind(0, "conversation"); err == nil {
		t.Error("引擎未初始化应报错")
	}
}

func TestGaeaFork(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)
	path := buildRewindSession(t, sessionDir)

	ctrl := rewindController(t, path)
	ga.ctrl = ctrl
	defer ctrl.Close()

	a := &App{}
	if err := a.GaeaFork(1); err != nil {
		t.Fatalf("GaeaFork(1): %v", err)
	}
	newPath := ctrl.SessionPath()
	if newPath == path {
		t.Fatal("Fork 后会话路径未切换")
	}
	// 新会话只含第 0 轮（system + user + assistant）
	hist := ctrl.History()
	turns := 0
	for _, m := range hist {
		if m.Role == provider.RoleUser {
			turns++
		}
	}
	if turns != 1 {
		t.Fatalf("分叉后回合数 = %d, want 1", turns)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("新会话文件不存在: %v", err)
	}
	// 分支 meta 记录父分支
	meta, ok, err := session.LoadBranchMeta(newPath)
	if err != nil || !ok {
		t.Fatalf("LoadBranchMeta: ok=%v err=%v", ok, err)
	}
	if meta.ParentID != session.BranchID(path) {
		t.Errorf("ParentID = %q, want %q", meta.ParentID, session.BranchID(path))
	}
	if meta.ForkTurn != 1 {
		t.Errorf("ForkTurn = %d, want 1", meta.ForkTurn)
	}
	// 分叉会话可再次恢复（日志完整）
	if _, err := ctrl.ResumeFromDisk(newPath); err != nil {
		t.Fatalf("分叉会话恢复失败: %v", err)
	}
}

func TestGaeaCheckpoints(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	path := buildRewindSession(t, gaeaConfig.WorkspaceSessionDir(ws))

	ctrl := rewindController(t, path)
	ga.ctrl = ctrl
	defer ctrl.Close()

	a := &App{}
	cps := a.GaeaCheckpoints()
	if len(cps) != 2 {
		t.Fatalf("Checkpoints = %d, want 2; %+v", len(cps), cps)
	}
	if cps[0].Turn != 0 || cps[0].Prompt != "帮我写周报" {
		t.Errorf("cp0 = %+v, want {Turn:0 Prompt:帮我写周报}", cps[0])
	}
	if len(cps[0].Files) != 1 || cps[0].Files[0] != "report.md" {
		t.Errorf("cp0.Files = %v, want [report.md]", cps[0].Files)
	}
	if cps[1].Turn != 1 || cps[1].Prompt != "改成英文" {
		t.Errorf("cp1 = %+v, want {Turn:1 Prompt:改成英文}", cps[1])
	}
	// 引擎未初始化返回空
	ga.ctrl = nil
	if cps := a.GaeaCheckpoints(); len(cps) != 0 {
		t.Errorf("引擎未初始化 Checkpoints = %+v, want 空", cps)
	}
	_ = strings.Contains
}
