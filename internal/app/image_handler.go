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

	"github.com/gaea/gaea/internal/ai"
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
func (a *App) GenerateFreeImage(prompt string, negative string, size string, style string, model string, seed int, n int, lora string) (map[string]interface{}, error) {
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
			Lora:     lora,
		}

		// 非 ComfyUI 后端不接受 size 参数（xAI 返回 400）
		if a.cfg.ImageBackend != "comfyui" {
			imgReq.Size = ""
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
func (a *App) SetImageBackend(backend string, comfyUIURL string, imageModel string, imageSaveDir string) error {
	if a.client == nil {
		return fmt.Errorf("AI 客户端未初始化")
	}
	// 设置图片保存目录
	if imageSaveDir != "" {
		a.cfg.ImageSaveDir = imageSaveDir
	}
	if a.cfg.ImageSaveDir == "" {
		a.cfg.ImageSaveDir = filepath.Join(os.Getenv("USERPROFILE"), "Pictures", "gaea")
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
		a.client.SetImageBackend(ai.NewComfyUIBackend(a.cfg.ComfyUIURL), "comfyui")
	case "xai":
		a.cfg.ImageBackend = "xai"
		a.cfg.ImageModel = "grok-imagine-image-quality" // 角色剧照默认高质量模型
		a.client.SetImageBackend(nil, "xai")
	case "herdsman":
		eng, ok := a.engineMgr.GetEngine("herdsman")
		if !ok || !eng.Enabled {
			return fmt.Errorf("Herdsman 引擎未启用，请先在模型中心启用")
		}
		a.cfg.ImageBackend = "herdsman"
		if imageModel != "" {
			a.cfg.ImageModel = imageModel
		}
		a.client.SetImageBackend(ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey), "herdsman")
	case "ollama":
		eng, ok := a.engineMgr.GetEngine("ollama")
		if !ok || !eng.Enabled {
			return fmt.Errorf("Ollama 引擎未启用，请先在模型中心启用")
		}
		a.cfg.ImageBackend = "ollama"
		if imageModel != "" {
			a.cfg.ImageModel = imageModel
		}
		a.client.SetImageBackend(ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey), "ollama")
	default:
		return fmt.Errorf("不支持的后端: %s（支持 xai / comfyui / herdsman / ollama）", backend)
	}
	return nil
}

// GetImageBackendConfig 返回当前图像后端配置（供角色剧照等场景使用）
func (a *App) GetImageBackendConfig() map[string]interface{} {
	backend := a.cfg.ImageBackend
	if backend == "" {
		backend = "xai"
	}
	currentModel := a.cfg.ImageModel
	if currentModel == "" {
		currentModel = "grok-imagine-image-quality"
	}

	// 根据后端类型构建可用模型列表
	var availableModels []map[string]string

	// 1. 从已启用的引擎中收集图像模型（Herdsman/Ollama/DeepSeek 等）
	if a.engineMgr != nil {
		for _, eng := range a.engineMgr.GetEngines() {
			if !eng.Enabled {
				continue
			}
			for _, m := range eng.Models {
				name := strings.ToLower(m.ID)
				if strings.Contains(name, "image") || strings.Contains(name, "zimage") ||
					strings.Contains(name, "flux") || strings.Contains(name, "krea") ||
					strings.Contains(name, "sd") || strings.Contains(name, "dalle") ||
					strings.Contains(name, "grok-imagine") {
					availableModels = append(availableModels, map[string]string{
						"engine": eng.Name,
						"model":  m.ID,
					})
				}
			}
		}
	}

	// 2. 根据当前后端补充默认模型列表
	switch backend {
	case "comfyui":
		comfyModels := []string{"krea2", "z-image-turbo", "flux"}
		hasCurrent := false
		for _, m := range comfyModels {
			if m == currentModel {
				hasCurrent = true
			}
			availableModels = append(availableModels, map[string]string{
				"engine": "ComfyUI",
				"model":  m,
			})
		}
		if !hasCurrent && currentModel != "" {
			availableModels = append(availableModels, map[string]string{
				"engine": "ComfyUI",
				"model":  currentModel,
			})
		}
	case "xai":
		availableModels = append(availableModels,
			map[string]string{"engine": "xAI", "model": "grok-imagine-image"},
			map[string]string{"engine": "xAI", "model": "grok-imagine-image-quality"},
		)
	}

	if len(availableModels) == 0 {
		availableModels = []map[string]string{
			{"engine": "xAI", "model": "grok-imagine-image"},
			{"engine": "xAI", "model": "grok-imagine-image-quality"},
		}
	}

	return map[string]interface{}{
		"backend":         backend,
		"currentModel":    currentModel,
		"availableModels": availableModels,
	}
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

	// 构建启动参数
	args := []string{"main.py", "--listen", "127.0.0.1", "--port", extractPort(a.cfg.ComfyUIURL)}
	// 使用内置 Python 时加 --windows-standalone-build
	if strings.Contains(pythonExe, "python\\python.exe") || strings.Contains(pythonExe, "python_embeded") {
		args = append(args, "--windows-standalone-build")
	}
	// 不强制指定 GPU 后端，让 ComfyUI 自动检测（支持 NVIDIA CUDA / AMD ROCm / DirectML）
	// 若需要强制 CPU 模式，可在设置中指定 `--cpu` 参数


	cmd := exec.CommandContext(ctx, pythonExe, args...)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8", "TQDM_DISABLE=1")
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
	a.comfyUICmd = cmd

	// 后台等待进程结束，记录退出原因
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("image: comfyui wait goroutine panic recovered", "panic", r)
			}
		}()
		if err := cmd.Wait(); err != nil {
			slog.Warn("ComfyUI 进程退出", "error", err)
		}
		a.comfyUICancel = nil
		a.comfyUICmd = nil
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
			filepath.Join(comfyUIPath, "..", "python", "python.exe"),    // 整合包 python/
			filepath.Join(comfyUIPath, "python_embeded", "python.exe"),  // 便携版
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
// StopComfyUI 停止 ComfyUI 服务（不管是谁启动的都能停）
func (a *App) StopComfyUI() error {
	port := extractPort(a.cfg.ComfyUIURL)

	// 1. 先通过 gaea 内部引用杀进程
	if a.comfyUICmd != nil && a.comfyUICmd.Process != nil {
		a.comfyUICmd.Process.Kill()
		a.comfyUICmd = nil
	}
	if a.comfyUICancel != nil {
		a.comfyUICancel()
		a.comfyUICancel = nil
	}

	// 2. 通过端口查找进程（不管是谁启动的），强制杀
	if pid := findProcessByPort(port); pid > 0 {
		proc, err := os.FindProcess(pid)
		if err == nil {
			proc.Kill()
		}
	}

	slog.Info("ComfyUI 已停止")
	return nil
}

