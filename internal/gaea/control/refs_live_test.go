package control

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveRefs_LiveVision 真实调用本地视觉模型（临时端到端验证）。
func TestResolveRefs_LiveVision(t *testing.T) {
	if os.Getenv("GAEA_LIVE_VISION_TEST") == "" {
		t.Skip("set GAEA_LIVE_VISION_TEST=1 to run")
	}
	srcAbs, err := filepath.Abs("../../../build/appicon.png")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".gaea/uploads", 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(srcAbs)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.Create(".gaea/uploads/paste-live.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatal(err)
	}
	dst.Close()

	c := &Controller{}
	block, errs := c.ResolveRefs(context.Background(), "识别 @.gaea/uploads/paste-live.png")
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if !strings.Contains(block, "【图片识别】") {
		t.Fatalf("block 未包含识别结果: %.200s", block)
	}
	t.Logf("识别结果: %.300s", block)
}
