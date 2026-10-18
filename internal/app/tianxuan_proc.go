package app

// tianxuan_proc.go — 天璇引擎（gufo-server，8094）进程生命周期：状态探测 / 启动 /
// 停止。天璇与 comfyui 同属「gaea 代启引擎」族：模型中心的引擎卡提供 启动/停止，
// 不必手跑 scripts/tianxuan-serve.ps1。启动参数即启动器 27b 预设（27B 是 35B 终局
// 后唯一档位，gufo 台账 §8）；停止走 netstat 找 8094 监听 PID 后 taskkill——与
// StopComfyUI 同口径「不管谁启动的都能停」。纪律：引擎生成中硬杀有 D 状态风险
// （gufo 台账 §6 设备互斥），UI 侧应在空闲时调用；这里不做生成中检测（chat 链路
// 与本文件无耦合，探活只看端口）。

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gaea/gaea/internal/config"
)

const (
	tianxuanPort      = 8094
	tianxuanStartWait = 120 * time.Second // 27B 冷装载 ~20-60s（2026-10-18 实测 15-33s），留足余量
	tianxuanProbeDial = 1500 * time.Millisecond
	// 二进制与模型路径 = scripts/tianxuan-serve.ps1 27b 预设（自构建优先、
	// 预构建兜底；模型用 herdsman 库已有文件——本机不下载纪律）。
	tianxuanBinSelfBuild = `C:\AI\wubigrok\clones\gufo\build\release\gufo-server.exe`
	tianxuanBinPrebuilt  = `C:\AI\gufo-win\gufo-windows-2026-10-03\bin\gufo.exe`
	tianxuanGGUF         = `C:\Users\wubi\.herdsman\models\Qwen3.8-27B\windows\amd64\amd\Qwen3.8-27B-UD-Q4_K_M.gguf`
	tianxuanMmproj       = `C:\Users\wubi\.herdsman\models\Qwen3.8-27B\windows\amd64\amd\mmproj-BF16.gguf`
	tianxuanDraftModel   = `C:\AI\gufo-win\Qwen3.8-27B-DFlash2-Q4_K_M.gguf`
)

// tianxuanProcRef 天璇引擎进程引用（仅登记 gaea 自己拉起的进程；外部启动器起的
// 实例走 netstat 停止路径）。快停-慢收（Wait goroutine 晚到）用 cmd 指针判身份，
// 与 comfyProcRefClearIfCurrent 同款。
type tianxuanProcRef struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func (a *App) tianxuanProcSet(cmd *exec.Cmd, cancel context.CancelFunc) {
	a.tianxuanProc.mu.Lock()
	a.tianxuanProc.cmd = cmd
	a.tianxuanProc.cancel = cancel
	a.tianxuanProc.mu.Unlock()
}

func (a *App) tianxuanProcClearIfCurrent(cmd *exec.Cmd) {
	a.tianxuanProc.mu.Lock()
	if a.tianxuanProc.cmd == cmd {
		a.tianxuanProc.cmd = nil
		a.tianxuanProc.cancel = nil
	}
	a.tianxuanProc.mu.Unlock()
}

// tianxuanProbe 探测 8094 上的天璇引擎：可达时返回 /v1/models 的模型名清单。
func tianxuanProbe() ([]string, bool) {
	return tianxuanProbeAt(fmt.Sprintf("http://127.0.0.1:%d", tianxuanPort))
}

// tianxuanProbeAt 探活本体（base 可注入，测试走 httptest）。
func tianxuanProbeAt(base string) ([]string, bool) {
	client := &http.Client{Timeout: tianxuanProbeDial}
	resp, err := client.Get(base + "/v1/models")
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return []string{}, true
	}
	ids := make([]string, 0, len(body.Data))
	for _, m := range body.Data {
		ids = append(ids, m.ID)
	}
	return ids, true
}

// TianxuanEngineStatus 天璇引擎运行状态（模型中心引擎卡的状态点与启停钮）。
func (a *App) TianxuanEngineStatus() map[string]interface{} {
	served, ok := tianxuanProbe()
	return map[string]interface{}{
		"running":       ok,
		"port":          tianxuanPort,
		"served_models": served,
	}
}

