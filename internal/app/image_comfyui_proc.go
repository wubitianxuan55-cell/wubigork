package app

// image_comfyui_proc.go — ComfyUI 进程管理与系统统计族（P5 破防专项拆自
// image_handler.go：进程引用/启停恢复/端口探测/系统资源统计——搬移零功能变更）。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/gaea/gaea/internal/netclient"
)

func (a *mediaState) comfyProcRefSet(cancel context.CancelFunc, cmd *exec.Cmd) {
	a.comfyProcMu.Lock()
	a.comfyUICancel, a.comfyUICmd = cancel, cmd
	a.comfyProcMu.Unlock()
}

// comfyProcRefClearIfCurrent 仅当登记的仍是 want 这一轮进程时清空引用。
// 快速「停-启」后上一轮的 cmd.Wait 回收 goroutine 可能晚到，不得误清新一轮登记。
// 身份判据用 cmd 指针（每轮 Start 必新建 exec.Cmd，func 值不可比较）；
// 仅在 cmd.Start 成功后调用，故 cmd 恒非 nil。
func (a *mediaState) comfyProcRefClearIfCurrent(cancel context.CancelFunc, cmd *exec.Cmd) {
	a.comfyProcMu.Lock()
	if cmd != nil && a.comfyUICmd == cmd {
		a.comfyUICancel, a.comfyUICmd = nil, nil
	}
	a.comfyProcMu.Unlock()
}

// comfyProcRefStop 杀掉登记的进程并清空引用（StopComfyUI 的内部引用路径）。
func (a *mediaState) comfyProcRefStop() {
	a.comfyProcMu.Lock()
	if a.comfyUICmd != nil && a.comfyUICmd.Process != nil {
		_ = a.comfyUICmd.Process.Kill()
	}
	if a.comfyUICancel != nil {
		a.comfyUICancel()
	}
	a.comfyUICancel, a.comfyUICmd = nil, nil
	a.comfyProcMu.Unlock()
}

// comfyProcRefClear 无条件清空引用（GetComfyUIStatus 探活失败时的监控清理）。
func (a *mediaState) comfyProcRefClear() {
	a.comfyProcMu.Lock()
	a.comfyUICancel, a.comfyUICmd = nil, nil
	a.comfyProcMu.Unlock()
}

