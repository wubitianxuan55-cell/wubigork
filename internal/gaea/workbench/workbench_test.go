package workbench

import (
	"archive/zip"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

func TestListBatchAndMutations(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("readme"), 0o644); err != nil {
		t.Fatal(err)
	}
	levels := ListBatch(root, []string{"", "docs", "missing"}, nil)
	if len(levels) != 3 {
		t.Fatalf("levels=%d, want 3", len(levels))
	}
	if levels[0].Error != "" || len(levels[0].Entries) != 2 {
		t.Fatalf("root level=%+v", levels[0])
	}
	if levels[1].Error != "" || len(levels[1].Entries) != 0 {
		t.Fatalf("docs level=%+v", levels[1])
	}
	if levels[2].Error == "" {
		t.Fatal("missing level should report an in-place error")
	}

	created, err := CreateFile(root, "docs", "plan.md")
	if err != nil || created.Path != "docs/plan.md" {
		t.Fatalf("CreateFile=%+v err=%v", created, err)
	}
	folder, err := CreateFolder(root, "docs", "assets")
	if err != nil || folder.Path != "docs/assets" || !folder.IsDir {
		t.Fatalf("CreateFolder=%+v err=%v", folder, err)
	}
	renamed, err := Rename(root, "docs/plan.md", "outline.md")
	if err != nil || renamed.Path != "docs/outline.md" {
		t.Fatalf("Rename=%+v err=%v", renamed, err)
	}
	if err := DeletePaths(root, []string{"docs/outline.md", "docs/assets"}); err != nil {
		t.Fatalf("DeletePaths: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "outline.md")); !os.IsNotExist(err) {
		t.Fatalf("renamed file still exists: %v", err)
	}
}

func TestMutationsRejectTraversalAndUnsafeNames(t *testing.T) {
	root := t.TempDir()
	if _, err := CreateFile(root, "", "../escape.txt"); err == nil {
		t.Fatal("CreateFile accepted traversal")
	}
	for _, name := range []string{"", ".", "..", `a/b`, `a\b`, "CON", "bad?.txt", "trail."} {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) accepted unsafe name", name)
		}
	}
	if err := ValidateName("normal-说明.md"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
}

func TestListBatchExcludeAndCompactChain(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"node_modules/pkg", "docs/private", "src/only/deep"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "src", "only", "deep", "main.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "only", "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "private", "secret.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	levels := ListBatch(root, []string{"", "docs"}, []string{"docs/private"})
	if len(levels) != 2 {
		t.Fatalf("levels=%+v", levels)
	}
	names := map[string]Entry{}
	for _, e := range levels[0].Entries {
		names[e.Name] = e
	}
	if _, ok := names["node_modules"]; ok {
		t.Fatal("default noise directory leaked")
	}
	if !names["src"].Compact {
		t.Fatalf("src should compact its sole directory child: %+v", names["src"])
	}
	if len(levels[1].Entries) != 0 {
		t.Fatalf("docs/private should be excluded: %+v", levels[1].Entries)
	}
}

func TestExportFileAndDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "a-copy.txt")
	if err := ExportPath(root, "docs/a.txt", outFile); err != nil {
		t.Fatalf("ExportPath file: %v", err)
	}
	if got, _ := os.ReadFile(outFile); string(got) != "alpha" {
		t.Fatalf("exported file=%q", got)
	}

	outZip := filepath.Join(t.TempDir(), "docs.zip")
	if err := ExportPath(root, "docs", outZip); err != nil {
		t.Fatalf("ExportPath dir: %v", err)
	}
	zr, err := zip.OpenReader(outZip)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	found := false
	for _, f := range zr.File {
		if f.Name == "docs/a.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ZIP missing docs/a.txt: %+v", zr.File)
	}
}

func TestReadWritePreservesEncodingAndEOLAndRejectsConflict(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	original := "第一行\r\n第二行\r\n"
	raw := fileenc.Encode(original, fileenc.GB18030)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	read, err := ReadText(root, "notes.txt")
	if err != nil {
		t.Fatalf("ReadText: %v", err)
	}
	if read.Encoding != "gb18030" || read.EOL != "crlf" || read.Text != original {
		t.Fatalf("read=%+v", read)
	}
	res, err := WriteText(root, "notes.txt", "新一\n新二\n", read.MTimeMs)
	if err != nil {
		t.Fatalf("WriteText: %v", err)
	}
	if res.MTimeMs == 0 {
		t.Fatal("write result missing mtime")
	}
	back, _ := os.ReadFile(path)
	decoded := string(fileenc.Decode(back, fileenc.GB18030))
	if decoded != "新一\r\n新二\r\n" {
		t.Fatalf("round trip=%q", decoded)
	}
	if _, err := WriteText(root, "notes.txt", "stale", read.MTimeMs); err == nil ||
		!strings.Contains(err.Error(), "磁盘上改变") {
		t.Fatalf("stale write should conflict, got %v", err)
	}
}

