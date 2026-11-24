package control

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/schedule"
)

// --- approval bridge (agent gate → events) ---

// proposalSaveApprovalSubject 展示将要写入方案库的方案：标题、类型、年度，以及
// 本次是新建、整册覆盖还是续录第几批、本批多少个标题节点、取自哪个源文件。
func proposalSaveApprovalSubject(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var p struct {
		Title       string `json:"title"`
		ProjectType string `json:"project_type"`
		Year        string `json:"year"`
		Mode        string `json:"mode"`
		Overwrite   bool   `json:"overwrite"`
		DryRun      bool   `json:"dry_run"`
		SourceFile  string `json:"source_file"`
		Sections    []struct {
			Level int `json:"level"`
		} `json:"sections"`
		Outline []struct {
			Level int `json:"level"`
		} `json:"outline"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ""
	}
	if strings.TrimSpace(p.Title) == "" {
		return ""
	}
	n := len(p.Outline)
	if n == 0 {
		n = len(p.Sections)
	}
	action := "新建"
	switch {
	case p.DryRun:
		action = "演练，不写库"
	case p.Overwrite:
		action = "整册覆盖"
	case strings.EqualFold(strings.TrimSpace(p.Mode), "append"):
		action = "续录"
	}
	parts := []string{"写入方案库（" + action + "）：" + strings.TrimSpace(p.Title)}
	if n > 0 {
		parts = append(parts, fmt.Sprintf("本批 %d 节", n))
	}
	if s := strings.TrimSpace(p.ProjectType); s != "" {
		parts = append(parts, "类型 "+s)
	}
	if s := strings.TrimSpace(p.Year); s != "" {
		parts = append(parts, "年度 "+s)
	}
	if s := strings.TrimSpace(p.SourceFile); s != "" {
		parts = append(parts, "源文件 "+filepath.Base(s))
	}
	return strings.Join(parts, " · ")
}

// proposalUpdateApprovalSubject 展示将要修改的方案与改哪一类：元数据（列出改成的标题）
// 还是某一小节（旧标题 → 新标题 / 整节替换正文）。
func proposalUpdateApprovalSubject(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var p struct {
		Proposal     string  `json:"proposal"`
		Title        string  `json:"title"`
		ProjectType  string  `json:"project_type"`
		Year         string  `json:"year"`
		Summary      string  `json:"summary"`
		Workspace    string  `json:"workspace"`
		SectionID    int64   `json:"section_id"`
		SectionTitle *string `json:"section_title"`
		Body         *string `json:"body"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Proposal) == "" {
		return ""
	}
	parts := []string{"修改方案库方案：" + strings.TrimSpace(p.Proposal)}
	if t := strings.TrimSpace(p.Title); t != "" {
		parts = append(parts, "改标题为「"+t+"」")
	}
	var fields []string
	for _, f := range []struct{ name, val string }{
		{"类型", p.ProjectType}, {"年度", p.Year}, {"摘要", p.Summary}, {"工作区", p.Workspace},
	} {
		if strings.TrimSpace(f.val) != "" {
			fields = append(fields, f.name)
		}
	}
	if len(fields) > 0 {
		parts = append(parts, "改"+strings.Join(fields, "/"))
	}
	if p.SectionID > 0 {
		what := fmt.Sprintf("修正小节 #%d", p.SectionID)
		if p.SectionTitle != nil {
			what += "标题→「" + strings.TrimSpace(*p.SectionTitle) + "」"
		}
		if p.Body != nil {
			what += "整节替换正文"
		}
		parts = append(parts, what)
	}
	return strings.Join(parts, " · ")
}

