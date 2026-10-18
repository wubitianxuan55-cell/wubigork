package app

// strata_proc.go — Strata 引擎（独立项目 C:\AI\strata\repo，8091）进程生命周期：
// 状态探测 / 启动 / 停止。集成边界与天璇一致：gaea 只引用外部项目的
// serve/server.py 与档位 JSON，不在 Go 常量里复制模型路径和引擎参数。
// 停止走「启动器进程树 → 8091 监听 PID」两级路径，外部启动的实例也能停。

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
	strataPort         = 8091
	strataStartWait    = 300 * time.Second
	strataProbeDial    = 1500 * time.Millisecond
	strataServeEntry   = `serve\server.py`
	strataPythonEntry  = `.venv\Scripts\python.exe`
	strataStopWait     = 5 * time.Second
	strataStopPollStep = 500 * time.Millisecond
)

// strataProfiles 对齐 C:\AI\strata\repo 现有的 setup 生成档位 JSON。
// 8091 由 gaea 统一传入，JSON 里的 port 字段只作项目自身默认值。
// 目录只保留 uncensored 档（2026-10-18 拍板：qwen3.8-flash IQ2 档与
// qwen3.6 普通 UD 档退役，仅留 qwen3.6-35b-a3b-uncensored-q4_k_p）。
var strataProfiles = []engineProfileOption{
	{ID: "qwen36_uncensored", Label: "qwen3.6-35b-a3b-uncensored-q4_k_p", ModelID: "qwen3.6-35b-a3b-uncensored-q4_k_p", Config: `strata-qwen36-uncensored-q4_k_p.json`},
}

// strataProjectPath 返回 Strata 独立引擎项目根目录。配置为空时回退本机固定工作区。
func (a *App) strataProjectPath() string {
	if a != nil && a.core != nil && a.cfg != nil {
		if p := strings.TrimSpace(a.cfg.StrataPath); p != "" {
			return filepath.Clean(p)
		}
	}
	return config.DefaultStrataPath
}

// strataServeInvocation 生成稳定的外部启动命令：项目自带的 Python 入口 +
// 档位 JSON。gaea 只传 profile / 端口 / 日志目标。
func strataServeInvocation(projectPath, profile string, port int) (string, []string, string, error) {
	spec, ok := engineProfileByID(strataProfiles, profile)
	if !ok {
		return "", nil, "", fmt.Errorf("未知 Strata 档位: %q", profile)
	}
	python := filepath.Join(projectPath, strataPythonEntry)
	server := filepath.Join(projectPath, strataServeEntry)
	cfgPath := filepath.Join(projectPath, spec.Config)
	return python, []string{
		server,
		"--engine", "strata",
		"--config", cfgPath,
		"--port", strconv.Itoa(port),
	}, cfgPath, nil
}

func strataProfileFromModels(served []string) string {
	for _, id := range served {
		if strings.Contains(strings.ToLower(id), "uncensored") {
			return "qwen36_uncensored"
		}
	}
	return ""
}

// strataProbe 探测 8091 上的 Strata 引擎：可达时返回 /v1/models 的模型名清单。
func strataProbe() ([]string, bool) {
	return strataProbeAt(fmt.Sprintf("http://127.0.0.1:%d/v1", strataPort))
}

// strataProbeAt 探活本体（base 可注入，测试走 httptest）。
func strataProbeAt(base string) ([]string, bool) {
	return probeOpenAIModelsAt(base, strataProbeDial)
}

// StrataEngineStatus Strata 引擎运行状态（模型中心引擎卡的状态点与启停钮）。
func (a *App) StrataEngineStatus() map[string]interface{} {
	t0 := time.Now()
	served, ok := strataProbe()
	latency := time.Since(t0).Milliseconds()
	profile := strataProfileFromModels(served)
	a.syncManagedCatalog("strata", strataProfiles, served, profile)
	if a.core != nil && a.engineMgr != nil {
		errMsg := ""
		if !ok {
			errMsg = "引擎未运行"
		}
		a.engineMgr.SetStatus("strata", managedEngineStatus("strata", ok, len(served), latency, errMsg))
	}
	project := a.strataProjectPath()
	return map[string]interface{}{
		"running":         ok,
		"port":            strataPort,
		"profile":         profile,
		"default_profile": "qwen36_uncensored",
		"profiles":        engineProfileJSON(strataProfiles),
		"project":         project,
		"serve_script":    filepath.Join(project, strataServeEntry),
		"served_models":   served,
	}
}

