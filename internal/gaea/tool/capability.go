// capability.go 是工具能力声明（codex 蒸馏 B6 折算，第四批 P8）。
//
// 三档能力（readOnly / 写盘 / 网络）：前两档已由 Tool 接口声明式自报
//（ReadOnly / PersistWrite），本文件补第三档——网络出口。不给 Tool 接口加
// 方法（全量工具面 retrofit 是纯成本）：注册表侧集中名单 + 前缀规则，
// 测试钉死防漂移——新增网络出口工具必须入表（sandbox 断网联动靠它闭合）。
//
// 注意：bash 不在名单——它有自己的网络开关（sandbox.Spec.Network 控制子
// 进程出网），整体拒绝 bash 会让「断网但本地命令可用」的形态消失。
package tool

import "strings"

// networkTools 是以网络出口为目的内建工具（显示名与注册名一致）。
var networkTools = map[string]bool{
	"web_search": true,
	"web_fetch":  true,
	// 受控浏览器全家：导航/读取/快照/交互/JS 注入/标签页——全部以网络页面
	// 为操作对象（browser_close 也须在断网会话一并收口，防半开状态）。
	"browser_navigate":   true,
	"browser_read":       true,
	"browser_snapshot":   true,
	"browser_click":      true,
	"browser_type":       true,
	"browser_press":      true,
	"browser_scroll":     true,
	"browser_close":      true,
	"browser_tabs":       true,
	"browser_new_tab":    true,
	"browser_switch_tab": true,
	"browser_run_js":     true,
}

// mcpPrefix 是 MCP 动态工具的命名空间前缀（plugin 包单源约定
// "mcp__<server>__<tool>"）——MCP 服务器可能是远程进程，一律按网络能力论。
const mcpPrefix = "mcp__"

// NetworkCapable 报告工具是否具备网络出口能力（P8 能力声明第三档）。
// 消费方：[sandbox] network=false 时断网闸拒绝（agent.WrapNetworkGate）。
func NetworkCapable(name string) bool {
	if networkTools[name] {
		return true
	}
	return strings.HasPrefix(name, mcpPrefix)
}