// proposalDeleteApprovalSubject 展示将要删除的方案。
func proposalDeleteApprovalSubject(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var p struct {
		Proposal string `json:"proposal"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Proposal) == "" {
		return ""
	}
	s := "从方案库删除（不可恢复）：" + strings.TrimSpace(p.Proposal)
	if r := strings.TrimSpace(p.Reason); r != "" {
		s += " · 理由：" + r
	}
	return s
}

// proposalArgsDryRun 方案库演练调用（dry_run=true）：只校验不写库，不必走逐条确认。
func proposalArgsDryRun(args json.RawMessage) bool {
	var p struct {
		DryRun  bool   `json:"dry_run"`
		Action  string `json:"action"`
		Outline []any  `json:"outline"`
	}
	return json.Unmarshal(args, &p) == nil && p.DryRun && strings.TrimSpace(p.Action) == ""
}

// hardAskTools 是必须逐条经用户确认的工具：写入成本库 / 记忆 / 方案库等
// 持久化数据，任何权限级别（含 yolo）都不自动放行，且不记忆会话放行。
var hardAskTools = map[string]bool{
	"cost_save": true,
	// cost_library（2026-11-25 造价数据库推倒重建）：导入工作簿/删定额/删工料机/
	// 批量调价/去重合并——删除与批量覆盖类逐条确认；演练与只读动作见
	// costLibraryArgsNoWrite（不弹卡）。
	"cost_library":          true,
	"remember":              true,
	"forget":                true,
	"proposal_save":         true,
	"proposal_update":       true,
	"proposal_delete":       true,
	"promote_session_facts": true,
	"install_skill":         true,
	// computer_key（电脑操控施工单 W4-2）：键盘可触发任意快捷键（含 Alt+F4、
	// 全局热键），必须逐条确认；W5 落工具后生效，此处先行占位无害。
	"computer_key": true,
}

// gateApprover adapts the Controller to permission.Approver. It is distinct
// from the public Approve command (different signature, different direction).
type gateApprover struct{ c *Controller }

func (g gateApprover) Approve(ctx context.Context, tool, subject string, args json.RawMessage) (bool, bool, string, error) {
	// Auto-allow without prompting while executing a just-approved plan (the plan
	// was the approval) or while YOLO/bypass mode is on. Deny rules already bit
	// before this point, so they still block.
	g.c.mu.Lock()
	autoApprove := g.c.autoApprove
	permLevel := g.c.permLevel
	g.c.mu.Unlock()
	auto := autoApprove || permLevel != "ask"
	// 方案库演练（dry_run=true）只校验、不写库：不弹确认卡，照常记进自动放行账本。
	if tool == "proposal_save" && proposalArgsDryRun(args) {
		g.c.emitAutoApproval(tool, subject, event.ApprovalSourceGate, event.AutoDecidedAuto)
		return true, false, "", nil
	}
	// 造价库演练与只读动作（dry_run=true / health / 未点名词干的 flywheel 预览）：
	// 只校验、匹配、统计，不写库——同样不弹确认卡，照常记进自动放行账本。
	if tool == "cost_library" && costLibraryArgsNoWrite(args) {
		g.c.emitAutoApproval(tool, subject, event.ApprovalSourceGate, event.AutoDecidedAuto)
		return true, false, "", nil
	}
	// 持久化写入（成本库/记忆/方案库）必须逐条经用户确认：
	// auto/yolo 权限级别也强制询问，且不记忆会话放行。集合按空间策略
	// 参数化（S1.5-A：play 产品默认空集 = 不弹审批卡；默认集 = 包级 hardAskTools）。
	if g.c.hardAskSet()[tool] {
		allow, _, denial, err := g.c.requestApproval(ctx, tool, approvalSubjectFor(tool, args), true, event.ApprovalSourceHardAsk)
		return allow, false, denial, err
	}
	// diff 确认闭环刀C：schedule_apply project 通道（args.project 非空=整计划
	// 替换/生成）毁伤半径最大——任何权限级别（含 auto/yolo）前置强制逐条
	// 确认，alwaysPrompt 不读不写会话放行（拍板项 2：禁会话记忆）。ops 增量
	// 通道维持现状闸门（拍板项 3：auto/yolo 豁免弹卡，后置回执+回滚兜底）。
	if tool == "schedule_apply" && scheduleApplyIsProjectChannel(args) {
		allow, _, denial, err := g.c.requestApproval(ctx, tool, approvalSubjectFor(tool, args), true, event.ApprovalSourceProjectPlan)
		return allow, false, denial, err
	}
	if auto {
		// 没弹卡，但这是一次真实的写入授权：记进自动放行账本（谁放的、
		// 拟执行什么）。不产生注意力事务——没有人被打断。
		decided := event.AutoDecidedAuto
		if autoApprove || permLevel == "yolo" {
			decided = event.AutoDecidedYolo
		}
		g.c.emitAutoApproval(tool, subject, event.ApprovalSourceGate, decided)
		return true, false, "", nil
	}
	// 会话内已授权（granted / argv 规则 / 追认落账）：授权已由人给出，直接
	// 放行——不进机器评审也不弹卡（guardian 在授权之后，不在授权之前）。
	g.c.mu.Lock()
	alreadyGranted := g.c.ruleGrantedLocked(tool, subject) || g.c.argvAllowedLocked(approvalPrompt{tool: tool, subject: subject})
	g.c.mu.Unlock()
	if !alreadyGranted {
		// T11 Guardian：常规 Ask 通道的用户卡之前先自动评审——allow 放行、
		// deny 带理由拒绝（可追认）；评审失败回落用户卡（红线：错误不冒充
		// 风险，也绝不因失败自动放行）。hardAsk/project 通道在上面已走
		// requestApproval，不经过这里（逐条确认硬纪律不受机器评审影响）。
		if path, denial := g.c.guardianReview(ctx, tool, subject, args); path != guardianFallthrough {
			return path == guardianAllowPath, false, denial, nil
		}
	}
	allow, remember, denial, err := g.c.requestApproval(ctx, tool, subject, false, event.ApprovalSourceGate)
	return allow, remember, denial, err
}

// approvalSubjectFor 为硬性审批工具生成可读的确认摘要。
func approvalSubjectFor(tool string, args json.RawMessage) string {
	switch tool {
	case "cost_save":
		return costSaveApprovalSubject(args)
	case "cost_library":
		return costLibraryApprovalSubject(args)
	case "remember":
		return rememberApprovalSubject(args)
	case "proposal_save":
		return proposalSaveApprovalSubject(args)
	case "proposal_update":
		return proposalUpdateApprovalSubject(args)
	case "proposal_delete":
		return proposalDeleteApprovalSubject(args)
	case "promote_session_facts":
		return "把本次会话沉淀的临时事实提升为永久记忆（跨会话自动加载）"
	case "schedule_apply":
		if scheduleApplyIsProjectChannel(args) {
			return scheduleApplyProjectSubject(args)
		}
		return permission.Subject(args)
	}
	return permission.Subject(args)
}

// scheduleApplyIsProjectChannel 判定 schedule_apply 是否 project 整量通道
// （args.project 非空 JSON 对象 = 整计划替换/生成）。ops 增量通道与解析
// 失败都按非 project 处理（走现状闸门）。
func scheduleApplyIsProjectChannel(args json.RawMessage) bool {
	var a struct {
		Project json.RawMessage `json:"project"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return false
	}
	s := strings.TrimSpace(string(a.Project))
	return s != "" && s != "null" && s != "{}"
}

