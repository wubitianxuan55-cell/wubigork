package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestSinFileRefImageVisionFallback(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ref.png")
	if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := sinVisionRecognize
	t.Cleanup(func() { sinVisionRecognize = orig })

	sinVisionRecognize = func(ctx context.Context, path, prompt string) (string, error) {
		return "一位红衣女子立于桥头。", nil
	}
	block, errs := sinFileRefBlock(context.Background(), "@"+p)
	if len(errs) != 0 || !strings.Contains(block, "【图片识别】") || !strings.Contains(block, "红衣女子") {
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