// StartComfyUI 启动 ComfyUI 服务
func (a *mediaState) StartComfyUI() error {
	if a.cfg.ComfyUIPath == "" {
		return fmt.Errorf("请先在设置中配置 ComfyUI 安装路径")
	}
	// 检查是否已运行
	if a.isComfyUIRunning() {
		return fmt.Errorf("ComfyUI 已在运行")
	}
	// 端口已被占用但 /system_stats 未就绪（实例正在启动/卡死）：直接复用现有进程，
	// 避免再拉起第二个实例导致 "Port 8188 already in use" 与 SQLite 数据库锁冲突。
	if pid := findProcessByPort(extractPort(a.cfg.ComfyUIURL)); pid > 0 {
		slog.Warn("ComfyUI 端口已被占用，跳过重复启动", "port", extractPort(a.cfg.ComfyUIURL), "pid", pid)
		return fmt.Errorf("端口 %s 已被进程 %d 占用（ComfyUI 可能正在启动），请稍候重试", extractPort(a.cfg.ComfyUIURL), pid)
	}

	// 检查 main.py 是否存在
	mainPy := filepath.Join(a.cfg.ComfyUIPath, "main.py")
	if _, err := os.Stat(mainPy); os.IsNotExist(err) {
		return fmt.Errorf("在 %s 中未找到 main.py，请确认 ComfyUI 安装路径正确", a.cfg.ComfyUIPath)
	}

	// 查找可用 Python 解释器
	pythonExe := findPython(a.cfg.ComfyUIPath, a.cfg.ComfyUIPythonPath)
	if pythonExe == "" {
		return fmt.Errorf("未找到 Python，请确认 Python 已安装。可在设置中指定 Python 解释器路径，或确保 python/py 在 PATH 中")
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.comfyProcRefSet(cancel, nil)

	// 构建启动参数
	args := []string{"main.py", "--listen", "127.0.0.1", "--port", extractPort(a.cfg.ComfyUIURL)}
	// 使用内置 Python / standalone-env 时加 --windows-standalone-build
	//（standalone-env 是 ROCm PyTorch 环境，Krea2/Z-Image-Turbo 必需；系统 Python 为 CPU-only）
	if strings.Contains(pythonExe, "python\\python.exe") || strings.Contains(pythonExe, "python_embeded") || strings.Contains(pythonExe, "standalone-env") { //nolint:misspell // python_embeded 系上游真实目录名
		args = append(args, "--windows-standalone-build")
	}
	// 不强制指定 GPU 后端，让 ComfyUI 自动检测（支持 NVIDIA CUDA / AMD ROCm / DirectML）
	// 若需要强制 CPU 模式，可在设置中指定 `--cpu` 参数

	cmd := exec.CommandContext(ctx, pythonExe, args...)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8", "TQDM_DISABLE=1")
	cmd.Dir = a.cfg.ComfyUIPath
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// stdout/stderr 重定向到日志文件（~/.gaea/logs/comfyui.log）：
	// 文件句柄在 gaea 进程退出后依然有效，避免孤儿 ComfyUI 实例的 stderr
	// 失效，导致生成时 tqdm flush 报 [Errno 22] Invalid argument。
	var logFile *os.File
	if home, err := os.UserHomeDir(); err == nil {
		logDir := filepath.Join(home, ".gaea", "logs")
		if err := os.MkdirAll(logDir, 0755); err == nil {
			if f, err := os.OpenFile(filepath.Join(logDir, "comfyui.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				logFile = f
			}
		}
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		cancel()
		a.comfyProcRefClear()
		if logFile != nil {
			_ = logFile.Close()
		}
		errMsg := ""
		if home, err := os.UserHomeDir(); err == nil {
			if b, err := os.ReadFile(filepath.Join(home, ".gaea", "logs", "comfyui.log")); err == nil {
				if len(b) > 4096 {
					b = b[len(b)-4096:]
				}
				errMsg = string(b)
			}
		}
		if len(errMsg) > 600 {
			errMsg = "..." + errMsg[len(errMsg)-600:]
		}
		if errMsg != "" {
			return fmt.Errorf("启动 ComfyUI 失败: %w\n%s", err, errMsg)
		}
		return fmt.Errorf("启动 ComfyUI 失败: %w（Python=%s, Dir=%s）", err, pythonExe, a.cfg.ComfyUIPath)
	}

	slog.Info("ComfyUI 已启动", "python", pythonExe, "dir", a.cfg.ComfyUIPath, "pid", cmd.Process.Pid)
	a.comfyProcRefSet(cancel, cmd)

	// 后台等待进程结束，记录退出原因
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("image: comfyui wait goroutine panic recovered", "panic", r)
			}
			if logFile != nil {
				_ = logFile.Close()
			}
		}()
		if err := cmd.Wait(); err != nil {
			slog.Warn("ComfyUI 进程退出", "error", err)
		}
		a.comfyProcRefClearIfCurrent(cancel, cmd)
	}()

	return nil
}

