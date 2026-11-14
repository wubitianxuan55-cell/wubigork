package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/pantheon"
)

func TestPantheonRecentWorkspacesKeepsFourMostRecentUniqueDirectories(t *testing.T) {
	root := t.TempDir()
	paths := make([]string, 6)
	for i := range paths {
		paths[i] = filepath.Join(root, "workspace-"+string(rune('a'+i)))
		if err := os.MkdirAll(paths[i], 0o755); err != nil {
			t.Fatal(err)
		}
	}
	missing := filepath.Join(root, "missing")
	sessions := []pantheon.SessionSummary{
		{ID: "old-a", Workspace: paths[0], UpdatedAt: 20},
		{ID: "new-a", Workspace: paths[0], UpdatedAt: 50},
		{ID: "b", Workspace: paths[1], UpdatedAt: 10},
		{ID: "c", Workspace: paths[2], UpdatedAt: 30},
		{ID: "d", Workspace: paths[3], UpdatedAt: 40},
		{ID: "e-old", Workspace: paths[4], UpdatedAt: 15},
		{ID: "e-new", Workspace: paths[4], UpdatedAt: 70},
		{ID: "f", Workspace: paths[5], UpdatedAt: 5},
		{ID: "missing", Workspace: missing, UpdatedAt: 100},
		{ID: "no-workspace", UpdatedAt: 90},
	}

	got := pantheonRecentWorkspaces(sessions, paths[3], 4)
	want := []string{paths[4], paths[0], paths[3], paths[2]}
	if len(got) != len(want) {
		t.Fatalf("got %d workspaces, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Path != want[i] {
			t.Fatalf("workspace[%d] = %q, want %q", i, got[i].Path, want[i])
		}
		if !got[i].Recent {
			t.Fatalf("workspace[%d] must be marked recent", i)
		}
	}
	if got[2].Current != true {
		t.Fatal("the current workspace must stay marked when it is in council history")
	}
	if got[0].Current || got[1].Current || got[3].Current {
		t.Fatal("only the matching current workspace may be marked current")
	}
}

func TestPantheonRecentWorkspacesDoesNotFallBackToUnusedCurrentDirectory(t *testing.T) {
	current := filepath.Join(t.TempDir(), "never-used")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := pantheonRecentWorkspaces(nil, current, 4); len(got) != 0 {
		t.Fatalf("unused current directory must not be offered: %#v", got)
	}
}
