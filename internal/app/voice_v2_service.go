// voice_v2_service.go — Voice V2（M4 产品接入刀）sidecar 依赖链监护。
//
// V2 实时语音链四依赖：breeze TTS(8012) / Qwen3-4B llama-server(8123) /
// Gaea 记忆网关(8792) / speech-to-speech sidecar(8791)。本文件只做「探测 + 拉起 + 状态上报」，
// 不改 V1 whisper 链任何路径（voice.backend=v2 才有消费方）。
// 路径口径：spike 装机位（C:\AI\gaea-voice-v2、C:\AI\uocr2-llama、
// scripts/voice_v2/start_spike.bat）；产品化分发拍板（M1 报告 §6）后收敛进配置。
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
	voiceV2ModelsRoot = `C:\AI\models`
	// 桌面副本的 exe 位于 Desktop，工作目录也不保证是仓库；保留当前装机根
	// 作为最后候选，确保双击启动仍能找到 sidecar 启动器。
	voiceV2DefaultRepoRoot = `C:\AI\wubigrok`
	voiceV2RepoRootEnv     = "GAEA_VOICE_V2_ROOT"
	voiceV2BaseRootEnv     = "GAEA_VOICE_V2_BASE"
)

// 相对布局（bundle/安装器目录形态）：base/{venv, models, bin} + scripts/voice_v2。
var (
	voiceV2LLMBinRel     = filepath.Join("bin", "llama-server.exe")
	voiceV2LLMModelRel   = filepath.Join("models", "Qwen3-4B-Q4_K_M.gguf")
	voiceV2VenvPythonRel = filepath.Join("venv", "Scripts", "python.exe")
)

// voiceV2ModelPath 按统一模型根解析模型文件；缺失回退历史 base/models 布局
// （老安装兼容）。modelsRoot 下的相对路径示例：qwen35-uncensored/xxx.gguf。
func voiceV2ModelPath(modelsRootRel string, legacyFallback string) string {
	if p := filepath.Join(voiceV2ModelsRoot, modelsRootRel); voiceV2FileExists(p) {
		return p
	}
	if legacyFallback != "" && voiceV2FileExists(legacyFallback) {
		return legacyFallback
	}
	return filepath.Join(voiceV2BaseRoot(), voiceV2LLMModelRel)
}

// voiceV2BaseRoot 语音 V2 依赖（venv/models/bin）的根目录：安装器/bundle 布局
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

// voiceV2VenvPythonPath 解析 sidecar venv 解释器：base 根 venv 优先，缺失回退
// 历史装机路径（现役机器不因 bundle 布局变化而被打断）。
func voiceV2VenvPythonPath() string {
	if p := filepath.Join(voiceV2BaseRoot(), voiceV2VenvPythonRel); voiceV2FileExists(p) {
		return p
	}
	return filepath.Join(voiceV2LegacyBaseRoot, voiceV2VenvPythonRel)
}

