package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/agent"
	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
)

// writeProjectSession 写一个有效会话（1 个用户回合，turns>0 才会被
// session.List 收录），并固定 mtime 以便断言排序。
func writeProjectSession(t *testing.T, dir, name, prompt string, mod time.Time) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: prompt})
	path := filepath.Join(dir, name+".jsonl")
	if err := s.Save(path); err != nil {
		t.Fatalf("保存会话 %s: %v", path, err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatalf("设置 mtime %s: %v", path, err)
	}
}

// TestGaeaListProjectSessions 验证按项目聚合：
//   - 当前工作区始终在最前（即使没有会话）；
//   - 最近工作区按最近会话时间倒序；
//   - 无会话的非当前工作区不出现；
//   - 不存在的路径不出现。
func TestGaeaListProjectSessions(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	oldCtrl := ga.ctrl
	ga.cfg = nil
	ga.ctrl = nil
	defer func() { ga.cfg = oldCfg; ga.ctrl = oldCtrl }()

	wsA := t.TempDir()
	wsB := t.TempDir()
	wsEmpty := t.TempDir()

	now := time.Now()
	writeProjectSession(t, gaeaConfig.WorkspaceSessionDir(wsA, ""), "s1", "整理季度报告", now.Add(-2*time.Hour))
	writeProjectSession(t, gaeaConfig.WorkspaceSessionDir(wsB, ""), "s1", "写市场方案", now.Add(-30*time.Minute))
	writeProjectSession(t, gaeaConfig.WorkspaceSessionDir(wsB, ""), "s2", "做竞品表格", now.Add(-3*time.Hour))

	ga.cfg = &gaeaConfig.Config{Workspace: wsA}
	gaeaConfig.TouchRecentWorkspace(wsB)
	gaeaConfig.TouchRecentWorkspace(wsEmpty) // 无会话 → 应被过滤
	gaeaConfig.TouchRecentWorkspace(filepath.Join(t.TempDir(), "不存在"))

	a := &App{}
	groups := a.GaeaListProjectSessions()

	if len(groups) != 2 {
		t.Fatalf("分组数量 = %d, want 2 (当前 + 最近有会话)", len(groups))
	}
	if !groups[0].Current || groups[0].Path != wsA {
		t.Errorf("groups[0] = %+v, want 当前工作区 %s", groups[0], wsA)
	}
	if groups[1].Path != wsB || groups[1].Current {
		t.Errorf("groups[1] = %+v, want 工作区 %s 且非当前", groups[1], wsB)
	}
	if len(groups[1].Sessions) != 2 {
		t.Errorf("工作区 B 会话数 = %d, want 2", len(groups[1].Sessions))
	}
	// B 的会话按新→旧
	if groups[1].Sessions[0].Preview != "写市场方案" {
		t.Errorf("B 最新会话 = %q, want 写市场方案", groups[1].Sessions[0].Preview)
	}
	// 当前工作区 A 有 1 个会话
	if len(groups[0].Sessions) != 1 {
		t.Errorf("工作区 A 会话数 = %d, want 1", len(groups[0].Sessions))
	}
}

// TestGaeaListProjectSessions_CurrentEmpty 验证当前工作区即使没有会话
// 也会作为空分组出现（前端展示「新建会话」空状态），而不是被过滤掉。
func TestGaeaListProjectSessions_CurrentEmpty(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	oldCtrl := ga.ctrl
	ga.cfg = nil
	ga.ctrl = nil
	defer func() { ga.cfg = oldCfg; ga.ctrl = oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}

	a := &App{}
	groups := a.GaeaListProjectSessions()
	if len(groups) != 1 {
		t.Fatalf("分组数量 = %d, want 1（当前空工作区）", len(groups))
	}
	if !groups[0].Current || len(groups[0].Sessions) != 0 {
		t.Errorf("当前空工作区分组 = %+v, want Current 且 0 会话", groups[0])
	}
}

// TestGaeaListSessionsFallback 验证重构后 GaeaListSessions 行为不变：
// 当前未落盘会话仍补为「(新会话)」条目。
func TestGaeaListSessionsFallback(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	oldCtrl := ga.ctrl
	ga.cfg = nil
	ga.ctrl = nil
	defer func() { ga.cfg = oldCfg; ga.ctrl = oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}

	a := &App{}
	// 无 controller → cur 为空 → 不补回退条目，且目录不存在也不报错
	if got := a.GaeaListSessions(); len(got) != 0 {
		t.Fatalf("无会话时 GaeaListSessions() = %+v, want 空", got)
	}
}

