package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseHerdsmanOpResult(t *testing.T) {
	r, err := parseHerdsmanOpResult([]byte(`{"ok":true,"result":{"status":"completed"}}`))
	if err != nil || !r.OK || r.Status != "completed" {
		t.Fatalf("completed 解析失败: r=%+v err=%v", r, err)
	}
	// 无 result.status 且 ok=true → 视为 completed（stop/uninstall 等短命令）。
	r, err = parseHerdsmanOpResult([]byte(`{"ok":true}`))
	if err != nil || r.Status != "completed" {
		t.Fatalf("ok=true 空状态解析失败: r=%+v err=%v", r, err)
	}
	// 失败路径：ok:false 带错误。
	r, err = parseHerdsmanOpResult([]byte(`{"ok":false,"result":{"status":"failed","error":"模型启动失败"}}`))
	if err == nil || !strings.Contains(err.Error(), "模型启动失败") {
		t.Fatalf("失败路径应返回错误: r=%+v err=%v", r, err)
	}
	// 非法 JSON。
	if _, err := parseHerdsmanOpResult([]byte(`nope`)); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

func TestParseHerdsmanLaunchPresets(t *testing.T) {
	dir := t.TempDir()
	f1 := `[
	  {"model_name":"Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2","inference_engine":"llama.cpp","start_time":"2026-08-13T10:00:00+08:00","port":8896,"status":"success","options":{"context_size":262144,"gpu_layers":99,"cache_type_k":"f16","no_mmap":true}},
	  {"model_name":"Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2","inference_engine":"llama.cpp","start_time":"2026-08-13T09:00:00+08:00","port":24331,"status":"success","options":{"context_size":4096}},
	  {"model_name":"old-model","inference_engine":"llama.cpp","start_time":"2026-08-12T08:00:00+08:00","port":1,"status":"failed","options":{}}
	]`
	f2 := `[{"model_name":"Gemma4:12B-IT","inference_engine":"llama.cpp","start_time":"2026-08-13T11:58:31+08:00","port":48503,"status":"success","options":{"context_size":262144,"mmproj":"mmproj-BF16.gguf"}}]`
	if err := os.WriteFile(filepath.Join(dir, "qwen.json"), []byte(f1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gemma.json"), []byte(f2), 0o644); err != nil {
		t.Fatal(err)
	}
	presets, err := parseHerdsmanLaunchPresets(dir)
	if err != nil {
		t.Fatalf("parseHerdsmanLaunchPresets: %v", err)
	}
	if len(presets) != 2 {
		t.Fatalf("len(presets) = %d, want 2", len(presets))
	}
	// 按模型名排序：Gemma4:12B-IT 在前。
	if presets[0].Model != "Gemma4:12B-IT" || presets[1].Model != "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2" {
		t.Fatalf("排序错误: %+v", presets)
	}
	// 每个模型取最近一次成功启动。
	if got := presets[1].Options["context_size"]; got != float64(262144) {
		t.Errorf("应取最近成功记录 context_size=%v", got)
	}
	if presets[1].Port != 8896 {
		t.Errorf("Port = %d, want 8896", presets[1].Port)
	}
}

func TestHerdsmanLaunchPresets_EmptyDir(t *testing.T) {
	presets, err := parseHerdsmanLaunchPresets(t.TempDir())
	if err != nil || len(presets) != 0 {
		t.Fatalf("空目录应返回空: presets=%v err=%v", presets, err)
	}
}

func TestHerdsmanDataDir(t *testing.T) {
	t.Setenv("HERDSMAN_DATA_DIR", `D:\herdsman-data`)
	if got := herdsmanDataDir(); got != `D:\herdsman-data` {
		t.Fatalf("herdsmanDataDir() = %q", got)
	}
}

func TestHerdsmanLifecycleHandlers(t *testing.T) {
	origList, origLong := herdsmanCLI, herdsmanCLIWithTimeout
	defer func() { herdsmanCLI, herdsmanCLIWithTimeout = origList, origLong }()

	herdsmanCLIWithTimeout = func(_ time.Duration, args ...string) ([]byte, error) {
		return []byte(`{"ok":true,"result":{"status":"completed"}}`), nil
	}
	a := &App{}
	for name, fn := range map[string]func(string) (HerdsmanOpResult, error){
		"start":     a.HerdsmanModelStart,
		"stop":      a.HerdsmanModelStop,
		"download":  a.HerdsmanModelDownload,
		"uninstall": a.HerdsmanModelUninstall,
	} {
		r, err := fn("bge-m3")
		if err != nil || !r.OK || r.Status != "completed" {
			t.Fatalf("%s 成功路径失败: r=%+v err=%v", name, r, err)
		}
		if _, err := fn(""); err == nil {
			t.Fatalf("%s 空模型名应报错", name)
		}
	}

	// CLI 失败透出。
	herdsmanCLIWithTimeout = func(_ time.Duration, args ...string) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	if _, err := a.HerdsmanModelStop("bge-m3"); err == nil {
		t.Fatal("CLI 失败应透出")
	}
}

// ─── AP9-02：超时必须杀整棵进程树（真实进程树，Windows）───────────
//
// 探针形态：Go 测试进程 → cmd.exe(/c 包装) → cmd.exe(内层) → ping.exe(孙进程)。
// ping 命令行带唯一 `-w <随机值>` 作 marker（-n 60 保证存活够久），孙进程按
// 「PING.EXE + marker」在 Win32_Process 里精确识别。

// probePingArgs 返回带唯一 marker 的 ping 参数与 marker 文本。
func probePingArgs() (args []string, marker string) {
	waitMS := 4000 + int(time.Now().UnixNano()%4000)
	marker = fmt.Sprintf("-w %d", waitMS)
	return []string{"-n", "60", "-w", strconv.Itoa(waitMS), "127.0.0.1"}, marker
}

// probeGrandchildPIDs 返回命令行命中 marker 的 ping.exe PID 列表。
func probeGrandchildPIDs(t *testing.T, marker string) []int {
	t.Helper()
	script := fmt.Sprintf(
		`Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'PING.EXE' -and $_.CommandLine -like '*%s*' } | ForEach-Object { $_.ProcessId }`,
		marker)
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		t.Fatalf("查询进程失败: %v", err)
	}
	var pids []int
	for _, f := range strings.Fields(strings.TrimSpace(string(out))) {
		if n, err := strconv.Atoi(f); err == nil {
			pids = append(pids, n)
		}
	}
	return pids
}

// processAlive 判断 PID 是否仍存活。
func processAlive(t *testing.T, pid int) bool {
	t.Helper()
	out, _ := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf(`if (Get-Process -Id %d -ErrorAction SilentlyContinue) { 'alive' } else { 'dead' }`, pid)).Output()
	return strings.Contains(string(out), "alive")
}

