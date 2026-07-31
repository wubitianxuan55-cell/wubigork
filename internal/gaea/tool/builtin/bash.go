package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/wubigork/wubigork/internal/gaea/jobs"
	"github.com/wubigork/wubigork/internal/gaea/sandbox"
	"github.com/wubigork/wubigork/internal/gaea/tool"
)

const bashTimeout = 300 * time.Second

func init() { tool.RegisterBuiltin(bash{}) }

// bash runs a shell command with a timeout to avoid hangs. sb, when it enforces,
// wraps the command in an OS sandbox; the zero value registered at init runs
// unconfined and is overridden per run by ConfineBash. shell is the resolved
// interpreter (real bash, or PowerShell on a Windows host without bash); the
// zero value resolves lazily. workDir, when non-empty, is the directory the
// command runs in (cmd.Dir); empty uses the process cwd.
type bash struct {
	sb      sandbox.Spec
	shell   sandbox.Shell
	workDir string
}

func (bash) Name() string { return "bash" }

func (b bash) Description() string {
	if b.resolved().Kind == sandbox.ShellPowerShell {
		return "Execute a command in the shell and return combined stdout/stderr. " +
			"NOTE: bash is not available on this host — commands run under Windows PowerShell, " +
			"so write PowerShell syntax (e.g. $null not /dev/null; ';' or separate calls, not '&&'; " +
			"Get-ChildItem/Select-String, not ls/grep). Use for builds, tests, git, etc."
	}
	return "Execute a shell command. 5-minute timeout. For long-running commands, use run_in_background=true. Set output_format=json to get structured result with separated stdout/stderr fields."
}

// resolved returns the bound shell, resolving lazily for the zero-value instance
// (e.g. a registry that never went through ConfineBash).
func (b bash) resolved() sandbox.Shell {
	if b.shell.Path != "" {
		return b.shell
	}
	return sandbox.ResolveShell()
}

func (bash) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","description":"Shell command to execute"},"run_in_background":{"type":"boolean","description":"Run detached: returns a job id immediately and keeps running across turns."},"output_format":{"type":"string","enum":["plain","json"],"description":"plain (default) returns raw merged output. json returns structured {ok, exit_code, duration_ms, stdout, stderr, command} with separated stdout/stderr fields."}},"required":["command"]}`)
}

// ReadOnly is false: bash's effect cannot be inferred from args (rm, curl,
// git commit, etc. are all reachable). Conservative even when a particular
// command happens to be read-only — the agent batch decision can't tell.
func (bash) ReadOnly() bool { return false }

func (bash) CompactDescription() string     { return compactDesc["bash"] }
func (bash) CompactSchema() json.RawMessage { return compactSchema["bash"] }