// TianxuanEngineStart 启动天璇引擎（27b 预设）。已在运行→幂等返回 already-running；
// 端口被占用→如实报 port-occupied（与其它引擎互斥纪律）。启动后轮询 /v1/models
// 至就绪（上限 120s），超时/早退如实报错并附日志路径。
func (a *App) TianxuanEngineStart() map[string]interface{} {
	if served, ok := tianxuanProbe(); ok {
		return map[string]interface{}{"started": false, "reason": "already-running", "served_models": served}
	}
	if ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", tianxuanPort)); err != nil {
		return map[string]interface{}{"started": false, "reason": "port-occupied", "port": tianxuanPort}
	} else {
		_ = ln.Close()
	}
	bin := tianxuanBinSelfBuild
	if _, err := os.Stat(bin); err != nil {
		bin = tianxuanBinPrebuilt
		if _, err := os.Stat(bin); err != nil {
			return map[string]interface{}{"started": false, "reason": "binary-not-found", "tried": []string{tianxuanBinSelfBuild, tianxuanBinPrebuilt}}
		}
	}
	for _, p := range []string{tianxuanGGUF, tianxuanMmproj, tianxuanDraftModel} {
		if _, err := os.Stat(p); err != nil {
			return map[string]interface{}{"started": false, "reason": "model-not-found", "path": p}
		}
	}
	cacheDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "gufo", "cache")
	_ = os.MkdirAll(cacheDir, 0o755)
	argv := []string{
		"serve", "llm", "-m", tianxuanGGUF,
		"-p", strconv.Itoa(tianxuanPort),
		"-j", "2", "-c", "32768",
		"--served-model-name", "Qwen3.8-27B-DFlash2",
		"--speculative", "dflash2",
		"--mmproj", tianxuanMmproj,
		"--dflash-model", tianxuanDraftModel,
		"--draft-tokens", "4",
		"--cache-disk", cacheDir,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, argv...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	logPath := filepath.Join(config.DataRoot(), "logs", fmt.Sprintf("tianxuan-serve-%s.log", time.Now().Format("20060102-150405")))
	logFile, err := os.Create(logPath)
	if err != nil {
		cancel()
		return map[string]interface{}{"started": false, "reason": "log-open-failed", "error": err.Error()}
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		cancel()
		_ = logFile.Close()
		return map[string]interface{}{"started": false, "reason": "spawn-failed", "error": err.Error()}
	}
	a.tianxuanProcSet(cmd, cancel)
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
		a.tianxuanProcClearIfCurrent(cmd)
	}()

	t0 := time.Now()
	deadline := t0.Add(tianxuanStartWait)
	for time.Now().Before(deadline) {
		if cmd.ProcessState != nil {
			return map[string]interface{}{"started": false, "reason": "exited-early", "log": logPath}
		}
		if served, ok := tianxuanProbe(); ok {
			return map[string]interface{}{
				"started":       true,
				"waited_ms":     time.Since(t0).Milliseconds(),
				"served_models": served,
				"log":           logPath,
			}
		}
		select {
		case <-ctx.Done():
			return map[string]interface{}{"started": false, "reason": "canceled", "log": logPath}
		case <-time.After(2 * time.Second):
		}
	}
	return map[string]interface{}{"started": false, "reason": "timeout", "log": logPath}
}

// TianxuanEngineStop 停止天璇引擎：优先杀 gaea 登记的进程，否则 netstat 找 8094
// 监听 PID（外部启动器起的实例也能停）。停止后轮询端口释放（上限 5s）。
func (a *App) TianxuanEngineStop() map[string]interface{} {
	a.tianxuanProc.mu.Lock()
	cmd := a.tianxuanProc.cmd
	cancel := a.tianxuanProc.cancel
	a.tianxuanProc.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		cancel()
		_ = cmd.Process.Kill()
		a.tianxuanProcClearIfCurrent(cmd)
	} else if pid := tianxuanPortPID(); pid != 0 {
		if out, err := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F").CombinedOutput(); err != nil {
			return map[string]interface{}{"stopped": false, "reason": "taskkill-failed", "pid": pid, "error": strings.TrimSpace(string(out))}
		}
	} else {
		return map[string]interface{}{"stopped": false, "reason": "not-running"}
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := tianxuanProbe(); !ok {
			return map[string]interface{}{"stopped": true}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return map[string]interface{}{"stopped": false, "reason": "port-still-busy"}
}

// tianxuanPortPID 找 8094 监听进程 PID（netstat 口径，与 comfyui 端口探测同款）。
func tianxuanPortPID() int {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return 0
	}
	want := fmt.Sprintf("127.0.0.1:%d", tianxuanPort)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] == "TCP" && fields[1] == want && fields[3] == "LISTENING" {
			if pid, err := strconv.Atoi(fields[4]); err == nil {
				return pid
			}
		}
	}
	return 0
}
