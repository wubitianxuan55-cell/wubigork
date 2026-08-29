# 语音朗读（VoxCPM.cpp 集成）实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 wubigork 增加本地 TTS 语音朗读功能，通过 VoxCPM.cpp 的 voxcpm-server 在本地 GPU 上合成语音。

**Architecture:** Go 端通过 `os/exec` 管理 voxcpm-server 子进程生命周期，通过 HTTP 调用 `/v1/audio/speech` 获取音频。前端章节页增加朗读按钮，音频通过 HTML5 Audio 播放。

**Tech Stack:** Go 1.26 + React 18 + TypeScript + VoxCPM.cpp (C++/ggml/CUDA)

## 全局约束

- 遵循现有 Wails 绑定模式：Go 方法 → `window.go.app.App.MethodName()`
- 遵循现有 Event 模式：`runtime.EventsEmit` + 前端 `window.runtime.EventsOn`
- 配置通过 Config struct + 环境变量管理
- 遵循文件即真相原则，无缓存层
- 纯本地运行，不依赖外部 API

---

## 文件结构

| 文件 | 职责 | 操作 |
|------|------|------|
| `internal/tts/server.go` | voxcpm-server 子进程管理（启动/停止/健康检查） | 新建 |
| `internal/tts/client.go` | HTTP 客户端：调用 /v1/audio/speech，保存音频文件 | 新建 |
| `internal/config/config.go` | 新增 TTS 配置项 | 修改 |
| `internal/app/tts_handler.go` | Wails 绑定方法（暴露给前端） | 新建 |
| `internal/app/app.go` | 新增 ttsServer 字段 | 修改 |
| `frontend/src/components/TTSPlayer.tsx` | 音频播放器组件（播放/暂停/语速/进度条） | 新建 |
| `frontend/src/pages/ChapterPage.tsx` | 工具栏加朗读按钮、集成 TTSPlayer | 修改 |
| `frontend/src/pages/SettingsPage.tsx` | 设置页加 TTS 配置区块 | 修改 |
| `frontend/src/types/index.ts` | 新增 TTSConfig 类型 | 修改 |

---

### Task 1: Go TTS 子进程管理 (`internal/tts/server.go`)

**Files:**
- Create: `internal/tts/server.go`

**Interfaces:**
- Produces: `type Server struct`, `func NewServer(modelPath string, port int, backend string, voiceDir string) *Server`, `func (s *Server) Start() error`, `func (s *Server) Stop() error`, `func (s *Server) HealthCheck() bool`, `func (s *Server) IsRunning() bool`

- [ ] **Step 1: 创建 internal/tts/server.go**

```go
package tts

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Server 管理 voxcpm-server 子进程
type Server struct {
	modelPath string
	port      int
	backend   string
	voiceDir  string

	cmd     *exec.Cmd
	cancel  context.CancelFunc
	running bool
}

// NewServer 创建 TTS 服务管理器
func NewServer(modelPath string, port int, backend string) *Server {
	if port <= 0 {
		port = 8765
	}
	if backend == "" {
		backend = "cuda"
	}
	voiceDir := filepath.Join(os.TempDir(), "wubigork-tts-voices")
	return &Server{
		modelPath: modelPath,
		port:      port,
		backend:   backend,
		voiceDir:  voiceDir,
	}
}

// Start 启动 voxcpm-server 子进程
func (s *Server) Start() error {
	if s.running {
		return fmt.Errorf("TTS 服务已在运行")
	}

	// 确保 voice 目录存在
	if err := os.MkdirAll(s.voiceDir, 0755); err != nil {
		return fmt.Errorf("创建 voice 目录失败: %w", err)
	}

	// 构建命令参数
	args := []string{
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", s.port),
		"--model-path", s.modelPath,
		"--model-name", "voxcpm-1.5",
		"--threads", "8",
		"--backend", s.backend,
		"--voice-dir", s.voiceDir,
		"--max-queue", "8",
		"--output-sample-rate", "24000",
		"--disable-auth",
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.cmd = exec.CommandContext(ctx, "voxcpm-server", args...)
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr

	slog.Info("启动 TTS 服务", "port", s.port, "model", s.modelPath, "backend", s.backend)
	if err := s.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("启动 voxcpm-server 失败（请确认已安装 VoxCPM.cpp 且 voxcpm-server 在 PATH 中）: %w", err)
	}

	// 等待服务就绪
	s.running = true
	go func() {
		err := s.cmd.Wait()
		s.running = false
		if err != nil {
			slog.Error("TTS 服务异常退出", "error", err)
		}
	}()

	// 健康检查轮询（最多等 30 秒）
	for i := 0; i < 60; i++ {
		if s.HealthCheck() {
			slog.Info("TTS 服务已就绪", "port", s.port)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	s.Stop()
	return fmt.Errorf("TTS 服务启动超时（30 秒未就绪）")
}

// Stop 停止 voxcpm-server
func (s *Server) Stop() error {
	if !s.running {
		return nil
	}
	slog.Info("正在停止 TTS 服务...")
	if s.cancel != nil {
		s.cancel()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	s.running = false
	return nil
}

// HealthCheck 检查服务是否可访问
func (s *Server) HealthCheck() bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", s.port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// IsRunning 返回服务是否在运行
func (s *Server) IsRunning() bool {
	return s.running && s.HealthCheck()
}

// Port 返回服务端口
func (s *Server) Port() int {
	return s.port
}
```

