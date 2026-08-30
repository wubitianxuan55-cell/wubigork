package control

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/permission"
)

// --- approval bridge (agent gate → events) ---

// hardAskTools 是必须逐条经用户确认的工具：写入成本库 / 记忆 / 知识库等
// 持久化数据，任何权限级别（含 yolo）都不自动放行，且不记忆会话放行。
var hardAskTools = map[string]bool{
	"cost_save":             true,
	"remember":              true,
	"forget":                true,
	"knowledge_add":         true,
	"promote_session_facts": true,
	"install_skill":         true,
}

// gateApprover adapts the Controller to permission.Approver. It is distinct
// from the public Approve command (different signature, different direction).
type gateApprover struct{ c *Controller }

func (g gateApprover) Approve(ctx context.Context, tool, subject string, args json.RawMessage) (bool, bool, error) {
	// Auto-allow without prompting while executing a just-approved plan (the plan
	// was the approval) or while YOLO/bypass mode is on. Deny rules already bit
	// before this point, so they still block.
	g.c.mu.Lock()
	auto := g.c.autoApprove || g.c.permLevel != "ask"
	g.c.mu.Unlock()
	// 持久化写入（成本库/记忆/知识库）必须逐条经用户确认：
	// auto/yolo 权限级别也强制询问，且不记忆会话放行。集合按空间策略
	// 参数化（S1.5-A：play 产品默认空集 = 不弹审批卡；默认集 = 包级 hardAskTools）。
	if g.c.hardAskSet()[tool] {
		return g.c.requestApproval(ctx, tool, approvalSubjectFor(tool, args), true)
	}
	if auto {
		return true, false, nil
	}
	return g.c.requestApproval(ctx, tool, subject, false)
}

// approvalSubjectFor 为硬性审批工具生成可读的确认摘要。
func approvalSubjectFor(tool string, args json.RawMessage) string {
	switch tool {
	case "cost_save":
		return costSaveApprovalSubject(args)
	case "remember":
		return rememberApprovalSubject(args)
	case "knowledge_add":
		return knowledgeAddApprovalSubject(args)
	case "promote_session_facts":
		return "把本次会话沉淀的临时事实提升为永久记忆（跨会话自动加载）"
	}
	return permission.Subject(args)
}

// approvalPrompt 参数化一次审批弹卡：常规工具闸门（gateApprover）与
// request_permission 权限升级申请共用同一条 ApprovalRequest 事件通道、
// 同一个 Approve(ID, decision) 应答与同一套超时/取消语义。
type approvalPrompt struct {
	tool    string
	subject string
	// reason 是权限升级申请的理由（仅 ruleRequest 携带），原样展示在审批卡。
	reason string
	// alwaysPrompt：hardAsk 逐条确认——不读不写 granted、不回写策略，批准
	// 仅对本次放行生效（任何级别都不自动放行的硬纪律）。
	alwaysPrompt bool
	// ruleRequest：request_permission 的「规则申请」——申请本身就是要求一条
	// 规则，因此 allow_once 与 allow_session 同效（写入会话 granted，否则
	// 批准没有任何可落地的效果）；事件带 Request 标记与 reason 供前端展示。
	ruleRequest bool
}

// requestApproval emits an ApprovalRequest and blocks until Approve(ID, …)
// answers or ctx is cancelled. A prior session grant for the same tool+subject
// short-circuits, unless alwaysPrompt is set (敏感写入必须逐条确认）。
// promptMu serialises outstanding prompts.
func (c *Controller) requestApproval(ctx context.Context, tool, subject string, alwaysPrompt bool) (bool, bool, error) {
	allow, remember, _, err := c.promptApproval(ctx, approvalPrompt{tool: tool, subject: subject, alwaysPrompt: alwaysPrompt})
	return allow, remember, err
}

