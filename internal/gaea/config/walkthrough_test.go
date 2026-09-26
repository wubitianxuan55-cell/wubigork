package config

import (
	"os"
	"path/filepath"
	"testing"
)

// GAEA_WALKTHROUGH=1 走查沙箱开关（v4.418）：
// 显式设置时 workspace 必须被强制指向固定沙箱目录——即使用户配置/项目配置
// 写了别的 workspace 也走沙箱（走查隔离优先于一切，2026-09-26 真机走查
// 险情的防复发刀：首条消息自动装配+自动恢复会把回合接进用户真实会话）。
func TestWalkthroughWorkspaceOverride(t *testing.T) {
	t.Setenv("GAEA_WALKTHROUGH", "1")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := filepath.Join(os.TempDir(), "gaea-walkthrough", "workspace")
	if cfg.Workspace != want {
		t.Fatalf("Workspace = %q, want 沙箱 %q", cfg.Workspace, want)
	}
	if st, err := os.Stat(want); err != nil || !st.IsDir() {
		t.Fatalf("沙箱目录应已创建: %v", err)
	}
}

// 未设置时不得触碰 workspace（零误伤：普通用户无感）。
func TestWalkthroughWorkspaceOff(t *testing.T) {
	t.Setenv("GAEA_WALKTHROUGH", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Workspace == filepath.Join(os.TempDir(), "gaea-walkthrough", "workspace") {
		t.Fatalf("未设开关时 workspace 不应指向沙箱: %q", cfg.Workspace)
	}
}