func TestReadTextTooLargeAndPage(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Repeat("x", 1023) + "\n"
	for i := 0; i < (EditableBytes/len(line))+20; i++ {
		if _, err := f.WriteString(line); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	full, err := ReadText(root, "large.log")
	if err != nil {
		t.Fatal(err)
	}
	if !full.TooLarge || full.Text != "" {
		t.Fatalf("large read=%+v", full)
	}
	page, err := ReadPage(root, "large.log", 0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Text == "" || !strings.HasSuffix(page.Text, "\n") || page.NextOffset <= 0 || page.EOF {
		t.Fatalf("first page=%+v", page)
	}
	next, err := ReadPage(root, "large.log", page.NextOffset)
	if err != nil {
		t.Fatal(err)
	}
	if next.Text == "" || next.Offset != page.NextOffset {
		t.Fatalf("next page=%+v", next)
	}
}

func TestDecodePreviewKeepsGB18030AtCutBoundary(t *testing.T) {
	text := strings.Repeat("中文记录\n", 300000)
	raw := fileenc.Encode(text, fileenc.GB18030)
	body, truncated, encoding, eol := DecodePreview(raw, 1024*1024+1)
	if !truncated || encoding != "gb18030" || eol != "lf" {
		t.Fatalf("meta truncated=%v encoding=%s eol=%s", truncated, encoding, eol)
	}
	if body == "" || strings.ContainsRune(body, '\uFFFD') {
		t.Fatalf("decoded preview should be non-empty and valid, len=%d", len(body))
	}
}

func TestUploadManagerChunked(t *testing.T) {
	root := t.TempDir()
	m := NewUploadManager()
	defer m.Close()
	token, err := m.Begin(root, "docs", "nested/report.txt")
	if err != nil {
		t.Fatal(err)
	}
	part1 := []byte("hello ")
	part2 := []byte("world")
	for _, part := range [][]byte{part1, part2} {
		if err := m.Append(token, base64.StdEncoding.EncodeToString(part)); err != nil {
			t.Fatal(err)
		}
	}
	res, err := m.Finish(token)
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != "docs/nested/report.txt" || res.Size != int64(len(part1)+len(part2)) {
		t.Fatalf("upload result=%+v", res)
	}
	got, _ := os.ReadFile(filepath.Join(root, "docs", "nested", "report.txt"))
	if string(got) != "hello world" {
		t.Fatalf("uploaded=%q", got)
	}
	if _, err := m.Begin(root, "docs", "nested/report.txt"); err == nil {
		t.Fatal("overwrite without explicit confirmation should be rejected")
	}
	if _, err := m.Begin(root, "", "../escape.txt"); err == nil {
		t.Fatal("upload traversal accepted")
	}
}

func TestImportFilesAndFolder(t *testing.T) {
	root := t.TempDir()
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "report.txt")
	if err := os.WriteFile(srcFile, []byte("native import"), 0o644); err != nil {
		t.Fatal(err)
	}
	results, err := ImportFiles(root, "docs", []string{srcFile})
	if err != nil {
		t.Fatalf("ImportFiles: %v", err)
	}
	if len(results) != 1 || results[0].Path != "docs/report.txt" {
		t.Fatalf("ImportFiles=%+v", results)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "docs", "report.txt")); string(got) != "native import" {
		t.Fatalf("imported file=%q", got)
	}
	if _, err := ImportFiles(root, "docs", []string{srcFile}); err == nil {
		t.Fatal("duplicate import should be refused")
	}

	folder := filepath.Join(srcDir, "bundle")
	if err := os.MkdirAll(filepath.Join(folder, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "nested", "a.md"), []byte("# A"), 0o644); err != nil {
		t.Fatal(err)
	}
	results, err = ImportFolder(root, "docs", folder)
	if err != nil {
		t.Fatalf("ImportFolder: %v", err)
	}
	if len(results) != 1 || results[0].Path != "docs/bundle/nested/a.md" {
		t.Fatalf("ImportFolder=%+v", results)
	}
}

func TestDirWatchReportsChangedLevel(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	changed := make(chan []string, 1)
	w, err := NewDirWatch(root, []string{"", "docs"}, nil, func(paths []string) {
		select {
		case changed <- paths:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(root, "docs", "new.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case paths := <-changed:
		if len(paths) != 1 || paths[0] != "docs" {
			t.Fatalf("watch paths=%v, want [docs]", paths)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for watch notice")
	}
}
