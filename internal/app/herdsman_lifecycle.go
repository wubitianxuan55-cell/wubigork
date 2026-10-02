package app

// Herdsman 模型生命周期（P2）：通过 herdsman.exe skill models 命令启动/停止/
// 下载/卸载模型，并读取 launch_records 生成本机实测启动参数预设。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// herdsmanOpMu 串行化 Herdsman 模型生命周期操作（E1-4）：
// herdsman 自身 model_scheduling.local_concurrency=1（同一时间只服务一个模型），
// 且 CLI 调用昂贵（下载最长 60 分钟、冷启动 20 分钟），并发发起会互相排队/冲突。
// 统一串行后，前端批量启停天然变为有序队列（配合前端 busy 状态展示）。
var herdsmanOpMu sync.Mutex

// herdsmanOpCLIResult 是生命周期命令的 JSON 信封。注意 error 字段两态并存：
// 字符串（老版本）与对象 {code,message}（v4.9.1 真机实测 unavailable 时为
// 对象）——用 RawMessage 兜底，文案提取走 herdsmanEnvelopeError。
type herdsmanOpCLIResult struct {
	OK     bool            `json:"ok"`
	Error  json.RawMessage `json:"error,omitempty"`
	Result struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	} `json:"result"`
}

// herdsmanEnvelopeError 从 CLI 输出提取 {ok:false, error:...} 的错误文案。
// error 字段字符串/对象两态兼容；ok=true 或无法解析返回 ""。
func herdsmanEnvelopeError(data []byte) string {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var probe struct {
		OK    bool            `json:"ok"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(data, &probe); err != nil || probe.OK {
		return ""
	}
	var msg string
	if json.Unmarshal(probe.Error, &msg) == nil {
		return strings.TrimSpace(msg)
	}
	var obj struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(probe.Error, &obj) == nil && obj.Message != "" {
		return strings.TrimSpace(obj.Message)
	}
	return ""
}

// herdsmanErrorHint 对高频故障给出定向修复提示。v4.9.1 真机实测：Herdsman
// 桌面端以管理员身份运行时，其 skill 控制管道（\\.\pipe\Herdsman-skill-v1）
// 的 DACL 只允许提权令牌访问，普通权限的 gaea 打开即 Access is denied——
// 旧文案「请确认桌面端已启动」完全误导（桌面端明明在跑）。
func herdsmanErrorHint(msg string) string {
	if strings.Contains(msg, "Access is denied") {
		return msg + "（疑似 Herdsman 以管理员权限运行：普通权限的 gaea 无权连接其控制管道，请用普通方式重启 Herdsman 桌面端后重试）"
	}
	return msg
}

// killProcessTree 强制终止整棵进程树（含全部子孙进程），返回清理结果。
//
// Windows：走 `taskkill /T /F`。**必须在父进程仍存活时调用**——/T 依赖活动的
// 父子关系枚举子孙；父进程一死，herdsman.exe 拉起的常驻模型/推理服务就成了
// 孤儿（不再有可回收句柄），显存只能手工排查。
// 非 Windows（跨平台构建/单测）：无 taskkill，退化为杀单进程。
func killProcessTree(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("非法 PID: %d", pid)
	}
	if runtime.GOOS != "windows" {
		p, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return p.Kill()
	}
	out, err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill /T /F /PID %d 失败: %w（输出: %s）", pid, err, truncateRunes(strings.TrimSpace(string(out)), 200))
	}
	return nil
}

// runHerdsmanCLI 执行 herdsman.exe 子命令。非零退出时优先透出 stdout JSON 里的
// 结构化错误（CLI 把 ok=false + error 写 stdout 后 exit 3 等）——此前用
// Output() 在失败路径丢弃 stdout，模型中心只显示「exit status 3」，真实原因
// 全被吞掉。
//
// 超时清理（审计 P0 AP9-02）：刻意**不用** exec.CommandContext——它的超时只会
// TerminateProcess 直接子进程，而 `skill models download/start` 拉起的是常驻
// 模型服务（子孙进程），父进程被单独杀掉后服务仍在占显存；用户重试再起一个，
// 显存打满、后续本地模型功能全失败。改为自行 Start + 等 ctx：超时时趁父进程
// 还在，用 taskkill /T 杀整棵树，随后回收句柄，并把「超时 + 清理结果」写进
// 返回的 error 文案与日志。
func runHerdsmanCLI(timeout time.Duration, args ...string) ([]byte, error) {
	exe := herdsmanExePath()
	if exe == "" {
		return nil, errors.New("未找到 herdsman.exe，请安装 Herdsman 或设置 HERDSMAN_EXE 环境变量")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.Command(exe, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("herdsman CLI 调用失败: %w", err)
	}
	// 带缓冲通道：超时分支即使不等 Wait 也不泄漏收尾 goroutine。
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()

	select {
	case err := <-waited:
		if err != nil {
			if msg := herdsmanEnvelopeError(stdout.Bytes()); msg != "" {
				return nil, fmt.Errorf("herdsman CLI 失败: %s", herdsmanErrorHint(msg))
			}
			if s := strings.TrimSpace(stderr.String()); s != "" {
				return nil, fmt.Errorf("herdsman CLI 调用失败: %w（stderr: %s）", err, truncateRunes(s, 200))
			}
			return nil, fmt.Errorf("herdsman CLI 调用失败: %w", err)
		}
		return stdout.Bytes(), nil
	case <-ctx.Done():
		pid := cmd.Process.Pid
		cleanup := "已清理整棵进程树"
		if killErr := killProcessTree(pid); killErr != nil {
			cleanup = "进程树清理失败: " + killErr.Error()
			slog.Error("herdsman CLI 超时清理进程树失败（模型服务可能仍在占显存）",
				"pid", pid, "timeout", timeout.String(), "error", killErr)
		} else {
			slog.Warn("herdsman CLI 超时，已清理整棵进程树", "pid", pid, "timeout", timeout.String(), "args", args)
		}
		// 回收：杀树成功后 Wait 立即返回；清理失败时不无限等（进程可能还要跑
		// 几十分钟），只做有界等待，句柄由带缓冲通道兜底不泄漏。
		//
		// 纪律：**只有 Wait 返回后**才允许读 stdout/stderr。cmd.Stdout/Stderr
		// 不是 *os.File，os/exec 会各起一个拷贝 goroutine 写这两个 bytes.Buffer
		// 并由 Wait join；5 秒兜底分支胜出时拷贝可能仍在写，此时读缓冲既是
		// data race、也只会拿到半截输出（进程未退出，输出本来就不完整）。
		select {
		case <-waited:
			if msg := herdsmanEnvelopeError(stdout.Bytes()); msg != "" {
				return nil, fmt.Errorf("herdsman CLI 超时（%s，PID %d，%s）: %s", timeout, pid, cleanup, herdsmanErrorHint(msg))
			}
			if s := strings.TrimSpace(stderr.String()); s != "" {
				return nil, fmt.Errorf("herdsman CLI 超时（%s，PID %d，%s）: %w（stderr: %s）", timeout, pid, cleanup, ctx.Err(), truncateRunes(s, 200))
			}
			return nil, fmt.Errorf("herdsman CLI 超时（%s，PID %d，%s）: %w", timeout, pid, cleanup, ctx.Err())
		case <-time.After(5 * time.Second):
			slog.Error("herdsman CLI 超时清理后进程仍未退出", "pid", pid, "args", args)
			return nil, fmt.Errorf("herdsman CLI 超时（%s，PID %d，%s）：进程在清理后仍未退出，输出不可用", timeout, pid, cleanup)
		}
	}
}

type herdsmanLaunchRecord struct {
	ModelName       string         `json:"model_name"`
	InferenceEngine string         `json:"inference_engine"`
	StartTime       string         `json:"start_time"`
	Port            int            `json:"port"`
	Status          string         `json:"status"`
	Options         map[string]any `json:"options"`
}

// herdsmanCLIWithTimeout 与 herdsmanCLI 相同，但可指定超时（长任务：下载/冷启动）。
var herdsmanCLIWithTimeout = func(timeout time.Duration, args ...string) ([]byte, error) {
	return runHerdsmanCLI(timeout, args...)
}

// HerdsmanOpResult 是一次生命周期操作的结果（对齐 CLI JSON）。
type HerdsmanOpResult struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HerdsmanLaunchPreset 是某模型在本机的实测启动参数（来自 launch_records）。
type HerdsmanLaunchPreset struct {
	Model     string         `json:"model"`
	Engine    string         `json:"engine"`
	Port      int            `json:"port"`
	StartedAt string         `json:"started_at"`
	Options   map[string]any `json:"options"`
}

// parseHerdsmanOpResult 解析生命周期命令的 JSON：ok=false 或 result.status
// 非 completed 时返回错误（携带 CLI 错误文案）。
func parseHerdsmanOpResult(data []byte) (HerdsmanOpResult, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var resp herdsmanOpCLIResult
	if err := json.Unmarshal(data, &resp); err != nil {
		return HerdsmanOpResult{}, fmt.Errorf("解析 herdsman 操作结果失败: %w", err)
	}
	out := HerdsmanOpResult{OK: resp.OK, Status: resp.Result.Status}
	if resp.Result.Status == "" && resp.OK {
		out.Status = "completed"
	}
	// 信封 error 两态兼容（字符串/对象）；result.error 为字符串。
	msg := herdsmanEnvelopeError(data)
	if msg == "" {
		msg = strings.TrimSpace(resp.Result.Error)
	}
	out.Message = msg
	if !resp.OK || (resp.Result.Status != "" && resp.Result.Status != "completed") {
		if out.Message == "" {
			out.Message = "操作未完成（status=" + out.Status + "）"
		}
		return out, errors.New(out.Message)
	}
	return out, nil
}

// herdsmanDataDir 定位 Herdsman 数据目录（HERDSMAN_DATA_DIR 优先）。
func herdsmanDataDir() string {
	if p := strings.TrimSpace(os.Getenv("HERDSMAN_DATA_DIR")); p != "" {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".herdsman")
	}
	return ""
}

// parseHerdsmanLaunchPresets 读取 launch_records/*.json，每个模型取最近一次
// 成功启动的参数。
func parseHerdsmanLaunchPresets(dir string) ([]HerdsmanLaunchPreset, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}
	latest := map[string]herdsmanLaunchRecord{}
	var order []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var records []herdsmanLaunchRecord
		if err := json.Unmarshal(data, &records); err != nil {
			continue
		}
		for _, r := range records {
			if r.Status != "success" || r.ModelName == "" {
				continue
			}
			cur, ok := latest[r.ModelName]
			if !ok || r.StartTime > cur.StartTime {
				if !ok {
					order = append(order, r.ModelName)
				}
				latest[r.ModelName] = r
			}
		}
	}
	sort.Strings(order)
	out := make([]HerdsmanLaunchPreset, 0, len(order))
	for _, name := range order {
		r := latest[name]
		out = append(out, HerdsmanLaunchPreset{
			Model:     r.ModelName,
			Engine:    r.InferenceEngine,
			Port:      r.Port,
			StartedAt: r.StartTime,
			Options:   r.Options,
		})
	}
	return out, nil
}

// HerdsmanLaunchPresets 返回本机所有模型的启动参数预设。
func (a *App) HerdsmanLaunchPresets() ([]HerdsmanLaunchPreset, error) {
	dir := filepath.Join(herdsmanDataDir(), "launch_records")
	if dir == filepath.Join("", "launch_records") || dir == `\launch_records` {
		return nil, errors.New("无法定位 Herdsman 数据目录")
	}
	presets, err := parseHerdsmanLaunchPresets(dir)
	if err != nil {
		return nil, fmt.Errorf("读取启动参数预设失败: %w", err)
	}
	return presets, nil
}

// HerdsmanModelStart 启动模型（--wait 等冷启动完成）。
func (a *App) HerdsmanModelStart(model string) (HerdsmanOpResult, error) {
	if strings.TrimSpace(model) == "" {
		return HerdsmanOpResult{}, errors.New("模型名不能为空")
	}
	herdsmanOpMu.Lock()
	defer herdsmanOpMu.Unlock()
	out, err := herdsmanCLIWithTimeout(20*time.Minute, "skill", "models", "start", "--model", model, "--wait")
	if err != nil {
		return HerdsmanOpResult{Message: err.Error()}, err
	}
	return parseHerdsmanOpResult(out)
}

// HerdsmanModelStop 停止模型。
func (a *App) HerdsmanModelStop(model string) (HerdsmanOpResult, error) {
	if strings.TrimSpace(model) == "" {
		return HerdsmanOpResult{}, errors.New("模型名不能为空")
	}
	herdsmanOpMu.Lock()
	defer herdsmanOpMu.Unlock()
	out, err := herdsmanCLIWithTimeout(3*time.Minute, "skill", "models", "stop", "--model", model, "--force")
	if err != nil {
		return HerdsmanOpResult{Message: err.Error()}, err
	}
	return parseHerdsmanOpResult(out)
}

// HerdsmanModelDownload 下载模型（--wait 等安装完成）。
func (a *App) HerdsmanModelDownload(model string) (HerdsmanOpResult, error) {
	if strings.TrimSpace(model) == "" {
		return HerdsmanOpResult{}, errors.New("模型名不能为空")
	}
	herdsmanOpMu.Lock()
	defer herdsmanOpMu.Unlock()
	out, err := herdsmanCLIWithTimeout(60*time.Minute, "skill", "models", "download", "--model", model, "--wait")
	if err != nil {
		return HerdsmanOpResult{Message: err.Error()}, err
	}
	return parseHerdsmanOpResult(out)
}

// HerdsmanModelUninstall 卸载已安装模型（前端需二次确认）。
func (a *App) HerdsmanModelUninstall(model string) (HerdsmanOpResult, error) {
	if strings.TrimSpace(model) == "" {
		return HerdsmanOpResult{}, errors.New("模型名不能为空")
	}
	herdsmanOpMu.Lock()
	defer herdsmanOpMu.Unlock()
	out, err := herdsmanCLIWithTimeout(5*time.Minute, "skill", "models", "uninstall", "--model", model, "--force")
	if err != nil {
		return HerdsmanOpResult{Message: err.Error()}, err
	}
	return parseHerdsmanOpResult(out)
}
