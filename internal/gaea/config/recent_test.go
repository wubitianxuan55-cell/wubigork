package config

import (
	"os"
	"reflect"
	"testing"
)

// TestTouchRecentWorkspace 验证最近工作区注册表的去重与置顶语义：
// 重复打开不产生重复项，最近使用的工作区排在最前。
func TestTouchRecentWorkspace(t *testing.T) {
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}()

	TouchRecentWorkspace("/ws/b")
	TouchRecentWorkspace("/ws/a")
	TouchRecentWorkspace("/ws/b") // 再次打开 b → 去重并置顶

	got := LoadRecentWorkspaces()
	want := []string{"/ws/b", "/ws/a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadRecentWorkspaces() = %v, want %v", got, want)
	}
}

// TestTouchRecentWorkspaceCap 验证列表长度上限：超出部分被丢弃。
func TestTouchRecentWorkspaceCap(t *testing.T) {
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}()

	for i := 0; i < maxRecentWorkspaces+3; i++ {
		TouchRecentWorkspace(string(rune('a'+i)))
	}
	got := LoadRecentWorkspaces()
	if len(got) != maxRecentWorkspaces {
		t.Fatalf("长度 = %d, want %d", len(got), maxRecentWorkspaces)
	}
	// 最新打开的最前
	if got[0] != string(rune('a'+maxRecentWorkspaces+2)) {
		t.Errorf("最新工作区应置顶, got[0] = %q", got[0])
	}
}