// waitProbeGrandchild 轮询等孙进程出现（CIM 查询与进程创建都有延迟）。
func waitProbeGrandchild(t *testing.T, marker string, timeout time.Duration) []int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if pids := probeGrandchildPIDs(t, marker); len(pids) > 0 {
			return pids
		}
		if time.Now().After(deadline) {
			t.Fatalf("探针孙进程未出现（marker=%q）", marker)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// waitProcessesGone 轮询等这批 PID 全部消失（taskkill 同步返回，但进程退出有
// 微小时延；有界轮询避免把时延误判成清理失败）。
func waitProcessesGone(t *testing.T, pids []int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		alive := 0
		for _, pid := range pids {
			if processAlive(t, pid) {
				alive++
			}
		}
		if alive == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("进程 %v 在 %s 内未消失（清理失败）", pids, timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// startProbeTree 起一棵真实三层进程树，返回父进程 PID 与孙进程 marker。
// Cleanup 用 taskkill /T 收拾残局（含未被断言的中间 cmd.exe）。
func startProbeTree(t *testing.T) (parentPID int, marker string) {
	t.Helper()
	pingArgs, marker := probePingArgs()
	parent := exec.Command("cmd", append([]string{"/c", "cmd", "/c", "ping"}, pingArgs...)...)
	if err := parent.Start(); err != nil {
		t.Fatalf("起探针进程树失败: %v", err)
	}
	go func() { _ = parent.Wait() }() // 立刻回收，避免僵尸进程与重复 Wait
	pid := parent.Process.Pid
	t.Cleanup(func() {
		_ = killProcessTree(pid)
		for _, g := range probeGrandchildPIDs(t, marker) {
			_ = killProcessTree(g)
		}
	})
	return pid, marker
}

// TestKillProcessTree_KillsGrandchild：killProcessTree 必须连孙进程一起清掉。
func TestKillProcessTree_KillsGrandchild(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("进程树清理走 taskkill /T，仅 Windows 实现")
	}
	parentPID, marker := startProbeTree(t)
	grandchildren := waitProbeGrandchild(t, marker, 15*time.Second)
	if err := killProcessTree(parentPID); err != nil {
		t.Fatalf("killProcessTree: %v", err)
	}
	waitProcessesGone(t, grandchildren, 10*time.Second)
	for _, p := range probeGrandchildPIDs(t, marker) {
		if processAlive(t, p) {
			t.Fatalf("孙进程 %d 仍在（marker=%q）", p, marker)
		}
	}
}

// TestParentOnlyKillLeavesGrandchild 钉死 AP9-02 的机理现场：只杀直接子进程
// （exec.CommandContext 超时的原行为）时孙进程继续存活——这正是必须 taskkill /T
// 的原因，也是本批把 CommandContext 换成「Start + 超时杀树」的依据。本用例
// 刻意不调用 killProcessTree（残局由 startProbeTree 的 Cleanup 收拾）。
func TestParentOnlyKillLeavesGrandchild(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("进程树清理走 taskkill /T，仅 Windows 实现")
	}
	parentPID, marker := startProbeTree(t)
	grandchildren := waitProbeGrandchild(t, marker, 15*time.Second)
	if err := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(parentPID)).Run(); err != nil {
		t.Fatalf("单进程 kill: %v", err)
	}
	time.Sleep(time.Second) // 等父子关系解体
	survivors := 0
	for _, p := range grandchildren {
		if processAlive(t, p) {
			survivors++
		}
	}
	if survivors == 0 {
		t.Fatal("前置失败：单进程 kill 后孙进程也消失了，无法锁定 AP9-02 现场")
	}
}

