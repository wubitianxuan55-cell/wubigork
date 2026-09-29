package config

import (
	"os"
	"path/filepath"
	"strings"
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

// TestWalkthroughOverrideIsolatesAppData v4.428.1 补漏锚：override 同时把 APPDATA
// 重定向到沙箱（UserConfigDir 数据面隔离）——只换 workspace 时 sin/chat 等
// %APPDATA%\gaea 数据根直连用户实盘（2026-09-29 走查实证）。
func TestWalkthroughOverrideIsolatesAppData(t *testing.T) {
	// 前序用例可能已把 APPDATA 重定向到定值沙箱——先归位到独立临时目录再验
	orig := t.TempDir()
	t.Setenv("APPDATA", orig)
	t.Setenv("GAEA_WALKTHROUGH", "1")
	cfg := &Config{Workspace: orig}
	ApplyWalkthroughOverride(cfg)
	got := os.Getenv("APPDATA")
	if got == orig {
		t.Fatalf("APPDATA 未重定向: %q", got)
	}
	if !strings.Contains(got, "gaea-walkthrough") {
		t.Fatalf("APPDATA 应指向走查沙箱: %q", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("沙箱 APPDATA 目录应已创建: %v", err)
	}
	// 关：未设环境变量时 APPDATA 原样不动
	os.Setenv("APPDATA", orig)
	t.Setenv("GAEA_WALKTHROUGH", "")
	cfg2 := &Config{Workspace: orig}
	ApplyWalkthroughOverride(cfg2)
	if os.Getenv("APPDATA") != orig {
		t.Fatalf("非走查模式不得动 APPDATA: %q", os.Getenv("APPDATA"))
	}
	if cfg2.Workspace != orig {
		t.Fatalf("非走查模式不得动 workspace: %q", cfg2.Workspace)
	}
}