- [ ] **Step 2: 编译验证**

Run: `cd D:\AI\wubigork && go build ./internal/tts/`
Expected: 编译成功，无错误

- [ ] **Step 3: Commit**

```bash
git add internal/tts/server.go
git commit -m "feat(tts): add voxcpm-server subprocess manager"
```

---

### Task 2: Go TTS HTTP 客户端 (`internal/tts/client.go`)

**Files:**
- Create: `internal/tts/client.go`

**Interfaces:**
- Consumes: `Server.Port()`, `Server.IsRunning()`
- Produces: `type Client struct`, `func NewClient(server *Server) *Client`, `func (c *Client) Synthesize(text string, speed float64) ([]byte, error)`, `func (c *Client) SynthesizeToFile(text string, speed float64, outputPath string) error`

- [ ] **Step 1: 创建 internal/tts/client.go**

```go
package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client 调用 voxcpm-server 的 HTTP 客户端
type Client struct {
	server *Server
	http   *http.Client
}

// NewClient 创建 TTS 客户端
func NewClient(server *Server) *Client {
	return &Client{
		server: server,
		http:   &http.Client{Timeout: 120 * time.Second},
	}
}

// Synthesize 合成语音，返回 WAV 字节
// 使用默认语音 "default"（无参考音频的通用中文语音）
func (c *Client) Synthesize(text string, speed float64) ([]byte, error) {
	if !c.server.IsRunning() {
		return nil, fmt.Errorf("TTS 服务未运行")
	}

	if speed < 0.25 {
		speed = 0.25
	}
	if speed > 4.0 {
		speed = 4.0
	}

	body := map[string]interface{}{
		"model":           "voxcpm-1.5",
		"input":           text,
		"voice":           "default",
		"response_format": "wav",
		"speed":           speed,
		"stream_format":   "audio",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/v1/audio/speech", c.server.Port())
	slog.Info("TTS 合成请求", "text_len", len([]rune(text)), "speed", speed)

	resp, err := c.http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("TTS 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS 服务返回错误 %d: %s", resp.StatusCode, string(errBody))
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取音频数据失败: %w", err)
	}

	slog.Info("TTS 合成完成", "bytes", len(audioBytes))
	return audioBytes, nil
}

// SynthesizeToFile 合成语音并保存到文件
func (c *Client) SynthesizeToFile(text string, speed float64, outputPath string) error {
	audio, err := c.Synthesize(text, speed)
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
```

- [ ] **Step 2: 编译验证**

Run: `cd D:\AI\wubigork && go build ./internal/tts/`
Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/tts/client.go
git commit -m "feat(tts): add voxcpm-server HTTP client"
```

---

### Task 3: 配置新增 TTS 项 (`internal/config/config.go`)

**Files:**
- Modify: `internal/config/config.go`

**Interfaces:**
- Produces: `Config.TTSServerPath string`, `Config.TTSModelPath string`, `Config.TTSPort int`, `Config.TTSBackend string`, `Config.TTSSpeed float64`

- [ ] **Step 1: 修改 Config struct 和 Load 函数**

在 `Config` struct 的 `ResourceDir string` 之后新增字段：

```go
	// TTS 语音朗读配置
	TTSServerPath string // voxcpm-server 可执行文件路径（默认依赖 PATH）
	TTSModelPath  string // GGUF 模型文件路径
	TTSPort       int    // TTS 服务端口（默认 8765）
	TTSBackend    string // 推理后端: cpu / cuda / vulkan（默认 cuda）
	TTSSpeed      float64 // 默认朗读语速（0.25-4.0，默认 1.0）