// scheduleApplyProjectSubject 为 project 通道整量替换生成可读确认摘要
// （diff 确认闭环刀C）：解析 args.project 跑 CPM，把「要写进来的东西」说
// 清楚——总工期/关键项数来自同一 schedule 引擎（与板块/回执同口径）。
// before 侧对比不在这里做（闸门无工作区上下文；改动明细对比由前端 diff
// 确认卡承担，v4.147 刀B）。解析失败/CPM 不过都诚实降级标注。
func scheduleApplyProjectSubject(args json.RawMessage) string {
	var a struct {
		Path    string          `json:"path"`
		Project json.RawMessage `json:"project"`
	}
	_ = json.Unmarshal(args, &a)
	target := "当前计划文件"
	if strings.TrimSpace(a.Path) != "" {
		target = strings.TrimSpace(a.Path)
	}
	if len(a.Project) == 0 {
		return fmt.Sprintf("整计划替换（%s）：参数解析失败，详见参数原文", target)
	}
	var p schedule.Project
	if err := json.Unmarshal(a.Project, &p); err != nil {
		return fmt.Sprintf("整计划替换（%s）：计划 JSON 无效，批准也将被引擎拒绝", target)
	}
	cpm, analysis := p.Analyze()
	if !cpm.OK {
		return fmt.Sprintf("整计划替换（%s）：%d 项工作，CPM 校验未过（%s）——批准也将被引擎拒绝",
			target, len(p.Tasks), cpm.Error)
	}
	return fmt.Sprintf("整计划替换（%s）：%d 项工作，总工期 %d 天，关键 %d 项；改动明细见 diff 预览",
		target, len(p.Tasks), cpm.Duration, len(analysis.Critical))
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
	// source 是触发来源判别字段（gate / hard_ask / project_plan /
	// permission_request），随 request 与 resolved 两侧事件落日志与轨迹。
	source string
}

// attentionLatency 把卡片请求发出 → 解决之间的墙钟时长折算为毫秒（时钟
// 回拨/负值按 0，不产生负数计数）。
func attentionLatency(start time.Time) int64 {
	if d := time.Since(start).Milliseconds(); d > 0 {
		return d
	}
	return 0
}

// approvalOutcome 把五决策语义 + ctx 取消归一为计数结果：allowed 合并三种
// 放行（明细 decision 仍保留），denied 含未知决策串的保守拒绝。
func approvalOutcome(decision string, err error) string {
	if err != nil {
		return event.OutcomeCancelled
	}
	switch decision {
	case DecisionAllowOnce, DecisionAllowSession, DecisionPersistAllow:
		return event.OutcomeAllowed
	case DecisionAbort:
		return event.OutcomeAborted
	case DecisionTimeout:
		return event.OutcomeTimeout
	default:
		return event.OutcomeDenied
	}
}

