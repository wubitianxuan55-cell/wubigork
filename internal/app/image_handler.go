package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wubigork/wubigork/internal/ai"
)

// imageItem 单张生成图片结果（包级共享，供移动端任务处理器提取）
type imageItem struct {
	Image  string  `json:"image"`
	Seed   int     `json:"seed"`
	Time   float64 `json:"time"`
	Prompt string  `json:"prompt"`
	Model  string  `json:"model"`
	Size   string  `json:"size"`
}

// GenerateFreeImage 自由图片生成 — 供 AI 绘梦 Tab 使用
// GenerateFreeImage 自由图片生成 — 供 AI 绘梦 Tab 使用
// 参数: prompt, negative, size, style, model, seed (0=随机), n (1-4)
func (a *App) GenerateFreeImage(prompt string, negative string, size string, style string, model string, seed int, n int) (map[string]interface{}, error) {
	if a.client == nil {
		return map[string]interface{}{"error": "AI 客户端未初始化，请先登录"}, nil
	}

	fullPrompt := prompt
	if style != "" {
		fullPrompt = prompt + "。风格: " + style
	}
	if size == "" {
		size = "1024x1024"
	}
	if n < 1 || n > 4 {
		n = 1
	}


	images := make([]imageItem, 0, n)
	var lastErr string

	for i := 0; i < n; i++ {
		genSeed := seed
		if genSeed == 0 {
			genSeed = int(time.Now().UnixNano()%1000000) + i*777
		}

		imgModel := a.cfg.ImageModel
		if model != "" {
			imgModel = model
		}

		imgReq := &ai.ImageGenerationRequest{
			Model:    imgModel,
			Prompt:   fullPrompt,
			Negative: negative,
			N:        1,
			Size:     size,
			Seed:     genSeed,
		}

		start := time.Now()
		resp, err := a.client.GenerateImage(a.ctx, imgReq)
		elapsed := time.Since(start).Seconds()

		if err != nil {
			slog.Warn("图片生成失败", "attempt", i+1, "error", err)
			lastErr = err.Error()
			continue
		}
		if len(resp.Data) == 0 {
			lastErr = "API 返回空结果"
			continue
		}

		imageData := resp.Data[0].URL
		if imageData == "" {
			imageData = resp.Data[0].B64JSON
		}

		images = append(images, imageItem{
			Image:  imageData,
			Seed:   genSeed,
			Time:   math.Round(elapsed*10) / 10,
			Prompt: fullPrompt,
			Model:  imgModel,
			Size:   size,
		})

		if a.cfg.ImageSaveDir != "" && imageData != "" {
			a.saveImageToDisk(imageData, fullPrompt)
		} else if imageData != "" {
			// 未配置专用目录时，自动保存到小说 images/ 目录
			a.saveToNovelImages(imageData, fullPrompt)
		}
	}

	if len(images) == 0 {
		msg := "图片生成失败"
		if lastErr != "" {
			msg = msg + "：" + lastErr
		}
		return map[string]interface{}{"error": msg}, nil
	}

	return map[string]interface{}{
		"images": images,
	}, nil
}