// TestSessionDirForPath 验证跨项目删除/重命名时的路径守卫：
// 只接受 <root>/.gaea/sessions/<file> 形态，其他路径一律拒绝。
func TestSessionDirForPath(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, ".gaea", "sessions", "a.jsonl")
	if got := sessionDirForPath(good); got != filepath.Dir(good) {
		t.Errorf("sessionDirForPath(%q) = %q, want %q", good, got, filepath.Dir(good))
	}
	archivedGood := filepath.Join(root, ".gaea", "sessions", "archive", "a.jsonl")
	if got := sessionDirForPath(archivedGood); got != filepath.Dir(archivedGood) {
		t.Errorf("sessionDirForPath(%q) = %q, want %q（归档路径应放行）", archivedGood, got, filepath.Dir(archivedGood))
	}
	for _, bad := range []string{
		filepath.Join(root, "elsewhere", "a.jsonl"),
		filepath.Join(root, ".gaea", "archive", "a.jsonl"),
		filepath.Join(root, ".gaea", "sessions", "sub", "a.jsonl"),
		filepath.Join(root, ".gaea", "sessions", "archive", "sub", "a.jsonl"),
		"relative/a.jsonl",
	} {
		if got := sessionDirForPath(bad); got != "" {
			t.Errorf("sessionDirForPath(%q) = %q, want 空（非法路径应拒绝）", bad, got)
		}
	}
}

// TestGaeaArchiveUnarchive 验证归档/恢复闭环：
// 归档后从活动列表消失并出现在 archived 分组，恢复后回到活动列表。
func TestGaeaArchiveUnarchive(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	oldCtrl := ga.ctrl
	ga.cfg = nil
	ga.ctrl = nil
	defer func() { ga.cfg = oldCfg; ga.ctrl = oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws, "")
	writeProjectSession(t, sessionDir, "s1", "季度总结", time.Now().Add(-time.Hour))

	a := &App{}
	active := a.GaeaListSessions()
	if len(active) != 1 {
		t.Fatalf("归档前活动会话数 = %d, want 1", len(active))
	}
	path := active[0].Path

	if err := a.GaeaArchiveSession(path); err != nil {
		t.Fatalf("GaeaArchiveSession: %v", err)
	}
	if got := a.GaeaListSessions(); len(got) != 0 {
		t.Fatalf("归档后活动会话数 = %d, want 0", len(got))
	}
	groups := a.GaeaListProjectSessions()
	if len(groups) != 1 || len(groups[0].Archived) != 1 {
		t.Fatalf("归档分组 = %+v, want 1 个项目且 archived=1", groups)
	}
	archivedPath := groups[0].Archived[0].Path

	restored, err := a.GaeaUnarchiveSession(archivedPath)
	if err != nil {
		t.Fatalf("GaeaUnarchiveSession: %v", err)
	}
	if restored != path {
		t.Errorf("恢复路径 = %q, want %q", restored, path)
	}
	if got := a.GaeaListSessions(); len(got) != 1 {
		t.Fatalf("恢复后活动会话数 = %d, want 1", len(got))
	}
}

// TestGaeaPinSession 验证置顶：置顶会话排在最前且标记 Pinned。
func TestGaeaPinSession(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	oldCtrl := ga.ctrl
	ga.cfg = nil
	ga.ctrl = nil
	defer func() { ga.cfg = oldCfg; ga.ctrl = oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws, "")
	now := time.Now()
	writeProjectSession(t, sessionDir, "old", "旧会话", now.Add(-2*time.Hour))
	writeProjectSession(t, sessionDir, "new", "新会话", now.Add(-time.Hour))

	a := &App{}
	all := a.GaeaListSessions()
	if len(all) != 2 {
		t.Fatalf("会话数 = %d, want 2", len(all))
	}
	// 默认新→旧
	if filepath.Base(all[0].Path) != "new.jsonl" {
		t.Errorf("默认排序首个 = %q, want new.jsonl", filepath.Base(all[0].Path))
	}
	oldPath := all[1].Path
	if err := a.GaeaPinSession(oldPath, true); err != nil {
		t.Fatalf("GaeaPinSession: %v", err)
	}
	after := a.GaeaListSessions()
	if !after[0].Pinned || filepath.Base(after[0].Path) != "old.jsonl" {
		t.Errorf("置顶后首个 = %+v, want old.jsonl 且 Pinned", after[0])
	}
	if err := a.GaeaPinSession(oldPath, false); err != nil {
		t.Fatalf("取消置顶: %v", err)
	}
	again := a.GaeaListSessions()
	if again[0].Pinned || filepath.Base(again[0].Path) != "new.jsonl" {
		t.Errorf("取消置顶后首个 = %+v, want new.jsonl 且未置顶", again[0])
	}
}