// requestApproval emits an ApprovalRequest and blocks until Approve(ID, …)
// answers or ctx is cancelled. A prior session grant for the same tool+subject
// short-circuits, unless alwaysPrompt is set (敏感写入必须逐条确认）。
// promptMu serialises outstanding prompts. denial 非空=拒绝/超时的回喂文案。
func (c *Controller) requestApproval(ctx context.Context, tool, subject string, alwaysPrompt bool, source string) (bool, bool, string, error) {
	res := c.promptApproval(ctx, approvalPrompt{
		tool: tool, subject: subject, alwaysPrompt: alwaysPrompt, source: source,
	})
	return res.allow, res.remember, res.denial, res.err
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
	autoApprove := c.autoApprove
	permLevel := c.permLevel
	auto := autoApprove || permLevel != "ask"
	if auto {
		c.grantRuleLocked(tool, subject)
	}
	c.mu.Unlock()
	if auto {
		// 规则直接按会话生效：不弹卡、不打扰人，但账本要留下「谁批准的
		// 这条规则、依据什么模式」，否则事后无法区分人批的规则与 auto/yolo
		// 下机器替人批的规则。
		decided := event.AutoDecidedAuto
		if autoApprove || permLevel == "yolo" {
			decided = event.AutoDecidedYolo
		}
		c.emitAutoApproval(tool, subject, event.ApprovalSourcePermissionRequest, decided)
		return true, "auto", nil
	}
	res := c.promptApproval(ctx, approvalPrompt{
		tool: tool, subject: subject, reason: reason, ruleRequest: true,
		source: event.ApprovalSourcePermissionRequest,
	})
	if res.err != nil {
		return false, "", res.err
	}
	decision := res.decision
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

// emitAutoApproval 记录一次「没弹卡、由机器放行」的授权（ApprovalAuto）。
// 注意力事务只记「人被打断」，这条账本记「机器替你决定了什么」：两条线
// 分列，互不冒充。sink 缺省（无事件总线的嵌用场景）静默跳过，不影响放行。
func (c *Controller) emitAutoApproval(tool, subject, source, decided string) {
	if c.sink == nil {
		return
	}
	c.sink.Emit(event.Event{Kind: event.ApprovalAuto, Auto: event.AutoApproval{
		Tool: tool, Subject: subject, Source: source, Decided: decided,
	}})
}

// autoSource 给自动放行账本补来源：promptApproval 的调用方都会显式传
// source，空值只可能来自更早的代码路径，按 gate 兜底（与读端同口径）。
func autoSource(p approvalPrompt) string {
	if p.source == "" {
		return event.ApprovalSourceGate
	}
	return p.source
}

// promptApproval emits one approval prompt per p and blocks until Approve(ID, …)
// answers, ctx is cancelled, or the configured approval timeout fires. It returns
// the resolution as an approvalResult: allow/remember for the gate, decision for
// RequestPermission, and denial（拒绝/超时的回喂文案，放行为空）——codex 蒸馏
// T3：拒绝必须带着理由进下一轮模型上下文，模型才知道换法子。
func (c *Controller) promptApproval(ctx context.Context, p approvalPrompt) approvalResult {
	key := p.tool + "\x00" + p.subject

	if !p.alwaysPrompt {
		c.mu.Lock()
		grantedAlready := c.ruleGrantedLocked(p.tool, p.subject) || c.argvAllowedLocked(p)
		c.mu.Unlock()
		if grantedAlready {
			// 会话内已有规则覆盖：不弹卡、不产生注意力事务，但账本要记下
			// 「这次写操作是被一条既有规则放行的」——否则人只看到结果，
			// 看不到机器替他做过的决定。
			c.emitAutoApproval(p.tool, p.subject, autoSource(p), event.AutoDecidedSessionGrant)
			return approvalResult{allow: true, remember: true, decision: DecisionAllowSession} // session grant was previously stored
		}
	}

	c.promptMu.Lock()
	defer c.promptMu.Unlock()

	// Re-check the grant: a session grant may have landed while we queued behind
	// another prompt for the same subject.
	c.mu.Lock()
	if !p.alwaysPrompt {
		if c.ruleGrantedLocked(p.tool, p.subject) || c.argvAllowedLocked(p) {
			c.mu.Unlock()
			c.emitAutoApproval(p.tool, p.subject, autoSource(p), event.AutoDecidedSessionGrant)
			return approvalResult{allow: true, remember: true, decision: DecisionAllowSession} // session grant stored while waiting
		}
	}
	c.nextID++
	id := strconv.Itoa(c.nextID)
	reply := make(chan approvalReply, 1)
	c.approvals[id] = reply
	c.mu.Unlock()

	c.sink.Emit(event.Event{Kind: event.ApprovalRequest, Approval: event.Approval{
		ID: id, Tool: p.tool, Subject: p.subject,
		Reason: p.reason, Request: p.ruleRequest, Source: p.source,
	}})
	// 注意力事务的结果侧：卡一旦真的弹出，就必须有对应解决事件（允许/拒绝/
	// 取消/超时/中止）+ 等待时长 + 来源，成对落日志与轨迹，使「这张卡后来
	// 怎样了」可重放、可计数。短路的会话放行不弹卡、不产生事务。
	startedAt := time.Now()
	var res approvalResult
	defer func() {
		c.sink.Emit(event.Event{Kind: event.ApprovalResolved, Resolved: event.ApprovalResolution{
			ID: id, Tool: p.tool, Subject: p.subject, Source: p.source,
			Decision: res.decision, Outcome: approvalOutcome(res.decision, res.err),
			LatencyMS: attentionLatency(startedAt),
		}})
	}()
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
			res.decision = DecisionAbort
			res.denial = "已拒绝：" + rejectionReason(r.reason) + "，本轮已中止"
			c.recordDeniedAction(id, p, res.decision, r.reason)
			c.Cancel()
			return res
		case DecisionPersistAllow:
			// 始终允许（codex ApprovedExecpolicyAmendment）：本会话内先记
			// granted（与 allow_session 同效），再把规则回写策略文件持久化，
			// 重启后由 [permissions].allow 直接放行。hardAsk 持久化写入（
			// alwaysPrompt）完全降级为 allow_session——不记 granted、不回写
			// （任何级别都不自动放行的硬纪律，frontend 也不渲染该按钮）。
			// 规则申请：记入 grantedRules（保留 glob 语义）而非精确 key。
			if !p.alwaysPrompt {
				c.mu.Lock()
				c.grantAllowLocked(key, p)
				c.mu.Unlock()
				c.writebackAllowRule(p.tool, p.subject)
			}
			res.decision = DecisionPersistAllow
			res.allow = true
			return res
		case DecisionAllowSession:
			if !p.alwaysPrompt {
				c.mu.Lock()
				c.grantAllowLocked(key, p)
				c.mu.Unlock()
			}
			res.decision = DecisionAllowSession
			res.allow = true
			return res
		case DecisionAllowOnce:
			// 常规闸门：允许一次不记忆（该次调用放行即完成语义）。规则申请：
			// 申请的对象本身就是一条规则——allow_once 与 allow_session 同效
			// （写入会话 granted），否则「允许一次」没有任何可落地的效果。
			if p.ruleRequest && !p.alwaysPrompt {
				c.mu.Lock()
				c.grantRuleLocked(p.tool, p.subject)
				c.mu.Unlock()
			}
			res.decision = DecisionAllowOnce
			res.allow = true
			return res
		default:
			// deny（含未知决策串的保守处理）：拒绝但回合继续。理由回喂是
			// 本次拒绝的正文（默认「用户拒绝」），一句话，不贴全量参数。
			res.decision = DecisionDeny
			res.denial = "已拒绝：" + rejectionReason(r.reason) + "——不要原样重试，换方案或结束回合说明"
			c.recordDeniedAction(id, p, res.decision, r.reason)
			return res
		}
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.approvals, id)
		c.mu.Unlock()
		res.err = ctx.Err()
		return res
	case <-timeoutCh:
		c.mu.Lock()
		delete(c.approvals, id)
		c.mu.Unlock()
		// 超时挂起（codex 蒸馏 T3）：按拒绝处理但不进 deny 账本（与用户主动
		// 拒绝可区分），动作挂「待追认账本」——回来后 ApproveDenied 可重放。
		res.decision = DecisionTimeout
		res.denial = "审批超时无人响应，已按拒绝处理——不要原样重试；换方案或结束回合说明，也可稍后追认"
		c.recordDeniedAction(id, p, res.decision, "")
		if p.subject != "" {
			c.notice(fmt.Sprintf("审批超时（%s 未响应），已按拒绝处理并挂起待追认：%s %s",
				c.approvalTimeout, p.tool, p.subject))
		} else {
			c.notice(fmt.Sprintf("审批超时（%s 未响应），已按拒绝处理并挂起待追认：%s",
				c.approvalTimeout, p.tool))
		}
		return res
	}
}

