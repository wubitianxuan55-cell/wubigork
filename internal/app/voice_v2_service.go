// voice_v2_service.go — Voice V2 依赖链监护（刀6：Go 原生管线=唯一路径）。
//
// V2 实时语音链依赖：breeze TTS(8012)；8791=进程内 Go realtime（WS 服务器，
// 会话池=1）；LLM 腿=聊天板块功能绑定进程内直连（8792 网关已退役，见
// voice_v2_go_legs.go）。本文件只做「探测 + 拉起 + 状态上报」，
// 不改 V1 whisper 链任何路径（voice.backend=v2 才有消费方）。
package app

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/gaea/gaea/internal/gaea/proc"
	"github.com/gaea/gaea/internal/voice/native"
	"github.com/gaea/gaea/internal/voice/realtime"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	voiceV2SidecarPort = 8791
	voiceV2LLMPort     = 8123
	voiceV2BreezePort  = 8012
	// 历史装机路径（P3 打包工程化前）：降级为 base 根缺失时的回退值。
	voiceV2LegacyLLMBin   = `C:\AI\uocr2-llama\bin\llama-server.exe`
	voiceV2LegacyBaseRoot = `C:\AI\gaea-voice-v2`
	// 模型统一根（P3 2026-11-05 用户拍板）：所有本地模型只存这一处，多处引用
	// 不复制（消除 26.8GB×N 重复）。llama/strata/herdsman 档位均指此根。
	voiceV2ModelsRoot  = `C:\AI\models`
	voiceV2BaseRootEnv = "GAEA_VOICE_V2_BASE"
)

// 相对布局（bundle/安装器目录形态）：base/{bin} —— venv（sidecar）与 4B gguf
// 已随刀6 sidecar/4B 退役出局，llama-server bin 布局保留（qwen35 自管引擎复用）。
var (
	voiceV2LLMBinRel = filepath.Join("bin", "llama-server.exe")
)

// voiceV2BaseRoot 语音 V2 依赖（bin）的根目录：安装器/bundle 布局
// 经 GAEA_VOICE_V2_BASE 指向；缺省=历史装机根（兼容现役机器，不迁移）。
func voiceV2BaseRoot() string {
	if v := strings.TrimSpace(os.Getenv(voiceV2BaseRootEnv)); v != "" {
		return v
	}
	return voiceV2LegacyBaseRoot
}

func voiceV2FileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// voiceV2LLMBinPath 解析 llama-server：base 根 bin 优先（bundle 布局），缺失
// 回退历史 uocr2-llama 装机；两者皆缺返回 bundle 预期路径（报错信息可读）。
func voiceV2LLMBinPath() string {
	bundled := filepath.Join(voiceV2BaseRoot(), voiceV2LLMBinRel)
	if voiceV2FileExists(bundled) {
		return bundled
	}
	if voiceV2FileExists(voiceV2LegacyLLMBin) {
		return voiceV2LegacyLLMBin
	}
	return bundled
}

// 探测/命令 seam：测试注入点（internal/app 测试隔离铁律——绝不真拉起 GPU 服务）。
var (
	voiceV2Probe = func(port int) bool {
		if port == voiceV2SidecarPort {
			// speech-to-speech 的 HTTP "/" 可能保持连接不返回；只要 TCP 已监听，
			// WebSocket 端点就具备连接条件。HTTP 探测会把已启动误判成未就绪。
			return voiceV2TCPProbe(fmt.Sprintf("127.0.0.1:%d", port), 800*time.Millisecond)
		}
		client := &http.Client{Timeout: 1500 * time.Millisecond}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return true // 任意 HTTP 应答（含 404）= 进程活着
	}
	voiceV2Spawn = func(name string, cmd *exec.Cmd) error {
		return cmd.Start()
	}
	voiceV2KillPort = func(port int) bool {
		probe := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf("(Get-NetTCPConnection -LocalPort %d -State Listen | Select-Object -First 1).OwningProcess", port))
		proc.HideWindow(probe) // Windows: 防止弹出 cmd 黑框（启动收旧/关闭收割路径）
		out, err := probe.Output()
		if err != nil {
			return false
		}
		pid := strings.TrimSpace(string(out))
		if pid == "" {
			return false
		}
		kill := exec.Command("taskkill", "/PID", pid, "/F")
		proc.HideWindow(kill) // Windows: 防止弹出 cmd 黑框
		return kill.Run() == nil
	}
)

