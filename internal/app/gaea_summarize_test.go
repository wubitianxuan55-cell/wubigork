package app

import (
	"os"
	"testing"
)

func TestGaeaSummarizeFile_MissingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	if _, err := a.GaeaSummarizeFile("nope.md", ""); err == nil {
		t.Fatal("missing file should error before any model call")
	}
}

func TestGaeaSummarizeFile_NoClient(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("ok.md", []byte("内容"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{}}
	if _, err := a.GaeaSummarizeFile("ok.md", ""); err == nil {
		t.Fatal("nil ai client should error")
	}
}