// saveImageToDisk 将图片数据保存到 ImageSaveDir，返回保存路径
func (a *App) saveImageToDisk(imageData string, prompt string) string {
	dir := a.cfg.ImageSaveDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}

	ts := time.Now().Format("20060102-150405")
	safePrompt := strings.TrimSpace(prompt)
	if r := []rune(safePrompt); len(r) > 20 {
		safePrompt = string(r[:20])
	}
	safePrompt = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>|`, r) {
			return '_'
		}
		return r
	}, safePrompt)
	filename := fmt.Sprintf("%s_%s.png", ts, safePrompt)
	fullPath := filepath.Join(dir, filename)

	if strings.HasPrefix(imageData, "data:") {
		commaIdx := strings.Index(imageData, ",")
		if commaIdx < 0 {
			return ""
		}
		data, err := base64.StdEncoding.DecodeString(imageData[commaIdx+1:])
		if err != nil {
			return ""
		}
		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			return ""
		}
		return fullPath
	}

	return "" // 远程 URL 不下载
}

// saveToNovelImages 将图片保存到当前小说的 images/ 目录
func (a *App) saveToNovelImages(imageData string, prompt string) {
	pm := a.getPM()
	if pm == nil {
		return
	}
	dir := filepath.Join(pm.Dir, "images")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	ts := time.Now().Format("20060102-150405")
	safePrompt := strings.TrimSpace(prompt)
	if r := []rune(safePrompt); len(r) > 20 {
		safePrompt = string(r[:20])
	}
	safePrompt = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>|`, r) {
			return '_'
		}
		return r
	}, safePrompt)
	filename := fmt.Sprintf("%s_%s.png", ts, safePrompt)
	fullPath := filepath.Join(dir, filename)

	if strings.HasPrefix(imageData, "data:") {
		commaIdx := strings.Index(imageData, ",")
		if commaIdx < 0 {
			return
		}
		data, err := base64.StdEncoding.DecodeString(imageData[commaIdx+1:])
		if err != nil {
			return
		}
		os.WriteFile(fullPath, data, 0644)
	}
}

// GetImageBackend 获取当前图片后端类型（供前端显示）
func (a *App) GetImageBackend() string {
	if a.client != nil {
		return a.client.GetImageBackendType()
	}
	return "xai"
}

// GetImageBackendInfo 获取当前图片后端类型和模型（供前端显示）
func (a *App) GetImageBackendInfo() map[string]string {
	return map[string]string{
		"backend": a.GetImageBackend(),
		"model":   a.cfg.ImageModel,
	}
}

// SetImageBackend 切换图片生成后端（供设置页调用）
func (a *App) SetImageBackend(backend string, comfyUIURL string, imageModel string) error {
	if a.client == nil {
		return fmt.Errorf("AI 客户端未初始化")
	}
	switch backend {
	case "comfyui":
		a.cfg.ImageBackend = "comfyui"
		if comfyUIURL != "" {
			a.cfg.ComfyUIURL = comfyUIURL
		}
		if imageModel != "" {
			a.cfg.ImageModel = imageModel
		}
		a.client.SetImageBackend(ai.NewComfyUIBackend(a.cfg.ComfyUIURL))
	case "xai":
		a.cfg.ImageBackend = "xai"
		a.client.SetImageBackend(nil)
	default:
		return fmt.Errorf("不支持的后端: %s（支持 xai / comfyui）", backend)
	}
	return nil
}

// ── ComfyUI 进程管理 ──────────────────────────────────────────