// TestRunHerdsmanCLI_TimeoutKillsProcessTree 端到端：把 HERDSMAN_EXE 指向
// cmd.exe 起一棵真实进程树，超时分支必须①返回「超时」文案②整棵树（含孙进程
// ping）消失。这条路径故意让杀树成功、进程很快退出（-race 下覆盖超时清理的
// 正常路径，5 秒兜底分支不触发）。
func TestRunHerdsmanCLI_TimeoutKillsProcessTree(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("进程树清理走 taskkill /T，仅 Windows 实现")
	}
	comspec := os.Getenv("COMSPEC")
	if comspec == "" {
		comspec = "cmd.exe"
	}
	t.Setenv("HERDSMAN_EXE", comspec)

	pingArgs, marker := probePingArgs()
	args := append([]string{"/c", "cmd", "/c", "ping"}, pingArgs...)
	done := make(chan error, 1)
	go func() {
		_, err := runHerdsmanCLI(5*time.Second, args...)
		done <- err
	}()

	grandchildren := waitProbeGrandchild(t, marker, 4*time.Second)
	var err error
	select {
	case err = <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("runHerdsmanCLI 未在超时后返回")
	}
	if err == nil {
		t.Fatal("超时分支必须返回错误")
	}
	if !strings.Contains(err.Error(), "超时") {
		t.Fatalf("错误文案要继续说明是超时: %v", err)
	}
	waitProcessesGone(t, grandchildren, 10*time.Second)
}
