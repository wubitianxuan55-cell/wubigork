package builtin

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestScreenCaptureThenVisionLive 端到端验证 agent 工具链：
// screen_capture 截图 → vision 识图（真实本地视觉模型）。
func TestScreenCaptureThenVisionLive(t *testing.T) {
	if os.Getenv("GAEA_LIVE_VISION_TEST") == "" {
		t.Skip("set GAEA_LIVE_VISION_TEST=1 to run")
	}
	t.Chdir(t.TempDir())

	out, err := screenCapture{}.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("screen_capture: %v", err)
	}
	start := strings.Index(out, ".gaea/uploads/")
	end := strings.Index(out, "（尺寸")
	if start < 0 || end <= start {
		t.Fatalf("无法解析截图输出: %q", out)
	}
	rel := out[start:end]
	t.Logf("截图: %s", rel)

	desc, err := visionTool{}.Execute(context.Background(), json.RawMessage(`{"image_path":"`+rel+`"}`))
	if err != nil {
		t.Fatalf("vision: %v", err)
	}
	if strings.TrimSpace(desc) == "" {
		t.Fatal("识图结果为空")
	}
	t.Logf("识图结果: %.200s", desc)
}
