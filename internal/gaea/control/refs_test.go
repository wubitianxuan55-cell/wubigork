package control

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/vision"
)

// stubOCR 注入全文 OCR 桩并在测试结束恢复。所有走 appendImageBlock 的测试
// 必须先调它：默认实现 ocrImageText=docmd.OCRImageText 会真拉起本地 uocr2
// 服务（同 internal/app 测试语义管线地雷形态）。
func stubOCR(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	orig := ocrImageText
	ocrImageText = fn
	t.Cleanup(func() { ocrImageText = orig })
}

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

// stubOfficeConv 注入办公文档转换桩并在测试结束恢复（默认实现会真走 docmd
// 转换/OCR 管线）。
func stubOfficeConv(t *testing.T, fn func(string, string, int) (string, int, bool, error)) {
	t.Helper()
	orig := convertOfficeDoc
	convertOfficeDoc = fn
	t.Cleanup(func() { convertOfficeDoc = orig })
}

// TestReadFileRef_OfficeConvertFailureCarriesReason 转换失败文案带真因
// （pdftoppm 缺失/引擎不可用一眼可见）并指向 format_convert。
func TestReadFileRef_OfficeConvertFailureCarriesReason(t *testing.T) {
	path := buildTextPDF(t, 1)
	stubOfficeConv(t, func(string, string, int) (string, int, bool, error) {
		return "", 1, false, errors.New("扫描件 PDF 需要 poppler 渲染（pdftoppm），但未找到")
	})
	content, _, err := readFileRef(path)
	if err != nil {
		t.Fatalf("readFileRef: %v", err)
	}
	if !strings.Contains(content, "pdftoppm") {
		t.Errorf("失败文案应带真因: %q", content)
	}
	if !strings.Contains(content, "format_convert") || !strings.Contains(content, "summarize_file") {
		t.Errorf("失败文案应指向深入读取工具: %q", content)
	}
	if strings.Contains(content, "scanned or compressed PDF") {
		t.Errorf("误导性固定文案应已被真因替换: %q", content)
	}
}