```

在 `Load()` 函数的默认值区域（`cfg := &Config{...}` 末尾）增加：

```go
		TTSPort:    8765,
		TTSBackend: "cuda",
		TTSSpeed:   1.0,
```

在环境变量覆盖区域增加：

```go
	if v := os.Getenv("WUBI_TTS_SERVER_PATH"); v != "" {
		cfg.TTSServerPath = v
	}
	if v := os.Getenv("WUBI_TTS_MODEL_PATH"); v != "" {
		cfg.TTSModelPath = v
	}
	if v := os.Getenv("WUBI_TTS_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.TTSPort = n
		}
	}
	if v := os.Getenv("WUBI_TTS_BACKEND"); v != "" {
		cfg.TTSBackend = v
	}
	if v := os.Getenv("WUBI_TTS_SPEED"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.TTSSpeed = f
		}
	}
```

- [ ] **Step 2: 编译验证**

Run: `cd D:\AI\wubigork && go build ./...`
Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(tts): add TTS config fields"
```

---

### Task 4: Wails 绑定方法 (`internal/app/tts_handler.go`)

**Files:**
- Create: `internal/app/tts_handler.go`
- Modify: `internal/app/app.go`（新增 ttsServer 字段）

**Interfaces:**
- Consumes: `App.cfg`, `App.ctx`, `tts.Server`, `tts.Client`
- Produces: `App.StartTTSServer()`, `App.StopTTSServer()`, `App.TTSSpeak(text, speed)`, `App.GetTTSStatus()`, `App.GetTTSConfig()`, `App.SaveTTSConfig()`

- [ ] **Step 1: 修改 app.go 新增字段**

在 `App` struct 的 `skillLoader` 之后新增：

```go
	// TTS 语音朗读
	ttsServer *tts.Server
	ttsClient *tts.Client
```

在 import 中增加 `"github.com/wubigork/wubigork/internal/tts"`

- [ ] **Step 2: 创建 internal/app/tts_handler.go**

