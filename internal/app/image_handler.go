package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/netclient"
)

// imageItem 单张生成图片结果（包级共享，供移动端任务处理器提取）
type imageItem struct {
	Image  string  `json:"image"`
	Seed   int     `json:"seed"`
	Time   float64 `json:"time"`
	Prompt string  `json:"prompt"`
	Model  string  `json:"model"`
	Size   string  `json:"size"`
	Kind   string  `json:"kind,omitempty"` // image | video
}

func (a *mediaState) beginImageGen(parent context.Context) (context.Context, context.CancelFunc, uint64) {
	a.imageGenMu.Lock()
	defer a.imageGenMu.Unlock()
	if parent == nil {
		parent = context.Background()
	}
	a.imageGenID++
	ctx, cancel := context.WithCancel(parent)
	a.imageGenCancel = cancel
	a.imageGenRunning = true
	return ctx, cancel, a.imageGenID
}

func (a *mediaState) endImageGen(id uint64, cancel context.CancelFunc) {
	a.imageGenMu.Lock()
	defer a.imageGenMu.Unlock()
	if a.imageGenID == id {
		a.imageGenCancel = nil
		a.imageGenRunning = false
		a.clearComfyTaskProgress()
	}
	if cancel != nil {
		cancel()
	}
}

// CancelImageGeneration 取消当前正在执行的图片/视频生成任务。
// 返回 true 表示存在可取消任务；前端生成队列会在任务报错后继续下一条。
func (a *mediaState) CancelImageGeneration() bool {
	a.imageGenMu.Lock()
	defer a.imageGenMu.Unlock()
	if !a.imageGenRunning || a.imageGenCancel == nil {
		return false
	}
	a.imageGenID++
	a.imageGenCancel()
	a.imageGenCancel = nil
	a.imageGenRunning = false
	return true
}

func (a *mediaState) updateComfyTaskProgress(status string, elapsedSeconds int) {
	a.comfyTaskMu.Lock()
	defer a.comfyTaskMu.Unlock()
	a.comfyTaskStatus = status
	a.comfyTaskElapsed = elapsedSeconds
}

func (a *mediaState) clearComfyTaskProgress() {
	a.comfyTaskMu.Lock()
	defer a.comfyTaskMu.Unlock()
	a.comfyTaskStatus = ""
	a.comfyTaskElapsed = 0
}

// GetComfyUITaskProgress 返回当前 ComfyUI 任务状态（前端轮询显示）。
func (a *mediaState) GetComfyUITaskProgress() map[string]interface{} {
	a.comfyTaskMu.RLock()
	defer a.comfyTaskMu.RUnlock()
	return map[string]interface{}{
		"status":  a.comfyTaskStatus,
		"elapsed": a.comfyTaskElapsed,
	}
}