// TestReadFileRef_OfficeConvertBudgetTimeout 转换超预算注入指引行，
// 消息发送路径不被分钟级 OCR 拖死。
func TestReadFileRef_OfficeConvertBudgetTimeout(t *testing.T) {
	path := buildTextPDF(t, 1)
	stubOfficeConv(t, func(string, string, int) (string, int, bool, error) {
		time.Sleep(200 * time.Millisecond) // 远超收缩后的预算
		return "迟到的正文，不应注入", 1, false, nil
	})
	origBudget := officeConvBudget
	officeConvBudget = 30 * time.Millisecond
	t.Cleanup(func() { officeConvBudget = origBudget })

	content, _, err := readFileRef(path)
	if err != nil {
		t.Fatalf("readFileRef: %v", err)
	}
	if !strings.Contains(content, "超预算") || !strings.Contains(content, "format_convert") {
		t.Errorf("超预算指引缺失: %q", content)
	}
	if strings.Contains(content, "迟到的正文") {
		t.Errorf("迟到正文不应注入: %q", content)
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
	if err := os.MkdirAll(filepath.Join(".gaea", "attachments"), 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.ToSlash(filepath.Join(".gaea", "attachments", "ref.png"))
	if err := os.WriteFile(rel, []byte("image"), 0o644); err != nil {
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
	stubOCR(t, func(string) (string, error) { return "", errors.New("stub: ocr off") })

	c := &Controller{}
	block, errs := c.ResolveRefs(context.Background(), "帮我看看 @"+rel)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if !strings.Contains(block, "【画面理解】") || !strings.Contains(block, "这是一张办公截图") {
		t.Errorf("block = %q, want 识别内容", block)
	}
	if strings.Contains(block, "【全文 OCR】") {
		t.Errorf("OCR 腿失败时不应注入全文块: %q", block)
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
	stubOCR(t, func(string) (string, error) { return "", errors.New("stub: ocr off") })

	c := &Controller{}
	block, errs := c.ResolveRefs(context.Background(), "识别 @"+rel)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if !strings.Contains(block, "【画面理解】") || !strings.Contains(block, "截图包含表格与标题") {
		t.Errorf("block = %q, want 识别内容", block)
	}
}

// TestResolveRefs_ImageRecognitionFallback 识图失败时回退为占位提示。
func TestResolveRefs_ImageRecognitionFallback(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join(".gaea", "attachments"), 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.ToSlash(filepath.Join(".gaea", "attachments", "fallback.png"))
	if err := os.WriteFile(rel, []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := visionRecognize
	visionRecognize = func(_ context.Context, _, _ string) (string, error) {
		return "", context.DeadlineExceeded
	}
	defer func() { visionRecognize = orig }()
	stubOCR(t, func(string) (string, error) { return "", errors.New("stub: ocr off") })

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

// TestResolveRefs_ImageDualInjection 识图与全文 OCR 两腿都成功时同块双注入。
func TestResolveRefs_ImageDualInjection(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join(".gaea", "attachments"), 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.ToSlash(filepath.Join(".gaea", "attachments", "dual.png"))
	if err := os.WriteFile(rel, []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := visionRecognize
	visionRecognize = func(_ context.Context, _, _ string) (string, error) {
		return "一张报销单照片，背景为办公桌面。", nil
	}
	defer func() { visionRecognize = orig }()
	stubOCR(t, func(string) (string, error) {
		return "报销单\n日期：2026-11-09\n金额：￥128.00", nil
	})

	c := &Controller{}
	block, errs := c.ResolveRefs(context.Background(), "看看 @"+rel)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if !strings.Contains(block, "【画面理解】") || !strings.Contains(block, "报销单照片") {
		t.Errorf("block missing vision leg: %q", block)
	}
	if !strings.Contains(block, "【全文 OCR】") || !strings.Contains(block, "金额：￥128.00") {
		t.Errorf("block missing OCR leg: %q", block)
	}
}

// TestResolveRefs_ImageOCRFailureKeepsVision OCR 腿失败只降级自己，识图块原样。
func TestResolveRefs_ImageOCRFailureKeepsVision(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".gaea/uploads", 0o755); err != nil {
		t.Fatal(err)
	}
	rel := ".gaea/uploads/leg.png"
	if err := os.WriteFile(rel, []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := visionRecognize
	visionRecognize = func(_ context.Context, _, _ string) (string, error) {
		return "纯画面描述，没有 OCR 也能看懂。", nil
	}
	defer func() { visionRecognize = orig }()
	stubOCR(t, func(string) (string, error) { return "", errors.New("uocr2 unavailable") })

	c := &Controller{}
	block, errs := c.ResolveRefs(context.Background(), "看 @"+rel)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if !strings.Contains(block, "【画面理解】") || !strings.Contains(block, "纯画面描述") {
		t.Errorf("vision leg broken: %q", block)
	}
	if strings.Contains(block, "【全文 OCR】") || strings.Contains(block, "uocr2 unavailable") {
		t.Errorf("OCR 腿失败应静默降级: %q", block)
	}
}

// TestResolveRefs_ImageOCRBudgetTimeout OCR 腿超预算时放弃注入并给提示行，
// 不拖死本轮消息（识图腿照常注入）。
func TestResolveRefs_ImageOCRBudgetTimeout(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".gaea/uploads", 0o755); err != nil {
		t.Fatal(err)
	}
	rel := ".gaea/uploads/slow.png"
	if err := os.WriteFile(rel, []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}

	origVision := visionRecognize
	visionRecognize = func(_ context.Context, _, _ string) (string, error) {
		return "超时场景里的画面描述。", nil
	}
	defer func() { visionRecognize = origVision }()
	stubOCR(t, func(string) (string, error) {
		time.Sleep(300 * time.Millisecond) // 远超收缩后的预算，模拟引擎冷启动
		return "这段全文来得太晚，不应注入。", nil
	})

	origBudget := visionOCRBudget
	visionOCRBudget = 30 * time.Millisecond
	defer func() { visionOCRBudget = origBudget }()

	c := &Controller{}
	block, errs := c.ResolveRefs(context.Background(), "看 @"+rel)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if !strings.Contains(block, "【画面理解】") {
		t.Errorf("vision leg missing: %q", block)
	}
	if !strings.Contains(block, "【全文 OCR】未就绪") || !strings.Contains(block, "ocr 工具") {
		t.Errorf("budget timeout hint missing: %q", block)
	}
	if strings.Contains(block, "来得太晚") {
		t.Errorf("迟到全文不应注入: %q", block)
	}
}

// TestTruncateVision_UTF8Safe truncateVision 按边界截断，不劈多字节字符。
func TestTruncateVision_UTF8Safe(t *testing.T) {
	s := "a" + strings.Repeat("汉", 600) // 1 + 1800 字节，切点 1200 落在"汉"中间
	got := truncateVision(s)
	if !utf8.ValidString(strings.TrimSuffix(got, "…[已截断]")) {
		t.Errorf("truncateVision produced invalid UTF-8")
	}
	head := strings.TrimSuffix(got, "…[已截断]")
	if utf8.RuneCountInString(head) != 400 { // "a" + 399 个"汉" = 1+1197 字节
		t.Errorf("head runes = %d, want 400", utf8.RuneCountInString(head))
	}
	if !strings.HasSuffix(got, "…[已截断]") {
		t.Errorf("missing truncation marker: %q", got)
	}
}

// TestOcrBlockText_Gates ocrBlockText 的注入门槛：噪声不注入、超限截断带补读指引。
func TestOcrBlockText_Gates(t *testing.T) {
	// 噪声（<12 字符）不注入。
	if got := ocrBlockText(ocrResult{text: "  hi  "}); got != "" {
		t.Errorf("noise text should be skipped, got %q", got)
	}
	// 失败不注入。
	if got := ocrBlockText(ocrResult{err: errors.New("x"), text: "足够长的文本内容示例"}); got != "" {
		t.Errorf("error result should be skipped, got %q", got)
	}
	// 超限截断：5000 个"汉"=15000 字节 > maxVisionOCRBytes。
	got := ocrBlockText(ocrResult{text: strings.Repeat("汉", 5000)})
	if !strings.Contains(got, "…[已截断；全文可调用 ocr 工具提取]") {
		t.Errorf("oversize text missing truncate+hint marker")
	}
	head := strings.TrimSuffix(strings.TrimPrefix(got, "【全文 OCR】\n"), "…[已截断；全文可调用 ocr 工具提取]")
	if len(head) > maxVisionOCRBytes {
		t.Errorf("head %d bytes > cap %d", len(head), maxVisionOCRBytes)
	}
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
