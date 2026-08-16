package app

// programming_web_test.go — 编程板块「DeepSeek Harness Web 进程管理」单元测试。
//
// 外部副作用全部经 probe* 探针注入（见 programming_web.go）：端口探测 / tasklist
// 存活 / taskkill 树杀 / LookPath / cmd.Start / 日志路径 / 等待超时，测试不触碰
// 真实 3080 端口与真实进程。所有包级变量改动经 t.Cleanup 恢复（同包顺序执行）。

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// restoreProbe 保存并恢复一个可替换探针/包级变量的当前值。
func restoreProbe(t *testing.T, set func(v interface{}), cur interface{}) {
	t.Helper()
	set(cur)
	t.Cleanup(func() { set(cur) })
}

// fakeCmd 返回一个带假 Pid、未真正启动的 *exec.Cmd（供 probeStartCmd 注入）。
func fakeCmd(pid int) *exec.Cmd {
	c := exec.Command("echo")
	c.Process = &os.Process{Pid: pid}
	return c
}

// ── GetProgrammingWebStatus ───────────────────────────────────────────────

func TestProgrammingWebStatusIdle(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probePIDAlive = v.(func(int) bool) }, probePIDAlive)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	restoreProbe(t, func(v interface{}) { harnessWebStartedAt = v.(time.Time) }, harnessWebStartedAt)
	probePortOpen = func() bool { return false }
	probePIDAlive = func(int) bool { return false }
	harnessWebPID = 0
	harnessWebStartedAt = time.Time{}

	s := (&App{}).GetProgrammingWebStatus()
	if s["running"] != false || s["owned"] != false {
		t.Fatalf("未运行应 running=false owned=false，got %+v", s)
	}
	if s["uptime_s"].(int64) != 0 {
		t.Fatalf("未运行 uptime_s 应为 0，got %v", s["uptime_s"])
	}
	if s["url"] != harnessWebURL {
		t.Fatalf("url 应为 %s，got %v", harnessWebURL, s["url"])
	}
	if s["log"] == nil || s["log"] == "" {
		t.Fatalf("log 字段缺失: %+v", s)
	}
}

func TestProgrammingWebStatusOwnedRunning(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probePIDAlive = v.(func(int) bool) }, probePIDAlive)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	restoreProbe(t, func(v interface{}) { harnessWebStartedAt = v.(time.Time) }, harnessWebStartedAt)
	probePortOpen = func() bool { return true }
	probePIDAlive = func(pid int) bool { return pid == 1234 }
	harnessWebPID = 1234
	harnessWebStartedAt = time.Now().Add(-65 * time.Second)

	s := (&App{}).GetProgrammingWebStatus()
	if s["running"] != true || s["owned"] != true || s["pid"].(int) != 1234 {
		t.Fatalf("自启运行中应 running/owned=true pid=1234，got %+v", s)
	}
	if s["uptime_s"].(int64) < 60 {
		t.Fatalf("运行 65s 后 uptime_s 应 ≥60，got %v", s["uptime_s"])
	}
}

func TestProgrammingWebStatusExternalRunning(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probePIDAlive = v.(func(int) bool) }, probePIDAlive)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	restoreProbe(t, func(v interface{}) { harnessWebStartedAt = v.(time.Time) }, harnessWebStartedAt)
	probePortOpen = func() bool { return true }
	probePIDAlive = func(int) bool { return false } // 记录的自启 pid 已死 / 从未记录
	harnessWebPID = 0
	harnessWebStartedAt = time.Now().Add(-10 * time.Second)

	s := (&App{}).GetProgrammingWebStatus()
	if s["running"] != true || s["owned"] != false {
		t.Fatalf("外部实例应 running=true owned=false，got %+v", s)
	}
	if s["uptime_s"].(int64) != 0 {
		t.Fatalf("非自启 uptime_s 应为 0（不得按陈旧 startedAt 计），got %v", s["uptime_s"])
	}
}

// ── StartProgrammingWeb ───────────────────────────────────────────────────

func TestStartProgrammingWebIdempotentOwned(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probePIDAlive = v.(func(int) bool) }, probePIDAlive)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	probePortOpen = func() bool { return true }
	probePIDAlive = func(pid int) bool { return pid == 77 }
	harnessWebPID = 77

	if err := (&App{}).StartProgrammingWeb(); err != nil {
		t.Fatalf("自启实例已在服务应幂等返回 nil，got %v", err)
	}
}

func TestStartProgrammingWebPortBusyExternal(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probePIDAlive = v.(func(int) bool) }, probePIDAlive)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	probePortOpen = func() bool { return true }
	probePIDAlive = func(int) bool { return false } // 不是自启实例
	harnessWebPID = 0

	err := (&App{}).StartProgrammingWeb()
	if err == nil || !strings.Contains(err.Error(), "已被其他进程占用") {
		t.Fatalf("外部占用 3080 应报错不抢端口，got %v", err)
	}
}

