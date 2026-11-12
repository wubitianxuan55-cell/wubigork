package app

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

func TestGaeaWorkbenchFileMutations(t *testing.T) {
	ws := armListDirWorkspace(t)
	if err := os.MkdirAll(filepath.Join(ws, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	levels := a.GaeaListDirBatch([]string{"", "docs", "missing"}, nil)
	if len(levels) != 3 || levels[0].Error != "" || levels[1].Error != "" ||
		levels[2].Error == "" {
		t.Fatalf("GaeaListDirBatch=%+v", levels)
	}
	created, err := a.GaeaCreateWorkspaceFile("docs", "plan.md")
	if err != nil || created.Path != "docs/plan.md" {
		t.Fatalf("GaeaCreateWorkspaceFile=%+v err=%v", created, err)
	}
	renamed, err := a.GaeaRenameWorkspacePath("docs/plan.md", "outline.md")
	if err != nil || renamed.Path != "docs/outline.md" {
		t.Fatalf("GaeaRenameWorkspacePath=%+v err=%v", renamed, err)
	}
	folder, err := a.GaeaCreateWorkspaceFolder("docs", "assets")
	if err != nil || !folder.IsDir {
		t.Fatalf("GaeaCreateWorkspaceFolder=%+v err=%v", folder, err)
	}
	if err := a.GaeaDeleteWorkspacePaths([]string{"docs/outline.md", "docs/assets"}); err != nil {
		t.Fatalf("GaeaDeleteWorkspacePaths: %v", err)
	}
}

func TestGaeaWorkbenchChunkedUpload(t *testing.T) {
	ws := armListDirWorkspace(t)
	a := &App{officeState: &officeState{app: &App{}}}
	token, err := a.GaeaWorkspaceUploadBegin("docs", "nested/report.txt")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	for _, chunk := range [][]byte{[]byte("hello "), []byte("world")} {
		if err := a.GaeaWorkspaceUploadAppend(token, base64.StdEncoding.EncodeToString(chunk)); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	res, err := a.GaeaWorkspaceUploadFinish(token)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if res.Path != "docs/nested/report.txt" || res.Size != 11 {
		t.Fatalf("Finish=%+v", res)
	}
	got, _ := os.ReadFile(filepath.Join(ws, "docs", "nested", "report.txt"))
	if string(got) != "hello world" {
		t.Fatalf("uploaded=%q", got)
	}
	a.closeWorkspaceWorkbench()
}

func TestGaeaWorkbenchTextConflictAndEncoding(t *testing.T) {
	ws := armListDirWorkspace(t)
	path := filepath.Join(ws, "a.txt")
	if err := os.WriteFile(path, fileenc.Encode("旧\r\n", fileenc.GB18030), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	read, err := a.GaeaReadTextFile("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if read.Encoding != "gb18030" || read.EOL != "crlf" {
		t.Fatalf("ReadTextFile=%+v", read)
	}
	if _, err := a.GaeaWriteWorkspaceFile("a.txt", "新\n", read.MTimeMs); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaWriteWorkspaceFile("a.txt", "旧稿\n", read.MTimeMs); err == nil {
		t.Fatal("stale mtime should be rejected")
	}
	back, _ := os.ReadFile(path)
	if got := string(fileenc.Decode(back, fileenc.GB18030)); got != "新\r\n" {
		t.Fatalf("round trip=%q", got)
	}
}
