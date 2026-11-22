package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// bash_output / kill_shell / wait operate the background jobs registered by
// bash(run_in_background) and task(run_in_background). They reach the session's
// job manager through the call context (jobs.FromContext) — the agent stamps it
// onto every tool call — and degrade to a clear error when it isn't available
// (a headless context with no manager). Together they poll a job's new output,
// terminate a job, and block until jobs finish.

func init() {
	tool.RegisterBuiltin(bashOutput{})
	tool.RegisterBuiltin(killShell{})
	tool.RegisterBuiltin(waitJob{})
}

// --- bash_output: poll a background job's new output (non-blocking) ---

type bashOutput struct{}

func (bashOutput) Name() string { return "bash_output" }

func (bashOutput) Description() string {
	return "Read new output from a background job. Set tail_lines to limit the output to the last N lines (max 500). Does not block."
}

func (bashOutput) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"job_id":{"type":"string","description":"The background job id (e.g. \"bash-1\") returned when it was started."},"filter":{"type":"string","description":"Optional regular expression; only matching lines of the new output are returned."},"tail_lines":{"type":"integer","description":"Return only the last N lines of new output (default 0 = all lines, max 500)."}},"required":["job_id"]}`)
}

func (bashOutput) ReadOnly() bool { return true }

func (bashOutput) CompactDescription() string     { return compactDesc["bash_output"] }
func (bashOutput) CompactSchema() json.RawMessage { return compactSchema["bash_output"] }

func (bashOutput) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		JobID     string `json:"job_id"`
		Filter    string `json:"filter"`
		TailLines int    `json:"tail_lines"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.JobID == "" {
		return "", fmt.Errorf("job_id is required")
	}
	jm, ok := jobs.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("background jobs are not available in this context")
	}
	text, status, found := jm.Output(p.JobID)
	if !found {
		return "", fmt.Errorf("no background job %q", p.JobID)
	}
	if p.Filter != "" && text != "" {
		filtered, err := filterLines(text, p.Filter)
		if err != nil {
			return "", err
		}
		text = filtered
	}
	// V10.12: tail_lines limits output to last N lines (clamped 1..500).
	if p.TailLines > 0 && text != "" {
		if p.TailLines > 500 {
			p.TailLines = 500
		}
		lines := strings.Split(text, "\n")
		if len(lines) > p.TailLines {
			text = fmt.Sprintf("... (showing last %d of %d lines) ...\n%s",
				p.TailLines, len(lines),
				strings.Join(lines[len(lines)-p.TailLines:], "\n"))
		}
	}
	header := fmt.Sprintf("[%s] %s", p.JobID, status)
	if strings.TrimSpace(text) == "" {
		return header + "\n(no new output)", nil
	}
	return header + "\n" + text, nil
}

// filterLines keeps only the lines of s matching the regular expression re.
func filterLines(s, re string) (string, error) {
	rx, err := regexp.Compile(re)
	if err != nil {
		return "", fmt.Errorf("invalid filter regexp: %w", err)
	}
	var keep []string
	for _, line := range strings.Split(s, "\n") {
		if rx.MatchString(line) {
			keep = append(keep, line)
		}
	}
	return strings.Join(keep, "\n"), nil
}

// --- kill_shell: terminate a running background job ---

type killShell struct{}

func (killShell) Name() string { return "kill_shell" }

func (killShell) Description() string {
	return "Terminate a running background job (bash or task) started with run_in_background. A no-op if the job has already finished or the id is unknown."
}

