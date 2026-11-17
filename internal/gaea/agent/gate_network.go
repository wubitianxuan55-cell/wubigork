// gate_network.go 是断网闸装饰器（codex 蒸馏 B6 折算，第四批 P8）。
//
// 越权拒绝的落点：[sandbox] network=false 时，网络能力工具（web/浏览器/
// MCP 动态工具）在闸门层拒绝并给出可读理由——此前断网只关 bash 子进程
// 出网，web/浏览器/MCP 照常出网是「半开沙箱」。bash 不拒（它有自己的
// 子进程网络开关，整体拒绝会消灭「断网但本地命令可用」形态）。
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// NetworkGate 在内层闸门前叠加断网拒绝（Allow=true 时零介入）。
type NetworkGate struct {
	Inner Gate
	Allow bool
}

// WrapNetworkGate 按开关系断网闸：allowNetwork=true 或内层为 nil 时原样
// 返回（现状零变化）。
func WrapNetworkGate(inner Gate, allowNetwork bool) Gate {
	if inner == nil || allowNetwork {
		return inner
	}
	return &NetworkGate{Inner: inner, Allow: false}
}

// Check 先判网络能力（拒绝文案可操作），再走内层闸门。
func (g *NetworkGate) Check(ctx context.Context, toolName string, args json.RawMessage, readOnly bool) (bool, string, error) {
	if !g.Allow && tool.NetworkCapable(toolName) {
		return false, fmt.Sprintf(
			"tool %s blocked: sandbox network is disabled ([sandbox] network=false) — network tools (web/browser/mcp) are refused in this session; local tools still work",
			toolName), nil
	}
	return g.Inner.Check(ctx, toolName, args, readOnly)
}
