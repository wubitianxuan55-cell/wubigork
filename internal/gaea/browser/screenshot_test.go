package browser

import (
	"context"
	"testing"
)

// ── 观察窗（v4.28 A2）：Observe 被动观察面 ────────────────────────────────

// TestObserveNotRunning 浏览器未运行时 Observe 返回 Available=false，且**绝不
// 拉起浏览器**（fake 服务零 ws 升级 = 零拨号/零启动；观察是被动动作）。
func TestObserveNotRunning(t *testing.T) {
	srv, f := newFakeDevtools(t)
	m := NewManager(Options{InjectHTTPBase: srv.URL})
	t.Cleanup(m.Shutdown)

	view := m.Observe(context.Background())
	if view.Available {
		t.Fatalf("未运行时 Available 应为 false，得到 %+v", view)
	}
	if view.URL != "" || view.Title != "" || view.Image != "" || view.Error != "" {
		t.Fatalf("未运行时其余字段应为空，得到 %+v", view)
	}
	if n := f.upgrades; n != 0 {
		t.Fatalf("观察不得拉起浏览器：ws 升级次数 = %d，应为 0", n)
	}
}

// TestObserveFrame 已就绪会话上观察一帧：jpeg data URL + 元信息 + 宽度上限
// 缩放（fake 页面 2560 宽 → clip.scale=0.5 → 位图 1280×600）。
func TestObserveFrame(t *testing.T) {
	m, f := newFakeManager(t)

	view := m.Observe(context.Background())
	if !view.Available {
		t.Fatalf("已 Ensure 后 Available 应为 true，得到 %+v", view)
	}
	if view.URL != "http://fake.local/page" || view.Title != "观察页" {
		t.Fatalf("URL/Title 未取到：%+v", view)
	}
	if want := "data:image/jpeg;base64,ZmFrZWpwZWc="; view.Image != want {
		t.Fatalf("Image = %q, want %q", view.Image, want)
	}
	if view.UpdatedAt <= 0 {
		t.Fatalf("UpdatedAt 应为正毫秒时间戳，得到 %d", view.UpdatedAt)
	}
	if view.Width != 1280 || view.Height != 600 {
		t.Fatalf("缩放后位图尺寸 = %dx%d, want 1280x600（2560×0.5 / 1200×0.5）", view.Width, view.Height)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.screenshots) != 1 {
		t.Fatalf("captureScreenshot 调用次数 = %d, want 1", len(f.screenshots))
	}
	params := f.screenshots[0]
	if params["format"] != "jpeg" {
		t.Fatalf("截图格式 = %v, want jpeg", params["format"])
	}
	if params["optimizeForSpeed"] != true {
		t.Fatalf("optimizeForSpeed 未开启：%v", params["optimizeForSpeed"])
	}
	clip, ok := params["clip"].(map[string]any)
	if !ok {
		t.Fatalf("缺 clip 参数：%+v", params)
	}
	const eps = 1e-9
	if got := clip["scale"].(float64); got < 0.5-eps || got > 0.5+eps {
		t.Fatalf("clip.scale = %v, want 0.5（1280/2560）", got)
	}
	if got := clip["width"].(float64); got < 2560-eps || got > 2560+eps {
		t.Fatalf("clip.width = %v, want 2560", got)
	}
}

// TestObserveScreenshotFailure 截图失败（CDP 回空帧）时 Available 仍为 true、
// Error 说明原因、Image 为空——URL/Title 元信息不因截图失败而丢。
func TestObserveScreenshotFailure(t *testing.T) {
	m, f := newFakeManager(t)
	f.mu.Lock()
	f.emptyShot = true
	f.mu.Unlock()

	view := m.Observe(context.Background())
	if !view.Available {
		t.Fatalf("截图失败不影响可用性判定，得到 %+v", view)
	}
	if view.Error == "" {
		t.Fatalf("截图失败时 Error 应说明原因：%+v", view)
	}
	if view.Image != "" {
		t.Fatalf("截图失败时 Image 应为空：%+v", view)
	}
	if view.URL != "http://fake.local/page" {
		t.Fatalf("元信息应照常返回：%+v", view)
	}
}
