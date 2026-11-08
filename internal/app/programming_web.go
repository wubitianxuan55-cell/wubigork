package app

// Programming board: managed DeepSeek Harness Web inside gaea.
//
// The Web runtime is intentionally isolated from the official desktop app:
//   root: C:\AI\dsh
//   home: C:\AI\dsh\home
//   port: 3091
//
// gaea starts the npm CLI directly (not start-web.cmd), injects only the four
// isolated path variables, captures the one-time process-token URL printed by
// DSH, and exposes that URL to the embedded ProgrammingPage iframe.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/proc"
)

const (
	defaultProgrammingWebRoot    = `C:\AI\dsh`
	programmingWebPort           = 3091
	programmingWebBaseURL        = "http://dsh.wails.localhost:3091"
	programmingWebTrustedHost    = "dsh.wails.localhost:3091"
	programmingWebExpected       = "0.2.2-rc.1"
	programmingWebReadyTimeout   = 30 * time.Second
	programmingWebMaxLogTailLine = 200
)

var (
	programmingWebMu           sync.Mutex
	programmingWebCmd          *exec.Cmd
	programmingWebJob          uintptr
	programmingWebPID          int
	programmingWebStartedAt    time.Time
	programmingWebToken        string
	programmingWebStarting     bool
	programmingWebLogWriteMu   sync.Mutex
	programmingWebTokenPattern = regexp.MustCompile(`https?://[^\s]+\?token=([A-Za-z0-9._~-]+)`)
)

var (
	programmingWebPortOpenFn = programmingWebPortOpen
	programmingWebPIDAliveFn = programmingWebPIDAlive
)

func programmingWebRoot() string {
	if v := strings.TrimSpace(os.Getenv("GAEA_DSH_ROOT")); v != "" {
		return filepath.Clean(v)
	}
	return defaultProgrammingWebRoot
}

func programmingWebHome(root string) string {
	if v := strings.TrimSpace(os.Getenv("GAEA_DSH_HOME")); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(root, "home")
}

func programmingWebAgentsHome(root string) string {
	if v := strings.TrimSpace(os.Getenv("GAEA_DSH_AGENTS_HOME")); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(root, "home", "agents")
}

func programmingWebBase(root string) string {
	return filepath.Join(root, "node_modules", "@deepseek-ai")
}

func programmingWebEntry(root string) string {
	return filepath.Join(programmingWebBase(root), "dsh", "lib", "bin.js")
}

func programmingWebLogPath(root string) string {
	if v := strings.TrimSpace(os.Getenv("GAEA_DSH_WEB_LOG")); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(programmingWebHome(root), "dsh-web.log")
}

func programmingWebPortOpen() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", programmingWebPort), 350*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func programmingWebPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS != "windows" {
		p, err := os.FindProcess(pid)
		return err == nil && p != nil
	}
	out, err := programmingWebPIDProbe(pid).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}

// programmingWebPIDProbe builds the liveness probe for the owned DSH Web
// process. It runs on every status poll (ProgrammingPage refreshes each 3.5s),
// so it must carry CREATE_NO_WINDOW — a bare console-subsystem spawn from a
// GUI process allocates a visible console and flashes a cmd window per poll.
func programmingWebPIDProbe(pid int) *exec.Cmd {
	probe := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
	proc.HideWindow(probe)
	return probe
}

func programmingWebOwnedLocked() bool {
	return programmingWebCmd != nil && programmingWebPID > 0 && programmingWebPIDAliveFn(programmingWebPID)
}

func programmingWebURL(token string) string {
	if strings.TrimSpace(token) == "" {
		return programmingWebBaseURL + "/"
	}
	return programmingWebBaseURL + "/?token=" + token
}

