package spaces

import (
	"path/filepath"
	"testing"
)

func TestValid(t *testing.T) {
	for _, s := range []string{SpaceWork, SpacePlay} {
		if !Valid(s) {
			t.Errorf("Valid(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "Work", "PLAY", "session", "sub"} {
		if Valid(s) {
			t.Errorf("Valid(%q) = true, want false", s)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"":          SpaceWork, // 空 → work（读端降级单点）
		"work":      SpaceWork, // = SpaceWork
		"play":      SpacePlay, // = SpacePlay
		"bogus":     SpaceWork, // 非法 → work 安全默认
		"Play":      SpaceWork, // 严格小写，不猜大小写
		" play":     SpaceWork,
		"workspace": SpaceWork,
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSpaceOr(t *testing.T) {
	if got := SpaceOr("", SpaceWork); got != SpaceWork {
		t.Errorf("SpaceOr(empty, work) = %q, want work", got)
	}
	if got := SpaceOr(SpacePlay, SpaceWork); got != SpacePlay {
		t.Errorf("SpaceOr(play, work) = %q, want play", got)
	}
	if got := SpaceOr("bogus", SpaceWork); got != SpaceWork {
		t.Errorf("SpaceOr(bogus, work) = %q, want work（非法值回退 fallback）", got)
	}
}

// TestWorkspaceSessionDir 验证会话目录分区映射：work/play 各自落分区，
// 平铺目录（mode=off 形态）由调用方以 "" 直取 base（config 层职责）。
func TestWorkspaceSessionDir(t *testing.T) {
	if got, want := WorkspaceSessionDir(`C:\ws`, SpaceWork), `C:\ws\.gaea\sessions\work`; got != want {
		t.Errorf("WorkspaceSessionDir(work) = %q, want %q", got, want)
	}
	if got, want := WorkspaceSessionDir(`C:\ws`, SpacePlay), `C:\ws\.gaea\sessions\play`; got != want {
		t.Errorf("WorkspaceSessionDir(play) = %q, want %q", got, want)
	}
	if got, want := WorkspaceSessionDir(`C:\ws`, ""), `C:\ws\.gaea\sessions\work`; got != want {
		t.Errorf("WorkspaceSessionDir(空) = %q, want %q（空归一为 work）", got, want)
	}
	if got, want := WorkspaceSessionDir(`C:\ws`, "bogus"), `C:\ws\.gaea\sessions\work`; got != want {
		t.Errorf("WorkspaceSessionDir(bogus) = %q, want %q", got, want)
	}
}

// TestExportsDir 验证产物分区：work 返回现状路径不动（兼容红线），
// play 落 .gaea/play/exports。
func TestExportsDir(t *testing.T) {
	if got, want := ExportsDir(`C:\ws`, SpaceWork), `C:\ws\.gaea\exports`; got != want {
		t.Errorf("ExportsDir(work) = %q, want %q（现状路径不得改变）", got, want)
	}
	if got, want := ExportsDir(`C:\ws`, ""), `C:\ws\.gaea\exports`; got != want {
		t.Errorf("ExportsDir(空) = %q, want %q", got, want)
	}
	if got, want := ExportsDir(`C:\ws`, SpacePlay), `C:\ws\.gaea\play\exports`; got != want {
		t.Errorf("ExportsDir(play) = %q, want %q", got, want)
	}
}

// TestWorkDir 验证过程文件目录分区：work 现状路径、play 分区。
func TestWorkDir(t *testing.T) {
	if got, want := WorkDir(`C:\ws`, SpaceWork), `C:\ws\.gaea\work`; got != want {
		t.Errorf("WorkDir(work) = %q, want %q（现状路径不得改变）", got, want)
	}
	if got, want := WorkDir(`C:\ws`, SpacePlay), `C:\ws\.gaea\play\work`; got != want {
		t.Errorf("WorkDir(play) = %q, want %q", got, want)
	}
}

// TestSpaceForDir 验证从会话目录推导空间归属（含 archive 向上取父）。
func TestSpaceForDir(t *testing.T) {
	cases := map[string]string{
		filepath.Join(`C:\ws`, ".gaea", "sessions"):                    SpaceWork, // 平铺兜底 = work
		filepath.Join(`C:\ws`, ".gaea", "sessions", "work"):            SpaceWork,
		filepath.Join(`C:\ws`, ".gaea", "sessions", "play"):            SpacePlay,
		filepath.Join(`C:\ws`, ".gaea", "sessions", "archive"):         SpaceWork, // 平铺 archive
		filepath.Join(`C:\ws`, ".gaea", "sessions", "work", "archive"): SpaceWork,
		filepath.Join(`C:\ws`, ".gaea", "sessions", "play", "archive"): SpacePlay, // 空间 archive 向上取父
		filepath.Join(`C:\ws`, ".gaea", "work"):                        SpaceWork, // 非会话目录不误判
		`C:\tmp\play-elsewhere`:                                        SpaceWork, // 仅精确目录名匹配
	}
	for dir, want := range cases {
		if got := SpaceForDir(dir); got != want {
			t.Errorf("SpaceForDir(%q) = %q, want %q", dir, got, want)
		}
	}
}
