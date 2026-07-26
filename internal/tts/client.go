package tts

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"
)

// Client 调用 voxcpm_tts CLI 进行语音合成
type Client struct {
	binaryPath string // voxcpm_tts 可执行文件路径
	modelPath  string // GGUF 模型路径
	backend    string
	threads    int
}

// NewClient 创建 TTS 客户端
func NewClient(binaryPath string, modelPath string, backend string) *Client {
	if backend == "" {
		backend = "cpu"
	}
	return &Client{
		binaryPath: binaryPath,
		modelPath:  modelPath,
		backend:    backend,
		threads:    8,
	}
}

const (
	TempDirName   = "wubigork-tts"
	OutputWAV     = "speech.wav"
	StreamDirName = "wubigork-tts-stream-*"
)

// Synthesize 合成语音，返回 WAV 字节。
func (c *Client) Synthesize(text string) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("朗读文本为空")
	}

	tmpDir := filepath.Join(os.TempDir(), TempDirName)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	outputPath := filepath.Join(tmpDir, OutputWAV)
	defer os.Remove(outputPath)

	args := []string{
		"--model-path", c.modelPath,
		"--text", text,
		"--output", outputPath,
		"--backend", c.backend,
		"--threads", fmt.Sprintf("%d", c.threads),
		"--inference-timesteps", "10",
		"--cfg-value", "2.0",
	}

	slog.Info("TTS 合成开始", "text_len", len([]rune(text)), "backend", c.backend)

	cmd := exec.Command(c.binaryPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("TTS 合成失败: %w\n%s", err, stderr.String())
	}

	audioBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("读取音频文件失败: %w", err)
	}

	slog.Info("TTS 合成完成", "bytes", len(audioBytes))
	return audioBytes, nil
}

// SplitSentences 将文本按中文标点拆分为句子（每句尽量短，利于流式播放）
func SplitSentences(text string) []string {
	// 先按主要断句标点拆分
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '\n'
	})

	var sentences []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// 超过 100 字的句子再按逗号/分号拆
		if utf8.RuneCountInString(p) > 100 {
			subParts := strings.FieldsFunc(p, func(r rune) bool {
				return r == '，' || r == '；' || r == '：'
			})
			for _, sp := range subParts {
				sp = strings.TrimSpace(sp)
				if sp != "" {
					sentences = append(sentences, sp)
				}
			}
		} else {
			sentences = append(sentences, p)
		}
	}
	return sentences
}

// SynthesizeToFile 合成语音并保存到文件
func (c *Client) SynthesizeToFile(text string, outputPath string) error {
	audio, err := c.Synthesize(text)
	if err != nil {
		return err
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	if err := os.WriteFile(outputPath, audio, 0644); err != nil {
		return fmt.Errorf("写入音频文件失败: %w", err)
	}

	slog.Info("音频已保存", "path", outputPath)
	return nil
}
