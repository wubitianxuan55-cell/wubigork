package app

import (
	"os"
	"path/filepath"
	"testing"
)

func isolateWenxueRuntime(t *testing.T) (*App, string) {
	t.Helper()
	restore := workspaceTestIsolate(t)
	t.Cleanup(restore)
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	t.Setenv("GAEA_TUWEN_ROOT", root)

	oldCfg := ga.cfg
	ga.cfg = nil
	ga.mu.Lock()
	oldCtrl := ga.ctrl
	ga.ctrl = nil
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		if ga.ctrl != nil {
			ga.ctrl.Close()
		}
		ga.ctrl = oldCtrl
		ga.cfg = oldCfg
		ga.mu.Unlock()
	})
	return &App{}, root
}

func TestWenxueCreateNovelStaysUnderTuwenRoot(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	created, err := a.WenxueCreateNovel("长夜灯火")
	if err != nil {
		t.Fatalf("WenxueCreateNovel: %v", err)
	}
	want := filepath.Join(root, "长夜灯火")
	if created.Path != want || created.Name != "长夜灯火" || !created.Current {
		t.Fatalf("created = %+v, want path=%q current", created, want)
	}
	if info, err := os.Stat(filepath.Join(want, ".gaea")); err != nil || !info.IsDir() {
		t.Fatalf("novel .gaea directory missing: info=%v err=%v", info, err)
	}

	list := a.WenxueListWorkspaces()
	if len(list) != 1 || list[0].Path != want || !list[0].Current {
		t.Fatalf("WenxueListWorkspaces = %+v, want the current novel only", list)
	}

	if _, err := a.WenxueCreateNovel("长夜灯火"); err == nil {
		t.Fatal("duplicate novel name should fail")
	}
	if _, err := a.WenxueCreateNovel(`..\逃逸`); err == nil {
		t.Fatal("path-like novel name should fail")
	}
	for _, bad := range []string{".隐藏", "CON", "NUL.md", "星夜."} {
		if _, err := a.WenxueCreateNovel(bad); err == nil {
			t.Fatalf("invalid Windows directory name %q should fail", bad)
		}
	}
}

func TestWenxueSwitchWorkspaceRejectsOutsideRoot(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	novel := filepath.Join(root, "甲小说")
	if err := os.MkdirAll(novel, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := a.WenxueSwitchWorkspace(novel); got != novel {
		t.Fatalf("WenxueSwitchWorkspace = %q, want %q", got, novel)
	}

	outside := t.TempDir()
	if got := a.WenxueSwitchWorkspace(outside); got != novel {
		t.Fatalf("outside switch = %q, want current %q", got, novel)
	}
	if got := wenxueWorkspace(); got != novel {
		t.Fatalf("wenxueWorkspace = %q, want %q", got, novel)
	}
}

func TestWenxueProjectGroupsOnlyContainNovelDirectories(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	for _, name := range []string{"乙小说", "甲小说"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "不是小说.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}

	current := filepath.Join(root, "甲小说")
	if got := a.WenxueSwitchWorkspace(current); got != current {
		t.Fatalf("WenxueSwitchWorkspace = %q, want %q", got, current)
	}
	groups := a.WenxueListProjectSessions()
	if len(groups) != 2 {
		t.Fatalf("WenxueListProjectSessions = %+v, want 2 novel groups", groups)
	}
	if groups[0].Path != current || !groups[0].Current {
		t.Fatalf("first group = %+v, want current 甲小说", groups[0])
	}
	if groups[0].Name == ".hidden" || groups[1].Name == ".hidden" {
		t.Fatalf("hidden directory leaked into groups: %+v", groups)
	}
	if sessions := a.WenxueListSessions(); len(sessions) != 0 {
		t.Fatalf("new novel sessions = %+v, want empty", sessions)
	}
}

func TestWenxueDeleteNovelOnlyDirectChildren(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	first, err := a.WenxueCreateNovel("旧城来信")
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.WenxueCreateNovel("长夜灯火")
	if err != nil {
		t.Fatal(err)
	}
	if first.Path == second.Path {
		t.Fatal("created novels should have distinct paths")
	}
	if err := a.WenxueDeleteNovel(first.Path); err != nil {
		t.Fatalf("delete non-current novel: %v", err)
	}
	if _, err := os.Stat(first.Path); !os.IsNotExist(err) {
		t.Fatalf("deleted novel still exists: %v", err)
	}
	if got := wenxueWorkspace(); got != second.Path {
		t.Fatalf("current novel changed after deleting another: got %q want %q", got, second.Path)
	}

	if err := a.WenxueDeleteNovel(root); err == nil {
		t.Fatal("deleting tuwen root should fail")
	}
	if err := a.WenxueDeleteNovel(t.TempDir()); err == nil {
		t.Fatal("deleting outside path should fail")
	}
	nested := filepath.Join(second.Path, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := a.WenxueDeleteNovel(nested); err == nil {
		t.Fatal("deleting nested path should fail")
	}

	if err := a.WenxueDeleteNovel(second.Path); err != nil {
		t.Fatalf("delete current novel: %v", err)
	}
	if got := wenxueWorkspace(); got != root {
		t.Fatalf("current workspace after deleting current novel = %q, want root %q", got, root)
	}
}