// findProcessByPort 查找监听指定端口的进程 PID（Windows netstat）
func findProcessByPort(port string) int {
	cmd := exec.Command("cmd", "/c",
		fmt.Sprintf("netstat -ano | findstr :%s | findstr LISTENING", port))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	// 输出格式: TCP  0.0.0.0:8188  0.0.0.0:0  LISTENING  12345
	fields := strings.Fields(string(out))
	if len(fields) > 0 {
		pidStr := fields[len(fields)-1]
		pid, _ := strconv.Atoi(pidStr)
		return pid
	}
	return 0
}

// GetComfyUIStatus 返回 ComfyUI 运行状态（含监控：检测到进程退出自动清理引用）
func (a *App) GetComfyUIStatus() map[string]interface{} {
	running := a.isComfyUIRunning()
	// 监控：如果进程不在运行但引用还在，自动清理
	if !running && (a.comfyUICancel != nil || a.comfyUICmd != nil) {
		a.comfyUICancel = nil
		a.comfyUICmd = nil
	}
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
	resp.Body.Close()
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
	dir := a.cfg.ImageSaveDir
	if dir == "" {
		dir = filepath.Join(os.Getenv("USERPROFILE"), "Pictures", "gaea")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("无法创建图片存放目录: %w", err)
	}
	return openDir(dir)
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
		"cpu":      getCPUUsage(),
		"memTotal": getTotalMemory(),
		"memUsed":  getUsedMemory(),
		"gpuName":  "",
		"gpuUsage": 0,
		"vramUsed": 0.0,
		"vramTotal": 0.0,
	}

	// GPU 信息：ComfyUI 运行时从 API 获取，否则用 nvidia-smi/wmic
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
						if v, ok := dev["vram_total"].(float64); ok {
							result["vramTotal"] = v / 1e9
						}
						if total, ok := dev["vram_total"].(float64); ok {
							if free, ok := dev["vram_free"].(float64); ok {
								result["vramUsed"] = (total - free) / 1e9
							}
						}
					}
				}
			}
		}
	} else {
		name, total, used := getGPUInfo()
		result["gpuName"] = name
		result["vramTotal"] = total
		result["vramUsed"] = used
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

// getTotalMemory 获取 Windows 总内存 (GB)
func getTotalMemory() float64 {
	cmd := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0
	}
	kb, err := strconv.ParseFloat(strings.TrimSpace(lines[1]), 64)
	if err != nil {
		return 0
	}
	return kb / 1e6
}

// getUsedMemory 获取 Windows 已用内存 (GB)
func getUsedMemory() float64 {
	total := getTotalMemory()
	cmd := exec.Command("wmic", "OS", "get", "FreePhysicalMemory")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0
	}
	freeKB, err := strconv.ParseFloat(strings.TrimSpace(lines[1]), 64)
	if err != nil {
		return 0
	}
	used := total - freeKB/1e6
	if used < 0 {
		used = 0
	}
	return used
}

// getGPUInfo 获取 GPU 名称、总显存、已用显存 (GB)
func getGPUInfo() (name string, totalGB float64, usedGB float64) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,memory.used", "--format=csv,noheader,nounits")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err == nil {
		line := strings.TrimSpace(string(out))
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			name = strings.TrimSpace(parts[0])
			totalMB, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			usedMB, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			return name, totalMB / 1024.0, usedMB / 1024.0
		}
	}
	// wmic 回退
	cmd2 := exec.Command("wmic", "path", "win32_VideoController", "get", "name")
	cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out2, err := cmd2.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out2)), "\n")
		if len(lines) >= 2 {
			name = strings.TrimSpace(lines[1])
		}
	}
	return
}

// hasNvidiaGPU 检测是否有 NVIDIA GPU
func hasNvidiaGPU() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}
