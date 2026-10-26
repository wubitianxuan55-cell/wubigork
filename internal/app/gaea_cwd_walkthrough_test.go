package app

import (
	"os"
	"path/filepath"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// gaeaCwd 走查态 fail-closed 守卫（2026-10-26 沙箱走查实弹事故回归锁）：
// gaea 引擎未装配（ga.cfg=nil，未登录走查壳即此形态）+ GAEA_WALKTHROUGH=1
// 时，工作目录必须落沙箱 workspace，绝不回退 os.Getwd()。事故形态：走查壳
// 以 cwd=仓库根 启动时 GaeaScheduleProjectCreate 静默实写了真实
// .gaea/schedule/index.json 与 进度计划/走查工程.gsched.json。
func TestGaeaCwdWalkthroughFailClosed(t *testing.T) {
	saved := ga.cfg
	ga.cfg = nil
	t.Cleanup(func() { ga.cfg = saved })

	t.Setenv("GAEA_WALKTHROUGH", "1")
	got := gaeaCwd()
	want := gaeaConfig.WalkthroughWorkspaceDir()
	if got != want {
		t.Fatalf("走查态 ga.cfg=nil：gaeaCwd=%q，须落沙箱 workspace %q（不得回退 cwd）", got, want)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("走查态 gaeaCwd=%q 须为绝对路径", got)
	}
}

// 非走查态：cfg 为 nil 回退 cwd 的既有口径不变（真机 cwd 语义保留）。
func TestGaeaCwdNonWalkthroughFallsBackToCwd(t *testing.T) {
	saved := ga.cfg
	ga.cfg = nil
	t.Cleanup(func() { ga.cfg = saved })

	t.Setenv("GAEA_WALKTHROUGH", "")
	cwd, _ := os.Getwd()
	if got := gaeaCwd(); got != cwd {
		t.Fatalf("非走查态回退口径被改变：got=%q want=%q", got, cwd)
	}
}
