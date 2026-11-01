package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gaea/gaea/internal/config"
)

// 本地 TTS 服务（模型中心“启动”按钮 / gaea 启动保活 / 合成前兜底共用）。
// T6-9.5：CosyVoice 路径/端口可配置（config.CosyVoiceDir / CosyVoicePort），
// 未配置时回退历史硬编码默认值（C:\AI\cosyvoice / 8010），行为与旧版本完全一致。

// ttsStartMaxRetries 启动失败后的最大重试次数（指数退避 1s/2s/4s）。
var ttsStartMaxRetries = 3

// ttsStartBackoff 每次重试前的等待时长（与 ttsStartMaxRetries 等长）。
var ttsStartBackoff = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}

// isLocalTTSAssetEngine 需要按需拉起本地服务进程的 TTS 引擎（合成前惰性
// ensure，已就绪零开销）：cosyvoice（python server.py）与 breeze（breeze-server.exe）。
func isLocalTTSAssetEngine(engineID string) bool {
	return engineID == "cosyvoice" || engineID == "breeze"
}

// waitLocalTTSReady 合成就绪有界等待：ensure 异步拉起后轮询探活至就绪
// （≤12s）。此前合成管线「发了就算」——冷启动窗口内 provider 必然
// connection refused，整段朗读滑落 edge（实弹日志：dial tcp refused）。
// 朗读按钮同步等待冷启动属可接受 UX（对比：滑落 edge = 错误音色）。
func (c *core) waitLocalTTSReady(engineID string) {
	if c.ttsReady(engineID) {
		return
	}
	c.ensureLocalTTSService(engineID)
	for i := 0; i < 30 && !c.ttsReady(engineID); i++ {
		time.Sleep(400 * time.Millisecond)
	}
}

// ttsCmdFactory 构造 CosyVoice 启动命令；抽为变量便于测试注入（断言参数/模拟启动失败）。
var ttsCmdFactory = func(cfg *config.Config) *exec.Cmd {
	return ttsCosyVoiceCmdFor(cfg)
}

// breezeCmdFactory 构造 breeze-server 启动命令；抽为变量便于测试注入。
// 启动失败（未安装/无模型）返回错误，由 startTTSServiceWithRetry 统一退避重试。
var breezeCmdFactory = func(cfg *config.Config) (*exec.Cmd, error) {
	return breezeCmdFor(cfg)
}

// ttsCosyVoiceCmdFor 由配置推导 CosyVoice 启动命令：python 与 server.py 均位于
// 配置目录下（venv\Scripts\python.exe / server.py），默认目录 C:\AI\cosyvoice。
func ttsCosyVoiceCmdFor(cfg *config.Config) *exec.Cmd {
	dir := config.DefaultCosyVoiceDir
	if cfg != nil && cfg.CosyVoiceDir != "" {
		dir = cfg.CosyVoiceDir
	}
	cmd := exec.Command(filepath.Join(dir, "venv", "Scripts", "python.exe"), filepath.Join(dir, "server.py"))
	cmd.Dir = dir
	return cmd
}

// breezeCmdFor 由配置推导 breeze-server 启动命令（Breeze-TTS-2，breezeCmdFactory 的实体）。
// exe 取 <dir>\bin\<版本>\breeze-server.exe（按目录名倒序取最新，容版本升级）；
// 模型取 <dir>\model\*.gguf（优先 q8_0 且非 -dd 实验档——dd 档深度解码量化后
// 约 45 秒持续生成起音质渐进劣化，不作默认）。
func breezeCmdFor(cfg *config.Config) (*exec.Cmd, error) {
	dir := config.DefaultBreezeDir
	if cfg != nil && cfg.BreezeDir != "" {
		dir = cfg.BreezeDir
	}
	exeMatches, err := filepath.Glob(filepath.Join(dir, "bin", "*", "breeze-server.exe"))
	if err != nil || len(exeMatches) == 0 {
		return nil, fmt.Errorf("未找到 breeze-server.exe（查找 %s\\bin\\*\\breeze-server.exe）", dir)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(exeMatches)))
	exe := exeMatches[0]

	model, err := pickBreezeModel(dir)
	if err != nil {
		return nil, err
	}

	port := config.DefaultBreezePort
	if cfg != nil && cfg.BreezePort > 0 {
		port = cfg.BreezePort
	}
	cmd := exec.Command(exe, model, "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Dir = filepath.Dir(exe)
	return cmd, nil
}

// pickBreezeModel 选择 breeze 模型 GGUF：<dir>\model\*.gguf 中优先
// q8_0（官方推荐档）且排除 -dd 实验档；无 q8_0 时取任意非 -dd 档。
func pickBreezeModel(dir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "model", "*.gguf"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("未找到 Breeze-TTS-2 模型 GGUF（查找 %s\\model\\*.gguf）", dir)
	}
	var q8, plain []string
	for _, m := range matches {
		l := strings.ToLower(filepath.Base(m))
		switch {
		case strings.Contains(l, "q8_0") && !strings.Contains(l, "-dd"):
			q8 = append(q8, m)
		case !strings.Contains(l, "-dd"):
			plain = append(plain, m)
		}
	}
	switch {
	case len(q8) > 0:
		sort.Strings(q8)
		return q8[0], nil
	case len(plain) > 0:
		sort.Strings(plain)
		return plain[0], nil
	default:
		sort.Strings(matches)
		return matches[0], nil
	}
}