// approvalResult 是一次审批弹卡的归一结果：allow/remember 供闸门，decision
// 供 request_permission 的决策串回传，denial 是拒绝/超时的回喂文案（放行为
// 空），err 仅在 ctx 取消（回合终止）时非 nil。
type approvalResult struct {
	allow    bool
	remember bool
	decision string
	denial   string
	err      error
}

// rejectionReason 归一拒绝理由：空白用默认「用户拒绝」；截到一句话（200 rune）
// ——理由是回喂模型的提示，不是参数回放。
func rejectionReason(reason string) string {
	r := strings.TrimSpace(reason)
	if r == "" {
		return "用户拒绝"
	}
	if runes := []rune(r); len(runes) > 200 {
		r = string(runes[:200]) + "…"
	}
	return r
}

// deniedActionCap 是待追认账本的封顶条数：超限滚掉最旧（内存账本，防长会话
// 无界增长）。
const deniedActionCap = 50

// deniedAction 是一条拒绝/超时挂账：记录原弹卡的全部重放原料，ApproveDenied
// 按原参数再弹一次卡走正常闸门。
type deniedAction struct {
	id           string    // 原弹卡 ID（账本键，nextID 单调不复用）
	tool         string    // 工具名
	subject      string    // 确认摘要 / 规则 subject（glob 语义原样保留）
	source       string    // 原来源（gate / hard_ask / project_plan / permission_request）
	alwaysPrompt bool      // hardAsk 逐条确认卡——追认放行也不写 granted（硬纪律）
	ruleRequest  bool      // 权限升级申请卡——追认放行走 grantRuleLocked
	decision     string    // deny / timeout（timeout 与用户拒绝可区分）
	reason       string    // 拒绝理由（超时为空）
	at           time.Time // 挂账时刻
	inFlight     bool      // 正在重放审批卡：双击/并发追认只能开一张卡
}

