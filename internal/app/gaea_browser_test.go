package app

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/gaea/browser"
)

// ── 浏览器观察窗绑定（v4.28 A2）──────────────────────────────────────────

// TestGaeaBrowserObserveUnavailable 默认实现下（无注入、浏览器从未拉起），
// 绑定返回 Available=false 的空视图且不报错——观察是被动动作，绝不拉起浏览器。
func TestGaeaBrowserObserveUnavailable(t *testing.T) {
	a := &App{}
	view := a.GaeaBrowserObserve()
	if view.Available {
		t.Fatalf("浏览器未运行时 Available 应为 false，得到 %+v", view)
	}
	if view.URL != "" || view.Image != "" || view.Error != "" {
		t.Fatalf("未运行时其余字段应为空，得到 %+v", view)
	}
}

// TestGaeaBrowserObserveSeam 注入 seam 生效：替换后绑定原样透传视图（字段零改动）。
func TestGaeaBrowserObserveSeam(t *testing.T) {
	prev := browserObserve
	browserObserve = func(context.Context) browser.ObserveView {
		return browser.ObserveView{
			Available: true,
			URL:       "http://fake.local/page",
			Title:     "观察页",
			Image:     "data:image/jpeg;base64,ZmFrZWpwZWc=",
			Width:     1280,
			Height:    600,
			UpdatedAt: 1725168000000,
		}
	}
	t.Cleanup(func() { browserObserve = prev })

	a := &App{}
	view := a.GaeaBrowserObserve()
	if !view.Available || view.URL != "http://fake.local/page" || view.Title != "观察页" ||
		view.Width != 1280 || view.Height != 600 {
		t.Fatalf("seam 注入未透传视图：%+v", view)
	}
}
