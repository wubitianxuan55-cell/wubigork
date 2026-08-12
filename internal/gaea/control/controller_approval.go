package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/event"
)

// --- approval bridge (agent gate → events) ---

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
	// 成本库写入必须逐条经用户确认：auto/yolo 权限级别也强制询问，
	// 且不记忆会话放行（避免后续 cost_save 静默入库）。
	if tool == "cost_save" {
		return g.c.requestApproval(ctx, tool, costSaveApprovalSubject(args), true)
	}
	if auto {
		return true, false, nil
	}
	return g.c.requestApproval(ctx, tool, subject, false)
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

	select {
	case r := <-reply:
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