func programmingWebDshVersion(root string) string {
	raw, err := os.ReadFile(filepath.Join(programmingWebBase(root), "dsh", "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return ""
	}
	return strings.TrimSpace(pkg.Version)
}

func programmingWebREAReady(root, home, agentsHome string) bool {
	config, err := os.ReadFile(filepath.Join(home, "cordis.patch.yml"))
	if err != nil || !strings.Contains(string(config), "serverName: rea") {
		return false
	}
	server := filepath.Join(root, "rea", "node_modules", "rea-agents", "scripts", "rea.mjs")
	skill := filepath.Join(agentsHome, "skills", "reverse-engineer-anything", "SKILL.md")
	return fileExists(server) && fileExists(skill)
}

func programmingWebWithin(child, root string) bool {
	rel, err := filepath.Rel(root, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// GetProgrammingWebStatus reports liveness and returns the tokenized iframe URL
// only for the process owned by this gaea instance.
func (a *App) GetProgrammingWebStatus() map[string]interface{} {
	programmingWebMu.Lock()
	pid := programmingWebPID
	token := programmingWebToken
	started := programmingWebStartedAt
	starting := programmingWebStarting
	owned := programmingWebOwnedLocked()
	programmingWebMu.Unlock()

	running := programmingWebPortOpenFn()
	if !running {
		owned = false
	}
	uptime := int64(0)
	if owned && !started.IsZero() {
		uptime = int64(time.Since(started).Seconds())
		if uptime < 0 {
			uptime = 0
		}
	}
	url := programmingWebBaseURL + "/"
	if owned {
		url = programmingWebURL(token)
	}
	root := programmingWebRoot()
	return map[string]interface{}{
		"running":  running,
		"owned":    owned,
		"starting": starting,
		"pid":      pid,
		"url":      url,
		"root":     root,
		"home":     programmingWebHome(root),
		"version":  programmingWebDshVersion(root),
		"log":      programmingWebLogPath(root),
		"uptime_s": uptime,
	}
}

// GetProgrammingWebPreflight validates the isolated install before starting.
func (a *App) GetProgrammingWebPreflight() map[string]interface{} {
	root := programmingWebRoot()
	home := programmingWebHome(root)
	agentsHome := programmingWebAgentsHome(root)
	base := programmingWebBase(root)
	nodePath, nodeErr := exec.LookPath("node")
	version := programmingWebDshVersion(root)
	installReady := fileExists(programmingWebEntry(root))
	profileReady := fileExists(filepath.Join(home, "profiles", "web", "package.json"))
	pluginReady := fileExists(filepath.Join(home, "profiles", "web", "node_modules", "dsh-wubi", "package.json"))
	reaReady := programmingWebREAReady(root, home, agentsHome)
	isolated := programmingWebWithin(home, root) &&
		programmingWebWithin(agentsHome, root) &&
		programmingWebWithin(base, root)
	portFree := !programmingWebPortOpenFn()
	versionOK := version == programmingWebExpected
	allReady := nodeErr == nil && installReady && profileReady && pluginReady && reaReady &&
		isolated && versionOK && portFree
	return map[string]interface{}{
		"node_found":    nodeErr == nil,
		"node_path":     nodePath,
		"install_ready": installReady,
		"profile_ready": profileReady,
		"plugin_ready":  pluginReady,
		"rea_ready":     reaReady,
		"isolated":      isolated,
		"version_ok":    versionOK,
		"port_free":     portFree,
		"all_ready":     allReady,
		"root":          root,
		"home":          home,
		"agents_home":   agentsHome,
		"base":          base,
		"version":       version,
		"expected":      programmingWebExpected,
	}
}

func programmingWebEnv(root string) []string {
	overrides := map[string]string{
		"DSH_HOME":               programmingWebHome(root),
		"DSH_AGENTS_HOME":        programmingWebAgentsHome(root),
		"DSH_BASE":               programmingWebBase(root),
		"DSH_WUBI_ISOLATED_ROOT": root,
	}
	out := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, replaced := overrides[key]; replaced {
				continue
			}
		}
		out = append(out, entry)
	}
	for key, value := range overrides {
		out = append(out, key+"="+value)
	}
	return out
}

func extractProgrammingWebToken(line string) string {
	match := programmingWebTokenPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func writeProgrammingWebLog(logFile *os.File, line, token string) {
	if logFile == nil || line == "" {
		return
	}
	safe := line
	if token != "" {
		safe = strings.ReplaceAll(safe, token, "<redacted>")
	}
	programmingWebLogWriteMu.Lock()
	_, _ = io.WriteString(logFile, safe)
	programmingWebLogWriteMu.Unlock()
}

func scanProgrammingWebOutput(r io.Reader, logFile *os.File, tokenCh chan<- string, onToken func(string)) {
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			token := extractProgrammingWebToken(line)
			writeProgrammingWebLog(logFile, line, token)
			if token != "" {
				onToken(token)
				select {
				case tokenCh <- token:
				default:
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// StartProgrammingWeb starts the isolated DSH Web. It is idempotent while the
// owned process or an in-progress start is active.
func (a *App) StartProgrammingWeb() error {
	programmingWebMu.Lock()
	if programmingWebStarting || programmingWebOwnedLocked() {
		programmingWebMu.Unlock()
		return nil
	}
	if programmingWebPortOpenFn() {
		programmingWebMu.Unlock()
		return fmt.Errorf("端口 %d 已被非 gaea 实例占用；为避免与桌面版或外部 Web 冲突，请先停止该进程", programmingWebPort)
	}
	programmingWebStarting = true
	programmingWebMu.Unlock()

	defer func() {
		programmingWebMu.Lock()
		programmingWebStarting = false
		programmingWebMu.Unlock()
	}()

	pre := a.GetProgrammingWebPreflight()
	if ready, _ := pre["all_ready"].(bool); !ready {
		return fmt.Errorf(
			"DeepSeek Harness 隔离环境未就绪：install=%v profile=%v plugin=%v rea=%v isolated=%v version=%v port_free=%v",
			pre["install_ready"], pre["profile_ready"], pre["plugin_ready"], pre["rea_ready"], pre["isolated"],
			pre["version_ok"], pre["port_free"],
		)
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return fmt.Errorf("未找到 Node.js：%w", err)
	}
	root := programmingWebRoot()
	logPath := programmingWebLogPath(root)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("创建 DSH Web 日志目录失败: %w", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("打开 DSH Web 日志失败: %w", err)
	}

	cmd := exec.Command(
		nodePath,
		programmingWebEntry(root),
		"web",
		"--no-open",
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(programmingWebPort),
		"--public-url", programmingWebBaseURL+"/",
		"--trusted-host", programmingWebTrustedHost,
	)
	cmd.Dir = root
	cmd.Env = programmingWebEnv(root)
	proc.HideWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = logFile.Close()
		return fmt.Errorf("接管 DSH Web stdout 失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = logFile.Close()
		return fmt.Errorf("接管 DSH Web stderr 失败: %w", err)
	}

	job, err := proc.StartTracked(cmd)
	if err != nil {
		_ = logFile.Close()
		return fmt.Errorf("启动 DSH Web 失败: %w", err)
	}
	pid := cmd.Process.Pid

	programmingWebMu.Lock()
	programmingWebCmd = cmd
	programmingWebJob = job
	programmingWebPID = pid
	programmingWebStartedAt = time.Now()
	programmingWebToken = ""
	programmingWebMu.Unlock()

	tokenCh := make(chan string, 1)
	waitCh := make(chan error, 1)
	var scanners sync.WaitGroup
	scanners.Add(2)

	onToken := func(token string) {
		programmingWebMu.Lock()
		if programmingWebCmd == cmd {
			programmingWebToken = token
		}
		programmingWebMu.Unlock()
	}
	go func() {
		defer scanners.Done()
		scanProgrammingWebOutput(stdout, logFile, tokenCh, onToken)
	}()
	go func() {
		defer scanners.Done()
		scanProgrammingWebOutput(stderr, logFile, tokenCh, onToken)
	}()
	go func() {
		waitErr := cmd.Wait()
		scanners.Wait()
		proc.KillTracked(cmd, job)
		_ = logFile.Close()

		programmingWebMu.Lock()
		if programmingWebCmd == cmd {
			programmingWebCmd = nil
			programmingWebJob = 0
			programmingWebPID = 0
			programmingWebStartedAt = time.Time{}
			programmingWebToken = ""
		}
		programmingWebMu.Unlock()
		waitCh <- waitErr
	}()

	timer := time.NewTimer(programmingWebReadyTimeout)
	defer timer.Stop()
	readyTick := time.NewTicker(200 * time.Millisecond)
	defer readyTick.Stop()
	sawToken := false
	for {
		select {
		case token := <-tokenCh:
			sawToken = token != ""
			if sawToken && programmingWebPortOpenFn() {
				return nil
			}
		case <-readyTick.C:
			if sawToken && programmingWebPortOpenFn() {
				return nil
			}
		case waitErr := <-waitCh:
			if waitErr == nil {
				waitErr = errors.New("进程提前退出")
			}
			return fmt.Errorf("DSH Web 启动失败（pid=%d）：%w；日志：%s", pid, waitErr, logPath)
		case <-timer.C:
			proc.KillTracked(cmd, job)
			return fmt.Errorf("DSH Web 已在 %s 内启动但未取得可用令牌（pid=%d）；日志：%s", programmingWebReadyTimeout, pid, logPath)
		}
	}
}

// StopProgrammingWeb stops only the gaea-owned process tree.
func (a *App) StopProgrammingWeb() error {
	programmingWebMu.Lock()
	cmd := programmingWebCmd
	job := programmingWebJob
	pid := programmingWebPID
	programmingWebMu.Unlock()

	if cmd == nil || pid <= 0 {
		if programmingWebPortOpenFn() {
			return fmt.Errorf("端口 %d 有非 gaea 实例在运行，为避免误杀请从对应启动器关闭", programmingWebPort)
		}
		return nil
	}
	proc.KillTracked(cmd, job)
	return nil
}

func programmingWebTailLines(s string, n int) []string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

func truncateProgrammingWebLogLine(line string, max int) string {
	if max <= 0 || utf8.RuneCountInString(line) <= max {
		return line
	}
	runes := []rune(line)
	return string(runes[:max]) + "…"
}

// ProgrammingWebLogTail returns the tail of the isolated Web log. Tokens are
// already redacted while the process output is captured.
func (a *App) ProgrammingWebLogTail(n int) map[string]interface{} {
	if n <= 0 {
		n = 50
	}
	if n > programmingWebMaxLogTailLine {
		n = programmingWebMaxLogTailLine
	}
	path := programmingWebLogPath(programmingWebRoot())
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]interface{}{
			"exists": false,
			"path":   path,
			"lines":  []string{},
			"error":  "日志文件尚未生成（第一次启动后出现）",
		}
	}
	lines := programmingWebTailLines(string(raw), n)
	for i, line := range lines {
		lines[i] = truncateProgrammingWebLogLine(line, 1200)
	}
	return map[string]interface{}{
		"exists": true,
		"path":   path,
		"lines":  lines,
		"error":  "",
	}
}
