// Package docmd — OCR Provider Seam 测试（Step 3c）。
//
// 固化 seam 行为：
//   - 注册表：ovis/tesseract 自注册、互斥注册 panic、未知 kind fail-closed
//   - 引擎顺序由 GAEA_OCR_ENGINE 配置驱动：auto = ovis→tesseract（保持现状）；
//     显式 = 单引擎；未知 = 报错（不静默回退）
//   - OCRImageText 按引擎链路由（单图降级行为由 ocr_test.go 固化，此处补显式
//     引擎与 fail-closed 用例）
package docmd

import (
	"errors"
	"os"
	"path/filepath"
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
	for _, want := range []string{"ovis", "tesseract"} {
		if !got[want] {
			t.Errorf("注册表缺少 kind %q（已注册: %v）", want, kinds)
		}
	}
}

func TestOCRProviderRegistry_Construct(t *testing.T) {
	for _, kind := range []string{"ovis", "tesseract"} {
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
	RegisterOCRProvider(OCRKindOvis, func() OCRProvider { return nil }) // 已注册 → panic
}

// TestOCREngineOrder_ConfigDriven GAEA_OCR_ENGINE 驱动引擎顺序。
func TestOCREngineOrder_ConfigDriven(t *testing.T) {
	cases := []struct {
		env  string
		want []string
		err  bool
	}{
		{"", []string{"ovis", "tesseract"}, false}, // 默认 = auto
		{"auto", []string{"ovis", "tesseract"}, false},
		{"AUTO", []string{"ovis", "tesseract"}, false}, // 大小写不敏感
		{"ovis", []string{"ovis"}, false},
		{"tesseract", []string{"tesseract"}, false},
		{"paddle", nil, true}, // 未知 → fail-closed
	}
	for _, c := range cases {
		t.Setenv("GAEA_OCR_ENGINE", c.env)
		got, err := ocrEngineOrder()
		if c.err {
			if err == nil {
				t.Errorf("GAEA_OCR_ENGINE=%q 应报错", c.env)
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

// TestOCRImageText_ExplicitTesseract 显式 tesseract：只走 tesseract 分支，
// OvisOCR2 环境变量缺失也不影响（配置驱动选择）。
func TestOCRImageText_ExplicitTesseract(t *testing.T) {
	t.Setenv("GAEA_OCR_ENGINE", "tesseract")
	// OvisOCR2 指向不存在目录（若被误用必然失败）
	t.Setenv("GAEA_OCR_DIR", filepath.Join(t.TempDir(), "no-such-ocr"))

	img := filepath.Join(t.TempDir(), "img.png")
	if err := os.WriteFile(img, []byte("fake-image"), 0o644); err != nil {
		t.Fatal(err)
	}

	restoreTesseractVars(t)
	tesseractLookPath = func(string) (string, error) { return "C:\\fake\\tesseract.exe", nil }
	tesseractImage = func(_, _ string) (string, error) { return "显式 tesseract 结果", nil }

	text, err := OCRImageText(img)
	if err != nil {
		t.Fatalf("OCRImageText: %v", err)
	}
	if text != "显式 tesseract 结果" {
		t.Fatalf("text = %q, want 显式 tesseract 结果", text)
	}
}

// TestOCRImageText_UnknownEngineFailsClosed 未知引擎：报错，且不静默降级到
// tesseract（即使 tesseract 可用）。
func TestOCRImageText_UnknownEngineFailsClosed(t *testing.T) {
	t.Setenv("GAEA_OCR_ENGINE", "paddle")

	restoreTesseractVars(t)
	tesseractLookPath = func(string) (string, error) { return "C:\\fake\\tesseract.exe", nil }
	tesseractImage = func(_, _ string) (string, error) { return "不应被使用", nil }

	_, err := OCRImageText("whatever.png")
	if err == nil {
		t.Fatal("未知引擎应报错（fail-closed，不得降级 tesseract）")
	}
	if !strings.Contains(err.Error(), "paddle") {
		t.Fatalf("错误应说明引擎名: %v", err)
	}
}

// TestOCRImageText_ExplicitEngineUnavailable 显式引擎不可用：报错，不静默回退。
func TestOCRImageText_ExplicitEngineUnavailable(t *testing.T) {
	t.Setenv("GAEA_OCR_ENGINE", "tesseract")
	restoreTesseractVars(t)
	tesseractLookPath = func(string) (string, error) { return "", errors.New("tesseract 未安装") }

	_, err := OCRImageText("whatever.png")
	if err == nil {
		t.Fatal("显式 tesseract 不可用应报错")
	}
	if !strings.Contains(err.Error(), "tesseract") {
		t.Fatalf("错误应说明引擎名: %v", err)
	}
}

// TestOCRPDFUnavailableError_ConfigAware 扫描件 PDF 引擎不可用错误按配置生成：
// auto 链保留原文案（含 OvisOCR2/tesseract 安装提示），显式单引擎给针对性文案。
func TestOCRPDFUnavailableError_ConfigAware(t *testing.T) {
	err := ocrPDFUnavailableError([]string{"ovis", "tesseract"})
	for _, want := range []string{"OvisOCR2", "tesseract", "安装"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("auto 错误缺少 %q: %v", want, err)
		}
	}
	err1 := ocrPDFUnavailableError([]string{"tesseract"})
	if !strings.Contains(err1.Error(), "tesseract") {
		t.Errorf("单引擎错误应说明引擎: %v", err1)
	}
}
