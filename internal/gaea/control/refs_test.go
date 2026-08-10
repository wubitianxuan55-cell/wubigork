package control

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/vision"
)

// buildTextPDF 生成带唯一文本的合成 PDF（文本流，无外部依赖）。
func buildTextPDF(t *testing.T, pages int) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	for i := 1; i <= pages; i++ {
		fmt.Fprintf(&sb, "/Type /Page\nBT /F1 12 Tf 72 720 Td (第 %d 页内容) Tj ET\n", i)
	}
	path := filepath.Join(t.TempDir(), "office.pdf")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	return path
}

// TestReadFileRef_OfficePDF @ 办公文档应转 Markdown 注入头部，而不是输出
// "binary not shown"；超过 refOfficeMaxPages 的 PDF 带页数截断标记。
func TestReadFileRef_OfficePDF(t *testing.T) {
	path := buildTextPDF(t, 30)
	content, isDir, err := readFileRef(path)
	if err != nil {
		t.Fatalf("readFileRef: %v", err)
	}
	if isDir {
		t.Fatal("isDir = true, want false")
	}
	if strings.Contains(content, "binary file") {
		t.Errorf("office PDF fell back to binary marker")
	}
	if !strings.Contains(content, "第 1 页内容") {
		t.Errorf("office PDF missing injected text head")
	}
	if !strings.Contains(content, "30 页") || !strings.Contains(content, "已注入前 20 页") {
		t.Errorf("office PDF marker missing page info: %q", content)
	}
	if !strings.Contains(content, "summarize_file") {
		t.Errorf("office PDF marker missing summarize_file hint")
	}
}

// TestReadFileRef_LargeTextMarker 大文本文件截断标记应指引 summarize_file/read_file。
func TestReadFileRef_LargeTextMarker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.log")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", maxFileRefBytes+100)), 0o644); err != nil {
		t.Fatalf("write big file: %v", err)
	}
	content, _, err := readFileRef(path)
	if err != nil {
		t.Fatalf("readFileRef: %v", err)
	}
	if !strings.Contains(content, "truncated") {
		t.Errorf("missing truncated marker")
	}
	if !strings.Contains(content, "summarize_file") {
		t.Errorf("missing summarize_file hint")
	}
}

func TestParseRefTokens(t *testing.T) {
	cases := []struct {
		line string
		want []string
	}{
		{"see @docs:doc://x and @src/main.go", []string{"docs:doc://x", "src/main.go"}},
		{"trailing @file.go.", []string{"file.go"}},
		{"dedup @a @a", []string{"a"}},
		{"no refs here", nil},
		{"email a@b.com keeps token", []string{"b.com"}},
	}
	for _, c := range cases {
		got := parseRefTokens(c.line)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseRefTokens(%q) = %v, want %v", c.line, got, c.want)
		}
	}
}

func TestClassifyRef(t *testing.T) {
	known := map[string]bool{"docs": true}
	files := map[string]bool{"src/main.go": true, "README.md": true, ".gaea/attachments/clipboard-20260601-010203.000000.png": true}
	exists := func(p string) bool { return files[p] }

	cases := []struct {
		token   string
		wantOK  bool
		wantKnd refKind
	}{
		{"docs:doc://style", true, refResource}, // known server + uri
		{"src/main.go", true, refFile},          // existing file
		{"README.md", true, refFile},            // existing file
		{".gaea/attachments/clipboard-20260601-010203.000000.png", true, refImage},
		{"ghost:issue://1", false, 0}, // unknown server, no such file
		{"missing.go", false, 0},      // nonexistent path → not a ref
		{"docs:", false, 0},           // empty uri → not a resource, no file
	}
	for _, c := range cases {
		r, ok := classifyRef(c.token, known, exists)
		if ok != c.wantOK {
			t.Errorf("classifyRef(%q) ok = %v, want %v", c.token, ok, c.wantOK)
			continue
		}
		if ok && r.kind != c.wantKnd {
			t.Errorf("classifyRef(%q) kind = %v, want %v", c.token, r.kind, c.wantKnd)
		}
	}
}