func (killShell) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"job_id":{"type":"string","description":"The background job id to terminate (e.g. \"bash-1\")."}},"required":["job_id"]}`)
}

func (killShell) ReadOnly() bool { return false }

func (killShell) CompactDescription() string     { return compactDesc["kill_shell"] }
func (killShell) CompactSchema() json.RawMessage { return compactSchema["kill_shell"] }

func (killShell) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.JobID == "" {
		return "", fmt.Errorf("job_id is required")
	}
	jm, ok := jobs.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("background jobs are not available in this context")
	}
	// 1. Cancel the job context (kills the immediate process).
	jm.Kill(p.JobID)

	// 2. On Windows, also attempt taskkill /T /F to handle orphaned child processes
	//    (e.g. Python/Node servers started by the background shell).
	killOutput := ""
	if runtime.GOOS == "windows" {
		if pid := jm.Pid(p.JobID); pid > 0 {
			killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
			hideBashWindow(killCmd) // 防止 taskkill 弹黑框（非 Windows 为空操作）
			killCmd.Stdout = io.Discard
			killCmd.Stderr = io.Discard
			if err := killCmd.Run(); err == nil {
				killOutput = " process tree terminated."
			}
		}
	}
	return fmt.Sprintf("Killed background job %q.%s", p.JobID, killOutput), nil
}

// --- wait: block until background jobs finish, then return their results ---

// subagentWaiter wait 收 sa_/mt_ 引用的注入缝（P1-7，2026-11-21 沿江反馈：wait
// 传 sa_ 报 No background jobs to wait for，主控只能 sleep 轮询撞 5 分钟硬超时）。
// builtin 不能反向 import agent（环）——boot 装配点注入 agent.WaitSubagents；
// 未接线保持旧行为（只认 bash job ids）。
var subagentWaiter func(ctx context.Context, refs []string, timeoutSec int) []SubagentWaitRow

// SubagentWaitRow 注入缝的行契约（agent.SubagentWaitResult 的镜像，避免环）。
type SubagentWaitRow struct {
	Ref    string
	Status string
	Output string
}

// SetSubagentWaiter boot 装配点接线（agent 包提供实现）。
func SetSubagentWaiter(f func(ctx context.Context, refs []string, timeoutSec int) []SubagentWaitRow) {
	subagentWaiter = f
}

type waitJob struct{}

func (waitJob) Name() string { return "wait" }

func (waitJob) Description() string {
	return "Block until background jobs finish, then return each job's status and final output/answer. Use to collect the result of a task(run_in_background) or bash(run_in_background) before continuing. Omit job_ids to wait for every running job. job_ids 也接受子代理引用（sa_…/mt_…）——阻塞等它们到终态（completed/failed/aborted），完成后用 read_subagent_result 取结果；进度可先看 subagent_list 的 last: 动作行。"
}

func (waitJob) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"job_ids":{"type":"array","items":{"type":"string"},"description":"Background job ids 或子代理引用（sa_…/mt_…）to wait for. Omit to wait for every currently-running job."},"timeout_seconds":{"type":"integer","description":"Optional maximum seconds to block before returning current progress. Omit to wait until the jobs finish.","minimum":1}}}`)
}

func (waitJob) ReadOnly() bool { return true }

func (waitJob) CompactDescription() string     { return compactDesc["wait"] }
func (waitJob) CompactSchema() json.RawMessage { return compactSchema["wait"] }

// isSubagentRefShape sa_/mt_ 前缀判定（与 agent.IsSubagentRef 同口径的本地镜像，
// 避免环；接线缝在场才分流）。
func isSubagentRefShape(ref string) bool {
	return len(ref) > 3 && (ref[:3] == "sa_" || ref[:3] == "mt_")
}

func (waitJob) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		JobIDs         []string `json:"job_ids"`
		TimeoutSeconds int      `json:"timeout_seconds"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
	}
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = 300 // 默认 5 分钟，防止无限等待
	}
	// P1-7 分流：sa_/mt_ 引用走子代理等待面（阻塞到终态），其余走 job 管理器。
	var saRefs, jobIDs []string
	for _, id := range p.JobIDs {
		if subagentWaiter != nil && isSubagentRefShape(id) {
			saRefs = append(saRefs, id)
		} else {
			jobIDs = append(jobIDs, id)
		}
	}
	var parts []string
	if len(jobIDs) > 0 || len(p.JobIDs) == 0 {
		jm, ok := jobs.FromContext(ctx)
		if !ok && len(p.JobIDs) > 0 {
			return "", fmt.Errorf("background jobs are not available in this context")
		}
		if ok {
			results := jm.Wait(ctx, jobIDs, timeout)
			for _, r := range results {
				label := r.ID
				if r.Label != "" {
					label = fmt.Sprintf("%s (%s)", r.ID, r.Label)
				}
				line := fmt.Sprintf("[%s] %s", label, r.Status)
				if strings.TrimSpace(r.Output) != "" {
					line += "\n" + r.Output
				}
				parts = append(parts, line)
			}
		}
	}
	if len(saRefs) > 0 && subagentWaiter != nil {
		for _, r := range subagentWaiter(ctx, saRefs, timeout) {
			line := fmt.Sprintf("[%s] %s", r.Ref, r.Status)
			if strings.TrimSpace(r.Output) != "" {
				line += "\n" + r.Output
			}
			parts = append(parts, line)
		}
	}
	if len(parts) == 0 {
		return "No background jobs to wait for.", nil
	}
	return strings.Join(parts, "\n\n"), nil
}
