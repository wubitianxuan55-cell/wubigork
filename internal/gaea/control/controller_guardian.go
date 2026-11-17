package control

// Guardian 自动评审接线（codex 蒸馏 T11 刀三）：常规 Ask 通道用户卡之前的
// 机器预判。三条路径——
//   - allow：放行 + guardian 账本（risk/authorization/rationale 三字段）；
//   - deny：带理由拒绝 + 落 T3 待追认账本（用户可 ApproveDenied 重放），
//     连续 3 次拒绝触发熔断（中断由本控制器执行，评审器只计数）；
//   - 评审失败（超时/预算/解析/调用）：回落用户审批卡——红线，错误绝不
//     冒充风险，也绝不因失败自动放行。

import (
	"context"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/guardian"
)

// guardianPath 是 guardianReview 的结论路径（调用方据此分流）。
type guardianPath int

const (
	// guardianFallthrough = 评审未决（关闭/失败/非适用通道）→ 走用户卡。
	guardianFallthrough guardianPath = iota
	// guardianAllowPath = 评审放行。
	guardianAllowPath
	// guardianDenyPath = 评审拒绝（denial 文案已带回）。
	guardianDenyPath
)

// guardianReview 跑一次自动评审。decided=false 表示回落用户流程。
// 调用点：gateApprover 常规 Ask 通道（hardAsk/project/auto 通道不进来）。
func (c *Controller) guardianReview(ctx context.Context, tool, subject string, args []byte) (path guardianPath, denial string) {
	c.mu.Lock()
	rev := c.guardian
	userInput := c.lastUserInput
	c.mu.Unlock()
	if rev == nil {
		return guardianFallthrough, ""
	}
	res := rev.Review(ctx, guardian.Request{
		Tool: tool, Subject: subject, Args: args, UserInstruction: userInput,
	})
	if res.Err != nil {
		// 失败分类透明上报，但不产生拒绝账本（asset-07 §四）——回落用户卡。
		c.notice(fmt.Sprintf("自动评审未完成（%v），已转人工审批", res.Err))
		c.mu.Lock()
		c.guardianBreaker.RecordAllow() // 连续性打断：失败不是拒绝
		c.mu.Unlock()
		return guardianFallthrough, ""
	}
	if res.Assessment.Outcome == guardian.OutcomeAllow {
		c.mu.Lock()
		c.guardianBreaker.RecordAllow()
		c.mu.Unlock()
		c.emitGuardianDecision(tool, subject, event.AutoDecidedGuardianAllow, res.Assessment)
		return guardianAllowPath, ""
	}
	// deny：三字段账本 + 待追认挂账 + 熔断计数（触发即中断回合，宿主执行）。
	c.emitGuardianDecision(tool, subject, event.AutoDecidedGuardianDeny, res.Assessment)
	c.mu.Lock()
	tripped := c.guardianBreaker.RecordDenial()
	c.nextID++ // 追认键：guardian 拒绝没有审批卡 id，用同一序列造唯一键
	gid := fmt.Sprintf("g%d", c.nextID)
	c.mu.Unlock()
	c.recordDeniedAction(gid, approvalPrompt{
		tool: tool, subject: subject, source: event.ApprovalSourceGuardian,
	}, "deny", res.Assessment.Rationale)
	if tripped {
		c.notice(fmt.Sprintf("自动评审连续 %d 次拒绝，已中断本轮——请检查任务方向或追认具体动作", guardian.MaxConsecutiveDenials))
		c.Cancel() // 熔断中断由宿主执行（asset-07 §五职责分界）
	}
	return guardianDenyPath, "自动评审拒绝：" + res.Assessment.Rationale +
		"——不要原样重试；换方案、结束回合说明，或请用户追认该动作"
}

// emitGuardianDecision 落 guardian 三字段账本（ApprovalAuto，Source=guardian）。
func (c *Controller) emitGuardianDecision(tool, subject, decided string, a guardian.Assessment) {
	if c.sink == nil {
		return
	}
	c.sink.Emit(event.Event{Kind: event.ApprovalAuto, Auto: event.AutoApproval{
		Tool: tool, Subject: subject,
		Source:        event.ApprovalSourceGuardian,
		Decided:       decided,
		Risk:          string(a.Risk),
		Authorization: string(a.Authorization),
		Rationale:     a.Rationale,
	}})
}
