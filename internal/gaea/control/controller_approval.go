package control

import (
	"context"
	"encoding/json"
	"fmt"
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
	// auto/yolo 权限级别也强制询问，且不记忆会话放行。
	if hardAskTools[tool] {
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

// requestApproval emits an ApprovalRequest and blocks until Approve(ID, …)
// answers or ctx is cancelled. A prior session grant for the same tool+subject
// short-circuits, unless alwaysPrompt is set (敏感写入必须逐条确认）。
// promptMu serialises outstanding prompts.
func (c *Controller) requestApproval(ctx context.Context, tool, subject string, alwaysPrompt bool) (bool, bool, error) {
	key := tool + "\x00" + subject

	if !alwaysPrompt {
		c.mu.Lock()
		if c.granted[key] {
			c.mu.Unlock()
			return true, true, nil // session grant was previously stored
		}
		c.mu.Unlock()
	}

	c.promptMu.Lock()
	defer c.promptMu.Unlock()

	// Re-check the grant: a session grant may have landed while we queued behind
	// another prompt for the same subject.
	c.mu.Lock()
	if !alwaysPrompt {
		if c.granted[key] {
			c.mu.Unlock()
			return true, true, nil // session grant stored while waiting
		}
	}
	c.nextID++
	id := strconv.Itoa(c.nextID)
	reply := make(chan approvalReply, 1)
	c.approvals[id] = reply
	c.mu.Unlock()

	c.sink.Emit(event.Event{Kind: event.ApprovalRequest, Approval: event.Approval{ID: id, Tool: tool, Subject: subject}})
	// The agent now needs the user's attention; a Notification hook can ping an
	// external channel (desktop notice, phone) while the run blocks on the reply.
	if subject != "" {
		go c.hooks.Notification(ctx, "approval needed: "+tool+" "+subject)
	} else {
		go c.hooks.Notification(ctx, "approval needed: "+tool)
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
		if r.abort {
			// 拒绝并终止本轮（codex ReviewDecision::Abort）：取消当前回合，
			// agent 循环在下一步采样/闸门检查处快速收敛退出；工具结果按
			// 拒绝落日志（「拒绝但继续」由 allow=false 表达，两者语义分离）。
			c.Cancel()
			return false, false, nil
		}
		if r.allow && r.session && !alwaysPrompt {
			c.mu.Lock()
			c.granted[key] = true
			c.mu.Unlock()
		}
		// remember=false: session grants live here, not in the on-disk policy.
		return r.allow, false, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.approvals, id)
		c.mu.Unlock()
		return false, false, ctx.Err()
	case <-timeoutCh:
		c.mu.Lock()
		delete(c.approvals, id)
		c.mu.Unlock()
		if subject != "" {
			c.notice(fmt.Sprintf("审批超时（%s 未响应），已按拒绝处理：%s %s",
				c.approvalTimeout, tool, subject))
		} else {
			c.notice(fmt.Sprintf("审批超时（%s 未响应），已按拒绝处理：%s", c.approvalTimeout, tool))
		}
		return false, false, nil
	}
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