// findPython 查找可用的 Python 解释器。
// 按优先级依次检查：用户配置 → ComfyUI 便携版 → 虚拟环境 → py 启动器 → PATH 中的 python。
func findPython(comfyUIPath string, cfgPythonPath string) string {
	// 1. 用户手动配置的 Python 路径（最高优先）
	if cfgPythonPath != "" {
		if _, err := os.Stat(cfgPythonPath); err == nil {
			return cfgPythonPath
		}
		slog.Warn("配置的 Python 路径不存在，尝试自动查找", "path", cfgPythonPath)
	}

	// 2. ComfyUI 便携版（standalone-env 为 ROCm PyTorch 环境，Krea2/ZIT 必需，优先级最高）
	if comfyUIPath != "" {
		candidates := []string{
			filepath.Join(comfyUIPath, "..", "standalone-env", "python.exe"), // standalone-env（ROCm PyTorch）
			filepath.Join(comfyUIPath, "..", "python", "python.exe"),         // 整合包 python/
			filepath.Join(comfyUIPath, "python_embeded", "python.exe"),       //nolint:misspell // python_embeded 系上游真实目录名
			filepath.Join(comfyUIPath, "venv", "Scripts", "python.exe"),
			filepath.Join(comfyUIPath, ".venv", "Scripts", "python.exe"),
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// 3. Windows py 启动器（始终在 PATH）
	if _, err := exec.LookPath("py"); err == nil {
		return "py"
	}

	// 4. 系统 PATH
	for _, name := range []string{"python", "python3"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

// StopComfyUI 停止 ComfyUI 服务（不管是谁启动的都能停）
func (a *mediaState) StopComfyUI() error {
	port := extractPort(a.cfg.ComfyUIURL)

	// 1. 先通过 gaea 内部引用杀进程
	a.comfyProcRefStop()

	// 2. 通过端口查找进程（不管是谁启动的），强制杀
	if pid := findProcessByPort(port); pid > 0 {
		proc, err := os.FindProcess(pid)
		if err == nil {
			_ = proc.Kill()
		}
	}

	slog.Info("ComfyUI 已停止")
	return nil
}

// recoverComfyUI 重启 ComfyUI 并等待就绪（用于孤儿实例 stderr 失效的自动恢复）。
// 最多等待约 90 秒；失败仅记录日志，由上层按原错误返回。ctx 为生成链请求级
// context（审计 P1 AP7-09）：等待期被取消即刻退出，不再替已取消的请求白等。
func (a *mediaState) recoverComfyUI(ctx context.Context) {
	if err := a.StopComfyUI(); err != nil {
		slog.Warn("自动恢复：停止 ComfyUI 失败", "error", err)
	}
	// 等待端口释放，避免立刻重启时端口仍被占用
	if !sleepCtx(ctx, 2*time.Second) {
		return
	}
	if err := a.StartComfyUI(); err != nil {
		slog.Warn("自动恢复：启动 ComfyUI 失败", "error", err)
		return
	}
	for i := 0; i < 30; i++ {
		if !sleepCtx(ctx, 3*time.Second) {
			return
		}
		if a.isComfyUIRunning() {
			slog.Info("ComfyUI 自动恢复完成")
			return
		}
	}
	slog.Warn("ComfyUI 自动恢复超时")
}

// sleepCtx 可取消的 sleep：false = ctx 已取消。
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if ctx == nil {
		time.Sleep(d)
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// ensureComfyUIRunning ComfyUI 未运行时拉起并等待就绪（v4.387）。有界等待
// 120s（冷启动 Python+节点注册可能 30~90s）；未配置安装路径/启动失败/等待
// 超时一律返回 false，调用方保留原错误口径失败——不吞错不换错。就绪判定
// 与 WarmComfyUI/isComfyUIRunning 同口径（/system_stats 200）。并发拉起竞
// 态由 StartComfyUI 的端口占用守卫兜底：第二个调用方拿到「端口已被占用」
// 后转入就绪等待（第一个实例正在起来）而非直接放弃。ctx 为生成链请求级
// context（审计 P1 AP7-09）：取消即刻放弃等待，不再替已取消的请求白等 120s。
func (a *mediaState) ensureComfyUIRunning(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	if a.isComfyUIRunning() {
		return true
	}
	if strings.TrimSpace(a.cfg.ComfyUIPath) == "" {
		slog.Warn("ComfyUI 未运行且未配置安装路径，放弃自动拉起")
		return false
	}
	if err := a.StartComfyUI(); err != nil {
		if !strings.Contains(err.Error(), "已被") && !strings.Contains(err.Error(), "已在运行") {
			slog.Warn("ComfyUI 自动拉起失败", "error", err)
			return false
		}
		// 端口已被占用（并发拉起/实例正在启动）或刚好已在运行：转入就绪等待。
	}
	for i := 0; i < 40; i++ {
		if !sleepCtx(ctx, 3*time.Second) {
			slog.Info("ComfyUI 自动拉起等待被取消")
			return false
		}
		if a.isComfyUIRunning() {
			return true
		}
	}
	slog.Warn("ComfyUI 自动拉起后等待就绪超时（120s）")
	return false
}

// findProcessByPort 查找监听指定端口的进程 PID（Windows netstat -ano）。
// T6-4.5：弃用 cmd/findstr 字符串拼接（可注入命令），改用参数数组 exec.Command
// 并解析输出；port 入参先做白名单校验（纯数字 1–65535），杜绝注入。
func findProcessByPort(port string) int {
	if !isValidPort(port) {
		return 0
	}
	cmd := exec.Command("netstat", "-ano")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	return parseNetstatPID(string(out), port)
}

// parseNetstatPID 解析 netstat -ano 输出，返回监听 port 的进程 PID（0 = 未找到）。
// 输出格式: TCP  0.0.0.0:8188  0.0.0.0:0  LISTENING  12345
// 仅匹配本地地址以 :port 结尾且状态为 LISTENING 的 TCP 行（避免 findstr 的子串误匹配）。
func parseNetstatPID(out string, port string) int {
	target := ":" + port
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if fields[0] != "TCP" {
			continue
		}
		if !strings.HasSuffix(fields[1], target) {
			continue
		}
		if fields[3] != "LISTENING" {
			continue
		}
		if pid, err := strconv.Atoi(fields[4]); err == nil && pid > 0 {
			return pid
		}
	}
	return 0
}

// isValidPort 校验端口号：纯数字且在 1–65535 范围内（T6-4.5 注入防护）。
func isValidPort(port string) bool {
	if port == "" || len(port) > 5 {
		return false
	}
	for _, r := range port {
		if r < '0' || r > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535
}

func (a *mediaState) GetSystemStats() map[string]interface{} {
	memTotal, memUsed := getMemoryStats()
	result := map[string]interface{}{
		"cpu":       getCPUUsage(),
		"memTotal":  memTotal,
		"memUsed":   memUsed,
		"gpuName":   "",
		"gpuUsage":  0,
		"vramUsed":  0.0,
		"vramTotal": 0.0,
	}

	// GPU 信息：ComfyUI 运行时从 API 获取，否则用 nvidia-smi/wmic
	if a.isComfyUIRunning() {
		client := netclient.NewSimpleClient(3 * time.Second)
		resp, err := client.Get(strings.TrimSuffix(a.cfg.ComfyUIURL, "/") + "/system_stats")
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var stats map[string]interface{}
			if json.Unmarshal(body, &stats) == nil {
				if devices, ok := stats["devices"].([]interface{}); ok && len(devices) > 0 {
					if dev, ok := devices[0].(map[string]interface{}); ok {
						result["gpuName"] = dev["name"]
						if v, ok := dev["vram_total"].(float64); ok {
							result["vramTotal"] = v / 1e9
						}
						if total, ok := dev["vram_total"].(float64); ok {
							if free, ok := dev["vram_free"].(float64); ok {
								result["vramUsed"] = (total - free) / 1e9
							}
						}
					}
				}
			}
		}
	} else {
		name, total, used := getGPUInfo()
		result["gpuName"] = name
		result["vramTotal"] = total
		result["vramUsed"] = used
	}

	return result
}

// ── Windows 系统资源采集（弃用 wmic——Win11 24H2 起已移除；
//    直接调 kernel32.dll 原生 API，不依赖 x/sys/windows 符号，全版本可用）──

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

type winFiletime struct{ LowDateTime, HighDateTime uint32 }

// winMemoryStatusEx 必须与 Windows MEMORYSTATUSEX 完全一致（64 字节）：
// 2×uint32 + 7×uint64。字段数不对会令 GlobalMemoryStatusEx 失败（返回 0）。
type winMemoryStatusEx struct {
	Length                       uint32
	MemoryLoad                   uint32
	TotalPhys, AvailPhys         uint64
	TotalPageFile, AvailPageFile uint64
	TotalVirtual, AvailVirtual   uint64
	AvailExtendedVirtual         uint64
}

// getCPUUsage 获取 Windows CPU 使用率（%）。
// GetSystemTimes 两次采样差值计算；首次返回 0（无历史基准），
// 短间隔（<1s）重复调用返回上一次结果而不是重置基准，避免多个轮询者叠加导致恒 0。
var (
	prevCPUTimes  [3]uint64 // idle, kernel, user（100ns 单位）
	prevCPUSample time.Time
	prevCPUInit   bool
	prevCPUUsage  int
)

func getCPUUsage() int {
	var idle, kernel, user winFiletime
	r1, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r1 == 0 {
		return prevCPUUsage // 保持上一次，避免偶尔失败闪烁为 0
	}
	now := time.Now()
	idleT := uint64(idle.HighDateTime)<<32 | uint64(idle.LowDateTime)
	kernelT := uint64(kernel.HighDateTime)<<32 | uint64(kernel.LowDateTime)
	userT := uint64(user.HighDateTime)<<32 | uint64(user.LowDateTime)
	if !prevCPUInit {
		prevCPUTimes = [3]uint64{idleT, kernelT, userT}
		prevCPUSample = now
		prevCPUInit = true
		return 0
	}
	if now.Sub(prevCPUSample) < time.Second {
		return prevCPUUsage
	}
	dIdle := idleT - prevCPUTimes[0]
	dKernel := kernelT - prevCPUTimes[1]
	dUser := userT - prevCPUTimes[2]
	prevCPUTimes = [3]uint64{idleT, kernelT, userT}
	prevCPUSample = now
	total := dKernel + dUser // kernel 含 idle
	if total == 0 {
		return 0
	}
	usage := int((total - dIdle) * 100 / total)
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	prevCPUUsage = usage
	return usage
}

// getMemoryStats 获取 Windows 总内存与已用内存 (GB)。GlobalMemoryStatusEx 原生 API。
func getMemoryStats() (totalGB, usedGB float64) {
	var ms winMemoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return 0, 0
	}
	totalGB = float64(ms.TotalPhys) / 1e9
	usedGB = float64(ms.TotalPhys-ms.AvailPhys) / 1e9
	if usedGB < 0 {
		usedGB = 0
	}
	return totalGB, usedGB
}

// GPU 信息缓存（15s TTL）：nvidia-smi / PowerShell 每次轮询都执行代价高，
// 且显存总量不会频繁变化；已用显存由 nvidia-smi（NVIDIA）或 ComfyUI 提供。
var (
	gpuCacheMu    sync.Mutex
	gpuCacheAt    time.Time
	gpuCacheName  string
	gpuCacheTotal float64
	gpuCacheUsed  float64
)

// getGPUInfo 获取 GPU 名称、总显存、已用显存 (GB)
func getGPUInfo() (name string, totalGB float64, usedGB float64) {
	gpuCacheMu.Lock()
	defer gpuCacheMu.Unlock()
	if !gpuCacheAt.IsZero() && time.Since(gpuCacheAt) < 15*time.Second {
		return gpuCacheName, gpuCacheTotal, gpuCacheUsed
	}

	// NVIDIA：nvidia-smi 直接给出名称 + 总/已用显存
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,memory.used", "--format=csv,noheader,nounits")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err == nil {
		line := strings.TrimSpace(string(out))
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			name = strings.TrimSpace(parts[0])
			totalMB, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			usedMB, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			gpuCacheName, gpuCacheTotal, gpuCacheUsed = name, totalMB/1024.0, usedMB/1024.0
			gpuCacheAt = time.Now()
			return name, totalMB / 1024.0, usedMB / 1024.0
		}
	}

	// AMD / 其他：注册表读真实 GPU 显存（HardwareInformation.qwMemorySize），
	// 跳过 Todesk / 虚拟显示器 / 基础显示适配器等虚拟设备。
	psScript := `
$vram=0; $best=''
Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}' -ErrorAction SilentlyContinue | ForEach-Object {
  $p = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
  $d = $p.'DriverDesc'
  if (-not $d) { return }
  if ($d -match 'Todesk|Virtual|Remote|Basic Display|RDP|Mirror') { return }
  $m = $p.'HardwareInformation.qwMemorySize'
  if (-not $m) { $m = $p.AdapterRAM }
  if ($m -and [uint64]$m -gt $vram) { $vram = [uint64]$m; $best = $d }
}
if ($best) { $best + '|' + $vram }
`
	cmd2 := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out2, err := cmd2.Output()
	if err == nil {
		line := strings.TrimSpace(string(out2))
		if i := strings.Index(line, "|"); i > 0 {
			gb, _ := strconv.ParseFloat(strings.TrimSpace(line[i+1:]), 64)
			if gb > 0 {
				name, totalGB = strings.TrimSpace(line[:i]), gb/1e9
				gpuCacheName, gpuCacheTotal, gpuCacheUsed = name, totalGB, 0
				gpuCacheAt = time.Now()
				return
			}
		}
	}

	// 兜底：CIM 名称（跳过虚拟设备），无显存信息
	cmd3 := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-CimInstance Win32_VideoController | Where-Object { $_.Name -notmatch 'Todesk|Virtual|Remote|Basic Display|RDP|Mirror' } | Select-Object -First 1).Name")
	cmd3.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out3, err := cmd3.Output(); err == nil {
		name = strings.TrimSpace(string(out3))
	}
	gpuCacheName, gpuCacheTotal, gpuCacheUsed = name, 0, 0
	gpuCacheAt = time.Now()
	return
}
