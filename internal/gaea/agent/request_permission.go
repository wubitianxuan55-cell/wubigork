package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// 审批决策串（逐字节对齐 control.ApproveDecision 五值 + C4 TimedOut 的
// "timeout"；agent 不能反向 import control——control 依赖 agent——故在此
// 私有镜像，双方注释互指，改动任一侧必须同步另一侧）。
const (
	decisionAllowOnce    = "allow_once"
	decisionAllowSession = "allow_session"
	decisionPersistAllow = "persist_allow"
	decisionDeny         = "deny"
	decisionAbort        = "abort"
	decisionTimeout      = "timeout"
)

// RequestPermissionTool lets the model escalate permissions explicitly when it
// lacks one it needs (对齐 codex request_permissions_for_environment 语义族):
// instead of firing the target tool and eating a denial, it first asks the user
// to grant a rule — "Tool" (whole tool) or "Tool(subject-glob)" — through the
// existing approval-card channel. The card shows the requested rule and the
// model's reason; the five decisions map to:
//
//	allow_once / allow_session → 写入会话 granted（本会话内规则生效）
//	persist_allow              → 同上 + 规则回写策略文件（跨会话）
//	deny                       → 工具结果记拒绝，回合继续
//	abort                      → 拒绝并终止本轮
//
// 硬纪律（两端共同守住，本工具自身不制造任何绕过）：批准授予的是「规则」，
// 后续真实工具调用仍走正常权限闸门（规则满足则自然放行，deny 规则依旧硬拒，
// hardAsk 工具不接受升级申请）；审批超时无人响应按既有语义拒绝。headless
// 运行（无 PermissionRequester）返回非交互结果，绝不阻塞自治运行。
type RequestPermissionTool struct{}

func NewRequestPermissionTool() *RequestPermissionTool { return &RequestPermissionTool{} }

func (*RequestPermissionTool) Name() string { return "request_permission" }

// Description 是模型每轮都付的常驻面（2026-11-25 办公工具梳理批 A2：69 个真实
// 会话里仅 4 次调用，却曾是三档最大的单项 1188 字符）。只留「调用前必须知道」
// 的规矩：何时用、规则形态、reason 必填且原样上卡、批准=规则不是绕过、被拒
// 别重试、不得替用户申请破坏性操作。各决策的后续动作（重发原调用 / 换路径 /
// 收尾说明）由 Execute 的回执逐形态给出；发布边界另由 PublishGuard 提示词块
// 单源陈述（publishguard.go），不在此重复。
func (*RequestPermissionTool) Description() string {
	return "Request a permission rule from the user — use when a call was blocked by the permission policy, or the next step needs a tool you know will be declined. " +
		"Rule: \"Tool\" or \"Tool(subject-glob)\" for one command prefix or path. " +
		"`reason` is required and shown verbatim: say what you will run and why. " +
		"A grant only adds a rule — re-issue the original call; it still passes the normal gate. " +
		"Denied: do not ask again; use an approach that needs no extra permission, or report the blocker. " +
		"Never request permissions for destructive operations the user has not asked for."
}

func (*RequestPermissionTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "tool":{
    "type":"string",
    "description":"Name of the tool you need the rule for, e.g. \"bash\". Must be a registered tool."
  },
  "subject":{
    "type":"string",
    "description":"Optional narrow subject the rule is scoped to, matched as a glob against the target call's subject (command / path / pattern). Omit to request the whole tool."
  },
  "reason":{
    "type":"string",
    "description":"Why you need this permission, shown to the user verbatim on the approval card. Be concrete: what you intend to run and why the task requires it."
  }
},
"required":["tool","reason"]
}`)
}

// ReadOnly is true: requesting a permission has no host side effects by itself
// (any grant is a user decision made on the approval card), so the call never
// gates on itself and stays available in plan mode.
func (*RequestPermissionTool) ReadOnly() bool { return true }

func (*RequestPermissionTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Tool    string `json:"tool"`
		Subject string `json:"subject"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	toolName := strings.TrimSpace(p.Tool)
	reason := strings.TrimSpace(p.Reason)
	subject := strings.TrimSpace(p.Subject)
	if toolName == "" {
		return "", fmt.Errorf("tool is required: name the tool you need the rule for")
	}
	// reason 必填：审批卡必须能看到申请理由——缺理由的申请直接退回重填，
	// 绝不在未展示理由的情况下把申请递给用户。
	if reason == "" {
		return "", fmt.Errorf("reason is required: the approval card shows it to the user verbatim — state what you need to run and why")
	}

	pr, ok := PermissionRequesterFromContext(ctx)
	if !ok {
		// Headless / no interactive user: don't block an autonomous run —
		// there is nobody to approve the escalation, and silently granting it
		// is forbidden. Same posture as the `ask` tool's Never-Ask fallback.
		return "[Never-Ask] No interactive user is available to answer a permission request (headless/CI mode). " +
			"The requested rule \"" + ruleString(toolName, subject) + "\" cannot be granted here and was NOT applied. " +
			"Continue with the least-privileged, non-interactive approach that does not need the extra permission, " +
			"or finish your turn and list the permission you would need and why.", nil
	}

	granted, decision, err := pr.RequestPermission(ctx, toolName, subject, reason)
	if err != nil {
		return "", fmt.Errorf("request_permission: %w", err)
	}
	rule := ruleString(toolName, subject)

	if granted {
		switch decision {
		case decisionPersistAllow:
			return "The user approved the permission request (decision=persist_allow). Rule \"" + rule + "\" is now in effect for this session AND has been written back to the permission policy file, so it persists across restarts. Re-issue the original tool call now — it still goes through the normal permission gate, which this rule now satisfies.", nil
		case decisionAllowOnce:
			return "The user approved the permission request (decision=allow_once). Rule \"" + rule + "\" is in effect for this session. Re-issue the original tool call now — it still goes through the normal permission gate, which this rule now satisfies.", nil
		case decisionAllowSession:
			return "The user approved the permission request (decision=allow_session). Rule \"" + rule + "\" is in effect for this session. Re-issue the original tool call now — it still goes through the normal permission gate, which this rule now satisfies.", nil
		default:
			// auto（auto/yolo 级别下无需打扰用户，直接生效）等其它授予形态。
			return "The permission request was granted without prompting (decision=" + decision + "). Rule \"" + rule + "\" is in effect for this session. Re-issue the original tool call now.", nil
		}
	}

	switch decision {
	case decisionDeny:
		return "The user DENIED the permission request (decision=deny). The rule \"" + rule + "\" was not granted and the turn continues. Do not repeat the same request — pick another approach that needs no extra permission, or explain the blocker in your final answer and let the user decide.", nil
	case decisionAbort:
		return "The user denied the permission request and aborted the turn (decision=abort). The rule \"" + rule + "\" was not granted.", nil
	case decisionTimeout:
		return "The permission request timed out with no user response and was treated as a denial (decision=timeout). The rule \"" + rule + "\" was not granted. Continue with a non-privileged approach.", nil
	default:
		// refused_*（hardAsk 逐条确认工具 / deny 规则目标 / 未知工具）等
		// 未弹卡即拒绝的形态：如实回传决策串，不掩盖原因。
		return "The permission request for \"" + rule + "\" was not granted (decision=" + decision + "). Do not repeat the same request — pick another approach that needs no extra permission.", nil
	}
}

// ruleString 渲染申请的规则串（与策略文件 "Tool" / "Tool(subject)" 格式一致）。
func ruleString(toolName, subject string) string {
	if subject == "" {
		return toolName
	}
	return toolName + "(" + subject + ")"
}
