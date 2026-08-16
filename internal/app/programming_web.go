package app

// 编程板块：DeepSeek Harness Web 进程管理。
//
// dsh（DeepSeek Harness）Web UI 默认服务在 http://127.0.0.1:3080。
// gaea 负责三件事：
//   - GetProgrammingWebStatus：探测端口是否在服务 + 记录自启进程 pid（owned 语义）
//     + 自启实例运行时长（uptime_s，非自启/未运行恒 0）；
//   - GetProgrammingWebPreflight：启动前置条件逐项检查（harness 目录有效 / pnpm
//     可用 / 依赖已装 / Web 构建产物就绪 / 端口空闲），启动引导视图据此渲染真实清单；
//   - StartProgrammingWeb：在 harness 根目录启动 `pnpm dsh web`（已运行幂等返回）；
//   - StopProgrammingWeb：只终止 gaea 自启的实例（外部实例不误杀，仅提示）；
//   - ProgrammingWebLogTail：读取 gaea 自启 dsh web 日志尾部（启动引导/运行中可查看）。
//
// harness 根目录默认 C:\AI\deepseek-harness，可用环境变量 GAEA_HARNESS_DIR 覆盖。
// 外部副作用（端口探测/tasklist/taskkill/LookPath/日志路径）全部经 probe* 可替换
// 探针，同包测试注入确定性实现（见 programming_web_test.go）。

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultHarnessRoot = `C:\AI\deepseek-harness`
	harnessWebPort     = 3080
	harnessWebURL      = "http://127.0.0.1:3080"
)

// harnessWebMu 保护 gaea 自启 dsh web 进程记录（owned 判定 + 停止只杀自启实例）
// 与自启时刻（uptime 计算）。
var (
	harnessWebMu        sync.Mutex
	harnessWebPID       int
	harnessWebStartedAt time.Time
)

// ── 可替换外部探针（同包测试注入；生产路径 = 真实实现） ──────────────────
var (
	probePortOpen = harnessPortOpen
	probePIDAlive = pidAlive
	probeLookPath = exec.LookPath
	probeKillTree = func(pid int) error {
		return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
	}
	// harnessLogPathFn 可替换（测试注入临时路径）；生产 = 系统临时目录。
	harnessLogPathFn = func() string {
		return filepath.Join(os.TempDir(), "gaea-dsh-web.log")
	}
	// probeStartCmd 可替换（测试不真起进程）；生产 = 真实 cmd.Start()。
	probeStartCmd = func(cmd *exec.Cmd) error { return cmd.Start() }
	// harnessWaitTimeout 端口就绪等待上限（测试可缩短，生产 20s）。
	harnessWaitTimeout = 20 * time.Second
)

// harnessRoot 返回 DeepSeek Harness 仓库根目录（环境变量可覆盖）。
func harnessRoot() string {
	if v := strings.TrimSpace(os.Getenv("GAEA_HARNESS_DIR")); v != "" {
		return v
	}
	return defaultHarnessRoot
}

// harnessLogPath 返回 gaea 自启 dsh web 的日志文件路径（GetStatus 与启动共用）。
func harnessLogPath() string {
	return harnessLogPathFn()
}

// harnessPortOpen 探测 3080 端口是否有进程在监听。
func harnessPortOpen() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", harnessWebPort), 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// pidAlive Windows 下用 tasklist 判断进程是否存活（os.FindProcess 无实际探测能力）。
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid)).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}

