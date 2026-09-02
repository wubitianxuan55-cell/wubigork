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
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
	"github.com/gaea/gaea/internal/netclient"
)

// imageItem 单张生成图片结果（包级共享，供移动端任务处理器提取）
type imageItem struct {
	Image    string  `json:"image"`
	Seed     int     `json:"seed"`
	Time     float64 `json:"time"`
	Prompt   string  `json:"prompt"`
	Model    string  `json:"model"`
	Size     string  `json:"size"`
	Kind     string  `json:"kind,omitempty"`      // image | video
	FilePath string  `json:"file_path,omitempty"` // T6-4.3：本地保存路径（历史图片可恢复）
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
//
// T6-4.1 取消真实生效：除 cancel context（令 gaea 轮询即刻退出）外，还会调用
// ComfyUI /interrupt 中断当前任务；本地取消标记会拒绝取消后的后续提交
// （ComfyUI 无删除排队任务的 API，见 ComfyUIBackend.Interrupt 的说明）。
// 幂等：首次取消后 imageGenCancel 置空，重复调用返回 false。
func (a *mediaState) CancelImageGeneration() bool {
	a.imageGenMu.Lock()
	if !a.imageGenRunning || a.imageGenCancel == nil {
		a.imageGenMu.Unlock()
		return false
	}
	a.imageGenID++
	cancel := a.imageGenCancel
	a.imageGenCancel = nil
	a.imageGenRunning = false
	a.imageGenMu.Unlock()

	cancel()
	a.interruptComfyUI()
	return true
}

// interruptComfyUI 调用 ComfyUI /interrupt 中断当前任务（T6-4.1）。
// 通过 ai.Client.GetImageBackend 类型断言取回真实后端实例；
// 失败仅记录日志，不掩盖（context 取消已令轮询退出，中断失败意味着
// ComfyUI 端任务会继续跑完，日志便于排查）。
func (a *mediaState) interruptComfyUI() {
	if a.cfg == nil || a.cfg.ImageBackend != "comfyui" || a.client == nil {
		return
	}
	ib, ok := a.client.GetImageBackend().(interface{ Interrupt(context.Context) error })
	if !ok {
		slog.Warn("取消生成：当前图片后端不支持中断", "backend", a.cfg.ImageBackend)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := ib.Interrupt(ctx); err != nil {
		slog.Warn("取消生成：ComfyUI /interrupt 调用失败", "error", err)
	}
}

// resetComfyCancel 新一轮生成开始时清除 ComfyUI 本地取消标记（T6-4.1），
// 保证取消后用户可正常发起新任务。
func (a *mediaState) resetComfyCancel() {
	if a.cfg == nil || a.cfg.ImageBackend != "comfyui" || a.client == nil {
		return
	}
	if ib, ok := a.client.GetImageBackend().(interface{ ResetCancel() }); ok {
		ib.ResetCancel()
	}
}

// updateComfyTaskProgress 更新 ComfyUI 任务状态（进度回调；status=queued/running）。
// percent<0 表示未知（保留上一次），node 为空表示无变化（保留上一次）。
func (a *mediaState) updateComfyTaskProgress(status string, elapsedSeconds int, percent int, node string) {
	a.comfyTaskMu.Lock()
	defer a.comfyTaskMu.Unlock()
	a.comfyTaskStatus = status
	a.comfyTaskElapsed = elapsedSeconds
	if percent >= 0 {
		a.comfyTaskPercent = percent
	}
	if node != "" {
		a.comfyTaskNode = node
	}
}

func (a *mediaState) clearComfyTaskProgress() {
	a.comfyTaskMu.Lock()
	defer a.comfyTaskMu.Unlock()
	a.comfyTaskStatus = ""
	a.comfyTaskElapsed = 0
	a.comfyTaskPercent = 0
	a.comfyTaskNode = ""
}

// GetComfyUITaskProgress 返回当前 ComfyUI 任务状态（前端轮询显示）。
func (a *mediaState) GetComfyUITaskProgress() map[string]interface{} {
	a.comfyTaskMu.RLock()
	defer a.comfyTaskMu.RUnlock()
	return map[string]interface{}{
		"status":  a.comfyTaskStatus,
		"elapsed": a.comfyTaskElapsed,
		"percent": a.comfyTaskPercent,
		"node":    a.comfyTaskNode,
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
		a.updateComfyTaskProgress("queued", 0, 0, "")
		a.resetComfyCancel()
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
	// S1.5-B play 内容护栏：image_safe_mode 提交前注入提示词安全段（后端
	// NSFW 开关位：ai 图片后端无 NSFW 透传字段，按后端能力缺省关，无法
	// 透传时仅注入 prompt 安全段）。未配置 = 零值 = 提示词原样。
	safePrompt := applyImageSafeMode(fullPrompt, playGuardrails().ImageSafeMode)

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
			Prompt:   safePrompt,
			Negative: negative,
			N:        1,
			Size:     size,
			Seed:     genSeed,
			Lora:     lora,
		}
		if a.cfg.ImageBackend == "comfyui" {
			imgReq.ProgressCallback = a.updateComfyTaskProgress
		}

		// xAI / Ollama 后端不接受 size 参数（xAI 返回 400）；herdsman 文档明确支持
		// size；GLM 官方 schema 同样接受 size（glm-image 默认 1280x1280）
		if a.cfg.ImageBackend != "comfyui" && a.cfg.ImageBackend != "herdsman" && a.cfg.ImageBackend != "glm" {
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

		item := imageItem{
			Image:  imageData,
			Seed:   genSeed,
			Time:   math.Round(elapsed*10) / 10,
			Prompt: fullPrompt,
			Model:  imgModel,
			Size:   size,
		}
		// T6-4.3：保存路径写入历史元数据（前端历史图片据此恢复本地文件）
		if a.cfg.ImageSaveDir != "" && imageData != "" {
			item.FilePath = a.saveImageToDisk(imageData, fullPrompt)
		} else if imageData != "" {
			// 未配置专用目录时，自动保存到小说 images/ 目录
			item.FilePath = a.saveToNovelImages(imageData, fullPrompt)
		}
		images = append(images, item)
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
		a.updateComfyTaskProgress("queued", 0, 0, "")
		a.resetComfyCancel()
	}
	var p mediaGenParams
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return map[string]interface{}{"error": "参数解析失败: " + err.Error()}, nil
	}
	mode := p.Mode
	if mode == "" {
		mode = "txt2img"
	}
	if mode == "t2v" && a.cfg.ImageBackend != "comfyui" {
		return map[string]interface{}{"error": "文生视频目前仅支持 ComfyUI 本地后端，请先在左侧切换引擎"}, nil
	}
	if mode == "img2img" && a.cfg.ImageBackend != "comfyui" && a.cfg.ImageBackend != "herdsman" && a.cfg.ImageBackend != "dashscope" {
		return map[string]interface{}{"error": "图生图目前支持 ComfyUI / Herdsman / 百炼(DashScope) 后端，请先在左侧切换引擎"}, nil
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
	// S1.5-B play 内容护栏：image_safe_mode 同 GenerateFreeImage（提交前
	// 注入提示词安全段；未配置 = 零值 = 提示词原样）。
	safePrompt := applyImageSafeMode(p.Prompt, playGuardrails().ImageSafeMode)
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
			Prompt:    safePrompt,
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
		// xAI / Ollama 后端不接受 size 参数（xAI 返回 400）；herdsman 文档明确支持
		// size；GLM 官方 schema 同样接受 size（glm-image 默认 1280x1280）；
		// dashscope 改图默认不传 size（输出宽高比随输入图，保持原图比例）
		if a.cfg.ImageBackend != "comfyui" && a.cfg.ImageBackend != "herdsman" && a.cfg.ImageBackend != "glm" && a.cfg.ImageBackend != "dashscope" {
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
		item := imageItem{
			Image:  imageData,
			Seed:   genSeed,
			Time:   math.Round(elapsed*10) / 10,
			Prompt: p.Prompt,
			Model:  imgModel,
			Size:   size,
			Kind:   kind,
		}
		// T6-4.3：保存路径写入历史元数据（前端历史图片据此恢复本地文件）
		if imageData != "" {
			if a.cfg.ImageSaveDir != "" {
				item.FilePath = a.saveMediaToDisk(imageData, p.Prompt, a.cfg.ImageSaveDir)
			} else {
				item.FilePath = a.saveMediaToNovelImages(imageData, p.Prompt)
			}
		}
		results = append(results, item)
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

// editImageFromCard 对话式改图（包内消费：intent_router execEditImage）。
// initImage 为参考图 data URL，prompt 为编辑指令；走当前图片后端 img2img
// 生成一张结果图，落盘（与 GenerateMedia 同口径：ImageSaveDir 优先，未配置
// 落当前小说 images/ 目录）后返回本地文件路径作为 cardPath；错误原样返回。
// 不导出=不进绑定面（仅供微信/意图链内部使用）。
func (a *mediaState) editImageFromCard(initImage, prompt string) (cardPath string, err error) {
	if a.client == nil {
		return "", fmt.Errorf("AI 客户端未初始化，请先登录")
	}
	// 仅 img2img 能力后端可走改图（其余引擎诚实报错，不静默降级为文生图）
	switch a.cfg.ImageBackend {
	case "comfyui", "herdsman", "dashscope":
	default:
		return "", fmt.Errorf("当前生图引擎不支持改图")
	}
	if strings.TrimSpace(initImage) == "" {
		return "", fmt.Errorf("改图需要先上传参考图")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("请输入编辑指令")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background() // 非 Wails 上下文（测试等）兜底，同 GetComfyUILoras
	}
	resp, err := a.client.GenerateImage(ctx, &ai.ImageGenerationRequest{
		Model:     a.cfg.ImageModel, // 空则后端回默认模型（如百炼 qwen-image-edit-plus）
		Prompt:    prompt,
		N:         1,
		Mode:      "img2img",
		InitImage: initImage,
		// Size 不传：改图输出宽高比随参考图，保持原图比例
	})
	if err != nil {
		return "", err
	}
	if len(resp.Data) == 0 {
		return "", fmt.Errorf("改图失败：API 返回空结果")
	}
	imageData := resp.Data[0].URL
	if imageData == "" {
		imageData = resp.Data[0].B64JSON
	}
	if imageData == "" {
		return "", fmt.Errorf("改图失败：结果为空")
	}
	// 落盘口径与 GenerateMedia 一致：ImageSaveDir 优先，未配置落小说 images/
	var filePath string
	if a.cfg.ImageSaveDir != "" {
		filePath = a.saveMediaToDisk(imageData, prompt, a.cfg.ImageSaveDir)
	} else {
		filePath = a.saveMediaToNovelImages(imageData, prompt)
	}
	if filePath == "" {
		return "", fmt.Errorf("改图结果保存失败，请检查图片存放目录权限")
	}
	return filePath, nil
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

// saveMediaToNovelImages 保存图片/视频到当前小说 images/ 目录，返回保存路径（失败返回空串）
func (a *mediaState) saveMediaToNovelImages(imageData string, prompt string) string {
	pm := a.app.getPM()
	if pm == nil {
		return ""
	}
	dir := filepath.Join(pm.Dir, "images")
	return a.saveMediaToDisk(imageData, prompt, dir)
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

// saveToNovelImages 将图片保存到当前小说的 images/ 目录，返回保存路径（失败返回空串）
func (a *mediaState) saveToNovelImages(imageData string, prompt string) string {
	pm := a.app.getPM()
	if pm == nil {
		return ""
	}
	dir := filepath.Join(pm.Dir, "images")
	return a.saveMediaToDisk(imageData, prompt, dir)
}

// GetImageBackend 获取当前图片后端类型（供前端显示）
func (a *mediaState) GetImageBackend() string {
	if a.client != nil {
		return a.client.GetImageBackendType()
	}
	return "xai"
}

// GetImageBackendInfo 获取当前图片后端类型和模型（供前端显示）。
// 兼容旧字段 backend/model，同时下发完整配置（image_model/comfyui_url/
// image_save_dir/comfyui_path/comfyui_python_path），模型中心据此恢复表单。
// isGLMImageModel 是否 GLM 官方生图模型（cogview 系 / glm-image 系）。
func isGLMImageModel(model string) bool {
	l := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(l, "cogview") || strings.HasPrefix(l, "glm-image")
}

func (a *mediaState) GetImageBackendInfo() map[string]string {
	imageModel := a.cfg.ImageModel
	switch {
	case a.cfg.ImageBackend == "comfyui" && imageModel == "":
		imageModel = "krea2"
	case a.cfg.ImageBackend == "glm":
		// 空模型或上一后端残留（如 grok-imagine-*）都归位 GLM 默认生图模型，
		// 避免表单带非官方模型名去请求（官方会报 model 不存在）。
		if !isGLMImageModel(imageModel) {
			imageModel = ai.GLMDefaultImageModel
		}
	case imageModel == "":
		imageModel = "grok-imagine-image-quality"
	}
	saveDir := a.cfg.ImageSaveDir
	if saveDir == "" {
		saveDir = filepath.Join(os.Getenv("USERPROFILE"), "Pictures", "gaea")
	}
	return map[string]string{
		"backend":             a.GetImageBackend(),
		"model":               imageModel,
		"image_model":         imageModel,
		"comfyui_url":         a.cfg.ComfyUIURL,
		"image_save_dir":      saveDir,
		"comfyui_path":        a.cfg.ComfyUIPath,
		"comfyui_python_path": a.cfg.ComfyUIPythonPath,
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

// SetImageBackend 切换图片生成后端（供设置页调用）。
// dashscopeKey 为百炼 API Key 明文（前端传入）：经 secure 加密后存
// cfg.DashScopeAPIKey 并落盘（存储口径同 GLM 等 Key 类配置）；传空时保留
// 已存 Key（切换其他后端不丢百炼 Key）。
func (a *mediaState) SetImageBackend(backend string, comfyUIURL string, imageModel string, imageSaveDir string, dashscopeKey string) error {
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
	case "glm":
		eng, ok := a.engineMgr.GetEngine("glm")
		if !ok || !eng.Enabled {
			return fmt.Errorf("GLM 引擎未启用，请先在模型中心启用")
		}
		key := a.engineMgr.GLMKey()
		if key == "" {
			return fmt.Errorf("GLM API Key 未配置，请先在模型中心 GLM 卡片保存 Key（open.bigmodel.cn 获取）")
		}
		a.cfg.ImageBackend = "glm"
		if imageModel != "" {
			a.cfg.ImageModel = imageModel
		}
		a.client.SetImageBackend(ai.NewGLMImageBackend(eng.BaseURL, key), "glm")
	case "dashscope":
		key := strings.TrimSpace(dashscopeKey)
		if key == "" {
			// 未传新 Key：回退已存的百炼 Key（密文解密），避免误切换报「未配置」
			stored, decErr := secure.DecryptString(a.cfg.DashScopeAPIKey)
			if decErr != nil {
				return fmt.Errorf("百炼 API Key 解密失败: %w", decErr)
			}
			key = strings.TrimSpace(stored)
		}
		if key == "" {
			return fmt.Errorf("百炼 API Key 未配置，请先在百炼控制台（dashscope.aliyuncs.com）获取并填入")
		}
		enc, err := secure.EncryptString(key)
		if err != nil {
			return fmt.Errorf("百炼 API Key 加密失败: %w", err)
		}
		a.cfg.DashScopeAPIKey = enc
		a.cfg.ImageBackend = "dashscope"
		if imageModel != "" {
			a.cfg.ImageModel = imageModel
		}
		ib, err := ai.NewImageBackend(ai.ImageBackendKindDashScope, ai.ImageBackendConfig{
			BaseURL: ai.DashScopeBaseURL,
			APIKey:  key,
		})
		if err != nil {
			return err
		}
		a.client.SetImageBackend(ib, "dashscope")
	default:
		return fmt.Errorf("不支持的后端: %s（支持 xai / comfyui / herdsman / ollama / glm / dashscope）", backend)
	}

	// 持久化绘梦配置，避免应用重启后回退到默认后端/模型/保存目录
	if err := config.Save(config.KeyImageBackend, backend); err != nil {
		slog.Warn("保存图片后端失败", "error", err)
	}
	// 百炼 Key：cfg 内为 secure 密文（存储口径同 GLM Key），传了新 Key 才落盘
	if dashscopeKey != "" {
		if err := config.Save(config.KeyDashScopeAPIKey, a.cfg.DashScopeAPIKey); err != nil {
			slog.Warn("保存百炼 Key 失败", "error", err)
		}
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
	// 端口已被占用但 /system_stats 未就绪（实例正在启动/卡死）：直接复用现有进程，
	// 避免再拉起第二个实例导致 "Port 8188 already in use" 与 SQLite 数据库锁冲突。
	if pid := findProcessByPort(extractPort(a.cfg.ComfyUIURL)); pid > 0 {
		slog.Warn("ComfyUI 端口已被占用，跳过重复启动", "port", extractPort(a.cfg.ComfyUIURL), "pid", pid)
		return fmt.Errorf("端口 %s 已被进程 %d 占用（ComfyUI 可能正在启动），请稍候重试", extractPort(a.cfg.ComfyUIURL), pid)
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

// findProcessByPort 查找监听指定端口的进程 PID（Windows netstat -ano）。
// T6-4.5：弃用 cmd/findstr 字符串拼接（可注入命令），改用参数数组 exec.Command
// 并解析输出；port 入参先做白名单校验（纯数字 1–65535），杜绝注入。
func findProcessByPort(port string) int {
	if !isValidPort(port) {
		return 0
	}
	cmd := exec.Command("netstat", "-ano")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	return parseNetstatPID(string(out), port)
}

// parseNetstatPID 解析 netstat -ano 输出，返回监听 port 的进程 PID（0 = 未找到）。
// 输出格式: TCP  0.0.0.0:8188  0.0.0.0:0  LISTENING  12345
// 仅匹配本地地址以 :port 结尾且状态为 LISTENING 的 TCP 行（避免 findstr 的子串误匹配）。
func parseNetstatPID(out string, port string) int {
	target := ":" + port
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if fields[0] != "TCP" {
			continue
		}
		if !strings.HasSuffix(fields[1], target) {
			continue
		}
		if fields[3] != "LISTENING" {
			continue
		}
		if pid, err := strconv.Atoi(fields[4]); err == nil && pid > 0 {
			return pid
		}
	}
	return 0
}

// isValidPort 校验端口号：纯数字且在 1–65535 范围内（T6-4.5 注入防护）。
func isValidPort(port string) bool {
	if port == "" || len(port) > 5 {
		return false
	}
	for _, r := range port {
		if r < '0' || r > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535
}

// GetComfyUIStatus 返回 ComfyUI 运行状态（含监控：检测到进程退出自动清理引用）
func (a *mediaState) GetComfyUIStatus() map[string]interface{} {
	running := a.isComfyUIRunning()
	// 监控：如果进程不在运行但引用还在，自动清理
	if !running && (a.comfyUICancel != nil || a.comfyUICmd != nil) {
		a.comfyUICancel = nil
		a.comfyUICmd = nil
	}
	p, _ := strconv.Atoi(extractPort(a.cfg.ComfyUIURL))
	return map[string]interface{}{
		"running": running,
		"url":     a.cfg.ComfyUIURL,
		"port":    p,
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

// ── Windows 系统资源采集（弃用 wmic——Win11 24H2 起已移除；
//    直接调 kernel32.dll 原生 API，不依赖 x/sys/windows 符号，全版本可用）──

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes        = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx  = kernel32.NewProc("GlobalMemoryStatusEx")
)

type winFiletime struct{ LowDateTime, HighDateTime uint32 }

// winMemoryStatusEx 必须与 Windows MEMORYSTATUSEX 完全一致（64 字节）：
// 2×uint32 + 7×uint64。字段数不对会令 GlobalMemoryStatusEx 失败（返回 0）。
type winMemoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys, AvailPhys uint64
	TotalPageFile, AvailPageFile   uint64
	TotalVirtual, AvailVirtual     uint64
	AvailExtendedVirtual uint64
}

// getCPUUsage 获取 Windows CPU 使用率（%）。
// GetSystemTimes 两次采样差值计算；首次返回 0（无历史基准），
// 短间隔（<1s）重复调用返回上一次结果而不是重置基准，避免多个轮询者叠加导致恒 0。
var (
	prevCPUTimes  [3]uint64 // idle, kernel, user（100ns 单位）
	prevCPUSample time.Time
	prevCPUInit   bool
	prevCPUUsage  int
)

func getCPUUsage() int {
	var idle, kernel, user winFiletime
	r1, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r1 == 0 {
		return prevCPUUsage // 保持上一次，避免偶尔失败闪烁为 0
	}
	now := time.Now()
	idleT := uint64(idle.HighDateTime)<<32 | uint64(idle.LowDateTime)
	kernelT := uint64(kernel.HighDateTime)<<32 | uint64(kernel.LowDateTime)
	userT := uint64(user.HighDateTime)<<32 | uint64(user.LowDateTime)
	if !prevCPUInit {
		prevCPUTimes = [3]uint64{idleT, kernelT, userT}
		prevCPUSample = now
		prevCPUInit = true
		return 0
	}
	if now.Sub(prevCPUSample) < time.Second {
		return prevCPUUsage
	}
	dIdle := idleT - prevCPUTimes[0]
	dKernel := kernelT - prevCPUTimes[1]
	dUser := userT - prevCPUTimes[2]
	prevCPUTimes = [3]uint64{idleT, kernelT, userT}
	prevCPUSample = now
	total := dKernel + dUser // kernel 含 idle
	if total == 0 {
		return 0
	}
	usage := int((total - dIdle) * 100 / total)
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	prevCPUUsage = usage
	return usage
}

// getMemoryStats 获取 Windows 总内存与已用内存 (GB)。GlobalMemoryStatusEx 原生 API。
func getMemoryStats() (totalGB, usedGB float64) {
	var ms winMemoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return 0, 0
	}
	totalGB = float64(ms.TotalPhys) / 1e9
	usedGB = float64(ms.TotalPhys-ms.AvailPhys) / 1e9
	if usedGB < 0 {
		usedGB = 0
	}
	return totalGB, usedGB
}

// GPU 信息缓存（15s TTL）：nvidia-smi / PowerShell 每次轮询都执行代价高，
// 且显存总量不会频繁变化；已用显存由 nvidia-smi（NVIDIA）或 ComfyUI 提供。
var (
	gpuCacheMu    sync.Mutex
	gpuCacheAt    time.Time
	gpuCacheName  string
	gpuCacheTotal float64
	gpuCacheUsed  float64
)

// getGPUInfo 获取 GPU 名称、总显存、已用显存 (GB)
func getGPUInfo() (name string, totalGB float64, usedGB float64) {
	gpuCacheMu.Lock()
	defer gpuCacheMu.Unlock()
	if !gpuCacheAt.IsZero() && time.Since(gpuCacheAt) < 15*time.Second {
		return gpuCacheName, gpuCacheTotal, gpuCacheUsed
	}

	// NVIDIA：nvidia-smi 直接给出名称 + 总/已用显存
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
			gpuCacheName, gpuCacheTotal, gpuCacheUsed = name, totalMB/1024.0, usedMB/1024.0
			gpuCacheAt = time.Now()
			return name, totalMB / 1024.0, usedMB / 1024.0
		}
	}

	// AMD / 其他：注册表读真实 GPU 显存（HardwareInformation.qwMemorySize），
	// 跳过 Todesk / 虚拟显示器 / 基础显示适配器等虚拟设备。
	psScript := `
$vram=0; $best=''
Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}' -ErrorAction SilentlyContinue | ForEach-Object {
  $p = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
  $d = $p.'DriverDesc'
  if (-not $d) { return }
  if ($d -match 'Todesk|Virtual|Remote|Basic Display|RDP|Mirror') { return }
  $m = $p.'HardwareInformation.qwMemorySize'
  if (-not $m) { $m = $p.AdapterRAM }
  if ($m -and [uint64]$m -gt $vram) { $vram = [uint64]$m; $best = $d }
}
if ($best) { $best + '|' + $vram }
`
	cmd2 := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out2, err := cmd2.Output()
	if err == nil {
		line := strings.TrimSpace(string(out2))
		if i := strings.Index(line, "|"); i > 0 {
			gb, _ := strconv.ParseFloat(strings.TrimSpace(line[i+1:]), 64)
			if gb > 0 {
				name, totalGB = strings.TrimSpace(line[:i]), gb/1e9
				gpuCacheName, gpuCacheTotal, gpuCacheUsed = name, totalGB, 0
				gpuCacheAt = time.Now()
				return
			}
		}
	}

	// 兜底：CIM 名称（跳过虚拟设备），无显存信息
	cmd3 := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-CimInstance Win32_VideoController | Where-Object { $_.Name -notmatch 'Todesk|Virtual|Remote|Basic Display|RDP|Mirror' } | Select-Object -First 1).Name")
	cmd3.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out3, err := cmd3.Output(); err == nil {
		name = strings.TrimSpace(string(out3))
	}
	gpuCacheName, gpuCacheTotal, gpuCacheUsed = name, 0, 0
	gpuCacheAt = time.Now()
	return
}