// RequestPermission 处理 request_permission 工具的「权限升级申请」（对齐 codex
// request_permissions_for_environment 语义族）：把一条规则申请（"Tool" /
// "Tool(subject-glob)"）连同模型给出的 reason 投递到既有审批卡通道并阻塞等待。
//
// 决策语义：allow_once / allow_session → 写入会话 granted（本会话内规则生效，
// 后续真实调用仍走正常闸门、规则满足则自然放行）；persist_allow → 同上并把
// 规则经 PersistAllowRule 回写策略文件；deny → 记拒绝、回合继续；abort →
// 拒绝并终止本轮；审批超时无人响应按既有语义拒绝（decision=timeout）。
//
// 硬纪律：hardAsk 逐条确认工具不接受升级申请（refused_hardask，任何级别都不
// 自动放行）；deny 规则命中的目标不弹卡直接拒绝（refused_deny_rule——批准的
// 规则盖不过 Decide 的 Deny 优先级，弹卡只会误导）；未注册工具拒绝
// （refused_unknown_tool）；auto/yolo 级别下真实调用本就不再询问，规则直接
// 按会话生效（decision=auto），不打扰用户。
//
// 返回 granted 与 decision 串；err 仅在 ctx 取消（回合终止）时非 nil。
func (c *Controller) RequestPermission(ctx context.Context, tool, subject, reason string) (bool, string, error) {
	subject = strings.TrimSpace(subject)
	// 硬纪律 1：hardAsk（逐条确认）工具不接受升级申请——它们的功能形态就是
	// 每条写操作单独确认，一条「规则」无法表达逐条纪律。
	if c.hardAskSet()[tool] {
		return false, "refused_hardask", nil
	}
	// 硬纪律 2：deny 规则任何模式硬拒——升级申请不能成为绕过面。
	if c.policy.Denies(tool, subject) {
		return false, "refused_deny_rule", nil
	}
	// 硬纪律 3：申请的目标必须是注册表里的真实工具，否则规则永远空转。
	if c.reg != nil {
		if _, ok := c.reg.Get(tool); !ok {
			return false, "refused_unknown_tool", nil
		}
	}
	// auto / yolo：真实调用本就不再询问（gateApprover 的 auto 短路），直接把
	// 规则记入会话规则表，不打扰用户。deny 硬拒与 hardAsk 逐条确认不受影响
	// （前者已在上方拦截，后者 granted 之外的路径恒弹卡）。
	c.mu.Lock()
	auto := c.autoApprove || c.permLevel != "ask"
	if auto {
		c.grantRuleLocked(tool, subject)
	}
	c.mu.Unlock()
	if auto {
		return true, "auto", nil
	}
	_, _, decision, err := c.promptApproval(ctx, approvalPrompt{
		tool: tool, subject: subject, reason: reason, ruleRequest: true,
	})
	if err != nil {
		return false, "", err
	}
	return decision == DecisionAllowOnce || decision == DecisionAllowSession ||
		decision == DecisionPersistAllow, decision, nil
}

// ruleGrantedLocked 报告会话内已有授权覆盖 (tool, subject)：精确 key 的
// granted（gate allow_session 记忆）或 grantedRules 中一条 glob 规则
// （request_permission 批准，"Tool" / "Tool(subject-glob)"，语义与策略文件
// 一致）。调用方必须持有 c.mu。
func (c *Controller) ruleGrantedLocked(tool, subject string) bool {
	if c.granted[tool+"\x00"+subject] {
		return true
	}
	for _, r := range c.grantedRules {
		if permission.RuleMatches(permission.Rule{Tool: r.tool, Subject: r.subject}, tool, subject) {
			return true
		}
	}
	return false
}

// grantRuleLocked 记录一条会话级规则（request_permission 批准）。subject
// 保留申请时的 glob 原文；subject 为空表示整工具。调用方必须持有 c.mu。
func (c *Controller) grantRuleLocked(tool, subject string) {
	c.grantedRules = append(c.grantedRules, grantedRule{tool: tool, subject: subject})
}

