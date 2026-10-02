package docmd

// ocr_kind_domain_test.go — 审计 U54/IN3-09 钉子：本 seam 的 kind 取值域 =
// 本地 OCR 引擎（ovis/tesseract）。远端 herdsman 链（internal/ocr 的
// PaddleOCR/MinerU，app/gaea_ocr.go 编排）刻意不在域内——若未来把 herdsman
// 注册成 provider kind 或写进 GAEA_OCR_ENGINE 取值域（审计原修法方向），
// 本文件红；须回 ocr.go seam 注释「口径边界」段重新论证分工后再动。

import (
	"strings"
	"testing"
)

// TestOCRKindDomain_LocalOnly 远端引擎/模型名不得混入本地 seam kind 取值域。
func TestOCRKindDomain_LocalOnly(t *testing.T) {
	for _, remote := range []string{"herdsman", "paddleocr", "paddleocr-ppocrv5-server", "minerU"} {
		for _, k := range OCRProviderKinds() {
			if strings.EqualFold(k, remote) {
				t.Errorf("kind %q 属远端 herdsman 链，不得注册进本地 seam（已注册: %v）", remote, OCRProviderKinds())
			}
		}
		if _, err := NewOCRProvider(remote); err == nil {
			t.Errorf("NewOCRProvider(%q) 应报错（fail-closed）", remote)
		} else if !strings.Contains(err.Error(), "unknown ocr provider kind") {
			t.Errorf("NewOCRProvider(%q) 错误不符: %v", remote, err)
		}
	}
}

// TestOCREngineOrder_RejectsRemoteKinds GAEA_OCR_ENGINE 填远端引擎/模型名
// fail-closed，不静默回退 auto 链。
func TestOCREngineOrder_RejectsRemoteKinds(t *testing.T) {
	for _, remote := range []string{"herdsman", "paddleocr-ppocrv5-server", "minerU"} {
		t.Setenv("GAEA_OCR_ENGINE", remote)
		got, err := ocrEngineOrder()
		if err == nil {
			t.Errorf("GAEA_OCR_ENGINE=%q 应报错，got %v", remote, got)
		}
	}
}
