package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// 本地 TTS 服务定义（模型中心“启动”按钮 / gaea 启动保活 / 合成前兜底共用）
const (
	ttsCosyVoiceURL    = "http://127.0.0.1:8010/v1/models"
	ttsCosyVoiceDir    = `C:\AI\cosyvoice`
	ttsCosyVoicePython = `C:\AI\cosyvoice\venv\Scripts\python.exe`
	ttsCosyVoiceScript = `C:\AI\cosyvoice\server.py`
)

var (
	ttsEnsureMu sync.Mutex
	ttsEnsuring = map[string]bool{}
)

func ttsReady(engineID string) bool {
	url := ttsCosyVoiceURL
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func startTTSService(engineID string) error {
	var cmd *exec.Cmd
	switch engineID {
	case "cosyvoice":
		cmd = exec.Command(ttsCosyVoicePython, ttsCosyVoiceScript)
		cmd.Dir = ttsCosyVoiceDir
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

// ensureLocalTTSService 幂等确保本地 TTS 服务就绪。
// 已就绪直接返回；未就绪则异步拉起并轮询，调用方不会被阻塞。
func (c *core) ensureLocalTTSService(engineID string) map[string]interface{} {
	if engineID != "cosyvoice" {
		return map[string]interface{}{"engine": engineID, "ready": false, "starting": false, "error": "unsupported engine"}
	}
	if ttsReady(engineID) {
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
			ttsEnsureMu.Lock()
			delete(ttsEnsuring, engineID)
			ttsEnsureMu.Unlock()
		}()
		if err := startTTSService(engineID); err != nil {
			slog.Warn("本地 TTS 服务启动失败", "engine", engineID, "error", err)
			c.emit("tts-service-status", map[string]interface{}{"engine": engineID, "ready": false, "error": err.Error()})
			return
		}
		timeout := 90 * time.Second
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			if ttsReady(engineID) {
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
