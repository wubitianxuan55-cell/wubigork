package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── A11 子代理完成裁决（Reasonix 2.x complete_subtask + evidence 蒸馏，v4.524）
//
// 病根：子代理的「我做完了」是模型自由陈述——父模型无从分辨真完成与口头完成。
// 上游机制：子代理经 complete_subtask 提交结构化报告（status + 验收判据 +
// 证据引用），宿主对照**自己观察到的回执**裁决——「只降不升」：引用的判据没有
// 回执支撑就降 unsatisfied，任一降级则 complete 整体降 partial（理由逐条列给
// 父模型）。工具提交永远成功，裁决只改父级看到的结论。
//
// 回执面（gaea 口径）：子会话消息即账本——verification 对「同名命令成功跑过」、
// diff 对「目标路径有成功写」、files 对「目标路径有成功读或写」；review 在 gaea
// 无对应工具=永不自证；manual 永不自证（上游同口径）。失败判定复用
// isErrorMessage 前缀族（error:/blocked:）。

const completeSubtaskName = "complete_subtask"

// completionEvidence 证据引用（上游 CompletionEvidence 同构）。
type completionEvidence struct {
	Kind    string   `json:"kind"` // verification|diff|files|review|manual
	Summary string   `json:"summary,omitempty"`
	Command string   `json:"command,omitempty"`
	Paths   []string `json:"paths,omitempty"`
}

// completionCriterion 单条验收判据。
type completionCriterion struct {
	ID       string               `json:"id"`
	Status   string               `json:"status"` // satisfied|unsatisfied
	Evidence []completionEvidence `json:"evidence,omitempty"`
}

// completionReport 完成报告（complete_subtask 参数形状）。
type completionReport struct {
	Status     string                `json:"status"` // complete|partial|blocked|failed
	Summary    string                `json:"summary"`
	Criteria   []completionCriterion `json:"acceptance_criteria,omitempty"`
	Unresolved []string              `json:"unresolved,omitempty"`
}

// parseCompletionReport 只验形状；真假是宿主的活（上游 ParseCompletionReport）。
func parseCompletionReport(raw string) (completionReport, error) {
	var r completionReport
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return r, fmt.Errorf("invalid complete_subtask JSON: %w", err)
	}
	r.Status = strings.ToLower(strings.TrimSpace(r.Status))
	switch r.Status {
	case "complete", "partial", "blocked", "failed":
	default:
		return r, fmt.Errorf("complete_subtask.status must be complete, partial, blocked, or failed")
	}
	r.Summary = strings.TrimSpace(r.Summary)
	if r.Summary == "" {
		return r, fmt.Errorf("complete_subtask.summary is required")
	}
	for i := range r.Criteria {
		c := &r.Criteria[i]
		c.Status = strings.ToLower(strings.TrimSpace(c.Status))
		switch c.Status {
		case "satisfied", "unsatisfied":
		default:
			return r, fmt.Errorf("criterion %q status must be satisfied or unsatisfied", c.ID)
		}
	}
	return r, nil
}

// completeSubtaskTool 子代理自报完成（只进子代理注册表——buildSubReg 常驻挂载）。
// 提交永远成功；裁决发生在宿主侧（runSubSession 收尾对照回执），子代理只收到
// 确认文案——「宿主可能把结论降到回执支撑的程度；没有什么能升它」（上游注释）。
type completeSubtaskTool struct{}

func (completeSubtaskTool) Name() string { return completeSubtaskName }

func (completeSubtaskTool) Description() string {
	return "Submit the structured completion report for this sub-task: status (complete/partial/blocked/failed), a summary, and acceptance_criteria — each with a status and evidence the host can verify from this session's own receipts (verification=cited command actually ran successfully here; diff=cited paths actually written; files=cited paths read or written; review/manual are never self-certifying). The host lowers any claim its receipts do not back; nothing is raised. Submitting always succeeds."
}

func (completeSubtaskTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"status":{"type":"string","enum":["complete","partial","blocked","failed"],"description":"Your claim about the run."},"summary":{"type":"string","description":"What was done, in a few sentences."},"acceptance_criteria":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"status":{"type":"string","enum":["satisfied","unsatisfied"]},"evidence":{"type":"array","items":{"type":"object","properties":{"kind":{"type":"string","enum":["verification","diff","files","review","manual"]},"summary":{"type":"string"},"command":{"type":"string","description":"For verification: the command that proves it (must have run successfully in this session)."},"paths":{"type":"array","items":{"type":"string"},"description":"For diff/files: the paths that prove it."}}}}}},"unresolved":{"type":"array","items":{"type":"string"},"description":"Known gaps or follow-ups."}},"required":["status","summary"]}`)
}

func (completeSubtaskTool) ReadOnly() bool { return true }

func (completeSubtaskTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	r, err := parseCompletionReport(string(args))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("completion report accepted (status=%s, %d criteria). The host verifies claims against this session's receipts and may lower the verdict in the result returned to the parent.", r.Status, len(r.Criteria)), nil
}

// lastCompletionReport 取子会话里最后一次 complete_subtask 提交（逆序扫描；
// 无提交 ok=false）。
func lastCompletionReport(msgs []provider.Message) (completionReport, bool) {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role != provider.RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			if tc.Name != completeSubtaskName {
				continue
			}
			if r, err := parseCompletionReport(tc.Arguments); err == nil {
				return r, true
			}
		}
	}
	return completionReport{}, false
}

// receiptPair 一对（调用，结果）——回执=调用发生过且结果不是失败前缀。
type receiptPair struct {
	Name     string
	Command  string
	Path     string
	Failed   bool
	ExitFail bool
}

// collectReceipts 把子会话折叠成回执清单：assistant 的工具调用与其 tool 结果
// 按 ToolCallID 配对。
func collectReceipts(msgs []provider.Message) []receiptPair {
	results := map[string]provider.Message{}
	for _, m := range msgs {
		if m.Role == provider.RoleTool && m.ToolCallID != "" {
			results[m.ToolCallID] = m
		}
	}
	var out []receiptPair
	for _, m := range msgs {
		if m.Role != provider.RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			p := receiptPair{Name: tc.Name}
			var a struct {
				Command string `json:"command"`
			}
			if json.Unmarshal([]byte(tc.Arguments), &a) == nil {
				p.Command = strings.TrimSpace(a.Command)
			}
			p.Path = extractFilePath(tc.Name, tc.Arguments)
			if res, ok := results[tc.ID]; ok {
				p.Failed = isErrorMessage(res)
			} else {
				p.Failed = true // 调用无结果回执（被中断/取消）=不构成支撑
			}
			out = append(out, p)
		}
	}
	return out
}

// receiptPathMatch 路径匹配：相等或引用是目录（写/读发生在其下）。
func receiptPathMatch(receipt, cited string) bool {
	norm := func(p string) string {
		p = filepath.Clean(strings.Trim(p, `"`))
		if runtime.GOOS == "windows" {
			p = strings.ToLower(p)
		}
		return p
	}
	x, y := norm(receipt), norm(cited)
	return x == y || strings.HasPrefix(x, y+string(filepath.Separator))
}