// TestResolveRefs_ImageRecognition 图片引用应自动识图并把描述注入上下文。
func TestResolveRefs_ImageRecognition(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := ensureAttachmentRoot(); err != nil {
		t.Fatal(err)
	}
	rel, f, err := createAttachmentFile(".png")
	if err != nil {
		t.Fatalf("createAttachmentFile: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	orig := visionRecognize
	visionRecognize = func(_ context.Context, imagePath, _ string) (string, error) {
		if !strings.HasSuffix(imagePath, ".png") {
			t.Errorf("imagePath = %q, want png", imagePath)
		}
		return "这是一张办公截图，包含标题和两段文字。", nil
	}
	defer func() { visionRecognize = orig }()

	c := &Controller{}
	block, errs := c.ResolveRefs(context.Background(), "帮我看看 @"+rel)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if !strings.Contains(block, "【图片识别】") || !strings.Contains(block, "这是一张办公截图") {
		t.Errorf("block = %q, want 识别内容", block)
	}
}

// TestResolveRefs_UploadsImage 普通图片路径（.gaea/uploads 等）也应识图注入。
func TestResolveRefs_UploadsImage(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".gaea/uploads", 0o755); err != nil {
		t.Fatal(err)
	}
	rel := ".gaea/uploads/paste-1.png"
	if err := os.WriteFile(rel, []byte("not-a-real-png"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := visionRecognize
	visionRecognize = func(_ context.Context, imagePath, _ string) (string, error) {
		if imagePath != rel {
			t.Errorf("imagePath = %q, want %q", imagePath, rel)
		}
		return "截图包含表格与标题。", nil
	}
	defer func() { visionRecognize = orig }()

	c := &Controller{}
	block, errs := c.ResolveRefs(context.Background(), "识别 @"+rel)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if !strings.Contains(block, "【图片识别】") || !strings.Contains(block, "截图包含表格与标题") {
		t.Errorf("block = %q, want 识别内容", block)
	}
}

// TestResolveRefs_ImageRecognitionFallback 识图失败时回退为占位提示。
func TestResolveRefs_ImageRecognitionFallback(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := ensureAttachmentRoot(); err != nil {
		t.Fatal(err)
	}
	rel, f, err := createAttachmentFile(".png")
	if err != nil {
		t.Fatalf("createAttachmentFile: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	orig := visionRecognize
	visionRecognize = func(_ context.Context, _, _ string) (string, error) {
		return "", context.DeadlineExceeded
	}
	defer func() { visionRecognize = orig }()

	c := &Controller{}
	block, _ := c.ResolveRefs(context.Background(), "看看 @"+rel)
	if !strings.Contains(block, "识图失败") {
		t.Errorf("block = %q, want 识图失败回退", block)
	}
	// 确保恢复原实现后类型正确
	if visionRecognize == nil {
		t.Fatal("visionRecognize 不应为 nil")
	}
	_ = vision.RecognizeImage
}

func TestReadFileRef(t *testing.T) {
	dir := t.TempDir()

	textPath := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(textPath, []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(dir, "blob.bin")
	if err := os.WriteFile(binPath, []byte{'a', 0x00, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	bigPath := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(bigPath, []byte(strings.Repeat("a", maxFileRefBytes+100)), 0o644); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(imagePath, []byte("\x89PNG\r\n\x1a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Text file: content verbatim, not a directory.
	if got, isDir, err := readFileRef(textPath); err != nil || isDir || got != "line one\nline two\n" {
		t.Errorf("text file = (%q, %v, %v)", got, isDir, err)
	}

	// Binary file: noted, not dumped.
	if got, _, err := readFileRef(binPath); err != nil || !strings.Contains(got, "binary file") {
		t.Errorf("binary file = (%q, %v), want a binary note", got, err)
	}

	// Image file: identified as image-specific guidance, not generic binary.
	if got, _, err := readFileRef(imagePath); err != nil || !strings.Contains(got, "image file") {
		t.Errorf("image file = (%q, %v), want an image note", got, err)
	}

	// Large file: truncated with a marker.
	if got, _, err := readFileRef(bigPath); err != nil || !strings.Contains(got, "truncated") {
		t.Errorf("big file should be truncated, got len=%d err=%v", len(got), err)
	}

	// Directory: recursive listing with relative paths including a trailing slash for subdirs.
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, isDir, err := readFileRef(dir)
	if err != nil || !isDir {
		t.Fatalf("dir = (isDir=%v, err=%v)", isDir, err)
	}
	if !strings.Contains(got, "hello.txt") || !strings.Contains(got, "sub/") || !strings.Contains(got, "sub/nested.txt") {
		t.Errorf("dir listing = %q, want hello.txt, sub/, and sub/nested.txt", got)
	}

	// Missing path: error.
	if _, _, err := readFileRef(filepath.Join(dir, "nope")); err == nil {
		t.Error("missing path should error")
	}
}
