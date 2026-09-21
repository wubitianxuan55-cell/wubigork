package app

// 清污绑定（v4.377）：auto-dream 旧直写路径（v4.376 及以前默认行为）不询问
// 就把每轮提炼写进长期记忆，错误/噪音事实污染数据。本组绑定按 dream 审计
// 日志（source=auto_dream 的 Names）与现存事实取交集，供用户预览+确认后
// 批量清除。用户显式接受过的建议（source=explicit）与手工沉淀条目不在
// 交集内，不会被误删。

import "fmt"

// GaeaDreamPurgePreview 预览「清理自动做梦直写」：返回 auto_dream 审计记录
// 过且当前仍存在的事实名。前端展示清单并由用户确认后再调 GaeaDreamPurge。
func (a *App) GaeaDreamPurgePreview() ([]string, error) {
	c := gaeaCtrl()
	if c == nil {
		return nil, fmt.Errorf("办公引擎未初始化")
	}
	return c.DreamAutoWritten(), nil
}

// GaeaDreamPurge 批量删除自动做梦直写过的事实（仅限预览交集，交集外静默
// 跳过——绑定面不可用于删除任意记忆）。返回实际删除条数。
func (a *App) GaeaDreamPurge(names []string) (int, error) {
	c := gaeaCtrl()
	if c == nil {
		return 0, fmt.Errorf("办公引擎未初始化")
	}
	return c.ForgetFacts(names)
}
