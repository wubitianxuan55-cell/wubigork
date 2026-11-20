package bridge

// W0 图片预算测试（电脑操控施工单 2026-11-19）：单请求上限 4 张，
// 超限丢最旧；编码失败跳过单张不拖垮请求；无图路径零编码。

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func writeTinyJPEG(t *testing.T, dir, name string) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 50}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return p
}

func TestToChatMessages_ImageBudgetKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	p := writeTinyJPEG(t, dir, "shot.jpg")
	msgs := make([]provider.Message, 6)
	for i := range msgs {
		msgs[i] = provider.Message{Role: provider.RoleTool, Content: fmt.Sprintf("m%d", i), Images: []string{p}}
	}
	out := toChatMessages(msgs)
	if len(out) != len(msgs) {
		t.Fatalf("消息数不变: got %d want %d", len(out), len(msgs))
	}
	for i := range out {
		if i < 2 {
			if len(out[i].Images) != 0 {
				t.Fatalf("最旧的两条应丢图: msg%d images=%d", i, len(out[i].Images))
			}
			continue
		}
		if len(out[i].Images) != 1 || out[i].Images[0][:22] != "data:image/jpeg;base64" {
			t.Fatalf("新四条应各带 1 张 data URL: msg%d %v", i, out[i].Images)
		}
	}
}

func TestToChatMessages_EncodeFailureSkipsImage(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleTool, Content: "boom", Images: []string{filepath.Join(t.TempDir(), "missing.png")}},
	}
	out := toChatMessages(msgs)
	if len(out) != 1 || out[0].Content != "boom" {
		t.Fatalf("编码失败不得影响消息本体: %+v", out)
	}
	if len(out[0].Images) != 0 {
		t.Fatalf("编码失败的单张应跳过: %v", out[0].Images)
	}
}

func TestToChatMessages_NoImagesUntouched(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "hi"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "t", Arguments: "{}"}}},
	}
	out := toChatMessages(msgs)
	for i, cm := range out {
		if cm.Images != nil {
			t.Fatalf("无图消息 Images 必须为 nil: msg%d %v", i, cm.Images)
		}
	}
}
