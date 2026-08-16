package app

// 编程板块：DeepSeek Harness Web 进程管理。
//
// dsh（DeepSeek Harness）Web UI 默认服务在 http://127.0.0.1:3080。
// gaea 负责三件事：
//   - GetProgrammingWebStatus：探测端口是否在服务 + 记录自启进程 pid（owned 语义）；
//   - StartProgrammingWeb：在 harness 根目录启动 `pnpm dsh web`（已运行幂等返回）；
//   - StopProgrammingWeb：只终止 gaea 自启的实例（外部实例不误杀，仅提示）。
//
// harness 根目录默认 C:\AI\deepseek-harness，可用环境变量 GAEA_HARNESS_DIR 覆盖。

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

// harnessWebMu 保护 gaea 自启 dsh web 进程记录（owned 判定 + 停止只杀自启实例）。
var (
	harnessWebMu  sync.Mutex
	harnessWebPID int
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
	return filepath.Join(os.TempDir(), "gaea-dsh-web.log")
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

// GetProgrammingWebStatus 返回 dsh web 运行状态（running/owned/pid/url/root）。
func (a *App) GetProgrammingWebStatus() map[string]interface{} {
	harnessWebMu.Lock()
	pid := harnessWebPID
	harnessWebMu.Unlock()
	running := harnessPortOpen()
	owned := pid > 0 && pidAlive(pid)
	return map[string]interface{}{
		"running": running,
		"owned":   owned,
		"pid":     pid,
		"url":     harnessWebURL,
		"root":    harnessRoot(),
		"log":     harnessLogPath(),
	}
}

// StartProgrammingWeb 启动 dsh web（已运行幂等返回；外部占用 3080 时明确报错不抢端口）。
func (a *App) StartProgrammingWeb() error {
	harnessWebMu.Lock()
	defer harnessWebMu.Unlock()

	if harnessPortOpen() {
		if harnessWebPID > 0 && pidAlive(harnessWebPID) {
			return nil // 自启实例已在服务，幂等
		}
		return fmt.Errorf("端口 %d 已被其他进程占用（非 gaea 自启实例），请先手动停止该进程", harnessWebPort)
	}

	root := harnessRoot()
	if fi, err := os.Stat(filepath.Join(root, "package.json")); err != nil || fi.IsDir() {
		return fmt.Errorf("找不到 DeepSeek Harness（%s），请确认目录存在或用 GAEA_HARNESS_DIR 指定", root)
	}
	pnpm, err := exec.LookPath("pnpm")
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
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 dsh web 失败: %w", err)
	}
	harnessWebPID = cmd.Process.Pid

	// 等待端口就绪（最长 20s），避免「点完没反应」。
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if harnessPortOpen() {
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
		if harnessPortOpen() {
			return fmt.Errorf("端口 %d 有外部实例在运行（非 gaea 自启），为避免误杀请手动停止", harnessWebPort)
		}
		return nil // 本就没在跑
	}
	pid := harnessWebPID
	harnessWebPID = 0
	if !pidAlive(pid) {
		return nil
	}
	// Windows：taskkill /T 连 pnpm→node 子进程树一起终止。
	if err := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run(); err != nil {
		return fmt.Errorf("停止 dsh web（pid=%d）失败: %w", pid, err)
	}
	return nil
}