func TestStartProgrammingWebHarnessMissing(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probeLookPath = v.(func(string) (string, error)) }, probeLookPath)
	probePortOpen = func() bool { return false }
	t.Setenv("GAEA_HARNESS_DIR", t.TempDir()) // 空目录，无 package.json

	err := (&App{}).StartProgrammingWeb()
	if err == nil || !strings.Contains(err.Error(), "找不到 DeepSeek Harness") {
		t.Fatalf("目录无效应报错，got %v", err)
	}
}

func TestStartProgrammingWebPnpmMissing(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probeLookPath = v.(func(string) (string, error)) }, probeLookPath)
	probePortOpen = func() bool { return false }
	probeLookPath = func(string) (string, error) { return "", errors.New("not found") }

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GAEA_HARNESS_DIR", root)

	err := (&App{}).StartProgrammingWeb()
	if err == nil || !strings.Contains(err.Error(), "未找到 pnpm") {
		t.Fatalf("pnpm 缺失应报错，got %v", err)
	}
}

func TestStartProgrammingWebSpawnAndReady(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probeLookPath = v.(func(string) (string, error)) }, probeLookPath)
	restoreProbe(t, func(v interface{}) { probeStartCmd = v.(func(*exec.Cmd) error) }, probeStartCmd)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	restoreProbe(t, func(v interface{}) { harnessWebStartedAt = v.(time.Time) }, harnessWebStartedAt)

	// 守卫检查时端口未开；spawn 后立刻就绪 → 正常返回。
	calls := 0
	probePortOpen = func() bool {
		calls++
		return calls > 1
	}
	probeLookPath = func(string) (string, error) { return "pnpm", nil }
	probeStartCmd = func(cmd *exec.Cmd) error {
		cmd.Process = &os.Process{Pid: 4242}
		return nil
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GAEA_HARNESS_DIR", root)

	if err := (&App{}).StartProgrammingWeb(); err != nil {
		t.Fatalf("spawn 后端口就绪应 nil，got %v", err)
	}
	if harnessWebPID != 4242 {
		t.Fatalf("应记录自启 pid 4242，got %d", harnessWebPID)
	}
	if harnessWebStartedAt.IsZero() {
		t.Fatal("应记录自启时刻（uptime 计算依赖）")
	}
}

func TestStartProgrammingWebPortTimeout(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probeLookPath = v.(func(string) (string, error)) }, probeLookPath)
	restoreProbe(t, func(v interface{}) { probeStartCmd = v.(func(*exec.Cmd) error) }, probeStartCmd)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	restoreProbe(t, func(v interface{}) { harnessWaitTimeout = v.(time.Duration) }, harnessWaitTimeout)

	probePortOpen = func() bool { return false } // 永不就绪
	probeLookPath = func(string) (string, error) { return "pnpm", nil }
	probeStartCmd = func(cmd *exec.Cmd) error {
		cmd.Process = &os.Process{Pid: 99}
		return nil
	}
	harnessWaitTimeout = 150 * time.Millisecond

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GAEA_HARNESS_DIR", root)

	err := (&App{}).StartProgrammingWeb()
	if err == nil || !strings.Contains(err.Error(), "端口未在") {
		t.Fatalf("端口超时应报错，got %v", err)
	}
	if harnessWebPID != 99 {
		t.Fatalf("超时也应记录 pid（供查日志），got %d", harnessWebPID)
	}
}

// ── StopProgrammingWeb ────────────────────────────────────────────────────

func TestStopProgrammingWebOwned(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePIDAlive = v.(func(int) bool) }, probePIDAlive)
	restoreProbe(t, func(v interface{}) { probeKillTree = v.(func(int) error) }, probeKillTree)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	restoreProbe(t, func(v interface{}) { harnessWebStartedAt = v.(time.Time) }, harnessWebStartedAt)

	killed := 0
	probePIDAlive = func(pid int) bool { return pid == 42 }
	probeKillTree = func(pid int) error { killed = pid; return nil }
	harnessWebPID = 42
	harnessWebStartedAt = time.Now()

	if err := (&App{}).StopProgrammingWeb(); err != nil {
		t.Fatalf("停止自启实例应 nil，got %v", err)
	}
	if killed != 42 {
		t.Fatalf("应 kill 自启 pid 42，got %d", killed)
	}
	if harnessWebPID != 0 {
		t.Fatalf("停止后记录应清空，got %d", harnessWebPID)
	}
	if !harnessWebStartedAt.IsZero() {
		t.Fatal("停止后 startedAt 应清零（uptime 不再累计）")
	}
}

func TestStopProgrammingWebExternal(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	probePortOpen = func() bool { return true }
	harnessWebPID = 0

	err := (&App{}).StopProgrammingWeb()
	if err == nil || !strings.Contains(err.Error(), "外部实例") {
		t.Fatalf("外部实例应提示不误杀，got %v", err)
	}
}

func TestStopProgrammingWebIdle(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { harnessWebPID = v.(int) }, harnessWebPID)
	probePortOpen = func() bool { return false }
	harnessWebPID = 0

	if err := (&App{}).StopProgrammingWeb(); err != nil {
		t.Fatalf("本就未运行应 nil，got %v", err)
	}
}