// GenerateFreeImage 自由图片生成 — 供 AI 绘梦 Tab 使用
// GenerateFreeImage 自由图片生成 — 供 AI 绘梦 Tab 使用
// 参数: prompt, negative, size, style, model, seed (0=随机), n (1-4)
func (a *mediaState) GenerateFreeImage(prompt string, negative string, size string, style string, model string, seed int, n int, lora string) (map[string]interface{}, error) {
	if a.client == nil {
		return map[string]interface{}{"error": "AI 客户端未初始化，请先登录"}, nil
	}
	genCtx, cancel, genID := a.beginImageGen(a.ctx)
	defer a.endImageGen(genID, cancel)
	if a.cfg.ImageBackend == "comfyui" {
		a.updateComfyTaskProgress("queued", 0)
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
	comfyRecovered := false

	for i := 0; i < n; i++ {
		genSeed := seed
		if genSeed == 0 {
			genSeed = int(time.Now().UnixNano()%1000000) + i*777
		} else if n > 1 {
			// 固定种子且一次生成多张时，每张用 seed+i，避免 n 张完全雷同
			genSeed = seed + i
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
		if a.cfg.ImageBackend == "comfyui" {
			imgReq.ProgressCallback = a.updateComfyTaskProgress
		}

		// 非 ComfyUI 后端不接受 size 参数（xAI 返回 400）
		if a.cfg.ImageBackend != "comfyui" {
			imgReq.Size = ""
		}
		start := time.Now()
		resp, err := a.client.GenerateImage(genCtx, imgReq)
		// 孤儿 ComfyUI 实例（stderr 失效）会在执行时报 [Errno 22]：
		// 自动重启一次后重试，避免用户手动处理
		if err != nil && !comfyRecovered && a.cfg.ImageBackend == "comfyui" && strings.Contains(err.Error(), "[Errno 22]") {
			slog.Warn("ComfyUI stderr 失效（疑似孤儿实例），自动重启后重试", "error", err)
			a.recoverComfyUI()
			comfyRecovered = true
			resp, err = a.client.GenerateImage(genCtx, imgReq)
		}
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

// mediaGenParams 绘梦多模式生成参数（GenerateMedia 入参，JSON 字符串）
type mediaGenParams struct {
	Prompt    string  `json:"prompt"`
	Negative  string  `json:"negative"`
	Size      string  `json:"size"`
	Model     string  `json:"model"`
	Seed      int     `json:"seed"`
	Lora      string  `json:"lora"`
	Count     int     `json:"count"`
	Mode      string  `json:"mode"`      // txt2img | img2img | t2v
	InitImage string  `json:"initImage"` // 图生图参考图 data URL
	Denoise   float64 `json:"denoise"`   // 重绘幅度 0-1
	Frames    int     `json:"frames"`    // 视频帧数
	FPS       int     `json:"fps"`       // 视频帧率
}

// GenerateMedia 多模式媒体生成：文生图 / 图生图 / 文生视频（供绘梦页使用）
func (a *mediaState) GenerateMedia(paramsJSON string) (map[string]interface{}, error) {
	if a.client == nil {
		return map[string]interface{}{"error": "AI 客户端未初始化，请先登录"}, nil
	}
	genCtx, cancel, genID := a.beginImageGen(a.ctx)
	defer a.endImageGen(genID, cancel)
	if a.cfg.ImageBackend == "comfyui" {
		a.updateComfyTaskProgress("queued", 0)
	}
	var p mediaGenParams
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return map[string]interface{}{"error": "参数解析失败: " + err.Error()}, nil
	}
	mode := p.Mode
	if mode == "" {
		mode = "txt2img"
	}
	if mode != "txt2img" && a.cfg.ImageBackend != "comfyui" {
		return map[string]interface{}{"error": "图生图 / 文生视频目前仅支持 ComfyUI 本地后端，请先在左侧切换引擎"}, nil
	}
	if mode == "img2img" && strings.TrimSpace(p.InitImage) == "" {
		return map[string]interface{}{"error": "图生图需要先上传参考图"}, nil
	}
	if strings.TrimSpace(p.Prompt) == "" {
		return map[string]interface{}{"error": "请输入画面描述"}, nil
	}

	size := p.Size
	if size == "" {
		size = "1024x1024"
		if mode == "t2v" {
			size = "768x512"
		}
	}
	n := p.Count
	if n < 1 || n > 4 {
		n = 1
	}
	if mode == "t2v" {
		n = 1 // 视频一次只生成一条
	}

	results := make([]imageItem, 0, n)
	var lastErr string
	for i := 0; i < n; i++ {
		genSeed := p.Seed
		if genSeed == 0 {
			genSeed = int(time.Now().UnixNano()%1000000) + i*777
		} else if n > 1 {
			// 固定种子且一次生成多张时，每张用 seed+i，避免 n 张完全雷同
			genSeed = p.Seed + i
		}
		imgModel := a.cfg.ImageModel
		if p.Model != "" {
			imgModel = p.Model
		}
		imgReq := &ai.ImageGenerationRequest{
			Model:     imgModel,
			Prompt:    p.Prompt,
			Negative:  p.Negative,
			N:         1,
			Size:      size,
			Seed:      genSeed,
			Lora:      p.Lora,
			Mode:      mode,
			InitImage: p.InitImage,
			Denoise:   p.Denoise,
			Frames:    p.Frames,
			FPS:       p.FPS,
		}
		if a.cfg.ImageBackend == "comfyui" {
			imgReq.ProgressCallback = a.updateComfyTaskProgress
		}
		if a.cfg.ImageBackend != "comfyui" {
			imgReq.Size = ""
		}
		start := time.Now()
		resp, err := a.client.GenerateImage(genCtx, imgReq)
		elapsed := time.Since(start).Seconds()
		if err != nil {
			slog.Warn("媒体生成失败", "mode", mode, "attempt", i+1, "error", err)
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
		kind := resp.Data[0].Kind
		if kind == "" {
			kind = "image"
		}
		results = append(results, imageItem{
			Image:  imageData,
			Seed:   genSeed,
			Time:   math.Round(elapsed*10) / 10,
			Prompt: p.Prompt,
			Model:  imgModel,
			Size:   size,
			Kind:   kind,
		})

		if imageData != "" {
			if a.cfg.ImageSaveDir != "" {
				a.saveMediaToDisk(imageData, p.Prompt, a.cfg.ImageSaveDir)
			} else {
				a.saveMediaToNovelImages(imageData, p.Prompt)
			}
		}
	}

	if len(results) == 0 {
		msg := "生成失败"
		if lastErr != "" {
			msg = msg + "：" + lastErr
		}
		return map[string]interface{}{"error": msg}, nil
	}
	return map[string]interface{}{"results": results, "mode": mode}, nil
}

// saveMediaToDisk 按 data URL 的 MIME 推断扩展名，保存图片/视频到指定目录
func (a *mediaState) saveMediaToDisk(imageData string, prompt string, dir string) string {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}
	ext := mediaExt(imageData)
	filename := mediaFilename(prompt, ext)
	fullPath := filepath.Join(dir, filename)
	if data, ok := decodeDataURL(imageData); ok {
		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			return ""
		}
		return fullPath
	}
	return ""
}

// saveMediaToNovelImages 保存图片/视频到当前小说 images/ 目录
func (a *mediaState) saveMediaToNovelImages(imageData string, prompt string) {
	pm := a.app.getPM()
	if pm == nil {
		return
	}
	dir := filepath.Join(pm.Dir, "images")
	a.saveMediaToDisk(imageData, prompt, dir)
}

// mediaExt 从 data URL 推断扩展名
func mediaExt(imageData string) string {
	switch {
	case strings.HasPrefix(imageData, "data:video/mp4"):
		return ".mp4"
	case strings.HasPrefix(imageData, "data:video/webm"):
		return ".webm"
	case strings.HasPrefix(imageData, "data:video/quicktime"):
		return ".mov"
	case strings.HasPrefix(imageData, "data:image/webp"):
		return ".webp"
	case strings.HasPrefix(imageData, "data:image/gif"):
		return ".gif"
	case strings.HasPrefix(imageData, "data:image/jpeg"), strings.HasPrefix(imageData, "data:image/jpg"):
		return ".jpg"
	default:
		return ".png"
	}
}

// mediaFilename 生成媒体文件名（时间戳 + 前 20 字提示词）
func mediaFilename(prompt string, ext string) string {
	ts := strconv.FormatInt(time.Now().UnixNano(), 10)
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
	return fmt.Sprintf("%s_%s%s", ts, safePrompt, ext)
}

// decodeDataURL 解码 data URL 内容
func decodeDataURL(imageData string) ([]byte, bool) {
	if !strings.HasPrefix(imageData, "data:") {
		return nil, false
	}
	commaIdx := strings.Index(imageData, ",")
	if commaIdx < 0 {
		return nil, false
	}
	data, err := base64.StdEncoding.DecodeString(imageData[commaIdx+1:])
	if err != nil {
		return nil, false
	}
	return data, true
}

// saveImageToDisk 将图片数据保存到 ImageSaveDir，返回保存路径
func (a *mediaState) saveImageToDisk(imageData string, prompt string) string {
	return a.saveMediaToDisk(imageData, prompt, a.cfg.ImageSaveDir)
}

// saveToNovelImages 将图片保存到当前小说的 images/ 目录
func (a *mediaState) saveToNovelImages(imageData string, prompt string) {
	pm := a.app.getPM()
	if pm == nil {
		return
	}
	dir := filepath.Join(pm.Dir, "images")
	a.saveMediaToDisk(imageData, prompt, dir)
}

// GetImageBackend 获取当前图片后端类型（供前端显示）
func (a *mediaState) GetImageBackend() string {
	if a.client != nil {
		return a.client.GetImageBackendType()
	}
	return "xai"
}

// GetImageBackendInfo 获取当前图片后端类型和模型（供前端显示）
func (a *mediaState) GetImageBackendInfo() map[string]string {
	return map[string]string{
		"backend": a.GetImageBackend(),
		"model":   a.cfg.ImageModel,
	}
}

// GetPortraitConfig 获取角色库剧照独立后端/模型（空 = 跟随绘梦）
func (a *App) GetPortraitConfig() map[string]string {
	return map[string]string{
		"backend": a.cfg.PortraitBackend,
		"model":   a.cfg.PortraitModel,
	}
}

// SetPortraitConfig 设置角色库剧照独立后端/模型（空 = 跟随绘梦）
func (a *App) SetPortraitConfig(backend, model string) error {
	a.cfg.PortraitBackend = backend
	a.cfg.PortraitModel = model
	if err := config.Save(config.KeyPortraitBackend, backend); err != nil {
		slog.Warn("保存剧照后端失败", "error", err)
		return err
	}
	if err := config.Save(config.KeyPortraitModel, model); err != nil {
		slog.Warn("保存剧照模型失败", "error", err)
		return err
	}
	slog.Info("角色库剧照绑定已设置", "backend", backend, "model", model)
	return nil
}

// SetImageBackend 切换图片生成后端（供设置页调用）
func (a *mediaState) SetImageBackend(backend string, comfyUIURL string, imageModel string, imageSaveDir string) error {
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

	// 持久化绘梦配置，避免应用重启后回退到默认后端/模型/保存目录
	if err := config.Save(config.KeyImageBackend, backend); err != nil {
		slog.Warn("保存图片后端失败", "error", err)
	}
	if comfyUIURL != "" {
		if err := config.Save(config.KeyComfyUIURL, comfyUIURL); err != nil {
			slog.Warn("保存 ComfyUI 地址失败", "error", err)
		}
	}
	if a.cfg.ImageModel != "" {
		if err := config.Save(config.KeyImageModel, a.cfg.ImageModel); err != nil {
			slog.Warn("保存图片模型失败", "error", err)
		}
	}
	if a.cfg.ImageSaveDir != "" {
		if err := config.Save(config.KeyImageSaveDir, a.cfg.ImageSaveDir); err != nil {
			slog.Warn("保存图片存放目录失败", "error", err)
		}
	}
	return nil
}

// GetImageBackendConfig 返回当前图像后端配置（供角色剧照等场景使用）
func (a *mediaState) GetImageBackendConfig() map[string]interface{} {
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

	// 2. 恒提供 ComfyUI 本地模型（角色剧照等场景可选择本地出图，本机单用户定位）
	comfyAlways := []string{"krea2", "z-image-turbo", "flux"}
	for _, cm := range comfyAlways {
		dup := false
		for _, m := range availableModels {
			if m["model"] == cm {
				dup = true
				break
			}
		}
		if !dup {
			availableModels = append(availableModels, map[string]string{"engine": "ComfyUI", "model": cm})
		}
	}

	// 3. 根据当前后端补充默认模型列表
	switch backend {
	case "comfyui":
		hasCurrent := false
		for _, m := range availableModels {
			if m["model"] == currentModel {
				hasCurrent = true
				break
			}
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
func (a *mediaState) StartComfyUI() error {
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
	// 使用内置 Python / standalone-env 时加 --windows-standalone-build
	//（standalone-env 是 ROCm PyTorch 环境，Krea2/Z-Image-Turbo 必需；系统 Python 为 CPU-only）
	if strings.Contains(pythonExe, "python\\python.exe") || strings.Contains(pythonExe, "python_embeded") || strings.Contains(pythonExe, "standalone-env") {
		args = append(args, "--windows-standalone-build")
	}
	// 不强制指定 GPU 后端，让 ComfyUI 自动检测（支持 NVIDIA CUDA / AMD ROCm / DirectML）
	// 若需要强制 CPU 模式，可在设置中指定 `--cpu` 参数

	cmd := exec.CommandContext(ctx, pythonExe, args...)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8", "TQDM_DISABLE=1")
	cmd.Dir = a.cfg.ComfyUIPath
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// stdout/stderr 重定向到日志文件（~/.gaea/logs/comfyui.log）：
	// 文件句柄在 gaea 进程退出后依然有效，避免孤儿 ComfyUI 实例的 stderr
	// 失效，导致生成时 tqdm flush 报 [Errno 22] Invalid argument。
	var logFile *os.File
	if home, err := os.UserHomeDir(); err == nil {
		logDir := filepath.Join(home, ".gaea", "logs")
		if err := os.MkdirAll(logDir, 0755); err == nil {
			if f, err := os.OpenFile(filepath.Join(logDir, "comfyui.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				logFile = f
			}
		}
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		cancel()
		a.comfyUICancel = nil
		if logFile != nil {
			logFile.Close()
		}
		errMsg := ""
		if home, err := os.UserHomeDir(); err == nil {
			if b, err := os.ReadFile(filepath.Join(home, ".gaea", "logs", "comfyui.log")); err == nil {
				if len(b) > 4096 {
					b = b[len(b)-4096:]
				}
				errMsg = string(b)
			}
		}
		if len(errMsg) > 600 {
			errMsg = "..." + errMsg[len(errMsg)-600:]
		}
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
			if logFile != nil {
				logFile.Close()
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

	// 2. ComfyUI 便携版（standalone-env 为 ROCm PyTorch 环境，Krea2/ZIT 必需，优先级最高）
	if comfyUIPath != "" {
		candidates := []string{
			filepath.Join(comfyUIPath, "..", "standalone-env", "python.exe"), // standalone-env（ROCm PyTorch）
			filepath.Join(comfyUIPath, "..", "python", "python.exe"),         // 整合包 python/
			filepath.Join(comfyUIPath, "python_embeded", "python.exe"),       // 便携版
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
func (a *mediaState) StopComfyUI() error {
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

// recoverComfyUI 重启 ComfyUI 并等待就绪（用于孤儿实例 stderr 失效的自动恢复）。
// 最多等待约 90 秒；失败仅记录日志，由上层按原错误返回。
func (a *mediaState) recoverComfyUI() {
	if err := a.StopComfyUI(); err != nil {
		slog.Warn("自动恢复：停止 ComfyUI 失败", "error", err)
	}
	// 等待端口释放，避免立刻重启时端口仍被占用
	time.Sleep(2 * time.Second)
	if err := a.StartComfyUI(); err != nil {
		slog.Warn("自动恢复：启动 ComfyUI 失败", "error", err)
		return
	}
	for i := 0; i < 30; i++ {
		time.Sleep(3 * time.Second)
		if a.isComfyUIRunning() {
			slog.Info("ComfyUI 自动恢复完成")
			return
		}
	}
	slog.Warn("ComfyUI 自动恢复超时")
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
func (a *mediaState) GetComfyUIStatus() map[string]interface{} {
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

// GetComfyUILoras 返回 ComfyUI 当前可用的 LoRA 列表（绘梦 LoRA 多选动态加载）
func (a *mediaState) GetComfyUILoras() ([]string, error) {
	if a.cfg.ComfyUIURL == "" {
		return nil, fmt.Errorf("ComfyUI 地址未配置")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	backend := ai.NewComfyUIBackend(a.cfg.ComfyUIURL)
	return backend.ListLoras(ctx)
}

// isComfyUIRunning 检查 ComfyUI 是否可连通
func (a *mediaState) isComfyUIRunning() bool {
	client := netclient.NewSimpleClient(2 * time.Second)
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
func (a *mediaState) OpenImageSaveDir() error {
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
func (a *mediaState) OpenNovelImagesDir() error {
	pm := a.app.getPM()
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
func (a *mediaState) GetSystemStats() map[string]interface{} {
	memTotal, memUsed := getMemoryStats()
	result := map[string]interface{}{
		"cpu":       getCPUUsage(),
		"memTotal":  memTotal,
		"memUsed":   memUsed,
		"gpuName":   "",
		"gpuUsage":  0,
		"vramUsed":  0.0,
		"vramTotal": 0.0,
	}

	// GPU 信息：ComfyUI 运行时从 API 获取，否则用 nvidia-smi/wmic
	if a.isComfyUIRunning() {
		client := netclient.NewSimpleClient(3 * time.Second)
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

// getMemoryStats 获取 Windows 总内存与已用内存 (GB)，一次 wmic 调用取两列
// getMemoryStats 获取 Windows 总内存与已用内存 (GB)，一次 wmic 调用取两列
func getMemoryStats() (totalGB, usedGB float64) {
	cmd := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, 0
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return 0, 0
	}
	totalKB, err1 := strconv.ParseFloat(fields[0], 64)
	freeKB, err2 := strconv.ParseFloat(fields[1], 64)
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	used := (totalKB - freeKB) / 1e6
	if used < 0 {
		used = 0
	}
	return totalKB / 1e6, used
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