// criterionBacked 判据是否被回执支撑（上游 criterionIsBacked 同构）：
// verification→同名命令成功跑过；diff→路径有成功写；files→路径有成功读或写；
// review/manual 永不自证。返回 (是否支撑, 缺失理由)。
func criterionBacked(c completionCriterion, receipts []receiptPair) (bool, string) {
	if len(c.Evidence) == 0 {
		return false, "claimed satisfied with no evidence"
	}
	var missing []string
	for _, e := range c.Evidence {
		switch strings.ToLower(strings.TrimSpace(e.Kind)) {
		case "verification":
			cmd := strings.TrimSpace(e.Command)
			backed := false
			if cmd != "" {
				for _, p := range receipts {
					if p.Name != "bash" || p.Failed || p.Command == "" {
						continue
					}
					if strings.Contains(p.Command, cmd) || strings.Contains(cmd, p.Command) {
						backed = true
						break
					}
				}
			}
			if !backed {
				missing = append(missing, fmt.Sprintf("no successful run of %q in this session", cmd))
			}
		case "diff":
			backed := false
			for _, p := range receipts {
				if p.Failed || !isWriteReceipt(p.Name) {
					continue
				}
				for _, cited := range e.Paths {
					if p.Path != "" && receiptPathMatch(p.Path, cited) {
						backed = true
					}
				}
			}
			if !backed {
				missing = append(missing, fmt.Sprintf("no successful write recorded for %v", e.Paths))
			}
		case "files":
			backed := false
			for _, p := range receipts {
				if p.Failed || (!isWriteReceipt(p.Name) && p.Name != "read_file") {
					continue
				}
				for _, cited := range e.Paths {
					if p.Path != "" && receiptPathMatch(p.Path, cited) {
						backed = true
					}
				}
			}
			if !backed {
				missing = append(missing, fmt.Sprintf("no successful read or write recorded for %v", e.Paths))
			}
		case "review", "manual":
			// gaea 子会话无 review 回执面；manual 按上游口径永不自证。
			missing = append(missing, fmt.Sprintf("%s evidence is never self-certifying", e.Kind))
		}
	}
	if len(missing) == 0 {
		return true, ""
	}
	return false, strings.Join(missing, "; ")
}

// isWriteReceipt 写族工具（与 batch_executor 冲突键同名单）。
func isWriteReceipt(name string) bool {
	switch name {
	case "edit_file", "write_file", "multi_edit", "edit_lines", "move_file":
		return true
	}
	return false
}

// adjudicateCompletion 把报告降到回执支撑的程度（上游 AdjudicateCompletion）：
// 引用判据无回执支撑降 unsatisfied；任一降级则 complete 整体降 partial，
// 理由逐条返回。partial/blocked/failed 不是完成主张，不裁决。
func adjudicateCompletion(r completionReport, receipts []receiptPair) (completionReport, []string) {
	if r.Status != "complete" {
		return r, nil
	}
	var reasons []string
	downgraded := false
	for i := range r.Criteria {
		c := &r.Criteria[i]
		if c.Status != "satisfied" {
			continue
		}
		if backed, why := criterionBacked(*c, receipts); !backed {
			c.Status = "unsatisfied"
			downgraded = true
			reasons = append(reasons, fmt.Sprintf("%s: %s", c.ID, why))
		}
	}
	if downgraded {
		r.Status = "partial"
	}
	return r, reasons
}

// childCompletionVerdict 收尾裁决入口：子会话无报告返回空串（不附裁语）；
// 有报告则附裁决结论（降级时逐条理由——父模型看到的是「为什么降」而非只看到降）。
func childCompletionVerdict(sess *Session) string {
	if sess == nil {
		return ""
	}
	report, ok := lastCompletionReport(sess.Messages)
	if !ok {
		return ""
	}
	receipts := collectReceipts(sess.Messages)
	final, reasons := adjudicateCompletion(report, receipts)
	if len(reasons) == 0 {
		return fmt.Sprintf("\n[completion verdict] status=%s accepted; %d criteria backed by session receipts.",
			final.Status, len(final.Criteria))
	}
	return fmt.Sprintf("\n[completion verdict] host lowered status %s → %s against this session's receipts:\n- %s",
		report.Status, final.Status, strings.Join(reasons, "\n- "))
}
