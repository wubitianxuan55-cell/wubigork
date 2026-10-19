package app

// gpu_perf_windows.go — Windows GPU 性能计数器采集（PowerShell Get-Counter）。
//
// GPU Engine(*)\Utilization Percentage 与 GPU Process Memory(*)\Local Usage 是
// Windows 图形栈的厂商中立计数器（AMD/NVIDIA/Intel 均注册，免管理员）。
// 此前 GPU 侧读数只有两条路：ComfyUI 在跑（/system_stats）或 nvidia-smi——
// 本机（AMD gfx1151）两者皆空，运行监测/模型中心资源条/首页矩阵的 GPU 负载
// 与显存占用常年「无遥测」。
//
// 为什么走 PowerShell 而不是直连 PDH：本机 pdh.dll（Win11 26200）的批量取值
// 族全线异常——PdhGetFormattedCounterValueW 导出名缺失（无后缀版存在）、
// PdhGetFormattedCounterArrayW 恒 PDH_INVALID_HANDLE 且传缓冲段错误、
// PdhExpandWildCardPathW 二次调用仍 MORE_DATA（2026-10-19 逐步实测，python
// 复现同症=系统库问题非绑定姿势）。Get-Counter 同源同数据且实测可用
// （真机：util_sum=1.4 / vram_gb=12.21）。房屋先例：getGPUInfo 的注册表
// 后备同为 PowerShell 子进程。
//
// 节拍：stale-while-revalidate——命中缓存（含过期）立即返回，后台单飞刷新
// （脚本 ~2-3s，TTL 15s）。GPU 负载是慢变量，监测卡 15s 步进完全够看；
// 绝不在 RuntimeMonitor 2s 轮询热路径上同步阻塞三秒。首次未取到时 ok=false
// （「无遥测」，不编造 0%）。

import (
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// gpuPerfCache 读数进程级缓存（后台单飞刷新）。
var gpuPerfCache struct {
	mu       sync.Mutex
	at       time.Time // 最近一次成功采集时间
	util     float64   // GPU 负载 %（全引擎实例求和，钳 0~100）
	vramUsed float64   // 专用显存占用 GB
	ok       bool      // 曾成功采集过至少一次
}

// gpuPerfRefreshing 后台刷新单飞闸（atomic CAS：0=空闲 1=进行中）。
var gpuPerfRefreshing int32

const gpuPerfTTL = 15 * time.Second

// gpuPerfScript 一次子进程取两计数器：全实例利用率求和 + 专用显存字节数求和。
const gpuPerfScript = `$u = Get-Counter '\GPU Engine(*)\Utilization Percentage' -ErrorAction SilentlyContinue
if ($u) { 'util_sum=' + [math]::Round(($u.CounterSamples | Measure-Object -Property CookedValue -Sum).Sum, 2) }
$m = Get-Counter '\GPU Process Memory(*)\Local Usage' -ErrorAction SilentlyContinue
if ($m) { 'vram_gb=' + [math]::Round(($m.CounterSamples | Measure-Object -Property CookedValue -Sum).Sum / 1GB, 2) }`

// parseGPUPerfOutput 解析脚本输出行（util_sum=… / vram_gb=…；缺行=该侧无读数）。
func parseGPUPerfOutput(out string) (util float64, vram float64, utilOK, vramOK bool) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if v, err := strconv.ParseFloat(strings.TrimPrefix(line, "util_sum="), 64); err == nil && strings.HasPrefix(line, "util_sum=") {
			util, utilOK = math.Min(math.Max(v, 0), 100), true
		}
		if v, err := strconv.ParseFloat(strings.TrimPrefix(line, "vram_gb="), 64); err == nil && strings.HasPrefix(line, "vram_gb=") {
			vram, vramOK = math.Max(v, 0), true
		}
	}
	return
}

// collectGPUPerfOnce 同步执行采集脚本（仅后台刷新路径调用）。
func collectGPUPerfOnce() {
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", gpuPerfScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return
	}
	util, vram, utilOK, vramOK := parseGPUPerfOutput(string(out))
	if !utilOK && !vramOK {
		return
	}
	gpuPerfCache.mu.Lock()
	defer gpuPerfCache.mu.Unlock()
	if utilOK {
		gpuPerfCache.util = util
	}
	if vramOK {
		gpuPerfCache.vramUsed = vram
	}
	gpuPerfCache.ok = true
	gpuPerfCache.at = time.Now()
}

// windowsGPUPerf 返回 (负载%, 专用显存已用GB, ok)。从未成功采集过 ok=false；
// 缓存过期时立即返回旧值并后台单飞刷新（脚本 2~3s，不阻塞调用方）。
func windowsGPUPerf() (float64, float64, bool) {
	gpuPerfCache.mu.Lock()
	stale := gpuPerfCache.ok && time.Since(gpuPerfCache.at) >= gpuPerfTTL
	util, vram, ok := gpuPerfCache.util, gpuPerfCache.vramUsed, gpuPerfCache.ok
	gpuPerfCache.mu.Unlock()
	if stale {
		if atomic.CompareAndSwapInt32(&gpuPerfRefreshing, 0, 1) {
			go func() {
				defer atomic.StoreInt32(&gpuPerfRefreshing, 0)
				collectGPUPerfOnce()
			}()
		}
	}
	return util, vram, ok
}
