// voice_v2_service.go — Voice V2（M4 产品接入刀）sidecar 依赖链监护。
//
// V2 实时语音链三依赖：breeze TTS(8012) → Qwen3-4B llama-server(8123) →
// speech-to-speech sidecar(8791)。本文件只做「探测 + 拉起 + 状态上报」，
// 不改 V1 whisper 链任何路径（voice.backend=v2 才有消费方）。
// 路径口径：spike 装机位（C:\AI\gaea-voice-v2、C:\AI\uocr2-llama、
// scripts/voice_v2/start_spike.bat）；产品化分发拍板（M1 报告 §6）后收敛进配置。
package app

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
	voiceV2LLMBin      = `C:\AI\uocr2-llama\bin\llama-server.exe`
	voiceV2LLMModel    = `C:\AI\gaea-voice-v2\models\Qwen3-4B-Q4_K_M.gguf`
	voiceV2VenvPython  = `C:\AI\gaea-voice-v2\venv\Scripts\python.exe`
)

// 探测/命令 seam：测试注入点（internal/app 测试隔离铁律——绝不真拉起 GPU 服务）。
var (
	voiceV2Probe = func(port int) bool {
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
		out, err := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf("(Get-NetTCPConnection -LocalPort %d -State Listen | Select-Object -First 1).OwningProcess", port)).Output()
		if err != nil {
			return false
		}
		pid := strings.TrimSpace(string(out))
		if pid == "" {
			return false
		}
		return exec.Command("taskkill", "/PID", pid, "/F").Run() == nil
	}
)

var (
	voiceV2EnsureMu   sync.Mutex
	voiceV2Ensuring   bool
	voiceV2LogDirOnce sync.Once
)

func voiceV2LogPath(name string) string {
	dir := filepath.Join(os.TempDir(), "gaea-voice-v2")
	voiceV2LogDirOnce.Do(func() { _ = os.MkdirAll(dir, 0o755) })
	return filepath.Join(dir, name)
}

// detachedCmd Windows 分离进程（父退出不连坐 + 无窗口）。
func detachedCmd(executable string, args []string, dir string) (*exec.Cmd, error) {
	if _, err := os.Stat(executable); err != nil {
		return nil, fmt.Errorf("可执行不存在 %s: %w", executable, err)
	}
	cmd := exec.Command(executable, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x08000000} // DETACHED_PROCESS | CREATE_NO_WINDOW
	logf, err := os.OpenFile(voiceV2LogPath(filepath.Base(executable)+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	cmd.Stdout, cmd.Stderr = logf, logf
	return cmd, nil
}

// repoRootFromExe 从可执行位置向上找 scripts/voice_v2/gaea-spike.yaml（真壳
// 工作目录不可靠；dev 跑在仓库内时两三跳即中）。
func repoRootFromExe() string {
	if wd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(wd, "scripts", "voice_v2", "gaea-spike.yaml")); err == nil {
			return wd
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	for dir := filepath.Dir(exe); dir != filepath.VolumeName(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "scripts", "voice_v2", "gaea-spike.yaml")); err == nil {
			return dir
		}
	}
	return ""
}

// VoiceV2Status 三依赖探测（幂等、只读）。
func (a *mediaState) VoiceV2Status() map[string]interface{} {
	return map[string]interface{}{
		"sidecar": voiceV2Probe(voiceV2SidecarPort),
		"llm":     voiceV2Probe(voiceV2LLMPort),
		"breeze":  voiceV2Probe(voiceV2BreezePort),
		"ready":   voiceV2Probe(voiceV2SidecarPort) && voiceV2Probe(voiceV2LLMPort) && voiceV2Probe(voiceV2BreezePort),
	}
}

// VoiceV2Ensure 幂等拉起三依赖链（异步，voice-v2-status 事件上报进度；
// 拉起顺序 breeze→llm→sidecar，前置不就绪不拉后置）。
func (a *mediaState) VoiceV2Ensure() map[string]interface{} {
	st := a.VoiceV2Status()
	if ready, _ := st["ready"].(bool); ready {
		return map[string]interface{}{"ready": true, "starting": false}
	}

	voiceV2EnsureMu.Lock()
	if voiceV2Ensuring {
		voiceV2EnsureMu.Unlock()
		return map[string]interface{}{"ready": false, "starting": true}
	}
	voiceV2Ensuring = true
	voiceV2EnsureMu.Unlock()

	go func() {
		defer func() {
			voiceV2EnsureMu.Lock()
			voiceV2Ensuring = false
			voiceV2EnsureMu.Unlock()
		}()
		emit := func(ready bool, errMsg string) {
			ev := map[string]interface{}{"ready": ready}
			if errMsg != "" {
				ev["error"] = errMsg
			}
			a.emit("voice-v2-status", ev)
		}
		// ① breeze：复用既有按需拉起线（自带 tts-service-status 事件）
		if !voiceV2Probe(voiceV2BreezePort) {
			a.ensureLocalTTSService("breeze")
		}
		// ② 4B llama-server
		if !voiceV2Probe(voiceV2LLMPort) {
			cmd, err := detachedCmd(voiceV2LLMBin, []string{
				"--host", "127.0.0.1", "--port", fmt.Sprintf("%d", voiceV2LLMPort),
				"-m", voiceV2LLMModel, "-ngl", "99", "--ctx-size", "8192",
				"--reasoning", "off", "-np", "1", "--cache-reuse", "256",
			}, "")
			if err == nil {
				err = voiceV2Spawn("llm", cmd)
			}
			if err != nil {
				emit(false, fmt.Sprintf("4B 模型拉起失败: %v", err))
				return
			}
		}
		if !voiceV2WaitPort(voiceV2LLMPort, 90*time.Second) {
			emit(false, "4B 模型 90s 未就绪")
			return
		}
		// ③ sidecar（start_spike.bat：venv python -m speech_to_speech.cli serve）
		if !voiceV2Probe(voiceV2SidecarPort) {
			root := repoRootFromExe()
			bat := filepath.Join(root, "scripts", "voice_v2", "start_spike.bat")
			if _, err := os.Stat(bat); err != nil {
				emit(false, fmt.Sprintf("sidecar 脚本缺失（%s）——先按 M1 报告装机", bat))
				return
			}
			if _, err := os.Stat(voiceV2VenvPython); err != nil {
				emit(false, fmt.Sprintf("sidecar venv 缺失（%s）——先按 M1 报告装机", voiceV2VenvPython))
				return
			}
			cmd, err := detachedCmd("cmd", []string{"/C", bat}, root)
			if err == nil {
				err = voiceV2Spawn("sidecar", cmd)
			}
			if err != nil {
				emit(false, fmt.Sprintf("sidecar 拉起失败: %v", err))
				return
			}
		}
		if !voiceV2WaitPort(voiceV2SidecarPort, 90*time.Second) {
			emit(false, "sidecar 90s 未就绪")
			return
		}
		emit(true, "")
	}()

	return map[string]interface{}{"ready": false, "starting": true}
}

// VoiceV2Stop 按端口击杀 sidecar 与 4B（breeze 是共享引擎不动）。
func (a *mediaState) VoiceV2Stop() map[string]interface{} {
	k1 := voiceV2KillPort(voiceV2SidecarPort)
	k2 := voiceV2KillPort(voiceV2LLMPort)
	return map[string]interface{}{"sidecar": k1, "llm": k2}
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
