package tts

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Windows TTS 使用系统自带 SAPI 语音合成（微软慧慧中文语音，零延迟）
type WinTTS struct {
	voice    string
	tempDir  string
}

func NewWinTTS() *WinTTS {
	return &WinTTS{
		voice:   "Microsoft Huihui Desktop",
		tempDir: filepath.Join(os.TempDir(), "wubigork-tts"),
	}
}

func (w *WinTTS) Synthesize(text string) ([]byte, error) {
	if err := os.MkdirAll(w.tempDir, 0755); err != nil {
		return nil, err
	}

	outputPath := filepath.Join(w.tempDir, "speech_win.wav")
	defer os.Remove(outputPath)

	// 将中文文本写入临时文件，避免 PowerShell 编码问题
	textPath := filepath.Join(w.tempDir, "tts_text.txt")
	if err := os.WriteFile(textPath, []byte(text), 0644); err != nil {
		return nil, err
	}
	defer os.Remove(textPath)

	script := fmt.Sprintf(
		`Add-Type -AssemblyName System.Speech
$s = New-Object System.Speech.Synthesis.SpeechSynthesizer
try { $s.SelectVoice('%s') } catch {}
$text = Get-Content '%s' -Encoding UTF8
$s.SetOutputToWaveFile('%s')
$s.Speak($text)
$s.Dispose()`,
		w.voice, escapePath(textPath), escapePath(outputPath))

	slog.Info("WinTTS 合成", "text_len", len([]rune(text)))

	cmd := exec.Command("powershell", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Windows TTS 失败: %w\n%s", err, stderr.String())
	}

	audio, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}

	slog.Info("WinTTS 完成", "bytes", len(audio))
	return audio, nil
}

func escapePath(p string) string {
	return p
}