// StartComfyUI 启动 ComfyUI 服务
func (a *App) StartComfyUI() error {
	if a.cfg.ComfyUIPath == "" {
		return fmt.Errorf("请先在设置中配置 ComfyUI 安装路径")
	}
	// 检查是否已运行
	if a.isComfyUIRunning() {
		return fmt.Errorf("ComfyUI 已在运行")
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
	a.comfyUICancel = cancel

	cmd := exec.CommandContext(ctx, pythonExe, "main.py",
		"--listen", "127.0.0.1",
		"--port", extractPort(a.cfg.ComfyUIURL),
		"--lowvram",
	)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
	cmd.Dir = a.cfg.ComfyUIPath
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// 捕获 stderr 用于诊断
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	
	if err := cmd.Start(); err != nil {
		cancel()
				a.comfyUICancel = nil
		errMsg := stderr.String()
		if len(errMsg) > 300 { errMsg = errMsg[:300] + "..." }
		if errMsg != "" {
			return fmt.Errorf("启动 ComfyUI 失败: %w\n%s", err, errMsg)
		}
		return fmt.Errorf("启动 ComfyUI 失败: %w（Python=%s, Dir=%s）", err, pythonExe, a.cfg.ComfyUIPath)
	}

	slog.Info("ComfyUI 已启动", "python", pythonExe, "dir", a.cfg.ComfyUIPath, "pid", cmd.Process.Pid)

	// 后台等待进程结束，记录退出原因
	go func() {
		if err := cmd.Wait(); err != nil {
			slog.Warn("ComfyUI 进程退出", "error", err)
		}
				a.comfyUICancel = nil
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

	// 2. ComfyUI 便携版
	if comfyUIPath != "" {
		candidates := []string{
			filepath.Join(comfyUIPath, "python_embeded", "python.exe"),
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

// StopComfyUI 停止 ComfyUI 服务
func (a *App) StopComfyUI() error {
	if a.comfyUICancel == nil {
		return fmt.Errorf("ComfyUI 未在运行")
	}
	a.comfyUICancel()
	a.comfyUICancel = nil
		slog.Info("ComfyUI 已停止")
	return nil
}

// GetComfyUIStatus 返回 ComfyUI 运行状态
func (a *App) GetComfyUIStatus() map[string]interface{} {
	running := a.isComfyUIRunning()
	return map[string]interface{}{
		"running": running,
		"url":     a.cfg.ComfyUIURL,
	}
}

// isComfyUIRunning 检查 ComfyUI 是否可连通
func (a *App) isComfyUIRunning() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(strings.TrimSuffix(a.cfg.ComfyUIURL, "/") + "/system_stats")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// extractPort 从 URL 提取端口号
func extractPort(url string) string {
	parts := strings.Split(url, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return "8188"
}

// ── 文件夹打开 ──────────────────────────────────────────────

// OpenImageSaveDir 在文件管理器中打开图片存放目录
func (a *App) OpenImageSaveDir() error {
	if a.cfg.ImageSaveDir == "" {
		return fmt.Errorf("未设置图片存放目录，请在设置中配置")
	}
	if err := os.MkdirAll(a.cfg.ImageSaveDir, 0755); err != nil {
		return fmt.Errorf("无法创建图片存放目录: %w", err)
	}
	return openDir(a.cfg.ImageSaveDir)
}

// OpenNovelImagesDir 在文件管理器中打开当前小说的图片目录
func (a *App) OpenNovelImagesDir() error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开小说")
	}
	imgDir := filepath.Join(pm.Dir, "images")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		return fmt.Errorf("无法创建小说图片目录: %w", err)
	}
	return openDir(imgDir)
}

// openDir 用系统文件管理器打开目录，目标不存在则返回明确错误
func openDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("目录不存在: %s", dir)
		}
		return fmt.Errorf("无法访问目录 %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("路径不是目录: %s", dir)
	}
	return exec.Command("explorer", dir).Start()
}

// GetSystemStats 获取系统状态（CPU + GPU）
func (a *App) GetSystemStats() map[string]interface{} {
	result := map[string]interface{}{
		"cpu":     getCPUUsage(),
		"gpuName": "",
		"gpuUsage": 0,
		"vramUsed": 0.0,
		"vramTotal": 0.0,
	}

	// 从 ComfyUI 获取 GPU 信息
	if a.isComfyUIRunning() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(strings.TrimSuffix(a.cfg.ComfyUIURL, "/") + "/system_stats")
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var stats map[string]interface{}
			if json.Unmarshal(body, &stats) == nil {
				if devices, ok := stats["devices"].([]interface{}); ok && len(devices) > 0 {
					if dev, ok := devices[0].(map[string]interface{}); ok {
						result["gpuName"] = dev["name"]
						result["vramTotal"] = float64(0)
						if v, ok := dev["vram_total"].(float64); ok {
							result["vramTotal"] = v / 1e9
						}
						result["vramUsed"] = float64(0)
						if total, ok := dev["vram_total"].(float64); ok {
							if free, ok := dev["vram_free"].(float64); ok {
								result["vramUsed"] = (total - free) / 1e9
							}
						}
					}
				}
			}
		}
	}

	return result
}

// getCPUUsage 获取 Windows CPU 使用率
func getCPUUsage() int {
	cmd := exec.Command("wmic", "cpu", "get", "loadpercentage")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return -1
	}
	val := strings.TrimSpace(lines[1])
	usage, err := strconv.Atoi(val)
	if err != nil {
		return -1
	}
	return usage
}
