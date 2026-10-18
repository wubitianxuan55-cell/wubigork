package app

// tianxuan_proc.go — 天璇引擎（独立项目 C:\AI\tianxuan，8094）进程生命周期：
// 状态探测 / 启动 / 停止。集成边界与 ComfyUI 同型：gaea 只保存外部项目路径，
// 调用项目自带的稳定入口 scripts\serve.ps1；二进制、模型、档位参数与日志约定
// 全部由 tianxuan 项目负责。gaea 不导入引擎源码，也不在 Go 常量里复制启动参数。
// 停止走「启动器进程树 → 8094 监听 PID」两级路径——与 StopComfyUI 同口径
// 「不管谁启动的都能停」。纪律：引擎生成中硬杀有 D 状态风险，UI 侧应在空闲时
// 调用；这里不做生成中检测（chat 链路与本文件无耦合，探活只看端口）。

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gaea/gaea/internal/config"
)

const (
	tianxuanPort         = 8094
	tianxuanStartWait    = 180 * time.Second
	tianxuanProbeDial    = 1500 * time.Millisecond
	tianxuanServeScript  = `scripts\serve.ps1`
	tianxuanStopWait     = 5 * time.Second
	tianxuanStopPollStep = 500 * time.Millisecond
)

// engineProfileOption 是外部引擎档位的展示/契约单一来源。ID 必须与外部项目
// 启动脚本的 ValidateSet 一致；Label 只用于模型中心展示。
type engineProfileOption struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Config  string `json:"-"` // Strata 档位 JSON（天璇为空）
	ModelID string `json:"-"` // 该档位实际服务的模型 id（目录/请求路由用）
}

// tianxuanProfiles 对齐 C:\AI\tianxuan\scripts\serve.ps1 V1.0 的 ValidateSet。
// 2026-10-18 拍板：base/fast（普通 Qwen3.6-35B 两档）退役删除；保留两个
// uncensored 档——uncensored（qwen3.6-35b-a3b-uncensored 主力）与 flash
// （qwen3.8-flash-next-uncensored，177B 消融深度档，262K 上下文）。
var tianxuanProfiles = []engineProfileOption{
	{ID: "uncensored", Label: "qwen3.6-35b-a3b-uncensored", ModelID: "qwen3.6-35b-a3b-uncensored"},
	{ID: "flash", Label: "qwen3.8-flash-next-uncensored", ModelID: "qwen3.8-flash-next-uncensored"},
}

func engineProfileByID(profiles []engineProfileOption, id string) (engineProfileOption, bool) {
	for _, p := range profiles {
		if p.ID == id {
			return p, true
		}
	}
	return engineProfileOption{}, false
}

func engineProfileJSON(profiles []engineProfileOption) []map[string]string {
	out := make([]map[string]string, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, map[string]string{"id": p.ID, "label": p.Label})
	}
	return out
}

// tianxuanProjectPath 返回独立引擎项目根目录。配置为空时回退本机固定工作区。
func (a *App) tianxuanProjectPath() string {
	if a != nil && a.core != nil && a.cfg != nil {
		if p := strings.TrimSpace(a.cfg.TianxuanPath); p != "" {
			return filepath.Clean(p)
		}
	}
	return config.DefaultTianxuanPath
}

// tianxuanServeInvocation 生成稳定的外部启动命令。项目脚本拥有全部档位语义；
// gaea 只传档位、端口和日志目标。独立成纯函数便于钉住绑定契约。
func tianxuanServeInvocation(projectPath, profile string, port int, stdoutLog, stderrLog string) (string, []string, error) {
	if _, ok := engineProfileByID(tianxuanProfiles, profile); !ok {
		return "", nil, fmt.Errorf("未知天璇档位: %q", profile)
	}
	script := filepath.Join(projectPath, tianxuanServeScript)
	return "powershell.exe", []string{
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-File", script,
		"-Profile", profile,
		"-Port", strconv.Itoa(port),
		"-StdoutLog", stdoutLog,
		"-StderrLog", stderrLog,
	}, nil
}