```go
package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wubigork/wubigork/internal/tts"
)

// ── TTS 语音朗读 ─────────────────────────────────────────────

// StartTTSServer 启动 voxcpm-server
func (a *App) StartTTSServer(modelPath string, port int, backend string) error {
	// 先停止已有实例
	if a.ttsServer != nil {
		a.ttsServer.Stop()
	}

	if modelPath == "" {
		modelPath = a.cfg.TTSModelPath
	}
	if port <= 0 {
		port = a.cfg.TTSPort
	}
	if backend == "" {
		backend = a.cfg.TTSBackend
	}

	if modelPath == "" {
		return fmt.Errorf("请先在设置中配置 GGUF 模型文件路径")
	}

	a.ttsServer = tts.NewServer(modelPath, port, backend)
	if err := a.ttsServer.Start(); err != nil {
		a.ttsServer = nil
		return err
	}

	a.ttsClient = tts.NewClient(a.ttsServer)
	slog.Info("TTS 服务已启动", "port", port, "backend", backend)
	return nil
}

// StopTTSServer 停止 voxcpm-server
func (a *App) StopTTSServer() error {
	if a.ttsServer == nil {
		return nil
	}
	err := a.ttsServer.Stop()
	a.ttsServer = nil
	a.ttsClient = nil
	return err
}

// TTSSpeak 合成语音并返回音频文件路径（前端用 HTML5 Audio 播放）
// 返回路径供前端 <audio> 标签直接使用
func (a *App) TTSSpeak(text string, speed float64) (string, error) {
	if a.ttsClient == nil {
		return "", fmt.Errorf("TTS 服务未启动，请先启动语音引擎")
	}

	if text == "" {
		return "", fmt.Errorf("朗读文本为空")
	}

	if speed <= 0 {
		speed = a.cfg.TTSSpeed
	}

	// 输出到临时目录
	outputPath := filepath.Join(os.TempDir(), "wubigork-tts", "speech.wav")
	if err := a.ttsClient.SynthesizeToFile(text, speed, outputPath); err != nil {
		return "", err
	}

	// Wails 前端无法直接访问本地文件路径（安全限制），
	// 所以用 base64 内联方式传递，或者通过 Assets 服务。
	// 这里返回文件路径，前端通过自定义协议读取
	return outputPath, nil
}

// TTSSpeakBase64 合成语音并返回 Base64 编码的 WAV 数据
// 前端可直接用于 <audio src="data:audio/wav;base64,..." />
func (a *App) TTSSpeakBase64(text string, speed float64) (map[string]interface{}, error) {
	if a.ttsClient == nil {
		return nil, fmt.Errorf("TTS 服务未启动")
	}

	if text == "" {
		return nil, fmt.Errorf("朗读文本为空")
	}

	if speed <= 0 {
		speed = a.cfg.TTSSpeed
	}

	audioBytes, err := a.ttsClient.Synthesize(text, speed)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"base64":     audioBytes,
		"mimeType":   "audio/wav",
		"sampleRate": 24000,
	}, nil
}

// GetTTSStatus 获取 TTS 服务状态
func (a *App) GetTTSStatus() map[string]interface{} {
	return map[string]interface{}{
		"running": a.ttsServer != nil && a.ttsServer.IsRunning(),
		"port":    func() int { if a.ttsServer != nil { return a.ttsServer.Port() }; return 0 }(),
	}
}

// GetTTSConfig 获取 TTS 配置
func (a *App) GetTTSConfig() map[string]interface{} {
	return map[string]interface{}{
		"modelPath":  a.cfg.TTSModelPath,
		"serverPath": a.cfg.TTSServerPath,
		"port":       a.cfg.TTSPort,
		"backend":    a.cfg.TTSBackend,
		"speed":      a.cfg.TTSSpeed,
	}
}

// SaveTTSConfig 保存 TTS 配置（写入 ~/.wubigork_config.json 的扩展字段）
func (a *App) SaveTTSConfig(modelPath string, serverPath string, port int, backend string, speed float64) error {
	if modelPath != "" {
		a.cfg.TTSModelPath = modelPath
	}
	if serverPath != "" {
		a.cfg.TTSServerPath = serverPath
	}
	if port > 0 {
		a.cfg.TTSPort = port
	}
	if backend != "" {
		a.cfg.TTSBackend = backend
	}
	if speed > 0 {
		a.cfg.TTSSpeed = speed
	}

	// 重启服务以应用新配置
	if a.ttsServer != nil && a.ttsServer.IsRunning() {
		var err error
		_ = a.ttsServer.Stop()
		a.ttsServer = nil
		a.ttsClient = nil
		// 尝试用新配置重启
		a.ttsServer = tts.NewServer(a.cfg.TTSModelPath, a.cfg.TTSPort, a.cfg.TTSBackend)
		if startErr := a.ttsServer.Start(); startErr != nil {
			a.ttsServer = nil
			slog.Warn("TTS 重启失败", "error", startErr)
			err = startErr
		} else {
			a.ttsClient = tts.NewClient(a.ttsServer)
		}
		if err != nil {
			return fmt.Errorf("配置已保存但 TTS 重启失败: %w", err)
		}
	}

	// 触发前端刷新
	runtime.EventsEmit(a.ctx, "tts-config-changed", nil)
	return nil
}
```

- [ ] **Step 3: 编译验证**

Run: `cd D:\AI\wubigork && go build ./...`
Expected: 编译成功（如果有 lint 错误，修复）

- [ ] **Step 4: Commit**

```bash
git add internal/app/tts_handler.go internal/app/app.go
git commit -m "feat(tts): add Wails bindings for TTS"
```

---

### Task 5: 前端 TTS 类型定义

**Files:**
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: 新增 TTSConfig 类型**

在文件末尾新增：

```typescript
// ── TTS 语音朗读 ────────────────────────────────────────
export interface TTSConfig {
  modelPath: string
  serverPath: string
  port: number
  backend: string
  speed: number
}

export interface TTSStatus {
  running: boolean
  port: number
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/types/index.ts
git commit -m "feat(tts): add TTS types"
```