func (b bash) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Command         string `json:"command"`
		RunInBackground bool   `json:"run_in_background"`
		OutputFormat    string `json:"output_format"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	sh := b.resolved()
	if !sh.SupportsChaining() && (hasUnquotedSeq(p.Command, "&&") || hasUnquotedSeq(p.Command, "||")) {
		return "", fmt.Errorf("this shell is Windows PowerShell, which does not parse '&&' or '||'. " +
			"Sequence with ';' (both run regardless of the first's result), use 'if ($?) { ... }' for " +
			"conditional chaining, or issue the commands as separate calls")
	}

	// Wrap in the OS sandbox when configured; otherwise argv is just the shell.
	argv, _ := sandbox.Command(b.sb, sh, p.Command)

	if p.RunInBackground {
		jm, ok := jobs.FromContext(ctx)
		if !ok {
			return "", fmt.Errorf("background execution is not available in this context")
		}
		workDir := b.workDir
		// The job runs under the manager's session context (no 120s timeout), so it
		// survives this turn; its combined output streams to the job buffer.
		job := jm.Start("bash", commandPreview(p.Command), func(jobCtx context.Context, out io.Writer) (string, error) {
			cmd := exec.CommandContext(jobCtx, argv[0], argv[1:]...)
			hideBashWindow(cmd) // Windows: 防止弹出 cmd 黑框
			cmd.Dir = workDir
			cmd.Stdout = out
			cmd.Stderr = out
			if err := cmd.Start(); err != nil {
				return "", err
			}
			// Record PID so kill_shell can fall back to taskkill /T on Windows.
			if id, ok := jobs.JobIDFromContext(jobCtx); ok {
				jm.SetPid(id, cmd.Process.Pid)
			}

			// V8.2: 后台任务也加上前台同款保护——jobCtx 取消时立刻强杀进程树，
			// 防止 cmd.Wait() 永久阻塞（软件启动后卡死或不正常退出）。
			go func() {
				<-jobCtx.Done()
				killProcessTree(cmd)
			}()

			// Try Windows Job Object for reliable process-tree cleanup.
			// When the job handle closes (defer), Windows kills all child/grandchild
			// processes recursively — even on kill_shell cancel or session close.
			job, jobErr := assignToJobObject(cmd)
			if jobErr == nil {
				defer syscall.CloseHandle(job)
			}
			err := cmd.Wait()
			if jobErr != nil {
				// Job Object failed (e.g. sandbox restriction); fall back to taskkill.
				killProcessTree(cmd)
			}
			return "", err
		})
		return fmt.Sprintf("Started background job %q. It keeps running across turns; read new output with bash_output(job_id=%q), wait for it with wait, or stop it with kill_shell(job_id=%q).", job.ID, job.ID, job.ID), nil
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, bashTimeout)
	defer cancel()

	cmd := exec.Command(argv[0], argv[1:]...)
	hideBashWindow(cmd) // Windows: 防止弹出 cmd 黑框
	cmd.Dir = b.workDir // "" lets exec use the process working directory

	// V10.5: json 模式下分离 stdout/stderr；plain 模式保持合并
	var stdoutBuf, stderrBuf bytes.Buffer
	if p.OutputFormat == "json" {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	} else {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stdoutBuf // merged in plain mode
	}

	err := cmd.Start()
	if err == nil {
		// earlyReturnCh 信号量：当检测到长期运行进程且决定提前返回时关闭，
		// 阻止 ctx 取消时 killProcessTree 误杀用户想保持运行的服务器进程。
		earlyReturnCh := make(chan struct{})

		go func() {
			select {
			case <-ctx.Done():
				select {
				case <-earlyReturnCh:
					// 早期返回——进程继续在后台运行，不杀
				default:
					killProcessTree(cmd)
				}
			}
		}()

		// Try Windows Job Object for reliable process-tree cleanup.
		// When the job handle closes (defer), Windows kills all child/grandchild
		// processes recursively — even on timeout or abrupt cancel.
		job, jobErr := assignToJobObject(cmd)
		if jobErr == nil {
			defer syscall.CloseHandle(job)
		}

		// ── 双路径等待：先等 8 秒，再判断是否长期运行进程 ──
		const earlyWait = 8 * time.Second
		waitCh := make(chan error, 1)
		go func() {
			waitCh <- cmd.Wait()
		}()

		select {
		case waitErr := <-waitCh:
			// 进程正常退出
			err = waitErr
		case <-time.After(earlyWait):
			// 8 秒后进程仍在运行 → 判断是否为长期运行进程
			output := stdoutBuf.String()
			if isLongRunningCommand(p.Command) || hasServerStartupOutput(output) {
				close(earlyReturnCh)

				// 收集已有输出，截断后返回
				if p.OutputFormat == "json" {
					stdoutStr := strings.TrimSpace(stdoutBuf.String())
					stderrStr := strings.TrimSpace(stderrBuf.String())
					const jsonStreamMaxBytes = 24 * 1024
					stdoutStr, _ = truncateStream(stdoutStr, jsonStreamMaxBytes)
					stderrStr, _ = truncateStream(stderrStr, jsonStreamMaxBytes)

					var buf2 bytes.Buffer
					enc := json.NewEncoder(&buf2)
					enc.SetEscapeHTML(false)
					result := map[string]any{
						"ok":          true,
						"running":     true,
						"exit_code":   0,
						"duration_ms": time.Since(start).Milliseconds(),
						"stdout":      stdoutStr,
						"stderr":      stderrStr,
						"command":     p.Command,
					}
					_ = enc.Encode(result)
					return strings.TrimSpace(buf2.String()), nil
				}

				// Plain mode
				out := stdoutBuf.String()
				const plainMaxBytes = 48 * 1024
				out, _ = truncateStream(out, plainMaxBytes)
				return out + "\n[进程仍在后台运行]", nil
			}

			// 不是长期运行进程——继续等待，直到进程退出或超时
			select {
			case waitErr := <-waitCh:
				err = waitErr
			case <-ctx.Done():
				err = ctx.Err()
			}
		case <-ctx.Done():
			err = ctx.Err()
		}

		// 进程已退出或 ctx 已取消——清理进程树
		if jobErr != nil {
			killProcessTree(cmd)
		}
	}

	// JSON output mode: return structured result with separated stdout/stderr.
	// Apply truncation to prevent large outputs from blowing up context window
	// (V10.12: previously JSON mode had NO truncation, risking massive blobs).
	if p.OutputFormat == "json" {
		ok := err == nil && ctx.Err() == nil
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		stdoutStr := strings.TrimSpace(stdoutBuf.String())
		stderrStr := strings.TrimSpace(stderrBuf.String())

		// Truncate each stream independently to ~24KB each (half of plain-mode 48KB).
		// Keeping both streams available is more useful than one large merged output.
		const jsonStreamMaxBytes = 24 * 1024
		stdoutStr, stdoutTrunc := truncateStream(stdoutStr, jsonStreamMaxBytes)
		stderrStr, stderrTrunc := truncateStream(stderrStr, jsonStreamMaxBytes)

		var buf2 bytes.Buffer
		enc := json.NewEncoder(&buf2)
		enc.SetEscapeHTML(false)
		result := map[string]any{
			"ok":          ok,
			"exit_code":   exitCode,
			"duration_ms": time.Since(start).Milliseconds(),
			"stdout":      stdoutStr,
			"stderr":      stderrStr,
			"command":     p.Command,
		}
		if stdoutTrunc {
			result["stdout_truncated"] = true
		}
		if stderrTrunc {
			result["stderr_truncated"] = true
		}
		_ = enc.Encode(result)
		return strings.TrimSpace(buf2.String()), nil
	}

	// Plain mode: merged output — apply same truncation as JSON mode for safety.
	out := stdoutBuf.String()
	const plainMaxBytes = 48 * 1024
	out, _ = truncateStream(out, plainMaxBytes)

	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("command timed out (> %s)", bashTimeout)
	}
	if err != nil {
		// Non-zero exit: feed output and error back so the model can self-correct.
		return out, fmt.Errorf("command exited: %w", err)
	}
	return out, nil
}

// hasUnquotedSeq reports whether seq appears in s outside any single- or
// double-quoted span, so a literal "a && b" string argument doesn't trip the
// PowerShell chaining guard.
func hasUnquotedSeq(s, seq string) bool {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if strings.HasPrefix(s[i:], seq) {
			return true
		}
	}
	return false
}

// commandPreview is a short single-line label for a background bash job, surfaced
// in the status bar and completion notices.
func commandPreview(cmd string) string {
	cmd = strings.TrimSpace(strings.ReplaceAll(cmd, "\n", " "))
	const max = 48
	r := []rune(cmd)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return cmd
}

// isLongRunningCommand 检测命令是否为服务器/长期运行进程。
// 匹配已知的 Dev 服务器、GUI 启动、文件监听等不自动退出的模式。
func isLongRunningCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	// 常见长期运行命令前缀
	longRunningPatterns := []string{
		"wails dev",
		"wails serve",
		"npm start",
		"npm run dev",
		"npm run serve",
		"npx vite",
		"npx next",
		"npx tsx watch",
		"pnpm dev",
		"pnpm start",
		"yarn dev",
		"yarn start",
		"go run",
		"start-process",
		"python -m http.server",
		"python -m flask",
		"python -m uvicorn",
		"python -m fastapi",
	}

	for _, pattern := range longRunningPatterns {
		if strings.HasPrefix(lower, pattern) {
			return true
		}
	}

	// Windows start 命令（启动新窗口运行程序）
	if strings.HasPrefix(lower, "start ") {
		return true
	}

	return false
}

// hasServerStartupOutput 检查输出中是否包含服务器启动特征。
// 用于判断一个不确定的命令是否已成功启动了服务。
func hasServerStartupOutput(output string) bool {
	lower := strings.ToLower(output)

	indicators := []string{
		"listening on",
		"serving at",
		"localhost:",
		"127.0.0.1:",
		"0.0.0.0:",
		"vite v", // Vite 开发服务器
		"compiled successfully",
		"press ctrl+c",
		"press ctrl-c",
		"server started",
		"server running",
		"running on",
		"started on port",
	}

	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}

	// http:// 且带端口号（如 http://localhost:5173）
	if strings.Contains(lower, "http://") {
		// 粗略检查后面有端口号
		idx := strings.Index(lower, "http://")
		rest := lower[idx+7:]
		colonIdx := strings.Index(rest, ":")
		if colonIdx > 0 && colonIdx < 20 {
			// 确认冒号后跟数字（端口）
			afterColon := rest[colonIdx+1:]
			if len(afterColon) > 0 && afterColon[0] >= '0' && afterColon[0] <= '9' {
				return true
			}
		}
	}

	return false
}

// killProcessTree 在命令执行完毕后清理 shell 可能残留的子进程树。
// Windows 上 shell 内部的 & 后台进程不会随 shell 退出而终止，
// taskkill /T 递归终止整个进程树避免孤儿进程和 wait 死锁。
func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS != "windows" {
	}
	if runtime.GOOS != "windows" {
		return
	}
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	killCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	killCmd.Stdout = io.Discard
	killCmd.Stderr = io.Discard
	_ = killCmd.Run() // 忽略错误（进程可能已正常退出）
}

// truncateStream applies head+tail truncation to a command output stream.
// Keeps the first N bytes and last N bytes, eliding the middle. Returns the
// truncated string and a boolean indicating whether truncation occurred.
// Uses simple byte-length truncation (not line-aware) for predictable sizing.
func truncateStream(s string, maxBytes int) (string, bool) {
	if len(s) <= maxBytes {
		return s, false
	}
	// ceil division: (maxBytes+1)/2 so an odd maxBytes doesn't lose a byte
	half := (maxBytes + 1) / 2
	// Adjust half to a valid UTF-8 boundary so we don't split multi-byte runes.
	for half > 0 && half < len(s) && !utf8.RuneStart(s[half]) {
		half--
	}
	head := s[:half]
	tailStart := len(s) - half
	if tailStart <= half {
		tailStart = half // prevent head/tail overlap when just barely over maxBytes
	}
	// Adjust tailStart to a valid UTF-8 boundary.
	for tailStart < len(s) && !utf8.RuneStart(s[tailStart]) {
		tailStart++
	}
	tail := s[tailStart:]
	result := head + fmt.Sprintf("\n... (%d bytes elided) ...\n", len(s)-maxBytes) + tail
	// If truncation hint makes the result longer than the original (input just
	// barely over maxBytes), return the original — truncation would be harmful.
	if len(result) >= len(s) {
		return s, false
	}
	return result, true
}
