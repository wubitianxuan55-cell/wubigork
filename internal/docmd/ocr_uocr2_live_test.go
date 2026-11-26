package docmd

// ocr_uocr2_live_test.go — uocr2 llama 后端实弹验收（默认跳过）：
//
//	GAEA_UOCR2_LIVE=1 go test ./internal/docmd -run TestUocr2LlamaLive -count=1 -v
//
// 真实链路：探测/按需拉起 C:\AI\gaea-OCR\uocr2-llama 工件（8139）→ chat 协议单页识别 +
// 2 页批识别（SR-SWA 跨页续排锚点：页 2 价格表应从第 24 行续排）。素材与
// 2026-10-29 三引擎对比/补丁验证同源（福建价格表第 2/3 页）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// liveSamplePaths 实弹素材的候选位置：引擎工件目录下的 testimg（GAEA_UOCR2_LLAMA_DIR
// 优先，默认 C:\AI\gaea-OCR\uocr2-llama\testimg——2026-11-26 从已删除的 python 后端
// 目录挪来）。
func liveSamplePaths() [][2]string {
	roots := []string{`C:\AI\gaea-OCR\uocr2-llama\testimg`}
	if dir := strings.TrimSpace(os.Getenv("GAEA_UOCR2_LLAMA_DIR")); dir != "" {
		roots = append([]string{filepath.Join(dir, "testimg")}, roots...)
	}
	var out [][2]string
	for _, r := range roots {
		out = append(out, [2]string{filepath.Join(r, "fj-02.png"), filepath.Join(r, "fj-03.png")})
	}
	return out
}

func TestUocr2LlamaLiveSingleAndBatch(t *testing.T) {
	if os.Getenv("GAEA_UOCR2_LIVE") != "1" {
		t.Skip("设 GAEA_UOCR2_LIVE=1 跑实弹（真 llama 后端，分钟级以内）")
	}
	p := &uocr2OCRProvider{}

	var fj2, fj3 string
	for _, pair := range liveSamplePaths() {
		if _, err := os.Stat(pair[0]); err == nil {
			if _, err := os.Stat(pair[1]); err == nil {
				fj2, fj3 = pair[0], pair[1]
				break
			}
		}
	}
	if fj2 == "" {
		t.Skip("实弹素材缺失：fj-02.png / fj-03.png（uocr2-llama\\testimg）")
	}

	t0 := time.Now()
	single, err := p.ExtractImage(fj2)
	if err != nil {
		t.Fatalf("单页实弹: %v", err)
	}
	for _, want := range []string{"水温", "流量", "总氰化物", "点、次"} {
		if !strings.Contains(single, want) {
			t.Errorf("单页结果缺少锚点词 %q", want)
		}
	}
	if strings.Contains(single, "<BLOCK>") || strings.Contains(single, "<PAGE>") {
		t.Errorf("单页结果泄漏结构标记（清洗器未生效）")
	}
	t.Logf("单页实弹 %.1fs, %d 字符", time.Since(t0).Seconds(), len(single))

	t0 = time.Now()
	pages, err := p.ExtractPages([]string{fj2, fj3})
	if err != nil {
		t.Fatalf("2 页批实弹: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("批结果应 2 页, got %d", len(pages))
	}
	// 跨页续排签名：页 1 收在 23 行（总氰化物），页 2 从第 24 行（挥发酚）续排。
	if !strings.Contains(pages[0], "总氰化物") {
		t.Errorf("页 1 应含第 23 行锚点「总氰化物」")
	}
	if !strings.Contains(pages[1], "挥发酚") {
		t.Errorf("页 2 应含续排锚点「挥发酚」（第 24 行，SR-SWA 跨页上下文）")
	}
	t.Logf("2 页批实弹 %.1fs, 页1 %d / 页2 %d 字符", time.Since(t0).Seconds(), len(pages[0]), len(pages[1]))
}
