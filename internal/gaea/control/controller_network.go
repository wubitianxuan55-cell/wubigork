// controller_network.go 是断网闸逐主机修正案的 controller 侧实现（codex
// 蒸馏差距档案刀1，codex NetworkPolicyAmendment 折算）：[sandbox]
// network=false 拦截的网络工具，弹卡四选（本次/本会话/永久此主机/拒绝），
// 「永久」经 PersistNetworkHost 回写 [network].gate_host_allow。台账匹配
// 沿用出网许可层约定：条目 == 目标 或目标是条目的子域（"."+entry 后缀）。
package control

import (
	"context"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
)

// networkAmender 实现 agent.NetworkAmendment。闭包持 controller，闸门层
// 不反向依赖 control。
type networkAmender struct{ c *Controller }

var (
	_ agent.NetworkAmendment = networkAmender{}
	_ agent.NetworkLedger    = networkHeadlessLedger{}
)

// wrapNetworkGate 是四个换级点的统一断网闸包装（P8 出口纪律 + 刀1 修正案）：
// 交互路径挂 controller 修正案（开关关时内部仍直接拒，行为逐字节等价）。
func (c *Controller) wrapNetworkGate(inner agent.Gate) agent.Gate {
	return agent.WrapNetworkGateWith(inner, !c.denyNetworkTools, networkAmender{c})
}

// matchEntry 台账匹配：条目 == 目标，或目标是条目的子域（".entry" 后缀），
// 与 netclient OutboundGuard 的白名单口径一致。工具名条目走精确相等。
func matchEntry(entry, target string) bool {
	return entry == target || strings.HasSuffix(target, "."+entry)
}

// Allowed 实现 agent.NetworkLedger：会话豁免（once+session）→ 持久台账。
func (n networkAmender) Allowed(toolName, target string) bool {
	c := n.c
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range []map[string]bool{c.networkExemptSession, c.networkExemptOnce} {
		if m[toolName] || (target != "" && m[target]) {
			return true
		}
	}
	for _, e := range c.networkHostAllow {
		if e == toolName || (target != "" && matchEntry(e, target)) {
			return true
		}
	}
	return false
}

// Enabled 实现 agent.NetworkAmendment：[network].gate_amendment。
func (n networkAmender) Enabled() bool { return n.c.networkAmendment }

// Ask 实现 agent.NetworkAmendment：复用既有审批卡通道（Source="gate"，
// 决策串不新增——persist_allow 在本语境 = 永久此主机）。弹卡走 promptApproval
// 直连（需要拿 decision 原值分流 once/session/persist）。denied/timeout 照常
// 挂待追认账本（recordDeniedAction 在 promptApproval 内已做），拒绝理由随
// 既有 T3 通道回喂。
func (n networkAmender) Ask(ctx context.Context, toolName, target string) (bool, bool) {
	c := n.c
	res := c.promptApproval(ctx, approvalPrompt{
		tool:    toolName,
		subject: target,
		source:  event.ApprovalSourceGate,
		reason:  fmt.Sprintf("sandbox network is disabled; allow %s → %s?", toolName, target),
	})
	if res.err != nil {
		return false, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	switch res.decision {
	case DecisionAllowOnce:
		if target != "" {
			c.networkExemptOnce[target] = true
		} else {
			c.networkExemptOnce[toolName] = true
		}
		return true, false
	case DecisionAllowSession, DecisionPersistAllow:
		// persist_allow 先给会话豁免；持久化由闸门回调 Persist（失败仅丢配置回写）。
		if target != "" {
			c.networkExemptSession[target] = true
		} else {
			c.networkExemptSession[toolName] = true
		}
		return true, res.decision == DecisionPersistAllow
	default:
		return false, false
	}
}

// Persist 实现 agent.NetworkAmendment：Load→追加→Save（与 PersistAllowRule
// 的 boot 回写同款）。回调 nil = 不支持持久化（退化为会话豁免，不报错）。
func (n networkAmender) Persist(toolOrHost string) error {
	if n.c.persistNetworkHost == nil {
		return nil
	}
	return n.c.persistNetworkHost(toolOrHost)
}

// networkHeadlessLedger 是 headless/无人值守路径的只读台账：没有 UI 可弹卡，
// 只认持久清单（Ask 永不调用——它没实现 NetworkAmendment 的弹卡半边）。
type networkHeadlessLedger struct{ allow []string }

func (h networkHeadlessLedger) Allowed(toolName, target string) bool {
	for _, e := range h.allow {
		if e == toolName || (target != "" && matchEntry(e, target)) {
			return true
		}
	}
	return false
}
