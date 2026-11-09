package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// ── 原罪附件 @引用解析（sin_file_refs.go）────────────────────────────────

func TestSinFileRefTextInjection(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "参考设定.txt")
	if err := os.WriteFile(p, []byte("主角名叫林晚。"), 0o644); err != nil {
		t.Fatal(err)
	}
	block, errs := sinFileRefBlock(context.Background(), "请参考 @"+p+" 写一段")
	if len(errs) != 0 {
		t.Fatalf("不应有失败: %v", errs)
	}
	if !strings.Contains(block, `<file path="`) || !strings.Contains(block, "主角名叫林晚。") {
		t.Fatalf("文本内容应注入块内: %q", block)
	}
}

func TestSinFileRefTruncatesLargeText(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "big.txt")
	big := strings.Repeat("甲", sinFileRefMaxBytes+4096)
	if err := os.WriteFile(p, []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	block, errs := sinFileRefBlock(context.Background(), "@"+p)
	if len(errs) != 0 {
		t.Fatalf("大文本是截断不是失败: %v", errs)
	}
	if !strings.Contains(block, "[已截断：仅注入前 64KB]") {
		t.Fatalf("截断标记缺失: %q", block[len(block)-80:])
	}
	// 头部内容确实被注入，且总量收敛（64KB + 标记）
	if !strings.Contains(block, strings.Repeat("甲", 1024)) {
		t.Fatal("截断后应保留头部内容")
	}
	if len(block) > sinFileRefMaxBytes+256 {
		t.Fatalf("注入块应收敛: %d", len(block))
	}
}

func TestSinFileRefBinaryNote(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bin.dat")
	raw := make([]byte, 64)
	raw[10] = 0x00 // 头 8KiB 含 NUL → 二进制
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	block, errs := sinFileRefBlock(context.Background(), "@"+p)
	if len(errs) != 0 {
		t.Fatalf("二进制是提示不是失败: %v", errs)
	}
	if !strings.Contains(block, "[二进制文件，不注入内容]") {
		t.Fatalf("二进制应提示不倾倒: %q", block)
	}
}

func TestSinFileRefIgnoresNonPaths(t *testing.T) {
	// 正文里的 @字样（@角色、邮箱、不存在的路径）零误伤：不产生块也不报错
	block, errs := sinFileRefBlock(context.Background(), "林晚 @林晚 发来邮件 lin@example.com，又提到 @C:\\不存在\\文件.txt")
	if block != "" || len(errs) != 0 {
		t.Fatalf("非路径 @token 应原样保留零误伤: block=%q errs=%v", block, errs)
	}
}

// stubSinOCR 注入全文 OCR 桩并在测试结束恢复。所有走 sinImageBlock 的测试
// 必须先调它：默认实现 sinOCRImageText=docmd.OCRImageText 会真拉起本地
// uocr2 服务（测试语义管线地雷形态）。
func stubSinOCR(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	orig := sinOCRImageText
	sinOCRImageText = fn
	t.Cleanup(func() { sinOCRImageText = orig })
}

func TestSinFileRefImageVisionFallback(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ref.png")
	if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := sinVisionRecognize
	t.Cleanup(func() { sinVisionRecognize = orig })
	stubSinOCR(t, func(string) (string, error) { return "", errors.New("stub: ocr off") })

	sinVisionRecognize = func(ctx context.Context, path, prompt string) (string, error) {
		return "一位红衣女子立于桥头。", nil
	}
	block, errs := sinFileRefBlock(context.Background(), "@"+p)
	if len(errs) != 0 || !strings.Contains(block, "【画面理解】") || !strings.Contains(block, "红衣女子") {
		t.Fatalf("识图结果应注入: block=%q errs=%v", block, errs)
	}

	sinVisionRecognize = func(ctx context.Context, path, prompt string) (string, error) {
		return "", context.DeadlineExceeded
	}
	block, errs = sinFileRefBlock(context.Background(), "@"+p)
	if len(errs) != 0 || !strings.Contains(block, "识图失败") {
		t.Fatalf("识图失败应回退占位: block=%q errs=%v", block, errs)
	}
}

