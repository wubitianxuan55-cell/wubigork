package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/agent"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
)

// TestSyncGoalForSessionWhileLocked 回归：GaeaInit 持 ga.mu 初始化时会走
// resumeLastSession → syncGoalForSession，因此 syncGoalForSession 不得再经
// gaeaCtrl() 对同一把非重入互斥锁二次加锁（v3.0.5 引入，办公板块首次打开
// 即永久死锁——「连接中…」、会话列表/工作空间不可用、无法新建会话）。
// 本测试直接模拟 GaeaInit 的持锁上下文调用，超时即视为死锁。
func TestSyncGoalForSessionWhileLocked(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	dir := gaeaConfig.WorkspaceSessionDir(ws)
	writeProjectSession(t, dir, "s1", "起草年度总结", time.Now().Add(-time.Hour))
	path := filepath.Join(dir, "s1.jsonl")

	a := &App{}
	if err := a.GaeaSetRequirement(path, "起草年度总结报告并输出 docx"); err != nil {
		t.Fatalf("设置目标: %v", err)
	}
	if err := a.GaeaSetRequirementAutoPursue(path, true); err != nil {
		t.Fatalf("开启自动追踪: %v", err)
	}

	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	ctrl := control.New(control.Options{Runner: noopStateRunner{}, Executor: exec, Sink: event.Discard})
	defer ctrl.Close()
	// 让控制器接管该会话（等价 GaeaResumeSession 的 ResumeFromDisk 路径），
	// 使 ctrl.SessionPath() 与 path 一致，syncGoalForSession 才会真正执行。
	ctrl.Resume(agent.NewSession("you are gaea"), path)

	// 模拟 GaeaInit：ga.mu 已持有（gaea_handler.go GaeaInit 顶部 Lock + defer Unlock）。
	ga.mu.Lock()
	done := make(chan struct{})
	go func() {
		a.syncGoalForSession(ctrl, path)
		close(done)
	}()
	select {
	case <-done:
		// 修复后：持锁上下文直接完成，不再二次取 ga.mu。
	case <-time.After(5 * time.Second):
		ga.mu.Unlock()
		t.Fatal("syncGoalForSession 在 ga.mu 持锁上下文死锁（gaeaCtrl 二次加锁）")
	}
	ga.mu.Unlock()

	if g := ctrl.Goal(); g != "起草年度总结报告并输出 docx" {
		t.Fatalf("goal = %q, want 目标文本", g)
	}
}

// TestGaeaInitAutoResumeWithSessions 回归用户场景：工作区存在会话时首次
// GaeaInit 必须完成（此前在 resumeLastSession → syncGoalForSession 处对
// ga.mu 二次加锁永久卡死），且初始化后新建会话/切换工作空间均可用。
func TestGaeaInitAutoResumeWithSessions(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}()

	bridge.SetClient(ai.NewClient(config.Load()))

	// 种子配置：工作区指向临时目录（含一个可恢复会话）。
	ws := t.TempDir()
	seed := gaeaConfig.Default()
	seed.DefaultModel = "gaea"
	seed.Workspace = ws
	seed.Providers = []gaeaConfig.ProviderEntry{{
		Name:          "gaea",
		Kind:          "wubigrok",
		Model:         "",
		ContextWindow: 1_000_000,
	}}
	seed.Tools.Enabled = nil
	seed.Sandbox.Bash = "off"
	if err := gaeaConfig.Save(seed); err != nil {
		t.Fatalf("种子配置保存失败: %v", err)
	}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)
	writeProjectSession(t, sessionDir, "s1", "起草年度总结", time.Now().Add(-time.Hour))
	path := filepath.Join(sessionDir, "s1.jsonl")
	a := &App{}
	if err := a.GaeaSetRequirement(path, "起草年度总结报告并输出 docx"); err != nil {
		t.Fatalf("设置目标: %v", err)
	}

	a = &App{core: &core{
		ctx:    context.Background(),
		cfg:    config.Load(),
		client: ai.NewClient(config.Load()),
	}}

	// 快照全局 gaea 运行时，成功路径结束后恢复，避免污染同包其他测试。
	// 注意：超时分支不能走 defer——修复前 GaeaInit 卡在 ga.mu 内，defer 的
	// ga.mu.Lock() 会再次挂起整个测试进程，必须立即退出测试二进制。
	ga.mu.Lock()
	oldCtrl, oldCfg := ga.ctrl, ga.cfg
	ga.ctrl, ga.cfg = nil, nil
	ga.mu.Unlock()
	restoreGaea := func() {
		ga.mu.Lock()
		defer ga.mu.Unlock()
		if ga.ctrl != nil && ga.ctrl != oldCtrl {
			ga.ctrl.Close()
		}
		ga.ctrl, ga.cfg = oldCtrl, oldCfg
		_ = db.CloseDatabase(gaeaConfig.MemoryUserDir())
	}

	// 带超时执行 GaeaInit：修复前这里永久卡死（办公板块「连接中…」）。
	done := make(chan error, 1)
	go func() { done <- a.GaeaInit() }()
	var initErr error
	select {
	case initErr = <-done:
	case <-time.After(30 * time.Second):
		// 死锁无法恢复：直接退出测试二进制，避免同包后续测试永久挂起。
		t.Log("GaeaInit 超时：resumeLastSession 在 ga.mu 持锁上下文死锁（回归）")
		os.Exit(1)
	}
	// 初始化已完成（死锁窗口已过），此后可安全 defer 清理。
	defer restoreGaea()
	if initErr != nil {
		t.Fatalf("GaeaInit 失败: %v", initErr)
	}

	ga.mu.Lock()
	ctrl := ga.ctrl
	ga.mu.Unlock()
	if ctrl == nil {
		t.Fatal("GaeaInit 后办公控制器仍为 nil")
	}

	// 自动恢复的会话应成为当前会话，且列表可见（「看不见工作空间」的修复点）。
	sessions := a.GaeaListSessions()
	if len(sessions) == 0 {
		t.Fatal("GaeaListSessions 为空，工作区会话不可见")
	}
	if !sessions[0].Current {
		t.Fatalf("自动恢复的会话未标记 current: %+v", sessions[0])
	}
	if sessions[0].Path != path {
		t.Fatalf("自动恢复会话路径 = %q, want %q", sessions[0].Path, path)
	}

	// 「无法新建会话」的修复点：GaeaNewSession 必须创建新的当前会话。
	if err := a.GaeaNewSession(); err != nil {
		t.Fatalf("GaeaNewSession: %v", err)
	}
	after := a.GaeaListSessions()
	if len(after) != 2 {
		t.Fatalf("新建会话后列表长度 = %d, want 2", len(after))
	}
	if !after[0].Current || after[0].Path == path {
		t.Fatalf("新建会话后当前会话异常: %+v", after[0])
	}

	// 「无法切换工作空间」的修复点：切到空工作区应返回新路径并生效。
	ws2 := t.TempDir()
	if got := a.GaeaSwitchWorkspace(ws2); got != ws2 {
		t.Fatalf("GaeaSwitchWorkspace = %q, want %q", got, ws2)
	}
	if got := a.GaeaMeta().Cwd; got != ws2 {
		t.Fatalf("切换后 cwd = %q, want %q", got, ws2)
	}
	if got := a.GaeaListSessions(); len(got) != 0 {
		t.Fatalf("切换后会话列表 = %+v, want 空（新工作区无会话）", got)
	}
}