// promptApproval emits one approval prompt per p and blocks until Approve(ID, …)
// answers, ctx is cancelled, or the configured approval timeout fires. It returns
// whether the call may proceed, whether a session grant was already in effect
// (the second bool of the legacy requestApproval signature), and the decision
// string (DecisionTimeout when nobody answered in time; "" when ctx cancelled).
func (c *Controller) promptApproval(ctx context.Context, p approvalPrompt) (allow, remember bool, decision string, err error) {
	key := p.tool + "\x00" + p.subject

	if !p.alwaysPrompt {
		c.mu.Lock()
		grantedAlready := c.ruleGrantedLocked(p.tool, p.subject)
		c.mu.Unlock()
		if grantedAlready {
			return true, true, DecisionAllowSession, nil // session grant was previously stored
		}
	}

	c.promptMu.Lock()
	defer c.promptMu.Unlock()

	// Re-check the grant: a session grant may have landed while we queued behind
	// another prompt for the same subject.
	c.mu.Lock()
	if !p.alwaysPrompt {
		if c.ruleGrantedLocked(p.tool, p.subject) {
			c.mu.Unlock()
			return true, true, DecisionAllowSession, nil // session grant stored while waiting
		}
	}
	c.nextID++
	id := strconv.Itoa(c.nextID)
	reply := make(chan approvalReply, 1)
	c.approvals[id] = reply
	c.mu.Unlock()

	c.sink.Emit(event.Event{Kind: event.ApprovalRequest, Approval: event.Approval{
		ID: id, Tool: p.tool, Subject: p.subject,
		Reason: p.reason, Request: p.ruleRequest,
	}})
	// The agent now needs the user's attention; a Notification hook can ping an
	// external channel (desktop notice, phone) while the run blocks on the reply.
	switch {
	case p.ruleRequest:
		go c.hooks.Notification(ctx, "permission requested: "+p.tool+" "+p.subject)
	case p.subject != "":
		go c.hooks.Notification(ctx, "approval needed: "+p.tool+" "+p.subject)
	default:
		go c.hooks.Notification(ctx, "approval needed: "+p.tool)
	}

	// C4 TimedOut（蒸馏 codex ReviewDecision::TimedOut）：配置了审批超时时，
	// 无人响应不永久阻塞——超时按拒绝处理（回合继续、工具结果记拒绝，不静默
	// 放行），并发 Notice 让用户回来能看到哪些步骤被超时拒绝。
	var timeoutCh <-chan time.Time
	if c.approvalTimeout > 0 {
		timer := time.NewTimer(c.approvalTimeout)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	select {
	case r := <-reply:
		switch r.decision {
		case DecisionAbort:
			// 拒绝并终止本轮（codex ReviewDecision::Abort）：取消当前回合，
			// agent 循环在下一步采样/闸门检查处快速收敛退出；工具结果按
			// 拒绝落日志（「拒绝但继续」由 deny 表达，两者语义分离）。
			c.Cancel()
			return false, false, DecisionAbort, nil
		case DecisionPersistAllow:
			// 始终允许（codex ApprovedExecpolicyAmendment）：本会话内先记
			// granted（与 allow_session 同效），再把规则回写策略文件持久化，
			// 重启后由 [permissions].allow 直接放行。hardAsk 持久化写入（
			// alwaysPrompt）完全降级为 allow_session——不记 granted、不回写
			// （任何级别都不自动放行的硬纪律，frontend 也不渲染该按钮）。
			// 规则申请：记入 grantedRules（保留 glob 语义）而非精确 key。
			if !p.alwaysPrompt {
				c.mu.Lock()
				if p.ruleRequest {
					c.grantRuleLocked(p.tool, p.subject)
				} else {
					c.granted[key] = true
				}
				c.mu.Unlock()
				c.writebackAllowRule(p.tool, p.subject)
			}
			return true, false, DecisionPersistAllow, nil
		case DecisionAllowSession:
			if !p.alwaysPrompt {
				c.mu.Lock()
				if p.ruleRequest {
					c.grantRuleLocked(p.tool, p.subject)
				} else {
					c.granted[key] = true
				}
				c.mu.Unlock()
			}
			return true, false, DecisionAllowSession, nil
		case DecisionAllowOnce:
			// 常规闸门：允许一次不记忆（该次调用放行即完成语义）。规则申请：
			// 申请的对象本身就是一条规则——allow_once 与 allow_session 同效
			// （写入会话 granted），否则「允许一次」没有任何可落地的效果。
			if p.ruleRequest && !p.alwaysPrompt {
				c.mu.Lock()
				c.grantRuleLocked(p.tool, p.subject)
				c.mu.Unlock()
			}
			return true, false, DecisionAllowOnce, nil
		default:
			// deny（含未知决策串的保守处理）：拒绝但回合继续。
			return false, false, DecisionDeny, nil
		}
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.approvals, id)
		c.mu.Unlock()
		return false, false, "", ctx.Err()
	case <-timeoutCh:
		c.mu.Lock()
		delete(c.approvals, id)
		c.mu.Unlock()
		if p.subject != "" {
			c.notice(fmt.Sprintf("审批超时（%s 未响应），已按拒绝处理：%s %s",
				c.approvalTimeout, p.tool, p.subject))
		} else {
			c.notice(fmt.Sprintf("审批超时（%s 未响应），已按拒绝处理：%s", c.approvalTimeout, p.tool))
		}
		return false, false, DecisionTimeout, nil
	}
}

