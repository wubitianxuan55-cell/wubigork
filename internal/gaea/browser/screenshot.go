// screenshot.go 浏览器观察窗观察面（v4.28 A2「浏览器观察窗」）。
//
// 设计（docs/research-2026-09-01/browser-observation.md）：观察窗走「截图步进
// 流」起版——CDP Page.captureScreenshot 单帧 jpeg（optimizeForSpeed），不做实时
// 帧流（远期再评估）。观察是**被动动作**：浏览器未运行时返回 Available=false，
// 绝不拉起浏览器（拉起只由 browser_* 工具的 Ensure 触发）。
package browser

import (
	"context"
	"fmt"
	"math"
	"time"
)

const (
	// maxObserveWidth 观察截图宽度上限（px）：页面更宽时按比例缩小（clip.scale
	// <1），控制 jpeg base64 体积（面板 <img> 与 Wails 桥通道都吃这个体积）。
	maxObserveWidth = 1280
	// observeJPEGQuality 观察截图 jpeg 质量（0-100）：观察窗以「看得清布局」
	// 为目标，不求 retina 保真；70 在体积与可读性之间（同屏截图经验值）。
	observeJPEGQuality = 70
)

// ObserveView 观察窗单帧视图：截图（data URL）+ 当前页上下文。json tag 全键
// 小驼峰（前端 bridge 直接消费）。
type ObserveView struct {
	// Available 受控浏览器当前是否有活动会话（false = 未运行；其余字段为空）。
	Available bool `json:"available"`
	// URL 当前活动页落点 URL（取不到时为空）。
	URL string `json:"url"`
	// Title 当前活动页标题。
	Title string `json:"title"`
	// Image 截图 data URL（"data:image/jpeg;base64,…"）；截图失败时为空串
	// （此时 Error 说明原因，URL/Title 仍可能有效）。
	Image string `json:"image"`
	// Width/Height 截图位图尺寸（px；缩放后的实际位图，非 CSS 尺寸）。
	Width  int `json:"width"`
	Height int `json:"height"`
	// UpdatedAt 本帧观察时刻（Unix 毫秒；前端展示「最后刷新」）。
	UpdatedAt int64 `json:"updatedAt"`
	// Error 本帧截图失败原因（Available=true 但截图失败时非空；浏览器未运行
	// 时为空——那种情况用 Available=false 表达，不算错误）。
	Error string `json:"error,omitempty"`
}

// Observe 对当前活动页做一次被动观察（截图 + URL/标题 + 时间戳）。
//
// 纪律：不 Ensure——活动会话不存在时直接返回 Available=false（观察窗刷新绝不
// 拉起浏览器；拉起是 browser_* 工具的职责）。截图走 Page.captureScreenshot
// （jpeg + optimizeForSpeed），页面比 maxObserveWidth 宽时经 clip.scale 等比
// 缩到上限内；页面尺寸取不到时退化为无 clip 的视口截图。
func (m *Manager) Observe(ctx context.Context) ObserveView {
	m.mu.Lock()
	conn := m.activeConn()
	m.mu.Unlock()
	if conn == nil {
		return ObserveView{Available: false} // 未运行：被动返回，绝不拉起
	}
	view := ObserveView{Available: true, UpdatedAt: time.Now().UnixMilli()}

	// 页面元信息 + 内容尺寸（一次 evaluate 拿全：title/url/scrollWidth/Height）。
	var meta struct {
		okField
		Title string  `json:"title"`
		URL   string  `json:"url"`
		W     float64 `json:"w"`
		H     float64 `json:"h"`
	}
	if err := m.evaluate(ctx, jsObserve, &meta); err == nil && meta.OK {
		view.Title = meta.Title
		view.URL = meta.URL
	} // 元信息失败不致命：截图仍可尝试（URL/Title 留空，下帧自愈）

	params := map[string]any{
		"format":           "jpeg",
		"quality":          observeJPEGQuality,
		"optimizeForSpeed": true,
	}
	// 内容尺寸已知 → clip 截整页并按宽度上限等比缩放；未知 → 无 clip 截视口。
	if w, h := meta.W, meta.H; w > 0 && h > 0 {
		scale := 1.0
		if w > maxObserveWidth {
			scale = maxObserveWidth / w
		}
		params["clip"] = map[string]any{
			"x": 0, "y": 0,
			"width": w, "height": h,
			"scale": scale,
		}
		view.Width = int(math.Round(w * scale))
		view.Height = int(math.Round(h * scale))
	}
	var res struct {
		Data string `json:"data"` // jpeg base64（无 data: 前缀）
	}
	if err := conn.Call(ctx, "Page.captureScreenshot", params, &res); err != nil {
		view.Error = fmt.Sprintf("截图失败: %v", err)
		return view
	}
	if res.Data == "" {
		view.Error = "截图失败: CDP 返回空帧"
		return view
	}
	view.Image = "data:image/jpeg;base64," + res.Data
	return view
}

// jsObserve 取观察帧元信息（title/url/内容宽高）。__gaeaObserve 为 fake CDP 的
// 匹配 token（同 jsMeta 的 __gaeaMeta 惯例）。scrollWidth/Height 取不到时退化
// 视口宽高（body 未撑开/怪异模式的兜底）。
const jsObserve = `(function(){try{var t='__gaeaObserve';return {ok:true,title:document.title||"",url:location.href,w:document.documentElement.scrollWidth||window.innerWidth,h:document.documentElement.scrollHeight||window.innerHeight};}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`
