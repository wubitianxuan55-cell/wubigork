package browser

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// LaunchOptions 启动选项。Headless 供测试与无界面场景使用；默认有头
// （可见 = 可信任，用户能看到受控浏览器在做什么）。
type LaunchOptions struct {
	Headless bool
}

// EdgeProcess 一个受控 Edge 实例：cmd/job 由 Launch 建立并交 KillTracked 收割，
// ProfileDir 是专用临时 profile（绝不指向用户主 profile）。
type EdgeProcess struct {
	Exe        string
	Cmd        *exec.Cmd
	Job        uintptr // Job Object 句柄（0 = 创建失败，仅剩 KillTree 兜底）
	Port       int
	ProfileDir string
}

// FindEdge 定位 msedge：GAEA_BROWSER_EXE（不校验存在，交给启动报错）→
// ProgramFiles(x86) → ProgramFiles → PATH。三段式与 herdsman_catalog 对齐。
func FindEdge() (string, error) {
	return findEdge(os.Getenv, exec.LookPath)
}

// findEdge 生产逻辑以 getenv/lookPath 注入，便于确定性测试。
func findEdge(getenv func(string) string, lookPath func(string) (string, error)) (string, error) {
	if p := strings.TrimSpace(getenv("GAEA_BROWSER_EXE")); p != "" {
		return p, nil
	}
	var candidates []string
	if pf := getenv("ProgramFiles(x86)"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "Microsoft", "Edge", "Application", "msedge.exe"))
	}
	if pf := getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "Microsoft", "Edge", "Application", "msedge.exe"))
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c, nil
		}
	}
	if exe, err := lookPath("msedge"); err == nil {
		return exe, nil
	}
	return "", errors.New("browser: 未找到 msedge.exe（可设环境变量 GAEA_BROWSER_EXE 指向 Edge 可执行文件）")
}

// freePort 取一个空闲 TCP 端口（取号即归还，存在竞窗但 MVP 足够）。
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// Launch 启动独立 Edge 实例：
//   - --remote-debugging-port 绑定专用端口（由 freePort 取号）；
//   - --user-data-dir 指向 os.TempDir()/gaea-browser-profile-<pid>（启动前清残，
//     绝不碰用户主 profile）；
//   - 关闭首跑向导/默认浏览器检查/翻译/侧边栏等干扰项。
//
// 进程绑进 Job Object（父进程异常退出也整体收割）；HideWindow 仅 headless 时
// 使用（有头模式窗口本身就是可见性的一部分）。ctx 不参与进程存活控制——
// 用 WithoutCancel 隔离单次工具调用的取消。
func Launch(ctx context.Context, exe string, port int, opts LaunchOptions) (*EdgeProcess, error) {
	profileDir := filepath.Join(os.TempDir(), fmt.Sprintf("gaea-browser-profile-%d", os.Getpid()))
	// 启动前清残：上次异常退出可能残留锁文件，导致本次起不来
	_ = os.RemoveAll(profileDir)
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return nil, fmt.Errorf("browser: 创建临时 profile 失败: %w", err)
	}
	args := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--user-data-dir=" + profileDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-features=Translate,msEdgeSidebarV2",
	}
	if opts.Headless {
		args = append(args, "--headless=new")
	}
	cmd := exec.CommandContext(context.WithoutCancel(ctx), exe, args...)
	if opts.Headless {
		proc.HideWindow(cmd)
	}
	job, err := proc.StartTracked(cmd)
	if err != nil {
		return nil, fmt.Errorf("browser: 启动 Edge 失败 (%s): %w", exe, err)
	}
	return &EdgeProcess{Exe: exe, Cmd: cmd, Job: job, Port: port, ProfileDir: profileDir}, nil
}