// writebackAllowRule 把 "ToolName" / "ToolName(subject)" 规则经回调回写策略
// 文件（Options.PersistAllowRule）。失败仅记日志——批准本身已生效（会话
// granted），持久化是增强而非依赖。不持 c.mu（回调做文件 I/O）。
func (c *Controller) writebackAllowRule(tool, subject string) {
	if c.persistAllowRule == nil {
		return
	}
	rule := tool
	if subject != "" {
		rule = tool + "(" + subject + ")"
	}
	if err := c.persistAllowRule(rule); err != nil {
		slog.Warn("approval: 策略回写失败（批准仍在本会话生效）", "rule", rule, "error", err)
		return
	}
	slog.Info("approval: 允许规则已回写策略文件", "rule", rule)
}

// costSaveApprovalSubject 把 cost_save 的参数整理成可读的确认摘要，
// 让审批卡显示要写入成本库的条目名称、单价、单位、规格与来源。
func costSaveApprovalSubject(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var p struct {
		Title    string  `json:"title"`
		Price    float64 `json:"price"`
		Unit     string  `json:"unit"`
		Spec     string  `json:"spec"`
		Source   string  `json:"source"`
		Category string  `json:"category"`
		Status   string  `json:"status"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ""
	}
	if strings.TrimSpace(p.Title) == "" {
		return ""
	}
	parts := []string{"写入成本库：" + strings.TrimSpace(p.Title)}
	parts = append(parts, fmt.Sprintf("单价 ¥%.2f", p.Price))
	if s := strings.TrimSpace(p.Unit); s != "" {
		parts = append(parts, "单位 "+s)
	}
	if s := strings.TrimSpace(p.Spec); s != "" {
		parts = append(parts, "规格 "+s)
	}
	if s := strings.TrimSpace(p.Category); s != "" {
		parts = append(parts, "分类 "+s)
	}
	if s := strings.TrimSpace(p.Source); s != "" {
		parts = append(parts, "来源 "+s)
	}
	if s := strings.TrimSpace(p.Status); s != "" {
		parts = append(parts, "状态 "+s)
	}
	return strings.Join(parts, " · ")
}

// rememberApprovalSubject 展示将要写入记忆的条目（名称/描述/类型/是否仅会话）。
func rememberApprovalSubject(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var p struct {
		Name        string `json:"name"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Session     bool   `json:"session"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ""
	}
	label := strings.TrimSpace(p.Title)
	if label == "" {
		label = strings.TrimSpace(p.Name)
	}
	if label == "" {
		label = strings.TrimSpace(p.Description)
	}
	if label == "" {
		return ""
	}
	scope := "永久记忆"
	if p.Session {
		scope = "仅本次会话"
	}
	parts := []string{"写入" + scope + "：" + label}
	if s := strings.TrimSpace(p.Description); s != "" && s != label {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(p.Type); s != "" {
		parts = append(parts, "类型 "+s)
	}
	return strings.Join(parts, " · ")
}

// knowledgeAddApprovalSubject 展示将要写入知识库的条目。
func knowledgeAddApprovalSubject(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var p struct {
		Title    string `json:"title"`
		Category string `json:"category"`
		Body     string `json:"body"`
		Tags     string `json:"tags"`
		Source   string `json:"source"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ""
	}
	if strings.TrimSpace(p.Title) == "" {
		return ""
	}
	parts := []string{"写入知识库：" + strings.TrimSpace(p.Title)}
	if s := strings.TrimSpace(p.Category); s != "" {
		parts = append(parts, "分类 "+s)
	}
	if s := strings.TrimSpace(p.Source); s != "" {
		parts = append(parts, "来源 "+s)
	}
	if s := strings.TrimSpace(p.Tags); s != "" {
		parts = append(parts, "标签 "+s)
	}
	if body := strings.TrimSpace(p.Body); body != "" {
		one := strings.SplitN(body, "\n", 2)[0]
		if r := []rune(one); len(r) > 80 {
			one = string(r[:80]) + "…"
		}
		parts = append(parts, one)
	}
	return strings.Join(parts, " · ")
}