---

### Task 6: 前端 TTSPlayer 组件

**Files:**
- Create: `frontend/src/components/TTSPlayer.tsx`

- [ ] **Step 1: 创建 TTSPlayer 组件**

```tsx
import React, { useState, useRef, useCallback } from 'react'
import { Button, Space, Slider, Typography, Tooltip, message } from 'antd'
import {
  SoundOutlined, PauseOutlined, PlayCircleOutlined,
  StopOutlined, LoadingOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'

interface TTSPlayerProps {
  getText: () => string
  onStatusChange?: (playing: boolean) => void
}

const TTSPlayer: React.FC<TTSPlayerProps> = ({ getText, onStatusChange }) => {
  const [playing, setPlaying] = useState(false)
  const [loading, setLoading] = useState(false)
  const [speed, setSpeed] = useState(1.0)
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const audioUrlRef = useRef<string | null>(null)

  const cleanupAudio = useCallback(() => {
    if (audioUrlRef.current) {
      URL.revokeObjectURL(audioUrlRef.current)
      audioUrlRef.current = null
    }
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current.src = ''
      audioRef.current = null
    }
  }, [])

  const handlePlay = async () => {
    const text = getText()
    if (!text || text.trim().length === 0) {
      message.warning('当前没有可朗读的内容')
      return
    }

    setLoading(true)
    try {
      // @ts-ignore
      const result = await window.go.app.App.TTSSpeakBase64(text, speed)
      if (!result?.base64) {
        throw new Error('合成失败')
      }

      cleanupAudio()

      // 将 base64 字节数组转换为 Blob
      const bytes = new Uint8Array(result.base64)
      const blob = new Blob([bytes], { type: result.mimeType || 'audio/wav' })
      const url = URL.createObjectURL(blob)
      audioUrlRef.current = url

      const audio = new Audio(url)
      audioRef.current = audio

      audio.onended = () => {
        setPlaying(false)
        onStatusChange?.(false)
      }

      audio.onerror = () => {
        message.error('音频播放失败')
        setPlaying(false)
        onStatusChange?.(false)
        cleanupAudio()
      }

      await audio.play()
      setPlaying(true)
      onStatusChange?.(true)
    } catch (err: any) {
      message.error(err?.message || '语音合成失败')
    } finally {
      setLoading(false)
    }
  }

  const handlePause = () => {
    audioRef.current?.pause()
    setPlaying(false)
    onStatusChange?.(false)
  }

  const handleStop = () => {
    audioRef.current?.pause()
    if (audioRef.current) {
      audioRef.current.currentTime = 0
    }
    setPlaying(false)
    onStatusChange?.(false)
  }

  const handleResume = () => {
    audioRef.current?.play()
    setPlaying(true)
    onStatusChange?.(true)
  }

  return (
    <Space size={4} align="center">
      {loading ? (
        <Button type="text" size="small" disabled
          icon={<LoadingOutlined spin style={{ color: '#60a5fa' }} />}
          style={{ color: C('color-text-secondary'), fontSize: 12 }}
        >
          合成中...
        </Button>
      ) : playing ? (
        <>
          <Tooltip title="暂停">
            <Button type="text" size="small"
              icon={<PauseOutlined style={{ color: '#f59e0b' }} />}
              onClick={handlePause}
            />
          </Tooltip>
          <Tooltip title="停止">
            <Button type="text" size="small"
              icon={<StopOutlined style={{ color: '#f87171' }} />}
              onClick={handleStop}
            />
          </Tooltip>
        </>
      ) : (
        <Tooltip title="朗读当前场景">
          <Button type="text" size="small"
            icon={<SoundOutlined style={{ color: '#4ade80' }} />}
            onClick={handlePlay}
            style={{ fontSize: 12 }}
          >
            朗读
          </Button>
        </Tooltip>
      )}

      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10, marginLeft: 4 }}>
        语速
      </Typography.Text>
      <Slider
        min={0.5}
        max={2.0}
        step={0.1}
        value={speed}
        onChange={setSpeed}
        style={{ width: 60, margin: 0 }}
        tooltip={{ formatter: (v) => `${v}x` }}
      />
    </Space>
  )
}

export default TTSPlayer
```