// recordDeniedAction 把拒绝/超时的动作挂进待追认账本（FIFO 封顶）。ctx 取消
// 不挂账（回合终止，模型没拿到结果，重放无意义）。
func (c *Controller) recordDeniedAction(id string, p approvalPrompt, decision, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deniedActions = append(c.deniedActions, deniedAction{
		id: id, tool: p.tool, subject: p.subject, source: p.source,
		alwaysPrompt: p.alwaysPrompt, ruleRequest: p.ruleRequest,
		decision: decision, reason: reason, at: time.Now(),
	})
	if len(c.deniedActions) > deniedActionCap {
		c.deniedActions = c.deniedActions[len(c.deniedActions)-deniedActionCap:]
	}
}

// PendingDeniedActions 返回待追认账本快照（时间正序）。前端「恢复时提示有 N
// 条待审」与追认列表都读这里；无人值守会话回来后按此展示。
func (c *Controller) PendingDeniedActions() []deniedAction {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]deniedAction, len(c.deniedActions))
	copy(out, c.deniedActions)
	return out
}

// DeniedActionInfo 是待追认账本的绑定面视图（JSON 字段名与前端 WireDeniedAction
// 对齐；decision=deny|timeout，timeout 与用户主动拒绝可区分）。
type DeniedActionInfo struct {
	ID       string `json:"id"`
	Tool     string `json:"tool"`
	Subject  string `json:"subject"`
	Source   string `json:"source"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
	At       string `json:"at"`
}

// DeniedActionList 返回待追认账本的绑定面快照（时间正序）。
func (c *Controller) DeniedActionList() []DeniedActionInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]DeniedActionInfo, 0, len(c.deniedActions))
	for _, a := range c.deniedActions {
		out = append(out, DeniedActionInfo{
			ID: a.id, Tool: a.tool, Subject: a.subject, Source: a.source,
			Decision: a.decision, Reason: a.reason, At: a.at.Format(time.RFC3339),
		})
	}
	return out
}

// ApproveDenied 追认一条已拒/超时的动作（codex 蒸馏 T3 approve-after-deny）：
// 按原参数（tool/subject/alwaysPrompt/source）再弹一次审批卡走正常闸门。
// 放行后写会话 granted（hardAsk 卡除外——逐条确认纪律，granted 永不写），
// 模型下次同参调用即自然放行、工具被重新执行；再次拒绝则账本保留可再追认。
func (c *Controller) ApproveDenied(id string) {
	var (
		act   deniedAction
		found bool
	)
	c.mu.Lock()
	for i := range c.deniedActions {
		if c.deniedActions[i].id == id {
			if c.deniedActions[i].inFlight {
				c.mu.Unlock()
				c.notice("追认：该动作的审批卡已经打开")
				return
			}
			c.deniedActions[i].inFlight = true
			act = c.deniedActions[i]
			found = true
			break
		}
	}
	c.mu.Unlock()
	if !found {
		c.notice("追认：未找到该动作（可能已滚动出账本或已被移除）")
		return
	}
	release := func() {
		c.mu.Lock()
		for i := range c.deniedActions {
			if c.deniedActions[i].id == id {
				c.deniedActions[i].inFlight = false
				break
			}
		}
		c.mu.Unlock()
	}
	// 重放弹卡：走与原始弹卡完全相同的通道与语义（hardAsk 逐条确认不降级）。
	res := c.promptApproval(context.Background(), approvalPrompt{
		tool: act.tool, subject: act.subject, alwaysPrompt: act.alwaysPrompt, ruleRequest: act.ruleRequest,
		source: act.source,
	})
	if res.err != nil {
		release()
		return
	}
	if !res.allow {
		release()
		c.notice(fmt.Sprintf("追认未通过（%s），动作保留在待审账本", res.decision))
		return
	}
	// 放行：写会话授权（hardAsk 除外），账本移除，回合里模型重试即通。
	c.mu.Lock()
	stillCurrent := false
	for i := range c.deniedActions {
		if c.deniedActions[i].id == id {
			stillCurrent = true
			break
		}
	}
	if !stillCurrent {
		c.mu.Unlock()
		c.notice("追认：会话已切换，旧动作不会授予到新会话")
		return
	}
	if !act.alwaysPrompt {
		if act.ruleRequest {
			c.grantRuleLocked(act.tool, act.subject)
		} else {
			c.granted[act.tool+"\x00"+act.subject] = true
		}
	}
	for i := range c.deniedActions {
		if c.deniedActions[i].id == id {
			c.deniedActions = append(c.deniedActions[:i], c.deniedActions[i+1:]...)
			break
		}
	}
	c.mu.Unlock()
	c.notice(fmt.Sprintf("已追认：%s %s——模型下次同参调用将放行", act.tool, act.subject))
}

// argvAllowedLocked 报告会话 argv 规则集是否放行该调用（调用方持锁）：
// bash 的「记住这条命令」形态（T10）。
func (c *Controller) argvAllowedLocked(p approvalPrompt) bool {
	return p.tool == "bash" && c.argvRules != nil && c.argvRules.Allowed(p.subject)
}

// grantAllowLocked 落一条「允许并记忆」的会话授权（allow_session / persist_allow
// 共用，调用方持锁）：规则申请走 glob 规则；命令类工具（bash）走 argv 前缀
// 规则（T10——记住命令形状而非逐字串，同前缀变体免重复审批，串接提权不匹配）；
// 其余工具保持精确 key。argvRules 惰性创建。
func (c *Controller) grantAllowLocked(key string, p approvalPrompt) {
	switch {
	case p.ruleRequest:
		c.grantRuleLocked(p.tool, p.subject)
	case p.tool == "bash":
		if r, ok := permission.ArgvRuleFromCommand(p.subject, 0); ok {
			if c.argvRules == nil {
				c.argvRules = permission.NewArgvRuleSet()
			}
			c.argvRules.Add(r)
			return
		}
		fallthrough // 命令无法安全切分（引号异常等）：回落精确 key，不猜
	default:
		c.granted[key] = true
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
	if tool == "bash" && subject != "" {
		// T10（codex 蒸馏）：命令类工具回写 argv 前缀规则而不是 glob——
		// `bash prefix ["git","status"]` 只匹配各 shell 段前缀都命中的命令，
		// `git status; rm -rf` 不会顺手放行。切分失败回落精确串。
		if r, ok := permission.ArgvRuleFromCommand(subject, 0); ok {
			rule = permission.ArgvRuleText(tool, r)
		} else {
			rule = tool + "(" + subject + ")"
		}
	} else if subject != "" {
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

// costLibraryArgsNoWrite 造价库的演练与只读调用：dry_run=true，或 action=health，
// 或 flywheel 没点名词干（只出预览）。这些调用不写库，不必走逐条确认。参数
// 解析不了按「会写」处理（fail-closed）。
func costLibraryArgsNoWrite(args json.RawMessage) bool {
	var p struct {
		DryRun bool     `json:"dry_run"`
		Action string   `json:"action"`
		Stems  []string `json:"stems"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return false
	}
	if p.DryRun {
		return true
	}
	switch strings.TrimSpace(p.Action) {
	case "health":
		return true
	case "flywheel":
		return len(p.Stems) == 0
	}
	return false
}

