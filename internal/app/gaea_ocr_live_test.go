package app

// gaea_ocr_live_test.go — GaeaOCRText 腿序实弹验收（默认跳过）：GAEA_OCR_LEG_ORDER
// 未设置时本地链先行，真打常驻 uocr2 服务（8138）识别真实图片。跑法：
//
//	GAEA_OCR_LIVE_IMAGE=<图片路径> go test ./internal/app -run TestGaeaOCRText_LiveLocalFirst -count=1
//
// 不设 GAEA_OCR_LIVE_IMAGE 即跳过；断言锚点词来自福建价格表（2026-10-29 三引擎
// 对比基准素材），预期 uocr2 返回 23 行价格表全表（CPU 约 60~90s）。

import (
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

func TestGaeaOCRText_LiveLocalFirst(t *testing.T) {
	img := os.Getenv("GAEA_OCR_LIVE_IMAGE")
	if img == "" {
		t.Skip("设 GAEA_OCR_LIVE_IMAGE=<图片路径> 跑实弹（真 uocr2 服务，CPU 约分钟级）")
	}
	if os.Getenv("GAEA_OCR_LEG_ORDER") != "" {
		t.Skip("实弹验收只针对默认腿序（GAEA_OCR_LEG_ORDER 未设置）")
	}

	a := &App{core: &core{engineMgr: modelengine.NewManager("", "")}} // herdsman 腿仅作兜底
	got, err := a.GaeaOCRText(img)
	if err != nil {
		t.Fatalf("GaeaOCRText 实弹: %v", err)
	}
	for _, want := range []string{"水温", "流量", "总氰化物", "点、次"} {
		if !strings.Contains(got, want) {
			t.Errorf("识别结果缺少锚点词 %q（腿序/清洗异常）", want)
		}
	}
	if strings.Contains(got, "<BLOCK>") || strings.Contains(got, "<FORK>") {
		t.Errorf("识别结果泄漏结构标记（清洗器未生效）")
	}
	t.Logf("实弹识别 %d 字符", len(got))
}