// ── ProgrammingWebLogTail / tailLines ─────────────────────────────────────

func TestProgrammingWebLogTailMissing(t *testing.T) {
	restoreProbe(t, func(v interface{}) { harnessLogPathFn = v.(func() string) }, harnessLogPathFn)
	harnessLogPathFn = func() string { return filepath.Join(t.TempDir(), "nope.log") }

	r := (&App{}).ProgrammingWebLogTail(50)
	if r["exists"] != false {
		t.Fatalf("日志不存在应 exists=false，got %+v", r)
	}
	if r["error"] == nil || r["error"] == "" {
		t.Fatalf("应给出提示文案，got %+v", r)
	}
	if lines, ok := r["lines"].([]string); !ok || len(lines) != 0 {
		t.Fatalf("lines 应为空切片，got %+v", r["lines"])
	}
}

func TestProgrammingWebLogTailReads(t *testing.T) {
	restoreProbe(t, func(v interface{}) { harnessLogPathFn = v.(func() string) }, harnessLogPathFn)
	logPath := filepath.Join(t.TempDir(), "dsh.log")
	harnessLogPathFn = func() string { return logPath }

	var b strings.Builder
	for i := 1; i <= 60; i++ {
		b.WriteString("line-")
		b.WriteString(strings.Repeat("x", i%7))
		b.WriteString("\r\n")
	}
	if err := os.WriteFile(logPath, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	r := (&App{}).ProgrammingWebLogTail(10)
	if r["exists"] != true {
		t.Fatalf("应 exists=true，got %+v", r)
	}
	lines := r["lines"].([]string)
	if len(lines) != 10 {
		t.Fatalf("n=10 应返回 10 行，got %d", len(lines))
	}
	if strings.Contains(lines[9], "\r") || !strings.HasPrefix(lines[9], "line-") {
		t.Fatalf("行尾不应残留 \\r，got %q", lines[9])
	}
	if !strings.HasPrefix(lines[0], "line-") {
		t.Fatalf("应取末尾 10 行（首行 line-51 起），got %q", lines[0])
	}
}

func TestTailLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want int
		last string
	}{
		{"空串", "", 10, 0, ""},
		{"只有换行", "\n\n", 10, 0, ""},
		{"不足 n", "a\nb\nc", 10, 3, "c"},
		{"恰好 n", "a\nb\nc", 3, 3, "c"},
		{"CRLF", "a\r\nb\r\nc\r\n", 2, 2, "c"},
		{"超量", "1\n2\n3\n4\n5", 2, 2, "5"},
	}
	for _, c := range cases {
		got := tailLines(c.in, c.n)
		if len(got) != c.want {
			t.Errorf("%s: len=%d want %d (%q)", c.name, len(got), c.want, got)
			continue
		}
		if c.want > 0 && got[len(got)-1] != c.last {
			t.Errorf("%s: 末行 %q want %q", c.name, got[len(got)-1], c.last)
		}
	}
}

// ── GetProgrammingWebPreflight ────────────────────────────────────────────

func TestGetProgrammingWebPreflightReady(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probeLookPath = v.(func(string) (string, error)) }, probeLookPath)
	probePortOpen = func() bool { return false }
	probeLookPath = func(string) (string, error) { return "pnpm", nil }

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "apps", "web", "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "apps", "web", "dist", "index.html"), []byte("<html/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GAEA_HARNESS_DIR", root)

	r := (&App{}).GetProgrammingWebPreflight()
	for _, k := range []string{"harness_valid", "pnpm_found", "deps_ready", "build_ready", "port_free"} {
		if r[k] != true {
			t.Errorf("%s 应为 true，got %+v", k, r)
		}
	}
	if r["all_ready"] != true {
		t.Fatalf("前置条件全满足应 all_ready=true，got %+v", r)
	}
	if r["root"] != root {
		t.Fatalf("root 应回显 harness 目录，got %v", r["root"])
	}
}

func TestGetProgrammingWebPreflightBroken(t *testing.T) {
	restoreProbe(t, func(v interface{}) { probePortOpen = v.(func() bool) }, probePortOpen)
	restoreProbe(t, func(v interface{}) { probeLookPath = v.(func(string) (string, error)) }, probeLookPath)
	probePortOpen = func() bool { return true } // 端口被占
	probeLookPath = func(string) (string, error) { return "", errors.New("not found") }

	root := t.TempDir() // 无 package.json / node_modules / dist
	t.Setenv("GAEA_HARNESS_DIR", root)

	r := (&App{}).GetProgrammingWebPreflight()
	if r["harness_valid"] != false || r["pnpm_found"] != false ||
		r["deps_ready"] != false || r["build_ready"] != false || r["port_free"] != false {
		t.Fatalf("缺前置条件应逐项 false，got %+v", r)
	}
	if r["all_ready"] != false {
		t.Fatalf("缺前置条件 all_ready 应为 false，got %+v", r)
	}
}
