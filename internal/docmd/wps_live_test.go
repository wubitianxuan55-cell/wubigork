package docmd

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveWPS_ScannedPDF 真调本机 WPS 的实弹，默认跳过（会消耗会员额度、依赖本机
// WPS 登录态）。GAEA_WPS_LIVE_PDF 指向一份「提取不到文字」的 PDF（扫描件 / 字形号
// 文字层）时验证整条链：WPS 在线 OCR 优先 → docx → Markdown，并打印耗时与结构计数。
// 只在本机手动跑：
//
//	GAEA_WPS_PDF=1 GAEA_WPS_LIVE_PDF='D:\path\扫描件.pdf' go test ./internal/docmd/ -run TestLiveWPS -v -count=1
//
// 可选 GAEA_WPS_LIVE_OUT 把 Markdown 落到指定路径供人工核对。
func TestLiveWPS_ScannedPDF(t *testing.T) {
	path := strings.TrimSpace(os.Getenv("GAEA_WPS_LIVE_PDF"))
	if path == "" {
		t.Skip("GAEA_WPS_LIVE_PDF 未设置：默认跳过（会真调 WPS）")
	}
	if os.Getenv("GAEA_WPS_PDF") != "1" {
		t.Fatal("TestMain 默认关了 WPS 通道：实弹须显式 GAEA_WPS_PDF=1")
	}
	forbidOCR(t) // 实弹只验证 WPS 通路：WPS 不通时不许落本地 OCR

	start := time.Now()
	md, total, truncated, images, err := ConvertPreservingImages(path, "", 0)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("WPS 在线 OCR 通路失败（%s）: %v", elapsed.Round(time.Millisecond), err)
	}
	st := CountStructure(md)
	cjk := 0
	for _, r := range md {
		if r >= 0x4E00 && r <= 0x9FFF {
			cjk++
		}
	}
	t.Logf("耗时=%s 总页数=%d 截断=%v 图片=%d 汉字=%d 标题行=%d 表格=%d",
		elapsed.Round(time.Millisecond), total, truncated, images, cjk, st.Headings, st.Tables)
	if total <= 0 || truncated || cjk < 1000 {
		t.Fatalf("产物不像一份可读文档：total=%d truncated=%v 汉字=%d", total, truncated, cjk)
	}
	if out := os.Getenv("GAEA_WPS_LIVE_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
			t.Fatalf("落盘 %s: %v", out, err)
		}
	}
}