func tianxuanProfileFromModels(served []string) string {
	for _, id := range served {
		l := strings.ToLower(id)
		switch {
		case strings.Contains(l, "flash-next"), strings.Contains(l, "flashnext"):
			// flash 档模型名（qwen3.8-flash-next-uncensored）同时含 uncensored
			// 字样，flash 判定必须在前。
			return "flash"
		case strings.Contains(l, "uncensored"):
			return "uncensored"
		}
	}
	return ""
}

// tianxuanProbe 探测 8094 上的天璇引擎：可达时返回 /v1/models 的模型名清单。
func tianxuanProbe() ([]string, bool) {
	return tianxuanProbeAt(fmt.Sprintf("http://127.0.0.1:%d", tianxuanPort))
}

// tianxuanProbeAt 探活本体（base 为服务根，自动拼 /v1/models；测试走 httptest）。
func tianxuanProbeAt(base string) ([]string, bool) {
	return probeOpenAIModelsAt(strings.TrimRight(base, "/")+"/v1", tianxuanProbeDial)
}

// TianxuanEngineStatus 天璇引擎运行状态（模型中心引擎卡的状态点与启停钮）。
func (a *App) TianxuanEngineStatus() map[string]interface{} {
	t0 := time.Now()
	served, ok := tianxuanProbe()
	latency := time.Since(t0).Milliseconds()
	profile := tianxuanProfileFromModels(served)
	a.syncManagedCatalog("tianxuan", tianxuanProfiles, served, profile)
	if a.core != nil && a.engineMgr != nil {
		errMsg := ""
		if !ok {
			errMsg = "引擎未运行"
		}
		a.engineMgr.SetStatus("tianxuan", managedEngineStatus("tianxuan", ok, len(served), latency, errMsg))
	}
	return map[string]interface{}{
		"running":         ok,
		"port":            tianxuanPort,
		"profile":         profile,
		"default_profile": "uncensored",
		"profiles":        engineProfileJSON(tianxuanProfiles),
		"project":         a.tianxuanProjectPath(),
		"serve_script":    filepath.Join(a.tianxuanProjectPath(), tianxuanServeScript),
		"served_models":   served,
	}
}

// TianxuanEngineStart 启动天璇引擎指定档位（空=唯一在册的 uncensored 档）。已在运行→幂等返回
// already-running；端口被占用→如实报 port-occupied（与其它引擎互斥纪律）。
// 启动后轮询 /v1/models 至就绪（上限 180s），超时/早退如实报错并附日志路径。
func (a *App) TianxuanEngineStart(profile string) map[string]interface{} {
	if strings.TrimSpace(profile) == "" {
		profile = "uncensored"
	}
	return a.tianxuanEngineStart(profile)
}

