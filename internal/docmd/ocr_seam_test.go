// Package docmd — OCR Provider Seam 测试（Step 3c）。
//
// 固化 seam 行为（2026-11-26 精简后本地唯一引擎是 uocr2；路由类用例用替身引擎
// ocrTestKind，定义见 ocr_test.go）：
//   - 注册表：uocr2 自注册、互斥注册 panic、未知 kind fail-closed；已退役引擎不得再注册
//   - 引擎由 GAEA_OCR_ENGINE 配置驱动：空 / auto / uocr2 = uocr2；显式 = 单引擎；
//     未知 = 报错；已退役（ovis / tesseract）= 报错并指路，不静默换挡
//   - OCRImageText 按引擎路由（显式引擎、未知引擎、已退役引擎、显式引擎不可用）
package docmd

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestOCRProviderRegistry_Kinds(t *testing.T) {
	kinds := OCRProviderKinds()
	got := map[string]bool{}
	for _, k := range kinds {
		got[k] = true
	}
	if !got[OCRKindUocr2] {
		t.Errorf("注册表缺少 kind %q（已注册: %v）", OCRKindUocr2, kinds)
	}
	for _, retired := range []string{"ovis", "tesseract"} {
		if got[retired] {
			t.Errorf("已退役的引擎 %q 不得再注册（已注册: %v）", retired, kinds)
		}
	}
}

func TestOCRProviderRegistry_Construct(t *testing.T) {
	for _, kind := range []string{OCRKindUocr2, ocrTestKind} {
		p, err := NewOCRProvider(kind)
		if err != nil {
			t.Fatalf("NewOCRProvider(%q): %v", kind, err)
		}
		if p == nil || p.Name() != kind {
			t.Errorf("%q Name = %v, want %q", kind, p, kind)
		}
	}
	if _, err := NewOCRProvider("no-such-engine"); err == nil {
		t.Fatal("未知 kind 应报错（fail-closed）")
	}
}

func TestOCRProviderRegistry_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic（互斥注册纪律）")
		}
	}()
	RegisterOCRProvider(OCRKindUocr2, func() OCRProvider { return nil }) // 已注册 → panic
}

