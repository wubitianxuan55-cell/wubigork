// gaea_browser.go 浏览器观察窗绑定（v4.28 A2「浏览器观察窗」）。
//
// 观察是被动动作：GaeaBrowserObserve 只读当前受控浏览器状态并截一帧，浏览器
// 未运行时返回 Available=false，绝不拉起（拉起只由 browser_* 工具触发）。
// 与 gaebraCwd 无关：受控 Edge 是进程级单例（browser.Default()，internal/gaea/
// tool/builtin/browser_tools.go 同源），无需按工作区隔离。
package app

import (
	"context"

	"github.com/gaea/gaea/internal/gaea/browser"
)

// browserObserve 注入 seam（测试替换；生产恒为 browser.Default().Observe）。
// internal/app 此前没有 browser manager 注入点（grep 确认：browser 只在
// tool/builtin 经 browser.Default() 单例消费），故最小注入 = 直接取进程级
// 单例，与既有 browser_* 工具同源同状态。
var browserObserve = func(ctx context.Context) browser.ObserveView {
	return browser.Default().Observe(ctx)
}

// GaeaBrowserObserve 浏览器观察窗单帧视图（v4.28 A2）：截图（jpeg data URL，
// ≤1280 宽）+ 当前 URL/标题 + 时间戳。不返回 error：浏览器未运行是观察窗的
// 正常态（Available=false），前端按空态渲染而非报错。
func (a *App) GaeaBrowserObserve() browser.ObserveView {
	return browserObserve(context.Background())
}
// OfficeB 门面 wrapper 已由 gen_bindings 写入 bindings_office.go（v4.27.2 起的
// 契约测试锁方法集一致；v4.28 集成时曾临时手补、生成后已删除）。