func (a *App) tianxuanEngineStart(profile string) map[string]interface{} {
	project := a.tianxuanProjectPath()
	if served, ok := tianxuanProbe(); ok {
		return map[string]interface{}{
			"started":       false,
			"reason":        "already-running",
			"profile":       tianxuanProfileFromModels(served),
			"profiles":      engineProfileJSON(tianxuanProfiles),
			"project":       project,
			"served_models": served,
		}
	}
	// 两台外部引擎都吃同一块 UMA/VRAM；同时驻留会直接把机器拖进换页。
	// 与其它本地引擎互斥：先停 Strata 再启天璇。
	if _, ok := strataProbe(); ok {
		return map[string]interface{}{
			"started": false,
			"reason":  "other-engine-running",
			"other":   "strata",
			"profile": profile,
			"project": project,
		}
	}
	if ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", tianxuanPort)); err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "port-occupied",
			"port":    tianxuanPort,
			"profile": profile,
			"project": project,
		}
	} else {
		_ = ln.Close()
	}

	info, err := os.Stat(project)
	if err != nil || !info.IsDir() {
		return map[string]interface{}{
			"started": false,
			"reason":  "project-not-found",
			"profile": profile,
			"project": project,
		}
	}
	script := filepath.Join(project, tianxuanServeScript)
	if _, err := os.Stat(script); err != nil {
		return map[string]interface{}{
			"started":      false,
			"reason":       "serve-script-not-found",
			"profile":      profile,
			"project":      project,
			"serve_script": script,
		}
	}

	logDir := filepath.Join(config.DataRoot(), "logs", "tianxuan")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "log-dir-create-failed",
			"profile": profile,
			"project": project,
			"error":   err.Error(),
		}
	}
	base := filepath.Join(logDir, fmt.Sprintf("%s-%s", profile, time.Now().Format("20060102-150405")))
	wrapperLog := base + ".wrapper.log"
	stdoutLog := base + ".stdout.log"
	stderrLog := base + ".stderr.log"

	bin, argv, err := tianxuanServeInvocation(project, profile, tianxuanPort, stdoutLog, stderrLog)
	if err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "invalid-profile",
			"profile": profile,
			"project": project,
			"error":   err.Error(),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, argv...)
	cmd.Dir = project
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	logFile, err := os.Create(wrapperLog)
	if err != nil {
		cancel()
		return map[string]interface{}{
			"started": false,
			"reason":  "log-open-failed",
			"profile": profile,
			"project": project,
			"error":   err.Error(),
		}
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		cancel()
		_ = logFile.Close()
		return map[string]interface{}{
			"started": false,
			"reason":  "spawn-failed",
			"profile": profile,
			"project": project,
			"error":   err.Error(),
		}
	}
	a.tianxuanProc.setWithJob(cmd, cancel)
	// 启动即记账「真实使用」：空闲自动退出从这一刻起算，模型拉起后一直没用
	// 也会在 N 分钟后被收掉（用户诉求：不要白占机器）。
	if a.core != nil && a.engineMgr != nil {
		a.engineMgr.TouchEngine("tianxuan")
	}
	waitDone := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
		a.tianxuanProc.clearIfCurrent(cmd)
		close(waitDone)
	}()

	t0 := time.Now()
	deadline := t0.Add(tianxuanStartWait)
	for time.Now().Before(deadline) {
		if served, ok := tianxuanProbe(); ok {
			profile = tianxuanProfileFromModels(served)
			a.syncManagedCatalog("tianxuan", tianxuanProfiles, served, profile)
			if a.core != nil && a.engineMgr != nil {
				a.engineMgr.SetStatus("tianxuan", managedEngineStatus("tianxuan", true, len(served), time.Since(t0).Milliseconds(), ""))
			}
			return map[string]interface{}{
				"started":       true,
				"waited_ms":     time.Since(t0).Milliseconds(),
				"served_models": served,
				"profile":       profile,
				"profiles":      engineProfileJSON(tianxuanProfiles),
				"project":       project,
				"log":           wrapperLog,
				"stdout_log":    stdoutLog,
				"stderr_log":    stderrLog,
				"serve_script":  script,
			}
		}
		select {
		case <-waitDone:
			return map[string]interface{}{
				"started":      false,
				"reason":       "exited-early",
				"error":        readLogTail(wrapperLog, 1600),
				"profile":      profile,
				"project":      project,
				"log":          wrapperLog,
				"stdout_log":   stdoutLog,
				"stderr_log":   stderrLog,
				"serve_script": script,
			}
		case <-ctx.Done():
			return map[string]interface{}{
				"started":      false,
				"reason":       "canceled",
				"profile":      profile,
				"project":      project,
				"log":          wrapperLog,
				"stdout_log":   stdoutLog,
				"stderr_log":   stderrLog,
				"serve_script": script,
			}
		case <-time.After(2 * time.Second):
		}
	}
	return map[string]interface{}{
		"started":      false,
		"reason":       "timeout",
		"profile":      profile,
		"project":      project,
		"log":          wrapperLog,
		"stdout_log":   stdoutLog,
		"stderr_log":   stderrLog,
		"serve_script": script,
	}
}

// TianxuanEngineStop 停止天璇引擎：先结束 gaea 登记的启动器进程树，再按
// 8094 监听 PID 兜底（外部启动器起的实例也能停）。停止后轮询端口释放。
func (a *App) TianxuanEngineStop() map[string]interface{} {
	res := stopManagedEngine(&a.tianxuanProc, tianxuanPort, tianxuanStopWait, tianxuanStopPollStep, func() bool {
		_, ok := tianxuanProbe()
		return ok
	})
	if stopped, _ := res["stopped"].(bool); stopped {
		a.markManagedCatalogStopped("tianxuan", tianxuanProfiles)
	}
	return res
}