// TestOCREngineOrder_ConfigDriven GAEA_OCR_ENGINE 驱动引擎选择：默认与显式 uocr2 都只有
// uocr2；已退役的 ovis / tesseract 与未知 kind 一样 fail-closed，但错误要指路。
func TestOCREngineOrder_ConfigDriven(t *testing.T) {
	cases := []struct {
		env    string
		want   []string
		errHas string
	}{
		{"", []string{OCRKindUocr2}, ""},
		{"auto", []string{OCRKindUocr2}, ""},
		{"AUTO", []string{OCRKindUocr2}, ""}, // 大小写不敏感
		{"uocr2", []string{OCRKindUocr2}, ""},
		{ocrTestKind, []string{ocrTestKind}, ""}, // 显式替身 kind（证明 seam 仍可插拔）
		{"ovis", nil, "退役"},
		{"tesseract", nil, "退役"},
		{"paddle", nil, "未知"}, // 未知 → fail-closed
	}
	for _, c := range cases {
		t.Setenv("GAEA_OCR_ENGINE", c.env)
		got, err := ocrEngineOrder()
		if c.errHas != "" {
			if err == nil || !strings.Contains(err.Error(), c.errHas) {
				t.Errorf("GAEA_OCR_ENGINE=%q 应报含 %q 的错误，得到 %v", c.env, c.errHas, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("GAEA_OCR_ENGINE=%q: %v", c.env, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("GAEA_OCR_ENGINE=%q → %v, want %v", c.env, got, c.want)
		}
	}
}

// TestOCRImageText_ExplicitEngine 显式引擎：只走该引擎（uocr2 不可用也不影响）。
func TestOCRImageText_ExplicitEngine(t *testing.T) {
	t.Setenv("GAEA_OCR_ENGINE", ocrTestKind)
	runs := useFakeOCR(t, "显式替身引擎结果", nil)

	text, err := OCRImageText(writeFakeImage(t))
	if err != nil {
		t.Fatalf("OCRImageText: %v", err)
	}
	if text != "显式替身引擎结果" || *runs != 1 {
		t.Fatalf("text = %q runs = %d, want 显式替身引擎结果 / 1", text, *runs)
	}
}

// TestOCRImageText_UnknownEngineFailsClosed 未知引擎：报错，且不静默落到任何可用引擎。
func TestOCRImageText_UnknownEngineFailsClosed(t *testing.T) {
	t.Setenv("GAEA_OCR_ENGINE", "paddle")
	runs := useFakeOCR(t, "不应被使用", nil)

	_, err := OCRImageText("whatever.png")
	if err == nil || !strings.Contains(err.Error(), "paddle") {
		t.Fatalf("未知引擎应报错并说明引擎名（fail-closed）: %v", err)
	}
	if *runs != 0 {
		t.Fatalf("未知引擎不得落到替身引擎（runs=%d）", *runs)
	}
}

// TestOCRImageText_RetiredEngineFailsClosed 已退役引擎（旧机器残留配置）：报错并指路，
// 不静默换挡到 uocr2。
func TestOCRImageText_RetiredEngineFailsClosed(t *testing.T) {
	for _, retired := range []string{"ovis", "tesseract"} {
		t.Setenv("GAEA_OCR_ENGINE", retired)
		runs := useFakeOCR(t, "不应被使用", nil)

		_, err := OCRImageText("whatever.png")
		if err == nil || !strings.Contains(err.Error(), "退役") || !strings.Contains(err.Error(), "uocr2") {
			t.Fatalf("GAEA_OCR_ENGINE=%s 应报「已退役」并指路 uocr2：%v", retired, err)
		}
		if *runs != 0 {
			t.Fatalf("GAEA_OCR_ENGINE=%s 不得落到替身引擎（runs=%d）", retired, *runs)
		}
	}
}

// TestOCRImageText_ExplicitEngineUnavailable 显式引擎不可用：报错，不静默回退。
func TestOCRImageText_ExplicitEngineUnavailable(t *testing.T) {
	t.Setenv("GAEA_OCR_ENGINE", ocrTestKind)
	useFakeOCR(t, "", errors.New("替身引擎不可用"))

	_, err := OCRImageText("whatever.png")
	if err == nil {
		t.Fatal("显式引擎不可用应报错")
	}
	if !strings.Contains(err.Error(), ocrTestKind) {
		t.Fatalf("错误应说明引擎名: %v", err)
	}
}

// TestOCRUnavailableErrors_ConfigAware 引擎不可用错误按配置生成：默认指路 Unlimited-OCR-2
// 且不再提已退役引擎；显式替身 kind 给针对性文案。
func TestOCRUnavailableErrors_ConfigAware(t *testing.T) {
	for name, err := range map[string]error{
		"图片":  ocrUnavailableError([]string{OCRKindUocr2}),
		"PDF": ocrPDFUnavailableError([]string{OCRKindUocr2}),
	} {
		for _, want := range []string{"Unlimited-OCR-2", "安装"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s错误缺少 %q: %v", name, want, err)
			}
		}
		for _, retired := range []string{"OvisOCR2", "tesseract"} {
			if strings.Contains(err.Error(), retired) {
				t.Errorf("%s错误不应再提已退役的引擎 %q: %v", name, retired, err)
			}
		}
	}
	if err1 := ocrPDFUnavailableError([]string{ocrTestKind}); !strings.Contains(err1.Error(), ocrTestKind) {
		t.Errorf("单引擎错误应说明引擎: %v", err1)
	}
	if err1 := ocrUnavailableError([]string{ocrTestKind}); !strings.Contains(err1.Error(), ocrTestKind) {
		t.Errorf("单引擎错误应说明引擎: %v", err1)
	}
}