- [ ] **Step 2: 编译验证**

Run: `cd D:\AI\wubigork\frontend && npx tsc --noEmit`
Expected: 无类型错误

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/TTSPlayer.tsx
git commit -m "feat(tts): add TTSPlayer component"
```

---

### Task 7: ChapterPage 集成朗读按钮

**Files:**
- Modify: `frontend/src/pages/ChapterPage.tsx`

- [ ] **Step 1: 在工具栏增加 TTSPlayer**

在 import 区新增：

```tsx
import TTSPlayer from '../components/TTSPlayer'
```

在工具栏区域（`{activeTab && (<> ...` 内部，streamSpeed 显示之后，generating 判断之前）增加 TTS 按钮。找到工具栏中合适位置（靠近语速/生成区域），在场景管理按钮旁边插入：

```tsx
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, justifyContent: 'flex-end' }}>
                <TTSPlayer
                  getText={() => activeTab?.scenes?.join('\n\n') || ''}
                />
              </div>
```

实际在 ChapterPage 的工具栏左上区域，`<Space size={4}>` 中 chapterNum/状态标签行的右侧。精确插入位置：在显示 streamSpeed 的 `</Typography.Text>` 之后、`</Space>` 之前插入 TTSPlayer。

- [ ] **Step 2: 编译验证**

Run: `cd D:\AI\wubigork\frontend && npx tsc --noEmit`
Expected: 无类型错误

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/ChapterPage.tsx
git commit -m "feat(tts): integrate TTS player into ChapterPage"
```

---

### Task 8: SettingsPage 增加 TTS 配置区块

**Files:**
- Modify: `frontend/src/pages/SettingsPage.tsx`

- [ ] **Step 1: 增加 TTS 配置 Card**

在 `SETTINGS_PAGE_END`（最后一个 `</Descriptions>` 闭合的 Card 之后、导出按钮之前）插入新的 Card：

```tsx
      <Card style={{ background: C('color-bg-container'), borderColor: C('color-border'), marginBottom: 24 }}>
        <Typography.Title level={5} style={{ color: C('color-text'), marginTop: 0 }}>
          🔊 语音朗读 (TTS)
        </Typography.Title>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginBottom: 12 }}>
          使用 VoxCPM 在本地 GPU 上合成语音。需先安装 <a href="https://github.com/bluryar/VoxCPM.cpp" target="_blank" rel="noopener noreferrer" style={{ color: '#60a5fa' }}>VoxCPM.cpp</a> 并下载 GGUF 模型。
        </Typography.Text>
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <div>
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
              voxcpm-server 路径
            </Typography.Text>
            <Input
              placeholder="留空使用 PATH 中的 voxcpm-server"
              value={ttsConfig.serverPath}
              onChange={(e) => setTTSConfig((prev: TTSConfigState) => ({ ...prev, serverPath: e.target.value }))}
              style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
            />
          </div>
          <div>
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
              GGUF 模型文件路径（必填）
            </Typography.Text>
            <Input
              placeholder="例如: D:\models\voxcpm1.5-q8_0-audiovae-f16.gguf"
              value={ttsConfig.modelPath}
              onChange={(e) => setTTSConfig((prev: TTSConfigState) => ({ ...prev, modelPath: e.target.value }))}
              style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
            />
          </div>
          <Space>
            <div>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>端口</Typography.Text>
              <Input
                type="number"
                value={ttsConfig.port}
                onChange={(e) => setTTSConfig((prev: TTSConfigState) => ({ ...prev, port: parseInt(e.target.value) || 8765 }))}
                style={{ width: 100, background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
              />
            </div>
            <div>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>后端</Typography.Text>
              <Select
                value={ttsConfig.backend}
                onChange={(val: string) => setTTSConfig((prev: TTSConfigState) => ({ ...prev, backend: val }))}
                style={{ width: 100 }}
                options={[
                  { label: 'CUDA', value: 'cuda' },
                  { label: 'CPU', value: 'cpu' },
                  { label: 'Vulkan', value: 'vulkan' },
                ]}
              />
            </div>
          </Space>
          <Button type="primary" onClick={handleSaveTTS}>
            💾 保存 TTS 配置
          </Button>
          <Button onClick={handleStartTTS} disabled={ttsStatus.running}>
            ▶️ 启动 TTS 服务
          </Button>
          <Button onClick={handleStopTTS} disabled={!ttsStatus.running} danger>
            ⏹️ 停止 TTS 服务
          </Button>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
            状态：{ttsStatus.running ? '🟢 运行中 (端口 ' + ttsStatus.port + ')' : '⚫ 未启动'}
          </Typography.Text>
        </Space>
      </Card>
```

