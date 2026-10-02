package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
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
	// 变体簇（阶段二刀 D）：登记回填——asset_id 本图台账条目；parent_id 源图
	// 条目（A→A' 同源链；空=非派生产物或源图不在台账）。
	AssetID  string `json:"asset_id,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
}

// CU1（蒸馏 unsloth §六-2）ComfyUI 预热：运行态武装位（Startup 置位，同
// imageHubRuntimeArmed 先例禁 cfg!=nil 惯性闸）+ 已启动闸（只在真正发起
// 预热时置位；首入时 ComfyUI 未就绪等可重试原因不消耗触发机会）。
var comfyWarmArmed atomic.Bool
var comfyWarmStarted atomic.Bool

// WarmComfyUI 绘梦页首入预热：后台提交一次 64×64 极小空跑把当前默认生图模型
// 预加载进显存（首图免几十秒惰性加载）。门控：引擎非 comfyui/模型未登记
// 工作流/ComfyUI 未运行/真实生成在途 → 静默跳过；预热结果只记日志，绝不弹错。
// 返回 {started, reason} 供测试断言与日志排查，前端 UI 不展示。
func (a *mediaState) WarmComfyUI() map[string]interface{} {
	if !comfyWarmArmed.Load() {
		return map[string]interface{}{"started": false, "reason": "unarmed"}
	}
	if a.cfg == nil || a.cfg.ImageBackend != "comfyui" {
		return map[string]interface{}{"started": false, "reason": "backend-not-comfyui"}
	}
	model := a.cfg.ImageModel
	if model == "" {
		model = "krea2" // 与生图路径的兜底口径一致
	}
	if !ai.ComfyUIWarmupSupported(model) {
		return map[string]interface{}{"started": false, "reason": "model-unsupported"}
	}
	if !a.isComfyUIRunning() {
		return map[string]interface{}{"started": false, "reason": "comfyui-not-running"}
	}
	a.imageGenMu.Lock()
	busy := a.imageGenRunning
	a.imageGenMu.Unlock()
	if busy {
		return map[string]interface{}{"started": false, "reason": "generation-in-flight"}
	}
	if !comfyWarmStarted.CompareAndSwap(false, true) {
		return map[string]interface{}{"started": false, "reason": "already-started"}
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("comfy-warm panic recovered", "panic", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		backend := ai.NewComfyUIBackend(a.cfg.ComfyUIURL)
		if err := backend.Warmup(ctx, model); err != nil {
			slog.Info("ComfyUI 预热未生效（静默降级）", "model", model, "error", err)
		}
	}()
	return map[string]interface{}{"started": true, "model": model}
}

// beginImageGen 进入生成区（进度/取消的 单槽 所有权）。审计 P0#14（源 AP7-01）：
// 改前无条件覆盖 imageGenCancel/imageGenRunning——并发第二个生成会偷走单槽，
// 先者的取消句柄被顶掉、进度互相串台。现拒绝并发：已在生成中时返回
// (nil, nil, 0)，调用方须判 genCtx == nil 并给用户中文 busy 回包
// （endImageGen(0, nil) 对零值幂等无害，defer 收尾写法不变）。
func (a *mediaState) beginImageGen(parent context.Context) (context.Context, context.CancelFunc, uint64) {
	a.imageGenMu.Lock()
	defer a.imageGenMu.Unlock()
	if a.imageGenRunning {
		return nil, nil, 0
	}
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
	return a.generateFreeImageProvenanced(prompt, negative, size, style, model, seed, n, lora, "imagegen", "")
}

// imageGenInternal 内部生成参数（绑定面不变：绘梦 / 原罪共用同一条生成链）。
// v4.258：把「产物归属」（sourceBoard/saveDir）与「T2 角色参考槽」
// （mode/refImages/refMethod/denoise/characterID）一并参数化，避免为参考图再抄一遍生成链。
type imageGenInternal struct {
	prompt   string
	negative string
	size     string
	style    string
	model    string
	lora     string
	seed     int
	n        int
	// 产物归属（v4.257 硬隔离刀）
	sourceBoard string
	saveDir     string
	// T2 角色参考槽（v4.258）：mode 空 = txt2img；refImages 为 data URL（本地路径
	// 由调用方先转 data URL——comfyui uploadImage 与 herdsman img2img 都只认 data URL）
	mode        string
	refImages   []string
	refMethod   string
	denoise     float64
	characterID string
	// 独立生图绑定（v4.388 原罪插图）：clientOverride 非空时本链走该客户端
	// （buildImageClientFor 按绑定后端构建，不改变全局绘梦后端）；backendOverride
	// 记录生效后端类型（进度回调/尺寸参数/自动拉起分支的判定依据），空=全局。
	clientOverride  *ai.Client
	backendOverride string
}

// generateFreeImageProvenanced 与 GenerateFreeImage 同一条生成链，把「产物
// 归属」参数化（v4.257 硬隔离刀）：
//   - sourceBoard：图像域台账的来源板块（绘梦=imagegen，原罪=sin）；
//   - saveDir：产物落盘目录覆盖（空 = 现状：ImageSaveDir 优先、其次小说 images/）。
//     原罪传自有目录（<用户配置目录>/gaea/sin/art），既不写办公工作区，也不依赖
//     办公的 ImageSaveDir 配置——原罪板块与办公板块硬隔离的落点之一。
//
// 行为等价性：saveDir="" 且 sourceBoard="imagegen" 时与改造前逐字节一致。
func (a *mediaState) generateFreeImageProvenanced(prompt string, negative string, size string, style string,
	model string, seed int, n int, lora string, sourceBoard string, saveDir string) (map[string]interface{}, error) {
	return a.generateImageInternal(imageGenInternal{
		prompt: prompt, negative: negative, size: size, style: style, model: model,
		seed: seed, n: n, lora: lora, sourceBoard: sourceBoard, saveDir: saveDir,
	})
}

// generateImageInternal 统一图片生成实现（含 T2 参考槽透传）。
func (a *mediaState) generateImageInternal(o imageGenInternal) (map[string]interface{}, error) {
	if a.client == nil {
		return map[string]interface{}{"error": "AI 客户端未初始化，请先登录"}, nil
	}
	prompt, negative, size, style, model, seed, n, lora := o.prompt, o.negative, o.size, o.style, o.model, o.seed, o.n, o.lora
	sourceBoard, saveDir := o.sourceBoard, o.saveDir
	// 独立生图绑定（v4.388）：override 客户端非空走绑定后端，否则全局客户端；
	// backendType 是本链生效后端（进度回调/尺寸参数/自动拉起分支的判定依据）。
	client := a.client
	backendType := a.cfg.ImageBackend
	if o.clientOverride != nil {
		client = o.clientOverride
		backendType = o.backendOverride
	}
	genCtx, cancel, genID := a.beginImageGen(a.ctx)
	if genCtx == nil {
		return map[string]interface{}{"error": "已有图片/视频任务正在生成，请等待完成或先取消当前任务"}, nil
	}
	defer a.endImageGen(genID, cancel)
	if backendType == "comfyui" {
		a.noteImageGenMemoryPressure()
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
	comfyBooted := false
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
			// T2 参考槽：文生图 + 参考图时由后端按 refMethod 决定是否转图生图
			Mode:      o.mode,
			RefImages: o.refImages,
			RefMethod: o.refMethod,
			Denoise:   o.denoise,
		}
		if backendType == "comfyui" {
			imgReq.ProgressCallback = a.updateComfyTaskProgress
		}

		// xAI / Ollama 后端不接受 size 参数（xAI 返回 400）；herdsman 文档明确支持
		// size；GLM 官方 schema 同样接受 size（glm-image 默认 1280x1280）
		if backendType != "comfyui" && backendType != "herdsman" && backendType != "glm" {
			imgReq.Size = ""
		}
		start := time.Now()
		resp, err := client.GenerateImage(genCtx, imgReq)
		// 孤儿 ComfyUI 实例（stderr 失效）会在执行时报 [Errno 22]：
		// 自动重启一次后重试，避免用户手动处理
		if err != nil && !comfyRecovered && backendType == "comfyui" && strings.Contains(err.Error(), "[Errno 22]") {
			slog.Warn("ComfyUI stderr 失效（疑似孤儿实例），自动重启后重试", "error", err)
			a.recoverComfyUI(genCtx)
			comfyRecovered = true
			resp, err = client.GenerateImage(genCtx, imgReq)
		}
		// ComfyUI 压根没跑（dial 连接被拒）且配置了安装路径：自动拉起+就绪
		// 等待后重试一次（本轮一次）。原罪插图与绘梦生成共用本链，此前
		// 服务未运行时直接以 connectex 原始错误失败，用户在原罪页无任何
		// 恢复入口（绘梦页才有启动按钮）。
		if err != nil && !comfyBooted && backendType == "comfyui" && strings.Contains(err.Error(), "连接 ComfyUI 失败") {
			comfyBooted = true
			if a.ensureComfyUIRunning(genCtx) {
				slog.Info("ComfyUI 未运行，已自动拉起，重试生成")
				resp, err = client.GenerateImage(genCtx, imgReq)
			}
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
		if saveDir != "" && imageData != "" {
			item.FilePath = a.saveMediaToDisk(imageData, fullPrompt, saveDir)
		} else if a.cfg.ImageSaveDir != "" && imageData != "" {
			item.FilePath = a.saveImageToDisk(imageData, fullPrompt)
		} else if imageData != "" {
			// 未配置专用目录时，自动保存到小说 images/ 目录
			item.FilePath = a.saveToNovelImages(imageData, fullPrompt)
		}
		// T0 图像域试点：落盘后登记（失败只 warn）；模式与角色 ID 如实登记，
		// 供画室按来源/角色回溯（参考槽生成标 img2img）。
		modeLabel := o.mode
		if modeLabel == "" {
			modeLabel = "txt2img"
		}
		a.recordImageHubGeneratedFor(item, modeLabel, o.characterID, sourceBoard, "")
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
	Prompt     string             `json:"prompt"`
	Negative   string             `json:"negative"`
	Size       string             `json:"size"`
	Model      string             `json:"model"`
	Seed       int                `json:"seed"`
	Lora       string             `json:"lora"`
	Count      int                `json:"count"`
	Mode       string             `json:"mode"`             // txt2img | img2img | edit | outpaint | t2v
	InitImage  string             `json:"initImage"`        // 图生图参考图 / 指令编辑原图 / 扩图原图（data URL）
	Mask       string             `json:"mask"`             // 蒙版局部重绘（阶段二刀 B）：灰度 PNG data URL，白=重绘区；仅 edit 消费
	Expand     *ai.OutpaintExpand `json:"expand,omitempty"` // 扩图四边百分比（阶段二刀 C）；仅 mode=outpaint 消费
	SourcePath string             `json:"sourcePath"`       // 编辑源图落盘路径（阶段二刀 D 变体簇）：edit/outpaint 消费，台账按 path 关联 ParentID
	Denoise    float64            `json:"denoise"`          // 重绘幅度 0-1
	Frames     int                `json:"frames"`           // 视频帧数
	FPS        int                `json:"fps"`              // 视频帧率
	// T2 角色参考槽：角色 ID + 参考图（data URL；首张作图生图种子）+ 一致性方法。
	CharacterID string   `json:"characterId"`
	RefImages   []string `json:"refImages"`
	RefMethod   string   `json:"refMethod"`
}

// GenerateMedia 多模式媒体生成：文生图 / 图生图 / 文生视频（供绘梦页使用）
func (a *mediaState) GenerateMedia(paramsJSON string) (map[string]interface{}, error) {
	if a.client == nil {
		return map[string]interface{}{"error": "AI 客户端未初始化，请先登录"}, nil
	}
	genCtx, cancel, genID := a.beginImageGen(a.ctx)
	if genCtx == nil {
		return map[string]interface{}{"error": "已有图片/视频任务正在生成，请等待完成或先取消当前任务"}, nil
	}
	defer a.endImageGen(genID, cancel)
	if a.cfg.ImageBackend == "comfyui" {
		a.noteImageGenMemoryPressure()
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
	if mode == "img2img" && a.cfg.ImageBackend != "comfyui" && a.cfg.ImageBackend != "herdsman" {
		return map[string]interface{}{"error": "图生图目前支持 ComfyUI / Herdsman 本地后端，请先在左侧切换引擎"}, nil
	}
	if mode == "img2img" && strings.TrimSpace(p.InitImage) == "" && len(p.RefImages) == 0 {
		return map[string]interface{}{"error": "图生图需要先上传参考图或选择角色参考"}, nil
	}
	// 指令编辑（阶段一刀 C）：原图必填；不加后端门——透传到后端按能力诚实
	// 报错（OpenAI 兼容=edits 端点；GLM/ComfyUI 拒绝文案见 internal/ai）。
	if mode == "edit" && strings.TrimSpace(p.InitImage) == "" {
		return map[string]interface{}{"error": "指令编辑需要原图"}, nil
	}
	// 扩图（阶段二刀 C）：合成画布+蒙版后转 edit 请求（复用 v4.393 edit+mask
	// 双后端通道）；用户 mask 与扩图蒙版是两条通道不混用（fail-closed，先于通用
	// mask 门给出定向文案）。
	var outpaintCanvas, outpaintMask string
	if mode == "outpaint" {
		if strings.TrimSpace(p.InitImage) == "" {
			return map[string]interface{}{"error": "扩图需要原图"}, nil
		}
		if p.Mask != "" {
			return map[string]interface{}{"error": "扩图不需要蒙版（扩展区由扩展量决定）"}, nil
		}
		var expand ai.OutpaintExpand
		if p.Expand != nil {
			expand = *p.Expand
		}
		canvas, mask, err := ai.ComposeOutpaint(p.InitImage, expand)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		outpaintCanvas, outpaintMask = canvas, mask
	}
	// 变体簇（阶段二刀 D）：编辑/扩图按源图落盘路径在台账同空间查 ParentID；
	// 查不到留空=诚实断链（源图非 gaea 产物/跨空间），不造假。
	variantParentID := ""
	if (mode == "edit" || mode == "outpaint") && strings.TrimSpace(p.SourcePath) != "" {
		variantParentID = imageHubAssetIDByPath(gaeaCwd(), gaeaEffectiveSpace(), p.SourcePath)
	}
	// 蒙版局部重绘（阶段二刀 B）：fail-closed——蒙版仅与指令编辑组合；
	// img2img+蒙版（纯局部重绘无指令语义）留观察池，不静默忽略。
	if p.Mask != "" && mode != "edit" {
		return map[string]interface{}{"error": "蒙版仅支持指令编辑模式（局部重绘）"}, nil
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
		// 指令编辑本地档（阶段二刀 A）：ComfyUI 编辑工作流固定走 Qwen-Image-Edit 族
		// ——请求 model 字段是生图模型名（krea2 等），不代表编辑引擎；元数据/台账
		// 如实记 qwen-image-edit（ai 层 edit 分支本就不消费该字段）。扩图（刀 C）
		// 转发同一编辑引擎，同口径。
		if mode == "edit" || mode == "outpaint" || (mode == "txt2img" && p.RefMethod == "qedit") {
			if a.cfg.ImageBackend == "comfyui" {
				imgModel = "qwen-image-edit"
			}
		}
		// 扩图转换（刀 C）：合成画布+蒙版后按 edit 请求下发——ai 层 fail-closed
		// （mask 仅 edit）天然满足；引擎层零改动。
		reqMode := mode
		reqInit := p.InitImage
		reqMask := p.Mask
		if mode == "outpaint" {
			reqMode = "edit"
			reqInit = outpaintCanvas
			reqMask = outpaintMask
		}
		imgReq := &ai.ImageGenerationRequest{
			Model:     imgModel,
			Prompt:    safePrompt,
			Negative:  p.Negative,
			N:         1,
			Size:      size,
			Seed:      genSeed,
			Lora:      p.Lora,
			Mode:      reqMode,
			InitImage: reqInit,
			Mask:      reqMask,
			Denoise:   p.Denoise,
			Frames:    p.Frames,
			FPS:       p.FPS,
			RefImages: p.RefImages,
			RefMethod: p.RefMethod,
		}
		if a.cfg.ImageBackend == "comfyui" {
			imgReq.ProgressCallback = a.updateComfyTaskProgress
		}
		// xAI / Ollama 后端不接受 size 参数（xAI 返回 400）；herdsman 文档明确支持
		// size；GLM 官方 schema 同样接受 size（glm-image 默认 1280x1280）
		if a.cfg.ImageBackend != "comfyui" && a.cfg.ImageBackend != "herdsman" && a.cfg.ImageBackend != "glm" {
			imgReq.Size = ""
		}
		// 扩图（刀 C）：画布尺寸即输出尺寸——size 会把结果强制重设，清空。
		if mode == "outpaint" {
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
				item.FilePath = a.saveToNovelImages(imageData, p.Prompt)
			}
		}
		// T0 图像域试点：多模式媒体落盘后登记（imagegen/media.generate；失败只 warn）；
		// 变体簇（刀 D）：编辑/扩图带 ParentID，登记后按路径回填 asset_id/parent_id
		// （运行态闸关闭时 lookup 空=字段空，不影响主流程）。
		a.recordImageHubGenerated(item, mode, p.CharacterID, variantParentID)
		if mode == "edit" || mode == "outpaint" {
			item.AssetID = imageHubAssetIDByPath(gaeaCwd(), gaeaEffectiveSpace(), item.FilePath)
			item.ParentID = variantParentID
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

// GetSinImageConfig 获取原罪插图独立生图后端/模型（v4.388；空 = 跟随全局
// 生图设置，即绘梦页当前后端）。
func (a *App) GetSinImageConfig() map[string]string {
	return map[string]string{
		"backend": a.cfg.SinImageBackend,
		"model":   a.cfg.SinImageModel,
	}
}

// SetSinImageConfig 设置原罪插图独立生图后端/模型（空 = 跟随全局）。
func (a *App) SetSinImageConfig(backend, model string) error {
	a.cfg.SinImageBackend = backend
	a.cfg.SinImageModel = model
	if err := config.Save(config.KeySinImageBackend, backend); err != nil {
		slog.Warn("保存原罪插图后端失败", "error", err)
		return err
	}
	if err := config.Save(config.KeySinImageModel, model); err != nil {
		slog.Warn("保存原罪插图模型失败", "error", err)
		return err
	}
	slog.Info("原罪插图生图绑定已设置", "backend", backend, "model", model)
	return nil
}

// SetImageBackend 切换图片生成后端（供设置页调用）。
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
	default:
		return fmt.Errorf("不支持的后端: %s（支持 xai / comfyui / herdsman / ollama / glm）", backend)
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

// comfyProcRefSet 登记一轮 ComfyUI 进程引用（cancel/cmd，cmd 可先为 nil 再补）。
func (a *mediaState) GetComfyUIStatus() map[string]interface{} {
	running := a.isComfyUIRunning()
	// 监控：如果进程不在运行但引用还在，自动清理
	if !running {
		a.comfyProcRefClear()
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

// GetComfyUIStatus 返回 ComfyUI 运行状态（含监控：检测到进程退出自动清理引用）