// StrataEngineStart 启动 Strata 指定档位（空=唯一在册的 uncensored 档）。已运行→already-running；
// 端口占用→port-occupied；启动后轮询 /v1/models 至就绪（上限 300s）。
func (a *App) StrataEngineStart(profile string) map[string]interface{} {
	if strings.TrimSpace(profile) == "" {
		profile = "qwen36_uncensored"
	}
	project := a.strataProjectPath()
	if served, ok := strataProbe(); ok {
		return map[string]interface{}{
			"started":       false,
			"reason":        "already-running",
			"profile":       strataProfileFromModels(served),
			"profiles":      engineProfileJSON(strataProfiles),
			"project":       project,
			"served_models": served,
		}
	}
	// 与天璇共用同一块 UMA/VRAM；同时驻留会直接拖进换页，先停天璇再启 Strata。
	if _, ok := tianxuanProbe(); ok {
		return map[string]interface{}{
			"started": false,
			"reason":  "other-engine-running",
			"other":   "tianxuan",
			"profile": profile,
			"project": project,
		}
	}
	if ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", strataPort)); err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "port-occupied",
			"port":    strataPort,
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
	python := filepath.Join(project, strataPythonEntry)
	if _, err := os.Stat(python); err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "python-not-found",
			"profile": profile,
			"project": project,
			"python":  python,
		}
	}
	server := filepath.Join(project, strataServeEntry)
	if _, err := os.Stat(server); err != nil {
		return map[string]interface{}{
			"started":      false,
			"reason":       "serve-script-not-found",
			"profile":      profile,
			"project":      project,
			"serve_script": server,
		}
	}

	logDir := filepath.Join(config.DataRoot(), "logs", "strata")
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
	stdoutLog := base + ".stdout.log"
	stderrLog := base + ".stderr.log"

	bin, argv, cfgPath, err := strataServeInvocation(project, profile, strataPort)
	if err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "invalid-profile",
			"profile": profile,
			"project": project,
			"error":   err.Error(),
		}
	}
	if _, err := os.Stat(cfgPath); err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "config-not-found",
			"profile": profile,
			"project": project,
			"config":  cfgPath,
		}
	}

	stdoutFile, err := os.Create(stdoutLog)
	if err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "log-open-failed",
			"profile": profile,
			"project": project,
			"error":   err.Error(),
		}
	}
	stderrFile, err := os.Create(stderrLog)
	if err != nil {
		_ = stdoutFile.Close()
		return map[string]interface{}{
			"started": false,
			"reason":  "log-open-failed",
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
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	if err := cmd.Start(); err != nil {
		cancel()
		_ = stdoutFile.Close()
		_ = stderrFile.Close()
		return map[string]interface{}{
			"started": false,
			"reason":  "spawn-failed",
			"profile": profile,
			"project": project,
			"error":   err.Error(),
		}
	}
	a.strataProc.setWithJob(cmd, cancel)
	// 启动即记账「真实使用」：空闲自动退出从这一刻起算（同 tianxuan_proc）。
	if a.core != nil && a.engineMgr != nil {
		a.engineMgr.TouchEngine("strata")
	}
	waitDone := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		_ = stdoutFile.Close()
		_ = stderrFile.Close()
		a.strataProc.clearIfCurrent(cmd)
		close(waitDone)
	}()

	t0 := time.Now()
	deadline := t0.Add(strataStartWait)
	for time.Now().Before(deadline) {
		if served, ok := strataProbe(); ok {
			detected := strataProfileFromModels(served)
			if detected == "" {
				detected = profile
			}
			a.syncManagedCatalog("strata", strataProfiles, served, detected)
			if a.core != nil && a.engineMgr != nil {
				a.engineMgr.SetStatus("strata", managedEngineStatus("strata", true, len(served), time.Since(t0).Milliseconds(), ""))
			}
			return map[string]interface{}{
				"started":       true,
				"waited_ms":     time.Since(t0).Milliseconds(),
				"served_models": served,
				"profile":       detected,
				"profiles":      engineProfileJSON(strataProfiles),
				"project":       project,
				"config":        cfgPath,
				"stdout_log":    stdoutLog,
				"stderr_log":    stderrLog,
				"serve_script":  server,
			}
		}
		select {
		case <-waitDone:
			return map[string]interface{}{
				"started":      false,
				"reason":       "exited-early",
				"error":        readLogTail(stderrLog, 1600),
				"profile":      profile,
				"project":      project,
				"config":       cfgPath,
				"stdout_log":   stdoutLog,
				"stderr_log":   stderrLog,
				"serve_script": server,
			}
		case <-ctx.Done():
			return map[string]interface{}{
				"started":      false,
				"reason":       "canceled",
				"profile":      profile,
				"project":      project,
				"config":       cfgPath,
				"stdout_log":   stdoutLog,
				"stderr_log":   stderrLog,
				"serve_script": server,
			}
		case <-time.After(2 * time.Second):
		}
	}
	return map[string]interface{}{
		"started":      false,
		"reason":       "timeout",
		"profile":      profile,
		"project":      project,
		"config":       cfgPath,
		"stdout_log":   stdoutLog,
		"stderr_log":   stderrLog,
		"serve_script": server,
	}
}

// StrataEngineStop 停止 Strata 引擎：登记进程树 → 8091 监听 PID 兜底。
func (a *App) StrataEngineStop() map[string]interface{} {
	res := stopManagedEngine(&a.strataProc, strataPort, strataStopWait, strataStopPollStep, func() bool {
		_, ok := strataProbe()
		return ok
	})
	if stopped, _ := res["stopped"].(bool); stopped {
		a.markManagedCatalogStopped("strata", strataProfiles)
	}
	return res
}