// TestGaeaNewSessionClearsGoal 验证新会话清空 goal gate：上个会话的
// 「持续工作到验收」目标不残留到新会话（手动 /goal 同样需要重设）。
func TestGaeaNewSessionClearsGoal(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	ctrl := control.New(control.Options{
		Runner:     noopStateRunner{},
		Executor:   exec,
		Sink:       event.Discard,
		SessionDir: t.TempDir(),
	})
	ga.ctrl = ctrl
	defer ctrl.Close()

	ctrl.SetGoal("旧目标")
	a := &App{}
	if err := a.GaeaNewSession(); err != nil {
		t.Fatalf("GaeaNewSession: %v", err)
	}
	if g := ctrl.Goal(); g != "" {
		t.Fatalf("新会话后 goal = %q, want 空", g)
	}
}

// TestGaeaHistoryToolEvents 验证历史接口带回工具事件：
// 恢复会话后过程卡与「变更」面板仍可见（Kun 可观察性闭环）。
func TestGaeaHistoryToolEvents(t *testing.T) {
	oldwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldwd) }()
	ws := t.TempDir()
	_ = os.Chdir(ws)

	sessionDir := gaeaConfig.WorkspaceSessionDir(ws, "")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "帮我改方案"})
	s.Add(provider.Message{
		Role:    provider.RoleAssistant,
		Content: "好的",
		ToolCalls: []provider.ToolCall{{ID: "call_1", Name: "edit_file", Arguments: `{"path":"方案.md","edits":[]}`}},
	})
	s.Add(provider.Message{Role: provider.RoleTool, Name: "edit_file", ToolCallID: "call_1", Content: "已更新 方案.md"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "完成"})
	path := filepath.Join(sessionDir, "s1.jsonl")
	if err := s.Save(path); err != nil {
		t.Fatalf("保存会话: %v", err)
	}

	bridge.SetClient(ai.NewClient(config.Load()))
	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "gaea"
	cfg.Providers = []gaeaConfig.ProviderEntry{{Name: "gaea", Kind: "wubigrok", Model: "", ContextWindow: 1_000_000}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	defer gaeaConfig.SetLoader(nil)

	ctrl, err := gaeaBoot.Build(context.Background(), gaeaBoot.Options{
		Model:      "gaea",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		MaxSteps:   0,
		SessionDir: sessionDir,
		Cwd:        ws,
	})
	if err != nil {
		t.Fatalf("办公引擎构建失败: %v", err)
	}
	defer ctrl.Close()
	loaded, err := agent.LoadSession(path)
	if err != nil {
		t.Fatalf("加载会话: %v", err)
	}
	ctrl.Resume(loaded, path)

	oldCtrl := ga.ctrl
	ga.ctrl = ctrl
	defer func() { ga.ctrl = oldCtrl }()

	a := &App{}
	hist := a.GaeaHistory()
	var dispatch, result *HistoryMessage
	for i := range hist {
		if hist[i].Role == "tool" && hist[i].ToolID == "call_1" {
			dispatch = &hist[i]
		}
		if hist[i].Role == "tool_result" && hist[i].ToolID == "call_1" {
			result = &hist[i]
		}
	}
	if dispatch == nil || dispatch.ToolName != "edit_file" || !strings.Contains(dispatch.ToolArgs, "方案.md") {
		t.Fatalf("缺少工具 dispatch 条目: %+v", hist)
	}
	if result == nil || !strings.Contains(result.ToolOutput, "已更新") {
		t.Fatalf("缺少工具结果条目: %+v", hist)
	}
}

// TestGaeaDeleteSessionClearsRegistries 验证删除会话时，标题、置顶两处注册表
// 都会清理干净，不会出现“删掉文件但留下孤儿条目”的回归。
func TestGaeaDeleteSessionClearsRegistries(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	oldCtrl := ga.ctrl
	ga.cfg = nil
	ga.ctrl = nil
	defer func() { ga.cfg = oldCfg; ga.ctrl = oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws, "")
	writeProjectSession(t, sessionDir, "s1", "年度总结", time.Now().Add(-time.Hour))

	a := &App{}
	path := a.GaeaListSessions()[0].Path
	base := filepath.Base(path)

	if err := a.GaeaRenameSession(path, "自定义标题"); err != nil {
		t.Fatalf("GaeaRenameSession: %v", err)
	}
	if err := a.GaeaPinSession(path, true); err != nil {
		t.Fatalf("GaeaPinSession: %v", err)
	}

	if err := a.GaeaDeleteSession(path); err != nil {
		t.Fatalf("GaeaDeleteSession: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("会话文件应被删除, stat err=%v", err)
	}
	if titles := loadSessionTitles(sessionDir); titles[base] != "" {
		t.Errorf("标题注册表残留: %q", titles[base])
	}
	if pinned := loadPinned(sessionDir); pinned[base] {
		t.Errorf("置顶注册表残留: %v", base)
	}
}
