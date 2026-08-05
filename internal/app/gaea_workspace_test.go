package app

import (
	"os"
	"path/filepath"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// workspaceTestIsolate 将配置与工作目录隔离到临时目录，避免污染真实环境。
// 返回恢复函数。
func workspaceTestIsolate(t *testing.T) func() {
	t.Helper()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	oldWD, _ := os.Getwd()
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_ = os.Chdir(t.TempDir())
	return func() {
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
		_ = os.Chdir(oldWD)
	}
}

// TestPersistWorkspace 验证工作空间切换持久化：
// persistWorkspaceLocked → 内存配置更新 → 用户配置文件往返 → gaeaCwd 跟随。
func TestPersistWorkspace(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	// 全局 ga.cfg 可能被其他测试污染，先清理
	oldCfg := ga.cfg
	ga.cfg = nil
	defer func() { ga.cfg = oldCfg }()

	a := &App{}
	workspace := t.TempDir()

	ga.mu.Lock()
	err := a.persistWorkspaceLocked(workspace)
	ga.mu.Unlock()
	if err != nil {
		t.Fatalf("persistWorkspaceLocked() 失败: %v", err)
	}

	// 1. 内存配置已更新
	if ga.cfg == nil || ga.cfg.Workspace != workspace {
		t.Fatalf("ga.cfg.Workspace = %v, want %v", ga.cfg.Workspace, workspace)
	}

	// 2. 重新加载（模拟下次启动）仍保持
	got, err := gaeaLoadConfig()
	if err != nil {
		t.Fatalf("gaeaLoadConfig() 失败: %v", err)
	}
	if got.Workspace != workspace {
		t.Errorf("持久化后重载 Workspace = %q, want %q", got.Workspace, workspace)
	}

	// 3. gaeaCwd 跟随工作空间
	if gaeaCwd() != workspace {
		t.Errorf("gaeaCwd() = %q, want %q", gaeaCwd(), workspace)
	}
}

// TestSwitchWorkspace_InvalidPath 验证无效路径保持当前工作空间不变。
func TestSwitchWorkspace_InvalidPath(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	ga.cfg = nil
	defer func() { ga.cfg = oldCfg }()

	a := &App{}
	cur := gaeaCwd()

	if got := a.GaeaSwitchWorkspace(""); got != cur {
		t.Errorf("GaeaSwitchWorkspace(\"\") = %q, want 当前 %q", got, cur)
	}
	if got := a.GaeaSwitchWorkspace(filepath.Join(t.TempDir(), "不存在")); got != cur {
		t.Errorf("GaeaSwitchWorkspace(无效路径) = %q, want 当前 %q", got, cur)
	}
}

// TestListWorkspaces 验证列表返回完整字段（path/name/current）。
func TestListWorkspaces(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	ga.cfg = nil
	defer func() { ga.cfg = oldCfg }()

	a := &App{}
	ws := a.GaeaListWorkspaces()
	if len(ws) != 1 {
		t.Fatalf("GaeaListWorkspaces() 长度 = %d, want 1", len(ws))
	}
	w := ws[0]
	if w.Path == "" {
		t.Error("WorkspaceView.Path 为空")
	}
	if w.Name == "" || w.Name != filepath.Base(w.Path) {
		t.Errorf("WorkspaceView.Name = %q, want base(%q)", w.Name, w.Path)
	}
	if !w.Current {
		t.Error("WorkspaceView.Current = false, want true（当前工作空间）")
	}
}

// TestSwitchWorkspace_Valid 验证有效目录切换成功并持久化。
func TestSwitchWorkspace_Valid(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	oldCfg := ga.cfg
	ga.cfg = nil
	ga.mu.Lock()
	oldCtrl := ga.ctrl
	ga.ctrl = nil
	ga.mu.Unlock()
	defer func() {
		ga.mu.Lock()
		if ga.ctrl != nil {
			ga.ctrl.Close()
		}
		ga.ctrl = oldCtrl
		ga.cfg = oldCfg
		ga.mu.Unlock()
	}()

	a := &App{}
	ws := t.TempDir()

	if got := a.GaeaSwitchWorkspace(ws); got != ws {
		t.Fatalf("GaeaSwitchWorkspace(有效目录) = %q, want %q", got, ws)
	}
	if gaeaCwd() != ws {
		t.Errorf("切换后 gaeaCwd() = %q, want %q", gaeaCwd(), ws)
	}

	// 列表标记新工作空间为 current
	wsList := a.GaeaListWorkspaces()
	if len(wsList) != 1 || wsList[0].Path != ws || !wsList[0].Current {
		t.Errorf("切换后列表 = %+v, want 单条 current path=%q", wsList, ws)
	}
}

// TestGaeaWorkspaceConfigRoundTrip 验证 Workspace 字段随办公配置 toml 往返。
func TestGaeaWorkspaceConfigRoundTrip(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	cfg, err := gaeaLoadConfig()
	if err != nil {
		t.Fatalf("gaeaLoadConfig() 失败: %v", err)
	}
	ws := t.TempDir()
	cfg.Workspace = ws
	if err := gaeaConfig.Save(cfg); err != nil {
		t.Fatalf("Save() 失败: %v", err)
	}

	got, err := gaeaLoadConfig()
	if err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	if got.Workspace != ws {
		t.Errorf("toml 往返后 Workspace = %q, want %q", got.Workspace, ws)
	}
}