func voiceV2TCPProbe(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

var (
	voiceV2EnsureMu   sync.Mutex
	voiceV2Ensuring   bool
	voiceV2LogDirOnce sync.Once
	voiceV2StateMu    sync.Mutex
	voiceV2LastError  string
	voiceV2Stage      string
)

func voiceV2LogPath(name string) string {
	dir := filepath.Join(os.TempDir(), "gaea-voice-v2")
	voiceV2LogDirOnce.Do(func() { _ = os.MkdirAll(dir, 0o755) })
	return filepath.Join(dir, name)
}

// detachedCmd Windows 分离进程（父退出不连坐 + 无窗口）。
func detachedCmd(executable string, args []string, dir string) (*exec.Cmd, error) {
	resolved, err := exec.LookPath(executable)
	if err != nil {
		return nil, fmt.Errorf("可执行不存在 %s: %w", executable, err)
	}
	cmd := exec.Command(resolved, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x08000000} // DETACHED_PROCESS | CREATE_NO_WINDOW
	logf, err := os.OpenFile(voiceV2LogPath(filepath.Base(resolved)+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	cmd.Stdout, cmd.Stderr = logf, logf
	return cmd, nil
}

func setVoiceV2Error(message string) {
	voiceV2StateMu.Lock()
	voiceV2LastError = strings.TrimSpace(message)
	voiceV2StateMu.Unlock()
}

func currentVoiceV2Error() string {
	voiceV2StateMu.Lock()
	defer voiceV2StateMu.Unlock()
	return voiceV2LastError
}

func setVoiceV2Stage(stage string) {
	voiceV2StateMu.Lock()
	voiceV2Stage = strings.TrimSpace(stage)
	voiceV2StateMu.Unlock()
}

func currentVoiceV2Stage() string {
	voiceV2StateMu.Lock()
	defer voiceV2StateMu.Unlock()
	return voiceV2Stage
}

func isVoiceV2Ensuring() bool {
	voiceV2EnsureMu.Lock()
	defer voiceV2EnsureMu.Unlock()
	return voiceV2Ensuring
}

// VoiceV2Status 依赖快照（幂等、只读；刀6 起 Go 原生管线=唯一路径，网关退役）。
// 8791=进程内 Go realtime（voiceV2GoModeActive 为准，端口探活看不到进程内态）、
// breeze 仍是 TTS 腿必需；LLM 腿=聊天板块功能绑定进程内直连（无本地端口可探）。
func (a *mediaState) VoiceV2Status() map[string]interface{} {
	rt := voiceV2GoModeActive()
	breeze := voiceV2Probe(voiceV2BreezePort)
	return map[string]interface{}{
		"realtime": rt,
		"breeze":   breeze,
		"ready":    rt && breeze,
		"starting": isVoiceV2Ensuring(),
		"stage":    currentVoiceV2Stage(),
		"error":    currentVoiceV2Error(),
	}
}

// voiceV2GoModeActive Go 原生管线灰度是否已接管 8791。
func voiceV2GoModeActive() bool {
	voiceV2RealtimeMu.Lock()
	defer voiceV2RealtimeMu.Unlock()
	return voiceV2RealtimeSrv != nil
}

// VoiceV2Ensure 幂等拉起依赖链（异步，voice-v2-status 事件上报进度；
// 拉起顺序 breeze→Go 原生管线；LLM 腿无本地依赖，前置不就绪不拉后置）。
func (a *mediaState) VoiceV2Ensure() map[string]interface{} {
	st := a.VoiceV2Status()
	if ready, _ := st["ready"].(bool); ready {
		setVoiceV2Error("")
		setVoiceV2Stage("")
		return map[string]interface{}{"ready": true, "starting": false}
	}

	voiceV2EnsureMu.Lock()
	if voiceV2Ensuring {
		voiceV2EnsureMu.Unlock()
		return map[string]interface{}{"ready": false, "starting": true}
	}
	voiceV2Ensuring = true
	voiceV2EnsureMu.Unlock()
	setVoiceV2Error("")
	setVoiceV2Stage("正在检测本地语音服务")

	go func() {
		defer func() {
			voiceV2EnsureMu.Lock()
			voiceV2Ensuring = false
			voiceV2EnsureMu.Unlock()
		}()
		emit := func(ready bool, stage, errMsg string) {
			if stage != "" {
				setVoiceV2Stage(stage)
			}
			setVoiceV2Error(errMsg)
			ev := map[string]interface{}{"ready": ready}
			if stage != "" {
				ev["stage"] = stage
			}
			if errMsg != "" {
				ev["error"] = errMsg
				slog.Warn("Voice V2 依赖拉起失败", "error", errMsg)
			} else if stage != "" {
				slog.Info("Voice V2 依赖拉起进度", "stage", stage)
			} else {
				slog.Info("Voice V2 依赖链就绪")
			}
			a.emit("voice-v2-status", ev)
		}
		// Go 原生实时管线（刀6 起唯一路径）：进程内 realtime 服务器顶 8791，
		// breeze 仍按需拉起（TTS 腿）。sidecar/4B/8792 网关拉起链已退役——
		// scripts/voice_v2 手动回滚物保留一版期。
		// TTS 腿依赖 breeze：未起先拉（8012；breeze 共享引擎，常驻时直过）
		if !voiceV2Probe(voiceV2BreezePort) {
			emit(false, "正在准备 Breeze 声线服务", "")
			a.ensureLocalTTSService("breeze")
			if !voiceV2WaitPort(voiceV2BreezePort, 90*time.Second) {
				emit(false, "正在准备 Breeze 声线服务", "Breeze TTS 90s 未就绪")
				return
			}
		}
		emit(false, "正在装配语音引擎（Go 原生）", "")
		start := time.Now()
		if err := voiceV2RealtimeStartFn(a); err != nil {
			emit(false, "正在装配语音引擎（Go 原生）", fmt.Sprintf("Go realtime 启动失败: %v", err))
			return
		}
		slog.Info("Voice V2 Go 管线就绪", "装配耗时", time.Since(start).String())
		emit(true, "", "")
	}()

	return map[string]interface{}{"ready": false, "starting": true}
}

// VoiceV2Stop 结束语音 V2：停进程内 Go realtime 服务器（8791 由本进程持有，
// 绝不能走端口杀=自杀；breeze 是共享引擎不动；sidecar/4B 拉起链已退役）。
func (a *mediaState) VoiceV2Stop() map[string]interface{} {
	voiceV2RealtimeStop()
	return map[string]interface{}{"realtime": true}
}

// Go 原生 realtime 服务器（刀6 起唯一路径）。
var (
	voiceV2RealtimeMu   sync.Mutex
	voiceV2RealtimeSrv  *realtime.Server
	voiceV2NativeEng    *native.Engine
	voiceV2NativeSilero *native.SileroVAD
	// voiceV2RealtimeStartFn 测试缝（probe/spawn 全桩纪律同族）。
	voiceV2RealtimeStartFn = func(a *mediaState) error { return voiceV2RealtimeStart(a) }
)

// voice 原生模型布局：bundle 根（GAEA_VOICE_V2_BASE）的 models/voice/ 优先
// （可重定位分发），缺省回落模型统一根 C:\AI\modelsoice\（现役装机口径）。
func voiceNativeModelPath(name string) string {
	if p := filepath.Join(voiceV2BaseRoot(), "models", "voice", name); voiceV2FileExists(p) {
		return p
	}
	return filepath.Join(voiceV2ModelsRoot, "voice", name)
}

func voiceNativeModelPaths() (silero, smartTurn, paraformer, tokens string) {
	return voiceNativeModelPath("silero_vad_v5_16k.onnx"),
		voiceNativeModelPath("smart-turn-v3.2-cpu.onnx"),
		voiceNativeModelPath("model.int8.onnx"),
		voiceNativeModelPath("tokens.txt")
}

// voiceEOTAdapter smart-turn 适配。
type voiceEOTAdapter struct{ e *native.Engine }

func (a voiceEOTAdapter) Predict(pcm []float32) (float32, time.Duration, error) {
	return a.e.EndOfTurnTiming(pcm, native.SampleRate)
}

// voiceSTTAdapter paraformer 适配。
type voiceSTTAdapter struct{ e *native.Engine }

func (a voiceSTTAdapter) Transcribe(pcm []float32, sampleRate int) (string, error) {
	return a.e.Transcribe(pcm, sampleRate)
}

func voiceV2RealtimeStart(a *mediaState) error {
	voiceV2RealtimeMu.Lock()
	defer voiceV2RealtimeMu.Unlock()
	if voiceV2RealtimeSrv != nil {
		return nil
	}
	if a == nil {
		return fmt.Errorf("mediaState 未就绪（进程内 LLM 腿需要 app 上下文）")
	}
	// 原生引擎装配（首次 ~1-2s：paraformer int8 加载+warmup）。
	engStart := time.Now()
	sileroOnnx, smartTurnOnnx, paraformerOnnx, paraformerTok := voiceNativeModelPaths()
	eng, err := native.New(native.Config{
		ORTRuntimePath:       native.OrtDLLPath(),
		SmartTurnModelPath:   smartTurnOnnx,
		ParaformerModelPath:  paraformerOnnx,
		ParaformerTokensPath: paraformerTok,
	})
	slog.Info("Voice V2 Go 原生引擎装配完成", "耗时", time.Since(engStart).String(), "error", err)
	if err == nil {
		var silero *native.SileroVAD
		silero, err = eng.NewSileroVAD(sileroOnnx)
		if err == nil {
			voiceV2NativeEng = eng
			voiceV2NativeSilero = silero
		}
	}
	if err != nil {
		return fmt.Errorf("语音原生引擎装配失败: %w", err)
	}
	var pipeRef *realtime.Pipeline
	legs := voiceResponseLegs{
		llm: voiceNativeLLMLeg{a: a},
		tts: voiceTTSLeg{base: "http://127.0.0.1:8012/v1", model: "breeze-tts-2",
			voice: func() string {
				if pipeRef != nil {
					return pipeRef.Voice()
				}
				return "char_gaea"
			}},
	}
	pipe := realtime.NewPipeline(realtime.PipelineConfig{
		VAD:            realtime.DefaultVADConfig(),
		Legs:           legs,
		CreateResponse: true,
	}, voiceV2NativeSilero, voiceEOTAdapter{eng}, voiceSTTAdapter{eng})
	pipeRef = pipe
	srv := &realtime.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", voiceV2SidecarPort),
		Handler: pipe,
	}
	if err := srv.Start(); err != nil {
		return err
	}
	voiceV2RealtimeSrv = srv
	slog.Info("Voice V2 Go realtime 服务器已监听", "addr", srv.Addr)
	return nil
}

func voiceV2RealtimeStop() {
	voiceV2RealtimeMu.Lock()
	srv := voiceV2RealtimeSrv
	voiceV2RealtimeSrv = nil
	voiceV2RealtimeMu.Unlock()
	if srv != nil {
		srv.Stop()
	}
	voiceV2RealtimeMu.Lock()
	eng, silero := voiceV2NativeEng, voiceV2NativeSilero
	voiceV2NativeEng, voiceV2NativeSilero = nil, nil
	voiceV2RealtimeMu.Unlock()
	if silero != nil {
		silero.Destroy()
	}
	if eng != nil {
		eng.Close()
	}
}

func voiceV2WaitPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if voiceV2Probe(port) {
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return false
}