// TestSinFileRefImageDualInjection 识图与全文 OCR 双腿都成功时同块双注入
// （与办公线 appendImageBlock 同语义）。
func TestSinFileRefImageDualInjection(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "dual.png")
	if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := sinVisionRecognize
	t.Cleanup(func() { sinVisionRecognize = orig })
	sinVisionRecognize = func(ctx context.Context, path, prompt string) (string, error) {
		return "一张手写信的照片。", nil
	}
	stubSinOCR(t, func(string) (string, error) {
		return "亲爱的编辑：\n新章已附上，盼复。", nil
	})

	block, errs := sinFileRefBlock(context.Background(), "@"+p)
	if len(errs) != 0 {
		t.Fatalf("不应有失败: %v", errs)
	}
	if !strings.Contains(block, "【画面理解】") || !strings.Contains(block, "手写信") {
		t.Fatalf("识图腿缺失: %q", block)
	}
	if !strings.Contains(block, "【全文 OCR】") || !strings.Contains(block, "亲爱的编辑") {
		t.Fatalf("OCR 腿缺失: %q", block)
	}
}

// TestSinFileRefImageOCRTimeout OCR 腿超预算时注入提示行且不拖死识图腿。
func TestSinFileRefImageOCRTimeout(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "slow.png")
	if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := sinVisionRecognize
	t.Cleanup(func() { sinVisionRecognize = orig })
	sinVisionRecognize = func(ctx context.Context, path, prompt string) (string, error) {
		return "超时场景里的画面。", nil
	}
	stubSinOCR(t, func(string) (string, error) {
		time.Sleep(300 * time.Millisecond) // 远超收缩后的预算
		return "这段全文来得太晚，不应注入。", nil
	})

	origBudget := sinVisionOCRBudget
	sinVisionOCRBudget = 30 * time.Millisecond
	t.Cleanup(func() { sinVisionOCRBudget = origBudget })

	block, errs := sinFileRefBlock(context.Background(), "@"+p)
	if len(errs) != 0 {
		t.Fatalf("不应有失败: %v", errs)
	}
	if !strings.Contains(block, "【画面理解】") {
		t.Fatalf("识图腿缺失: %q", block)
	}
	if !strings.Contains(block, "【全文 OCR】未就绪") {
		t.Fatalf("超时提示行缺失: %q", block)
	}
	if strings.Contains(block, "来得太晚") {
		t.Fatalf("迟到全文不应注入: %q", block)
	}
}

// TestSinOCRBlockText_Gates sinOCRBlockText 注入门槛：噪声/失败不注入、
// 超限截断带补读指引。
func TestSinOCRBlockText_Gates(t *testing.T) {
	if got := sinOCRBlockText(sinOCRResult{text: "  短  "}); got != "" {
		t.Errorf("噪声文本应跳过，得到 %q", got)
	}
	if got := sinOCRBlockText(sinOCRResult{err: errors.New("x"), text: "足够长的文本内容示例"}); got != "" {
		t.Errorf("失败结果应跳过，得到 %q", got)
	}
	got := sinOCRBlockText(sinOCRResult{text: strings.Repeat("墨", 5000)}) // 15000 字节 > 8KB
	if !strings.Contains(got, "…[已截断；全文可调用 ocr 工具提取]") {
		t.Errorf("超限文本缺截断+补读标记: %q", got[:120])
	}
	head := strings.TrimSuffix(strings.TrimPrefix(got, "【全文 OCR】\n"), "…[已截断；全文可调用 ocr 工具提取]")
	if len(head) > sinFileRefMaxOCRBytes {
		t.Errorf("头部 %d 字节超上限 %d", len(head), sinFileRefMaxOCRBytes)
	}
	if !utf8.ValidString(head) {
		t.Errorf("截断后出现非法 UTF-8")
	}
}

func TestSinFileRefDirHonestNote(t *testing.T) {
	dir := t.TempDir()
	block, errs := sinFileRefBlock(context.Background(), "@"+dir)
	if len(errs) != 1 || !strings.Contains(errs[0], "目录不注入") {
		t.Fatalf("目录应如实说明: errs=%v", errs)
	}
	if !strings.Contains(block, "目录不注入内容") {
		t.Fatalf("目录块缺失: %q", block)
	}
}

func TestSinFileRefDedupeAndPunctTrim(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("内容X"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 同一文件引用两次 + ASCII 尾标 → 只注入一块
	block, errs := sinFileRefBlock(context.Background(), "@"+p+" 和 @"+p+".")
	if len(errs) != 0 {
		t.Fatalf("不应有失败: %v", errs)
	}
	if got := strings.Count(block, "<file "); got != 1 {
		t.Fatalf("去重后应只有一块，得到 %d: %q", got, block)
	}
}
