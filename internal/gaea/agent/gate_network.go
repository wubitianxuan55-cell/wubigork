// gate_network.go 是断网闸装饰器（codex 蒸馏 B6 折算，第四批 P8）。
//
// 越权拒绝的落点：[sandbox] network=false 时，网络能力工具（web/浏览器/
// MCP 动态工具）在闸门层拒绝并给出可读理由——此前断网只关 bash 子进程
// 出网，web/浏览器/MCP 照常出网是「半开沙箱」。bash 不拒（它有自己的
// 子进程网络开关，整体拒绝会消灭「断网但本地命令可用」形态）。
//
// 逐主机修正案（codex 蒸馏差距档案刀1，NetworkPolicyAmendment 折算）：
// Amendment 非空且 Enabled() 时，被拦调用先查台账（会话豁免+持久清单，
// host 精确/子域后缀匹配，亦可记工具名），未命中弹卡四选（本次/本会话/
// 永久此主机/拒绝）。「永久」由 Persist 写 [network].gate_host_allow。
// 开关关或 Amendment 为 nil = 拦截行为与现状逐字节等价。
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// NetworkLedger 是修正案的只读台账面（headless 路径只需要这一半）。
type NetworkLedger interface {
	// Allowed 报告工具名或目标（host）是否已被豁免/列入持久台账。
	Allowed(toolName, target string) bool
}

// NetworkAmendment 在台账之上叠加弹卡与持久化（交互路径）。Enabled=false
// 时闸门跳过弹卡直接拒（现状语义）。
type NetworkAmendment interface {
	NetworkLedger
	Enabled() bool
	// Ask 弹卡询问；返回 (放行, 是否要求持久化)。拒绝/超时/取消都归 (false,false)。
	Ask(ctx context.Context, toolName, target string) (allowed bool, persist bool)
	// Persist 把条目写入持久台账；失败不影响本次放行（会话豁免已生效）。
	Persist(toolOrHost string) error
}

// NetworkGate 在内层闸门前叠加断网拒绝（Allow=true 时零介入）。
type NetworkGate struct {
	Inner     Gate
	Allow     bool
	Amendment NetworkAmendment
}

// WrapNetworkGate 按开关系断网闸：allowNetwork=true 或内层为 nil 时原样
// 返回（现状零变化）。
func WrapNetworkGate(inner Gate, allowNetwork bool) Gate {
	return WrapNetworkGateWith(inner, allowNetwork, nil)
}

// WrapNetworkGateWith 同 WrapNetworkGate 并挂修正案（amend 为 nil = 现状）。
func WrapNetworkGateWith(inner Gate, allowNetwork bool, amend NetworkAmendment) Gate {
	if inner == nil || allowNetwork {
		return inner
	}
	return &NetworkGate{Inner: inner, Allow: false, Amendment: amend}
}

// Check 先判网络能力（拒绝文案可操作），再走修正案台账/弹卡，最后内层闸门。
// 台账放行与弹卡放行都继续走内层闸——deny 规则等内层语义不被修正案盖过。
func (g *NetworkGate) Check(ctx context.Context, toolName string, args json.RawMessage, readOnly bool) (bool, string, error) {
	if !g.Allow && tool.NetworkCapable(toolName) {
		if g.Amendment != nil {
			target := amendmentTarget(toolName, args)
			if g.Amendment.Allowed(toolName, target) {
				return g.Inner.Check(ctx, toolName, args, readOnly)
			}
			if g.Amendment.Enabled() {
				allowed, persist := g.Amendment.Ask(ctx, toolName, target)
				if allowed {
					if persist {
						_ = g.Amendment.Persist(target) // 失败只丢持久化，会话豁免已生效
					}
					return g.Inner.Check(ctx, toolName, args, readOnly)
				}
				return false, fmt.Sprintf(
					"tool %s blocked: sandbox network is disabled and the host amendment for %q was declined — enable [sandbox] network or [network] gate_amendment to change this policy",
					toolName, target), nil
			}
		}
		return false, fmt.Sprintf(
			"tool %s blocked: sandbox network is disabled ([sandbox] network=false) — network tools (web/browser/mcp) are refused in this session; local tools still work",
			toolName), nil
	}
	return g.Inner.Check(ctx, toolName, args, readOnly)
}

// NetworkLedgerOnly 把一份静态清单（[network].gate_host_allow 快照）包成
// NetworkAmendment：只读台账、Enabled 恒 false（headless/无人值守路径——
// 没有可弹卡的 UI，清单命中放行，其余照旧拒绝）。
func NetworkLedgerOnly(entries []string) NetworkAmendment {
	return staticLedger(entries)
}

type staticLedger []string

func (s staticLedger) Allowed(toolName, target string) bool {
	for _, e := range s {
		if e == toolName || (target != "" && (target == e || strings.HasSuffix(target, "."+e))) {
			return true
		}
	}
	return false
}

func (s staticLedger) Enabled() bool { return false }

func (s staticLedger) Ask(context.Context, string, string) (bool, bool) { return false, false }

func (s staticLedger) Persist(string) error { return nil }

// amendmentTarget 提取修正案条目：args 的 url 系 subject → host（复用
// permission.Subject 的键位优先表）；无 url/解析不出 host 时退化为工具名级
// 条目（MCP 无参工具走这条）。
func amendmentTarget(toolName string, args json.RawMessage) string {
	subject := permission.Subject(args)
	if subject == "" {
		return toolName
	}
	if u, err := url.Parse(subject); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return toolName
}