// costLibraryApprovalSubject 把 cost_library 的参数整理成可读的确认摘要：
// 动作 + 对象 + 规模，让审批卡一眼看出要导入/删除/改价的是什么。
func costLibraryApprovalSubject(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var p struct {
		Action       string   `json:"action"`
		Force        bool     `json:"force"`
		Path         string   `json:"path"`
		ProjectID    int64    `json:"project_id"`
		ItemIDs      []int64  `json:"item_ids"`
		Codes        []string `json:"codes"`
		IDs          []int64  `json:"ids"`
		ResourceID   int64    `json:"resource_id"`
		ResourceCode string   `json:"resource_code"`
		Price        *float64 `json:"price"`
		Stems        []string `json:"stems"`
		KeepID       int64    `json:"keep_id"`
		MergeIDs     []int64  `json:"merge_ids"`
		Rows         []struct {
			Title string  `json:"title"`
			Price float64 `json:"price"`
		} `json:"rows"`
		Quota struct {
			Code  string `json:"code"`
			Title string `json:"title"`
		} `json:"quota"`
		Resource struct {
			ID           int64    `json:"id"`
			Code         string   `json:"code"`
			Title        string   `json:"title"`
			CurrentPrice *float64 `json:"current_price"`
			BasePrice    *float64 `json:"base_price"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ""
	}
	var label string
	switch p.Action {
	case "import_project":
		label = "导入五表工作簿"
	case "save_quota":
		label = "保存定额"
	case "delete_quota":
		label = "删除定额"
	case "save_resource":
		label = "保存工料机"
	case "delete_resource":
		label = "删除工料机"
	case "set_price":
		label = "工料机调价"
	case "bulk_price":
		label = "批量调价"
	case "delete_items":
		label = "删除清单行"
	case "delete_project":
		label = "删除项目"
	case "seed":
		label = "资料条目资源化"
	case "flywheel":
		label = "自生成企业定额"
	case "dedupe":
		label = "工料机去重"
	default:
		return permission.Subject(args)
	}
	parts := []string{"造价库：" + label}
	switch p.Action {
	case "import_project":
		if s := strings.TrimSpace(p.Path); s != "" {
			parts = append(parts, filepath.Base(s))
		}
	case "save_quota", "delete_quota":
		if s := strings.TrimSpace(p.Quota.Code); s != "" {
			parts = append(parts, s)
		}
		if s := strings.TrimSpace(p.Quota.Title); s != "" {
			parts = append(parts, "「"+s+"」")
		}
		parts = append(parts, fmt.Sprintf("%d 条", len(p.Codes)))
	case "save_resource":
		switch {
		case p.Resource.Title != "":
			parts = append(parts, "「"+strings.TrimSpace(p.Resource.Title)+"」")
		case p.Resource.Code != "":
			parts = append(parts, p.Resource.Code)
		}
		if p.Resource.CurrentPrice != nil {
			parts = append(parts, fmt.Sprintf("现行价 ¥%.2f", *p.Resource.CurrentPrice))
		} else if p.Resource.BasePrice != nil {
			parts = append(parts, fmt.Sprintf("基准价 ¥%.2f", *p.Resource.BasePrice))
		}
	case "delete_resource":
		if n := len(p.IDs) + len(p.Codes); n > 0 {
			parts = append(parts, fmt.Sprintf("%d 条", n))
		}
	case "set_price":
		if p.ResourceCode != "" {
			parts = append(parts, p.ResourceCode)
		} else if p.ResourceID > 0 {
			parts = append(parts, fmt.Sprintf("#%d", p.ResourceID))
		}
		if p.Price != nil {
			parts = append(parts, fmt.Sprintf("→ ¥%.2f", *p.Price))
		}
	case "bulk_price":
		parts = append(parts, fmt.Sprintf("%d 行", len(p.Rows)))
	case "delete_items":
		parts = append(parts, fmt.Sprintf("项目 #%d 的 %d 行", p.ProjectID, len(p.ItemIDs)))
	case "delete_project":
		parts = append(parts, fmt.Sprintf("项目 #%d（清单与独占定额一并删除）", p.ProjectID))
	case "flywheel":
		parts = append(parts, fmt.Sprintf("%d 个簇", len(p.Stems)))
	case "dedupe":
		parts = append(parts, fmt.Sprintf("保留 #%d，合并 %d 条", p.KeepID, len(p.MergeIDs)))
	}
	if p.Force {
		parts = append(parts, "连带解除引用")
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