// voiceV2LLMModelPath 解析 4B gguf：base 根 models 优先，缺失回退历史路径。
func voiceV2LLMModelPath() string {
	if p := filepath.Join(voiceV2BaseRoot(), voiceV2LLMModelRel); voiceV2FileExists(p) {
		return p
	}
	return filepath.Join(voiceV2LegacyBaseRoot, voiceV2LLMModelRel)
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

func hasVoiceV2RepoFiles(root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	for _, rel := range []string{
		filepath.Join("scripts", "voice_v2", "gaea-spike.yaml"),
		filepath.Join("scripts", "voice_v2", "gaea_sidecar_launcher.py"),
	} {
		if info, err := os.Stat(filepath.Join(root, rel)); err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

// findRepoRootFromStarts 从工作目录和可执行文件目录向上查找仓库脚本。
// Windows 卷根目录的 filepath.Dir 会返回自身；必须用 parent==dir 终止，
// 不能拿 C:\ 与 VolumeName 返回的 C: 比较（旧实现会在根目录永久 Stat）。
func findRepoRootFromStarts(starts []string, valid func(string) bool) string {
	seen := make(map[string]bool)
	for _, start := range starts {
		dir := filepath.Clean(strings.TrimSpace(start))
		if dir == "." || dir == "" {
			continue
		}
		for {
			key := strings.ToLower(dir)
			if !seen[key] {
				seen[key] = true
				if valid(dir) {
					return dir
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
}

// repoRootFromExe 优先使用显式配置，再检查工作目录/可执行文件祖先，最后回退
// 到本机 spike 装机根。所有分支均保证在卷根目录终止。
func repoRootFromExe() string {
	var starts []string
	if root := strings.TrimSpace(os.Getenv(voiceV2RepoRootEnv)); root != "" {
		starts = append(starts, root)
	}
	if wd, err := os.Getwd(); err == nil {
		starts = append(starts, wd)
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	starts = append(starts, voiceV2DefaultRepoRoot)
	return findRepoRootFromStarts(starts, hasVoiceV2RepoFiles)
}

// VoiceV2Status 四依赖探测（幂等、只读）。
func (a *mediaState) VoiceV2Status() map[string]interface{} {
	// 同一快照内每个端口只探一次：旧实现 ready 再探三项，既放大冷探测延迟，
	// 也可能出现 sidecar=true/ready=false 的自相矛盾快照（进程刚好退出时）。
	sidecar := voiceV2Probe(voiceV2SidecarPort)
	llm := voiceV2Probe(voiceV2LLMPort)
	breeze := voiceV2Probe(voiceV2BreezePort)
	gateway := voiceV2Probe(voiceV2GatewayPort)
	return map[string]interface{}{
		"sidecar":  sidecar,
		"llm":      llm,
		"breeze":   breeze,
		"gateway":  gateway,
		"ready":    sidecar && llm && breeze && gateway,
		"starting": isVoiceV2Ensuring(),
		"stage":    currentVoiceV2Stage(),
		"error":    currentVoiceV2Error(),
	}
}

// VoiceV2Ensure 幂等拉起依赖链（异步，voice-v2-status 事件上报进度；
// 拉起顺序 breeze→llm→sidecar，前置不就绪不拉后置）。
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
		if !voiceV2Probe(voiceV2GatewayPort) {
			emit(false, "检查记忆网关", "Gaea Voice V2 记忆网关未就绪（检查 8792 端口）")
			return
		}
		// Go 原生实时管线灰度模式（刀2~5 开发期）：GAEA_VOICE_REALTIME_GO=1 时
		// 进程内 realtime 服务器直接顶 8791（协议回声/后续真管线），不再拉
		// breeze/4B/sidecar。刀6 默认切换并删除本分支。
		if os.Getenv("GAEA_VOICE_REALTIME_GO") != "" {
			if err := voiceV2RealtimeStart(); err != nil {
				emit(false, "正在准备语音识别引擎", fmt.Sprintf("Go realtime 启动失败: %v", err))
				return
			}
			emit(true, "", "")
			return
		}
		// ① breeze：复用既有按需拉起线（自带 tts-service-status 事件）
		emit(false, "正在准备 Breeze 声线服务", "")
		if !voiceV2Probe(voiceV2BreezePort) {
			a.ensureLocalTTSService("breeze")
		}
		if !voiceV2WaitPort(voiceV2BreezePort, 90*time.Second) {
			emit(false, "准备 Breeze 声线服务", "Breeze TTS 90s 未就绪")
			return
		}
		// ② 4B llama-server
		emit(false, "正在准备快速模型", "")
		if !voiceV2Probe(voiceV2LLMPort) {
			cmd, err := detachedCmd(voiceV2LLMBinPath(), []string{
				"--host", "127.0.0.1", "--port", fmt.Sprintf("%d", voiceV2LLMPort),
				"-m", voiceV2ModelPath(
					filepath.Join("qwen35-uncensored", "Qwen3-4B-Q4_K_M.gguf"),
					filepath.Join(voiceV2LegacyBaseRoot, voiceV2LLMModelRel),
				), "-ngl", "99", "--ctx-size", "8192",
				"--reasoning", "off", "-np", "1", "--cache-reuse", "256",
			}, "")
			if err == nil {
				err = voiceV2Spawn("llm", cmd)
			}
			if err != nil {
				emit(false, "准备快速模型", fmt.Sprintf("4B 模型拉起失败: %v", err))
				return
			}
		}
		if !voiceV2WaitPort(voiceV2LLMPort, 90*time.Second) {
			emit(false, "准备快速模型", "4B 模型 90s 未就绪")
			return
		}
		// ③ sidecar：直接执行 venv Python，绕开 cmd.exe 对批处理路径的解析，
		// 启动失败能带着具体脚本路径上屏。
		// ②b 自管 35B（P2 续拍板）：voice_chat_llm=qwen35 时先拉 35B 再启
		// sidecar——sidecar 启动自检经网关走 qwen35 直代腿，8124 必须先就绪。
		if a.cfg != nil && a.cfg.VoiceChatLLM == "qwen35" && !qwen35Running() {
			emit(false, "正在准备自管 35B", "")
			res := a.app.qwen35Ensure()
			if res["running"] != true {
				emit(false, "准备自管 35B", fmt.Sprintf("%v", res["error"]))
				return
			}
		}
		emit(false, "正在准备语音识别引擎", "")
		if !voiceV2Probe(voiceV2SidecarPort) {
			root := repoRootFromExe()
			if root == "" {
				emit(false, "准备语音识别引擎", fmt.Sprintf(
					"sidecar 脚本缺失——设置 %s 指向 gaea 仓库（当前尝试含 %s）",
					voiceV2RepoRootEnv, voiceV2DefaultRepoRoot,
				))
				return
			}
			launcher := filepath.Join(root, "scripts", "voice_v2", "gaea_sidecar_launcher.py")
			config := filepath.Join(root, "scripts", "voice_v2", "gaea-spike.yaml")
			if _, err := os.Stat(voiceV2VenvPythonPath()); err != nil {
				emit(false, "准备语音识别引擎", fmt.Sprintf(
					"sidecar venv 缺失（%s）——先按 M1 报告装机", voiceV2VenvPythonPath(),
				))
				return
			}
			cmd, err := detachedCmd(voiceV2VenvPythonPath(), []string{
				launcher, "serve", "-f", config,
			}, root)
			if err == nil {
				err = voiceV2Spawn("sidecar", cmd)
			}
			if err != nil {
				emit(false, "准备语音识别引擎", fmt.Sprintf("sidecar 拉起失败: %v", err))
				return
			}
		}
		if !voiceV2WaitPort(voiceV2SidecarPort, 180*time.Second) {
			emit(false, "准备语音识别引擎", "sidecar 180s 未就绪")
			return
		}
		emit(true, "", "")
	}()

	return map[string]interface{}{"ready": false, "starting": true}
}

// VoiceV2Stop 按端口击杀 sidecar 与 4B（breeze 是共享引擎不动）。
// Go 原生 realtime 模式下 8791 由本进程持有——绝不能走端口杀（等于自杀），
// 只 Stop 服务器对象。
func (a *mediaState) VoiceV2Stop() map[string]interface{} {
	voiceV2RealtimeMu.Lock()
	goOwns8791 := voiceV2RealtimeSrv != nil
	voiceV2RealtimeMu.Unlock()
	var k1, k2 bool
	if goOwns8791 {
		voiceV2RealtimeStop()
		k1 = true
	} else {
		k1 = voiceV2KillPort(voiceV2SidecarPort)
	}
	k2 = voiceV2KillPort(voiceV2LLMPort)
	return map[string]interface{}{"sidecar": k1, "llm": k2}
}

// Go 原生 realtime 服务器（GAEA_VOICE_REALTIME_GO 灰度模式，刀2~5 开发期）。
var (
	voiceV2RealtimeMu   sync.Mutex
	voiceV2RealtimeSrv  *realtime.Server
	voiceV2NativeEng    *native.Engine
	voiceV2NativeSilero *native.SileroVAD
)

// voice 原生模型布局（模型统一根 voice/ 子目录）。
const (
	voiceNativeSileroOnnx     = `C:\AI\models\voice\silero_vad_v5_16k.onnx`
	voiceNativeSmartTurnOnnx  = `C:\AI\models\voice\smart-turn-v3.2-cpu.onnx`
	voiceNativeParaformerOnnx = `C:\AI\models\voice\model.int8.onnx`
	voiceNativeParaformerTok  = `C:\AI\models\voice\tokens.txt`
)

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

func voiceV2RealtimeStart() error {
	voiceV2RealtimeMu.Lock()
	defer voiceV2RealtimeMu.Unlock()
	if voiceV2RealtimeSrv != nil {
		return nil
	}
	// 原生引擎装配（首次 ~1-2s：paraformer int8 加载+warmup）。
	eng, err := native.New(native.Config{
		ORTRuntimePath:       native.OrtDLLPath(),
		SmartTurnModelPath:   voiceNativeSmartTurnOnnx,
		ParaformerModelPath:  voiceNativeParaformerOnnx,
		ParaformerTokensPath: voiceNativeParaformerTok,
	})
	if err == nil {
		var silero *native.SileroVAD
		silero, err = eng.NewSileroVAD(voiceNativeSileroOnnx)
		if err == nil {
			voiceV2NativeEng = eng
			voiceV2NativeSilero = silero
		}
	}
	if err != nil {
		return fmt.Errorf("语音原生引擎装配失败: %w", err)
	}
	pipe := realtime.NewPipeline(realtime.PipelineConfig{
		VAD: realtime.DefaultVADConfig(),
	}, voiceV2NativeSilero, voiceEOTAdapter{eng}, voiceSTTAdapter{eng})
	srv := &realtime.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", voiceV2SidecarPort),
		Handler: pipe,
	}
	if err := srv.Start(); err != nil {
		return err
	}
	voiceV2RealtimeSrv = srv
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
