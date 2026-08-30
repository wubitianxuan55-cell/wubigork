package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// v4.8.3 微信识图升级：多模态 Qwen 主模型优先（visionRecognize seam 注入），
// PaddleOCR 三级链（GaeaOCRText）降为兜底。

func TestVisionOCRText_PrefersMultimodal(t *testing.T) {
	orig := visionRecognize
	t.Cleanup(func() { visionRecognize = orig })
	visionRecognize = func(ctx context.Context, path, prompt string) (string, error) {
		return "识别文本", nil
	}

	a := &App{}
	got, err := a.visionOCRText("whatever.png")
	if err != nil {
		t.Fatalf("visionOCRText: %v", err)
	}
	if got != "识别文本" {
		t.Fatalf("got %q, want 主模型识别文本", got)
	}
}

func TestVisionOCRText_FallsBackOnMultimodalFailure(t *testing.T) {
	orig := visionRecognize
	t.Cleanup(func() { visionRecognize = orig })
	visionRecognize = func(ctx context.Context, path, prompt string) (string, error) {
		return "", errors.New("vision busy")
	}

	a := &App{}
	_, err := a.visionOCRText("no-such-file.png")
	// 主模型失败后必须到达 GaeaOCRText（其 os.Stat 报「图片不存在」）——
	// 错误文案证明回退链被走到，而不是把 vision 的错误原样上抛。
	if err == nil || !strings.Contains(err.Error(), "图片不存在") {
		t.Fatalf("应回退到 GaeaOCRText 并报图片不存在, got %v", err)
	}
}