var (
	ttsEnsureMu sync.Mutex
	ttsEnsuring = map[string]bool{}
)

// ttsURL 由配置推导 CosyVoice 探活 URL（http://127.0.0.1:<port>/v1/models），
// 默认端口 8010，与历史硬编码一致（T6-9.5）。
func (c *core) ttsURL() string {
	port := config.DefaultCosyVoicePort
	if c.cfg != nil && c.cfg.CosyVoicePort > 0 {
		port = c.cfg.CosyVoicePort
	}
	return fmt.Sprintf("http://127.0.0.1:%d/v1/models", port)
}

// breezeURL breeze-server 探活 URL（/health，模型加载完成才监听——200 即热）。
func (c *core) breezeURL() string {
	port := config.DefaultBreezePort
	if c.cfg != nil && c.cfg.BreezePort > 0 {
		port = c.cfg.BreezePort
	}
	return fmt.Sprintf("http://127.0.0.1:%d/health", port)
}

func (c *core) ttsReady(engineID string) bool {
	probeURL := c.ttsURL()
	if engineID == "breeze" {
		probeURL = c.breezeURL()
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(probeURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// startTTSService 单次启动本地 TTS 服务（启动失败由 startTTSServiceWithRetry 负责重试）。
func (c *core) startTTSService(engineID string) error {
	var cmd *exec.Cmd
	switch engineID {
	case "cosyvoice":
		cmd = ttsCmdFactory(c.cfg)
	case "breeze":
		breezeCmd, err := breezeCmdFactory(c.cfg)
		if err != nil {
			return err
		}
		cmd = breezeCmd
	default:
		return fmt.Errorf("unsupported local tts engine: %s", engineID)
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags = 0x08000000 // CREATE_NO_WINDOW：不弹控制台窗口
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // 回收句柄；服务进程独立存活
	return nil
}

// startTTSServiceWithRetry 启动失败按指数退避（1s/2s/4s）重试，最多 ttsStartMaxRetries
// 次；全部失败返回最后一次错误（仍失败才由调用方 emit 错误）。
func (c *core) startTTSServiceWithRetry(engineID string) error {
	var lastErr error
	for attempt := 0; attempt <= ttsStartMaxRetries; attempt++ {
		if lastErr = c.startTTSService(engineID); lastErr == nil {
			return nil
		}
		if attempt < ttsStartMaxRetries {
			delay := ttsStartBackoff[0]
			if attempt < len(ttsStartBackoff) {
				delay = ttsStartBackoff[attempt]
			}
			slog.Warn("本地 TTS 服务启动失败，等待重试", "engine", engineID, "attempt", attempt+1, "retry_in", delay.String(), "error", lastErr)
			time.Sleep(delay)
		}
	}
	return lastErr
}

// ensureLocalTTSService 幂等确保本地 TTS 服务就绪。
// 已就绪直接返回；未就绪则异步拉起并轮询，调用方不会被阻塞。
func (c *core) ensureLocalTTSService(engineID string) map[string]interface{} {
	if !isLocalTTSAssetEngine(engineID) {
		return map[string]interface{}{"engine": engineID, "ready": false, "starting": false, "error": "unsupported engine"}
	}
	if c.ttsReady(engineID) {
		return map[string]interface{}{"engine": engineID, "ready": true, "starting": false}
	}

	ttsEnsureMu.Lock()
	if ttsEnsuring[engineID] {
		ttsEnsureMu.Unlock()
		return map[string]interface{}{"engine": engineID, "ready": false, "starting": true}
	}
	ttsEnsuring[engineID] = true
	ttsEnsureMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("本地 TTS 服务启动 panic", "engine", engineID, "recover", r)
				c.emit("tts-service-status", map[string]interface{}{"engine": engineID, "ready": false, "error": "启动异常"})
			}
			ttsEnsureMu.Lock()
			delete(ttsEnsuring, engineID)
			ttsEnsureMu.Unlock()
		}()
		if err := c.startTTSServiceWithRetry(engineID); err != nil {
			slog.Warn("本地 TTS 服务启动失败", "engine", engineID, "error", err)
			c.emit("tts-service-status", map[string]interface{}{"engine": engineID, "ready": false, "error": err.Error()})
			return
		}
		timeout := 90 * time.Second
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			if c.ttsReady(engineID) {
				c.emit("tts-service-status", map[string]interface{}{"engine": engineID, "ready": true})
				slog.Info("本地 TTS 服务已就绪", "engine", engineID)
				return
			}
			time.Sleep(3 * time.Second)
		}
		c.emit("tts-service-status", map[string]interface{}{"engine": engineID, "ready": false, "error": "timeout"})
	}()

	return map[string]interface{}{"engine": engineID, "ready": false, "starting": true}
}

// StartLocalTTSService Wails 绑定：模型中心“启动”按钮调用。
func (a *mediaState) StartLocalTTSService(engineID string) map[string]interface{} {
	return a.ensureLocalTTSService(engineID)
}
