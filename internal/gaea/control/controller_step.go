// controller_step.go 是回合设置快照与受限更新（codex 蒸馏 B4，第四批 P5）。
//
// 机制同构 codex 的「step 设置是不可变快照 + 运行中受限更新」：回合开始时
// 把会撕裂回合的设置钉住，运行中变更被拒并给出可读理由——当前回合按它
// 被准入时的权威跑完，不半路变卦。
//
// 拍板（2026-11-17）：**权限算快照维度**。gaea 的禁止清单三档折算：
//   - 审批政策：SetPermLevel（活的 gate/policy 热切换面）；
//   - 模型：SetContextWindow（模型切换链的回落面；换引擎主体走重建，
//     重建在回合外）；
//   - 沙箱：AccessPolicy 构造期固定，无运行中变更面——无可守护项（记档）。
//
// 三态折算：Applied（回合外，立即生效）/ Rejected（回合内+维度在禁止清单，
// 附理由）。codex 的 TargetUnavailable 在 gaea 无对应面（controller 重建后
// 旧引用整体失效，不存在「更新打到旧目标」的窗口）。
package control

import (
	"fmt"

	"github.com/gaea/gaea/internal/gaea/permission"
)

// StepUpdateStatus 是受限更新的结果态。
type StepUpdateStatus string

const (
	// StepApplied：已生效（回合外路径，行为与改动前完全一致）。
	StepApplied StepUpdateStatus = "applied"
	// StepRejected：回合内且维度在禁止清单——拒绝并附理由，回合结束后重试。
	StepRejected StepUpdateStatus = "rejected"
)

// turnStepSnapshot 在回合启动时记录准入快照（permLevel）。由 mu 保护；
// 回合收尾清空。PermLevel 本身受 SetPermLevelReported 守护、回合内不会变，
// 快照是「本回合按什么权威被准入」的显式证据（审计与测试锚）。
type stepSnapshot struct {
	permLevel string
}

// SetPermLevelReported 是带三态回报的权限级别切换（P5）：回合进行中拒绝
// ——运行中的回合按回合开始时的审批权威跑完，不半路换门。
//
// 现有 SetPermLevel（无回报）保持签名不变、内部转调本方法：slash 路径
// （/perm）本就只在回合外派发；UI 双板路径应转调带回报版本把拒绝理由
// 透给用户。
func (c *Controller) SetPermLevelReported(level string) (StepUpdateStatus, string) {
	c.mu.Lock()
	if c.running {
		current := c.turnStep.permLevel
		c.mu.Unlock()
		return StepRejected, fmt.Sprintf(
			"回合进行中，审批级别切换被拒（当前回合按开始时的 %q 级别跑完，不会半路换门）；请等回合结束后再切到 %q",
			current, level)
	}
	c.applyPermLevelLocked(level)
	c.mu.Unlock()
	return StepApplied, ""
}

// applyPermLevelLocked 执行既有切换逻辑（policy.Mode + executor gate 重建）。
// 调用方必须已持 c.mu 且已确认 !running。
func (c *Controller) applyPermLevelLocked(level string) {
	c.permLevel = level
	// 持久化写入（记忆/方案库/造价库）必须逐条确认：
	// 任何权限级别都保留 AlwaysAsk 硬门（按空间策略参数化，S1.5-A）。
	switch level {
	case "auto":
		c.policy.Mode = permission.Allow
		if c.executor != nil {
			c.executor.SetGate(c.wrapNetworkGate(c.newGate()))
		}
	case "yolo":
		if c.executor != nil {
			// yolo 保持"跳过一切门禁"，但持久化写入例外仍必须确认：
			// 用空策略 + AlwaysAsk 的 gate 替代 nil gate。
			g := c.newGate()
			g.Policy = permission.New("allow", nil, nil, nil)
			c.executor.SetGate(c.wrapNetworkGate(g))
		}
	default: // "ask"
		c.policy.Mode = permission.Ask
		if c.executor != nil {
			c.executor.SetGate(c.wrapNetworkGate(c.newGate()))
		}
	}
}

// SetContextWindowReported 是带三态回报的上下文窗口同步（P5 模型维度）：
// 模型切换链的最后一跳，回合内执行会撕裂在跑回合的压缩判定——拒绝并给
// 理由（切换的其余步骤——默认引擎落盘——不撕裂回合，照常执行；调用方
// 收到 Rejected 时应提示重试窗口同步）。
func (c *Controller) SetContextWindowReported(window int) (StepUpdateStatus, string) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return StepRejected, "回合进行中，模型窗口同步被拒（在跑回合的压缩判定按旧窗口跑完）；请等回合结束后重试切换"
	}
	exec := c.executor
	c.mu.Unlock()
	if exec != nil {
		exec.SetContextWindow(window)
	}
	return StepApplied, ""
}

// PermLevelSnapshot 返回在跑回合的准入权限快照；空串=当前无回合。
func (c *Controller) PermLevelSnapshot() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return ""
	}
	return c.turnStep.permLevel
}

// boolTag 是观测维度的布尔串形态（P7，与 agent 包同口径两值形态）。
func boolTag(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