需要在 import 中添加 `Input, Select`，新增状态变量和回调函数：

```tsx
interface TTSConfigState {
  modelPath: string
  serverPath: string
  port: number
  backend: string
  speed: number
}
```

```tsx
  const [ttsConfig, setTTSConfig] = useState<TTSConfigState>({
    modelPath: '', serverPath: '', port: 8765, backend: 'cuda', speed: 1.0,
  })
  const [ttsStatus, setTTSStatus] = useState<{ running: boolean; port: number }>({ running: false, port: 0 })
```

在 `loadConfig` 之后增加加载 TTS 配置和状态的逻辑，`handleSaveTTS`、`handleStartTTS`、`handleStopTTS` 回调函数。

- [ ] **Step 2: 新增状态变量和回调函数**

在 `loadConfig` 函数之后增加：

```tsx
  const [ttsConfig, setTTSConfig] = useState<TTSConfigState>({
    modelPath: '', serverPath: '', port: 8765, backend: 'cuda', speed: 1.0,
  })
  const [ttsStatus, setTTSStatus] = useState<TTSStatus>({ running: false, port: 0 })

  const loadTTSConfig = async () => {
    try {
      // @ts-ignore
      const cfg = await window.go.app.App.GetTTSConfig()
      if (cfg) setTTSConfig(cfg as TTSConfigState)
      // @ts-ignore
      const status = await window.go.app.App.GetTTSStatus()
      if (status) setTTSStatus(status as TTSStatus)
    } catch (_) {}
  }

  // 在 loadConfig 的 useEffect 中也调用 loadTTSConfig
  useEffect(() => {
    loadConfig()
    loadTTSConfig()
  }, [])

  const handleSaveTTS = async () => {
    try {
      // @ts-ignore
      await window.go.app.App.SaveTTSConfig(
        ttsConfig.modelPath, ttsConfig.serverPath, ttsConfig.port, ttsConfig.backend, ttsConfig.speed
      )
      message.success('TTS 配置已保存')
    } catch (err: any) { message.error(err?.message || '保存失败') }
  }

  const handleStartTTS = async () => {
    try {
      // @ts-ignore
      await window.go.app.App.StartTTSServer(ttsConfig.modelPath, ttsConfig.port, ttsConfig.backend)
      setTTSStatus({ running: true, port: ttsConfig.port })
      message.success('TTS 服务已启动')
    } catch (err: any) { message.error(err?.message || '启动失败') }
  }

  const handleStopTTS = async () => {
    try {
      // @ts-ignore
      await window.go.app.App.StopTTSServer()
      setTTSStatus({ running: false, port: 0 })
      message.success('TTS 服务已停止')
    } catch (err: any) { message.error(err?.message || '停止失败') }
  }
```

在 import 区新增 `Input, Select, message`（如尚未导入）和 `TTSConfigState, TTSStatus` 类型导入。

在文件顶部函数外新增：

```tsx
interface TTSConfigState {
  modelPath: string
  serverPath: string
  port: number
  backend: string
  speed: number
}
```

从 `'../types'` 导入 `TTSStatus`。

- [ ] **Step 3: 编译验证**

Run: `cd D:\AI\wubigork\frontend && npx tsc --noEmit`
Expected: 无类型错误

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/SettingsPage.tsx
git commit -m "feat(tts): add TTS config to SettingsPage"
```

---

### Task 9: 端到端验证

- [ ] **Step 1: 完整编译 Go 后端**

Run: `cd D:\AI\wubigork && go build -o wubigork.exe .`
Expected: 编译成功

- [ ] **Step 2: 编译前端**

Run: `cd D:\AI\wubigork\frontend && npm run build`
Expected: 构建成功

- [ ] **Step 3: 验证 Wails 构建**

Run: `cd D:\AI\wubigork && wails build`
Expected: 打包成功

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat(tts): complete VoxCPM TTS integration"
```
