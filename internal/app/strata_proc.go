package app

// strata_proc.go — Strata 引擎（独立项目 C:\AI\gaea-ENGINES\strata\repo，8091）进程生命周期：
// 状态探测 / 启动 / 停止。集成边界与天璇一致：gaea 只引用外部项目的
// terminal/serve/server.py（0.2.16+ 布局；兼容旧 serve/server.py）与档位 JSON，
// 不在 Go 常量里复制模型路径和引擎参数。
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
	strataPort               = 8091
	strataStartWait          = 300 * time.Second
	strataProbeDial          = 1500 * time.Millisecond
	strataServeEntryCurrent  = `terminal\serve\server.py`
	strataServeEntryLegacy   = `serve\server.py`
	strataPythonEntryCurrent = `terminal\.venv\Scripts\python.exe`
	strataPythonEntryLegacy  = `.venv\Scripts\python.exe`
	strataColdStartSeconds   = 180
	strataStopWait           = 5 * time.Second
	strataStopPollStep       = 500 * time.Millisecond
)

// strataServeEntryPath 选择当前项目布局的 server.py。0.2.16 起控制端与
// requirements/venv 移入 terminal/；保留旧布局探测，避免尚未升级的用户断链。
// 两边都不存在时返回当前布局路径，让调用方按“文件缺失”如实报错。
func strataServeEntryPath(project string) string {
	current := filepath.Join(project, strataServeEntryCurrent)
	if _, err := os.Stat(current); err == nil {
		return current
	}
	legacy := filepath.Join(project, strataServeEntryLegacy)
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return current
}

// strataPythonPath 与 strataServeEntryPath 同口径：新布局 terminal\.venv 优先，
// 旧根目录 .venv 兜底。
func strataPythonPath(project string) string {
	current := filepath.Join(project, strataPythonEntryCurrent)
	if _, err := os.Stat(current); err == nil {
		return current
	}
	legacy := filepath.Join(project, strataPythonEntryLegacy)
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return current
}

// strataProfiles 对齐 C:\AI\gaea-ENGINES\strata\repo 现有档位 JSON。
// 8091 由 gaea 统一传入，JSON 里的 port 字段只作项目自身默认值。
// 2026-11-24 用户拍板：Inkling-2-Mini 删除；Strata 只保留 Qwen3.8。
// Qwen 冷启动要预填 55.43GiB 专家缓存（实测约 175s），不适合请求内热启动：
// 由模型中心显式启动，启动后常驻，直到用户手动停止。
var strataProfiles = []engineProfileOption{
	{ID: "qwen38fn_flash", Label: "qwen3.8-flash-next-uncensored (UD-IQ4_XS)", ModelID: "qwen3.8-flash-next-uncensored", Config: `strata-qwen38fn-flash.json`},
}

// strataPreflightProbe 供聊天/图文预检与空闲策略注入；生产直接探 8091。
var strataPreflightProbe = strataProbe

// strataColdStartOnly：该档无法把 2–3 分钟冷启动藏进一次聊天/写作请求。
func strataColdStartOnly(profileID string) bool {
	return profileID == "qwen38fn_flash"
}

// strataExplicitStartError 给请求路径即时、可操作反馈，而不是无反馈地等三分钟。
func strataExplicitStartError(profile engineProfileOption) error {
	return fmt.Errorf("%s 冷启动约 3 分钟，已禁用请求内热启动；请先到模型中心 → 语言模型点击“启动”，就绪后再发送", profile.Label)
}

// strataQwenKeepResident：活跃 Qwen 档不参与空闲收割。冷启动成本远高于
// 常驻占用；引擎卡仍提供显式“停止引擎”。
func strataQwenKeepResident() bool {
	served, ok := strataPreflightProbe()
	return ok && strataProfileFromModels(served) == "qwen38fn_flash"
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
	python := strataPythonPath(projectPath)
	server := strataServeEntryPath(projectPath)
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
		lower := strings.ToLower(id)
		if strings.Contains(lower, "flash") {
			return "qwen38fn_flash"
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
		"running":            ok,
		"port":               strataPort,
		"profile":            profile,
		"default_profile":    "qwen38fn_flash",
		"profiles":           engineProfileJSON(strataProfiles),
		"project":            project,
		"serve_script":       strataServeEntryPath(project),
		"served_models":      served,
		"cold_start_seconds": strataColdStartSeconds,
		"auto_start":         false,
		"keep_resident":      true,
	}
}

// StrataEngineStart 显式启动 Strata 指定档位。空 profile 不隐式选择默认模型，
// 返回 model-required；Qwen 冷启动只能由模型中心显式触发，聊天/图文预检不再调用。
// 已运行→already-running；端口占用→port-occupied；就绪轮询上限 300s。
func (a *App) StrataEngineStart(profile string) map[string]interface{} {
	if strings.TrimSpace(profile) == "" {
		return map[string]interface{}{
			"started":  false,
			"reason":   "model-required",
			"profiles": engineProfileJSON(strataProfiles),
			"error":    "Strata 必须由功能绑定指定具体模型档位，不能只启动空引擎",
		}
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
	// 自管 llama（8124，语音线用）与本引擎（8091，聊天用）分板块并行：
	// 2026-11-04 用户拍板「两个都能启动，用的板块不一样」——删除原「同驻换页」
	// 互斥守卫，两引擎可同时驻留（内存吃紧由用户自行取舍启停）。
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
	python := strataPythonPath(project)
	if _, err := os.Stat(python); err != nil {
		return map[string]interface{}{
			"started": false,
			"reason":  "python-not-found",
			"profile": profile,
			"project": project,
			"python":  python,
		}
	}
	server := strataServeEntryPath(project)
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