// dirExists 判断路径是否为存在的目录。
func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// tailLines 取文本最后 n 行（按 \n 切分，忽略末尾空行，兼容 CRLF；UTF-8 安全）。
func tailLines(s string, n int) []string {
	lines := strings.Split(s, "\n")
	// 去掉末尾空段（结尾换行产生）与行尾 \r（CRLF）。
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// GetProgrammingWebStatus 返回 dsh web 运行状态（running/owned/pid/url/root/
// log/uptime_s）。
func (a *App) GetProgrammingWebStatus() map[string]interface{} {
	harnessWebMu.Lock()
	pid := harnessWebPID
	started := harnessWebStartedAt
	harnessWebMu.Unlock()
	running := probePortOpen()
	owned := pid > 0 && probePIDAlive(pid)
	uptime := int64(0)
	if owned && running && !started.IsZero() {
		uptime = int64(time.Since(started).Seconds())
		if uptime < 0 {
			uptime = 0
		}
	}
	return map[string]interface{}{
		"running":  running,
		"owned":    owned,
		"pid":      pid,
		"url":      harnessWebURL,
		"root":     harnessRoot(),
		"log":      harnessLogPath(),
		"uptime_s": uptime,
	}
}

// GetProgrammingWebPreflight 返回 dsh web 启动前置条件逐项检查结果。
// 前端启动引导视图据此渲染绿/红清单（替代静态使用说明）；all_ready 为真
// 才建议一键启动，单项失败给出可操作的修复方向。
func (a *App) GetProgrammingWebPreflight() map[string]interface{} {
	root := harnessRoot()
	harnessValid := fileExists(filepath.Join(root, "package.json"))
	_, pnpmErr := probeLookPath("pnpm")
	pnpmFound := pnpmErr == nil
	depsReady := dirExists(filepath.Join(root, "node_modules"))
	buildReady := fileExists(filepath.Join(root, "apps", "web", "dist", "index.html"))
	portFree := !probePortOpen()
	return map[string]interface{}{
		"harness_valid": harnessValid,
		"pnpm_found":    pnpmFound,
		"deps_ready":    depsReady,
		"build_ready":   buildReady,
		"port_free":     portFree,
		"all_ready":     harnessValid && pnpmFound && depsReady && buildReady && portFree,
		"root":          root,
	}
}

// StartProgrammingWeb 启动 dsh web（已运行幂等返回；外部占用 3080 时明确报错不抢端口）。
func (a *App) StartProgrammingWeb() error {
	harnessWebMu.Lock()
	defer harnessWebMu.Unlock()

	if probePortOpen() {
		if harnessWebPID > 0 && probePIDAlive(harnessWebPID) {
			return nil // 自启实例已在服务，幂等
		}
		return fmt.Errorf("端口 %d 已被其他进程占用（非 gaea 自启实例），请先手动停止该进程", harnessWebPort)
	}

	root := harnessRoot()
	if fi, err := os.Stat(filepath.Join(root, "package.json")); err != nil || fi.IsDir() {
		return fmt.Errorf("找不到 DeepSeek Harness（%s），请确认目录存在或用 GAEA_HARNESS_DIR 指定", root)
	}
	pnpm, err := probeLookPath("pnpm")
	if err != nil {
		return fmt.Errorf("未找到 pnpm，请先安装 Node.js ≥22 并启用 corepack（corepack enable）")
	}

	logPath := harnessLogPath()
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("无法打开 dsh web 日志 %s: %w", logPath, err)
	}
	defer logFile.Close()

	cmd := exec.Command(pnpm, "dsh", "web")
	cmd.Dir = root
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := probeStartCmd(cmd); err != nil {
		return fmt.Errorf("启动 dsh web 失败: %w", err)
	}
	harnessWebPID = cmd.Process.Pid
	harnessWebStartedAt = time.Now()

	// 等待端口就绪（最长 harnessWaitTimeout），避免「点完没反应」。
	deadline := time.Now().Add(harnessWaitTimeout)
	for time.Now().Before(deadline) {
		if probePortOpen() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("dsh web 进程已启动（pid=%d）但端口未在 20s 内就绪，请查看日志：%s", cmd.Process.Pid, logPath)
}

// StopProgrammingWeb 仅停止 gaea 自启的 dsh web 实例（外部实例不误杀）。
func (a *App) StopProgrammingWeb() error {
	harnessWebMu.Lock()
	defer harnessWebMu.Unlock()

	if harnessWebPID <= 0 {
		if probePortOpen() {
			return fmt.Errorf("端口 %d 有外部实例在运行（非 gaea 自启），为避免误杀请手动停止", harnessWebPort)
		}
		return nil // 本就没在跑
	}
	pid := harnessWebPID
	harnessWebPID = 0
	harnessWebStartedAt = time.Time{}
	if !probePIDAlive(pid) {
		return nil
	}
	// Windows：taskkill /T 连 pnpm→node 子进程树一起终止。
	if err := probeKillTree(pid); err != nil {
		return fmt.Errorf("停止 dsh web（pid=%d）失败: %w", pid, err)
	}
	return nil
}

// ProgrammingWebLogTail 返回 gaea 自启 dsh web 日志尾部（最多 n 行，
// n 钳制 [1,200]，默认 50）；日志文件尚未生成时 exists=false + 提示文案。
func (a *App) ProgrammingWebLogTail(n int) map[string]interface{} {
	if n <= 0 {
		n = 50
	}
	if n > 200 {
		n = 200
	}
	path := harnessLogPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]interface{}{
			"exists": false,
			"path":   path,
			"lines":  []string{},
			"error":  "日志文件尚未生成（第一次启动后出现）",
		}
	}
	return map[string]interface{}{
		"exists": true,
		"path":   path,
		"lines":  tailLines(string(raw), n),
		"error":  "",
	}
}
