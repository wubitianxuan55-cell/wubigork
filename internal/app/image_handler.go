package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	if a.cfg == nil || a.cfg.ImageBackend != ai.ImageBackendTypeComfyUI {
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
	if a.cfg == nil || a.cfg.ImageBackend != ai.ImageBackendTypeComfyUI || a.clientRef() == nil {
		return
	}
	ib, ok := a.clientRef().GetImageBackend().(interface{ Interrupt(context.Context) error })
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
	if a.cfg == nil || a.cfg.ImageBackend != ai.ImageBackendTypeComfyUI || a.clientRef() == nil {
		return
	}
	if ib, ok := a.clientRef().GetImageBackend().(interface{ ResetCancel() }); ok {
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

// imageGenLoopSpec 共享生成循环的链差异面（AP7-02 收敛：generateImageInternal
// 与 GenerateMedia 两份近 100 行平行循环的骨架——种子派生→模型覆盖→size 清空→
// 提交→落盘→登记——收拢为 runImageGenLoop 一份，差异全在此参数结构体；除注明
// 「能力开关」的两位外均为两链既有语义，行为逐字段冻结）。
type imageGenLoopSpec struct {
	// 提交面：client/backend 是本链生效客户端与后端（internal 链=override 解析值；
	// media 链恒全局 clientRef()+cfg.ImageBackend）。
	client  *ai.Client
	backend string
	genCtx  context.Context
	// reqTemplate 请求模板：除 Model/Seed/ProgressCallback/size 后端清空外全部
	// 由调用方定死（Prompt 传提交值 safePrompt；Size 传归一值，链内特有清空如
	// outpaint 已由调用方在模板置空）。循环每轮浅拷贝后只改 Model/Seed——改前
	// 两链本就每轮新建请求、除 Seed 外字段轮间不变，语义一致。
	reqTemplate ai.ImageGenerationRequest
	// modelOverride 调用方显式模型（空=全局 cfg.ImageModel）；modelForced 链内
	// 强制模型（media 链 edit/outpaint/qedit 在 comfyui 下强制 qwen-image-edit；
	// internal 恒空）。非空时后者覆盖前者（与改前覆盖顺序一致）。
	modelOverride string
	modelForced   string
	n             int
	seed          int
	// metaPrompt 历史/落盘/登记用的展示 prompt（internal 含风格后缀的 fullPrompt；
	// media 为原文）——与提交给后端的 safePrompt 是两个字段，两链改前即如此。
	metaPrompt string
	// size 是归一后的展示尺寸：item.Size 恒取它，与 req.Size 的后端清空无关
	//（两链改前一致：清空只影响下发请求，不回写历史元数据）。
	size string
	// 失败日志（internal：「图片生成失败」无附加字段；media：「媒体生成失败」
	// 带 mode 字段；字段顺序 mode→attempt→error 与改前一致）。
	logPrefix string
	logAttrs  []any
	// 能力开关（行为冻结，审计 AP7-02）：comfyRetries——Errno22 孤儿实例
	// recover 重试 + 未运行自动拉起重试（各一次/整轮）。批 48 起两链同开
	// （round-21 留池申报项实施：同后端同失败模式，media 链补齐同款防线）。
	comfyRetries bool
	// 能力开关：wantKind 仅 media 链开——提取 resp.Data[0].Kind（空→"image"）
	// 写 item.Kind（json kind,omitempty，internal 不置即不出键，改前一致）。
	wantKind bool
	// save 落盘钩子（imageData 非空才调）：两链分支不同——internal 有 saveDir
	// 覆盖三路（saveDir→ImageSaveDir→小说 images/）；media 两路（ImageSaveDir→
	// 小说 images/，无覆盖通道）。返回 FilePath。
	save func(imageData string) string
	// register 登记钩子（两链登记口径不同：internal=recordImageHubGeneratedFor
	// 带 sourceBoard/生效后端；media=recordImageHubGenerated+变体簇回填），返回
	//（可能回填后的）item。
	register func(item imageItem) imageItem
}

// runImageGenLoop 两链共享的生成循环（AP7-02）：种子派生→模型覆盖→size 清空→
// 提交（comfyui 防线按能力开关）→落盘→登记。返回 (items, lastErr)；空结果
// 文案/终态 map 组装仍归各链（两链终态文案与键不同，行为冻结）。
func (a *mediaState) runImageGenLoop(spec imageGenLoopSpec) ([]imageItem, string) {
	items := make([]imageItem, 0, spec.n)
	var lastErr string
	comfyRecovered := false
	comfyBooted := false
	for i := 0; i < spec.n; i++ {
		genSeed := spec.seed
		if genSeed == 0 {
			genSeed = int(time.Now().UnixNano()%1000000) + i*777
		} else if spec.n > 1 {
			// 固定种子且一次生成多张时，每张用 seed+i，避免 n 张完全雷同
			genSeed = spec.seed + i
		}
		imgModel := a.cfg.ImageModel
		if spec.modelOverride != "" {
			imgModel = spec.modelOverride
		}
		if spec.modelForced != "" {
			imgModel = spec.modelForced
		}
		imgReq := spec.reqTemplate
		imgReq.Model = imgModel
		imgReq.Seed = genSeed
		if spec.backend == ai.ImageBackendTypeComfyUI {
			imgReq.ProgressCallback = a.updateComfyTaskProgress
		}
		// xAI / Ollama 后端不接受 size 参数（xAI 返回 400）；herdsman 文档明确支持
		// size；GLM 官方 schema 同样接受 size（glm-image 默认 1280x1280）
		if spec.backend != ai.ImageBackendTypeComfyUI && spec.backend != ai.ImageBackendTypeHerdsman && spec.backend != ai.ImageBackendTypeGLM {
			imgReq.Size = ""
		}
		start := time.Now()
		resp, err := spec.client.GenerateImage(spec.genCtx, &imgReq)
		// 孤儿 ComfyUI 实例（stderr 失效）会在执行时报 [Errno 22]：
		// 自动重启一次后重试，避免用户手动处理（能力开关：仅 internal 链）
		if spec.comfyRetries && err != nil && !comfyRecovered && spec.backend == ai.ImageBackendTypeComfyUI && strings.Contains(err.Error(), "[Errno 22]") {
			slog.Warn("ComfyUI stderr 失效（疑似孤儿实例），自动重启后重试", "error", err)
			a.recoverComfyUI(spec.genCtx)
			comfyRecovered = true
			resp, err = spec.client.GenerateImage(spec.genCtx, &imgReq)
		}
		// ComfyUI 压根没跑（dial 连接被拒）且配置了安装路径：自动拉起+就绪
		// 等待后重试一次（本轮一次）。（能力开关：仅 internal 链）
		if spec.comfyRetries && err != nil && !comfyBooted && spec.backend == ai.ImageBackendTypeComfyUI && strings.Contains(err.Error(), "连接 ComfyUI 失败") {
			comfyBooted = true
			if a.ensureComfyUIRunning(spec.genCtx) {
				slog.Info("ComfyUI 未运行，已自动拉起，重试生成")
				resp, err = spec.client.GenerateImage(spec.genCtx, &imgReq)
			}
		}
		elapsed := time.Since(start).Seconds()

		if err != nil {
			slog.Warn(spec.logPrefix, append(append([]any{}, spec.logAttrs...), "attempt", i+1, "error", err)...)
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
			Prompt: spec.metaPrompt,
			Model:  imgModel,
			Size:   spec.size,
		}
		if spec.wantKind {
			kind := resp.Data[0].Kind
			if kind == "" {
				kind = "image"
			}
			item.Kind = kind
		}
		// T6-4.3：保存路径写入历史元数据（前端历史图片据此恢复本地文件）
		if imageData != "" {
			item.FilePath = spec.save(imageData)
		}
		items = append(items, spec.register(item))
	}
	return items, lastErr
}

// generateImageInternal 统一图片生成实现（含 T2 参考槽透传）。生成循环走
// runImageGenLoop（AP7-02 收敛）；本链差异面=clientOverride/后台 comfyui 防线
// （能力开关开）/saveDir 三路落盘/台账按来源板块登记。
func (a *mediaState) generateImageInternal(o imageGenInternal) (map[string]interface{}, error) {
	if a.clientRef() == nil {
		return map[string]interface{}{"error": "AI 客户端未初始化，请先登录"}, nil
	}
	prompt, negative, size, style, model, seed, n, lora := o.prompt, o.negative, o.size, o.style, o.model, o.seed, o.n, o.lora
	sourceBoard, saveDir := o.sourceBoard, o.saveDir
	// 独立生图绑定（v4.388）：override 客户端非空走绑定后端，否则全局客户端；
	// backendType 是本链生效后端（进度回调/尺寸参数/自动拉起分支的判定依据）。
	client := a.clientRef()
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
	if backendType == ai.ImageBackendTypeComfyUI {
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

	// S1.5-B play 内容护栏：image_safe_mode 提交前注入提示词安全段（后端
	// NSFW 开关位：ai 图片后端无 NSFW 透传字段，按后端能力缺省关，无法
	// 透传时仅注入 prompt 安全段）。未配置 = 零值 = 提示词原样。
	safePrompt := applyImageSafeMode(fullPrompt, playGuardrails().ImageSafeMode)

	images, lastErr := a.runImageGenLoop(imageGenLoopSpec{
		client: client, backend: backendType, genCtx: genCtx,
		reqTemplate: ai.ImageGenerationRequest{
			Prompt:   safePrompt,
			Negative: negative,
			N:        1,
			Size:     size,
			Lora:     lora,
			// T2 参考槽：文生图 + 参考图时由后端按 refMethod 决定是否转图生图
			Mode:      o.mode,
			RefImages: o.refImages,
			RefMethod: o.refMethod,
			Denoise:   o.denoise,
		},
		modelOverride: model,
		n:             n,
		seed:          seed,
		metaPrompt:    fullPrompt,
		size:          size,
		logPrefix:     "图片生成失败",
		comfyRetries:  true,
		save: func(imageData string) string {
			if saveDir != "" {
				return a.saveMediaToDisk(imageData, fullPrompt, saveDir)
			} else if a.cfg.ImageSaveDir != "" {
				return a.saveImageToDisk(imageData, fullPrompt)
			}
			// 未配置专用目录时，自动保存到小说 images/ 目录
			return a.saveToNovelImages(imageData, fullPrompt)
		},
		register: func(item imageItem) imageItem {
			// T0 图像域试点：落盘后登记（失败只 warn）；模式与角色 ID 如实登记，
			// 供画室按来源/角色回溯（参考槽生成标 img2img）。后端/模型传**生效值**
			// （审计 AP7-05）：backendType 是 override 客户端通道的解析结果，imgModel
			// 是本轮真正下发的模型名——此前登记固定取全局 ImageBackend，会把独立绑定
			// 的产物记成全局后端，消耗报表按 {model,backend} 分组即失真。
			modeLabel := o.mode
			if modeLabel == "" {
				modeLabel = "txt2img"
			}
			a.recordImageHubGeneratedFor(item, modeLabel, o.characterID, sourceBoard, backendType, "")
			return item
		},
	})

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
	if a.clientRef() == nil {
		return map[string]interface{}{"error": "AI 客户端未初始化，请先登录"}, nil
	}
	genCtx, cancel, genID := a.beginImageGen(a.ctx)
	if genCtx == nil {
		return map[string]interface{}{"error": "已有图片/视频任务正在生成，请等待完成或先取消当前任务"}, nil
	}
	defer a.endImageGen(genID, cancel)
	if a.cfg.ImageBackend == ai.ImageBackendTypeComfyUI {
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
	if mode == "t2v" && a.cfg.ImageBackend != ai.ImageBackendTypeComfyUI {
		return map[string]interface{}{"error": "文生视频目前仅支持 ComfyUI 本地后端，请先在左侧切换引擎"}, nil
	}
	if mode == "img2img" && a.cfg.ImageBackend != ai.ImageBackendTypeComfyUI && a.cfg.ImageBackend != ai.ImageBackendTypeHerdsman {
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

	// S1.5-B play 内容护栏：image_safe_mode 同 GenerateFreeImage（提交前
	// 注入提示词安全段；未配置 = 零值 = 提示词原样）。
	safePrompt := applyImageSafeMode(p.Prompt, playGuardrails().ImageSafeMode)
	results, lastErr := func() ([]imageItem, string) {
		// 指令编辑本地档（阶段二刀 A）：ComfyUI 编辑工作流固定走 Qwen-Image-Edit 族
		// ——请求 model 字段是生图模型名（krea2 等），不代表编辑引擎；元数据/台账
		// 如实记 qwen-image-edit（ai 层 edit 分支本就不消费该字段）。扩图（刀 C）
		// 转发同一编辑引擎，同口径。（收敛为 modelForced 传入共享循环）
		modelForced := ""
		if mode == "edit" || mode == "outpaint" || (mode == "txt2img" && p.RefMethod == "qedit") {
			if a.cfg.ImageBackend == ai.ImageBackendTypeComfyUI {
				modelForced = "qwen-image-edit"
			}
		}
		// 扩图转换（刀 C）：合成画布+蒙版后按 edit 请求下发——ai 层 fail-closed
		// （mask 仅 edit）天然满足；引擎层零改动。
		reqMode, reqInit, reqMask := mode, p.InitImage, p.Mask
		if mode == "outpaint" {
			reqMode, reqInit, reqMask = "edit", outpaintCanvas, outpaintMask
		}
		reqTemplate := ai.ImageGenerationRequest{
			Model:     "", // 循环内按 全局/覆盖/强制 口径填（共享循环职责）
			Prompt:    safePrompt,
			Negative:  p.Negative,
			N:         1,
			Size:      size,
			Seed:      0,
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
		// 扩图（刀 C）：画布尺寸即输出尺寸——size 会把结果强制重设，清空。
		if mode == "outpaint" {
			reqTemplate.Size = ""
		}
		// AP7-02 收敛：循环骨架（种子派生→模型覆盖→size 清空→提交→落盘→登记）
		// 与 generateImageInternal 共用 runImageGenLoop；本链差异面=无 override
		// 客户端/Kind 提取/两路落盘/台账按 mode 登记+变体簇回填。comfyui 错误
		// 重试防线批 48 补齐（round-21 留池申报项实施）：两链同后端同失败模式，
		// Errno22 孤儿实例 recover + 未运行自动拉起（各一次/整轮）同 internal 语义。
		return a.runImageGenLoop(imageGenLoopSpec{
			client: a.clientRef(), backend: a.cfg.ImageBackend, genCtx: genCtx,
			reqTemplate:   reqTemplate,
			modelOverride: p.Model,
			modelForced:   modelForced,
			n:             n,
			seed:          p.Seed,
			metaPrompt:    p.Prompt,
			size:          size,
			logPrefix:     "媒体生成失败",
			logAttrs:      []any{"mode", mode},
			comfyRetries:  true,
			wantKind:      true,
			save: func(imageData string) string {
				if a.cfg.ImageSaveDir != "" {
					return a.saveMediaToDisk(imageData, p.Prompt, a.cfg.ImageSaveDir)
				}
				return a.saveToNovelImages(imageData, p.Prompt)
			},
			register: func(item imageItem) imageItem {
				// T0 图像域试点：多模式媒体落盘后登记（imagegen/media.generate；失败只 warn）；
				// 变体簇（刀 D）：编辑/扩图带 ParentID，登记后按路径回填 asset_id/parent_id
				// （运行态闸关闭时 lookup 空=字段空，不影响主流程）。
				a.recordImageHubGenerated(item, mode, p.CharacterID, variantParentID)
				if mode == "edit" || mode == "outpaint" {
					item.AssetID = imageHubAssetIDByPath(gaeaCwd(), gaeaEffectiveSpace(), item.FilePath)
					item.ParentID = variantParentID
				}
				return item
			},
		})
	}()

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
	if a.clientRef() != nil {
		return a.clientRef().GetImageBackendType()
	}
	return ai.ImageBackendTypeXAI
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
	case a.cfg.ImageBackend == ai.ImageBackendTypeComfyUI && imageModel == "":
		imageModel = "krea2"
	case a.cfg.ImageBackend == ai.ImageBackendTypeGLM:
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
	if a.clientRef() == nil {
		return fmt.Errorf("AI 客户端未初始化")
	}
	// 设置图片保存目录
	if imageSaveDir != "" {
		a.cfg.ImageSaveDir = imageSaveDir
	}
	if a.cfg.ImageSaveDir == "" {
		a.cfg.ImageSaveDir = filepath.Join(os.Getenv("USERPROFILE"), "Pictures", "gaea")
	}
	// 构造统一走 resolveImageBackend（IN2-03 收敛）。cfg 落盘顺序保留收敛前
	// 口径：comfyui 先并 URL 再构造；herdsman/ollama/glm 先校验（引擎/Key 原文案
	// 原顺序）再落配置——校验失败不动任何配置。xai 无实例（client 内置管线）。
	var r resolvedImageBackend
	switch backend {
	case ai.ImageBackendTypeComfyUI:
		a.cfg.ImageBackend = ai.ImageBackendTypeComfyUI
		if comfyUIURL != "" {
			a.cfg.ComfyUIURL = comfyUIURL
		}
		if imageModel != "" {
			a.cfg.ImageModel = imageModel
		}
		rr, rerr := resolveImageBackend(ai.ImageBackendTypeComfyUI, a.cfg, a.engineMgr)
		if rerr != nil {
			// 收敛前此处无前置校验（URL 空也照常构造，生成时才暴露）；现按
			// 其余四份副本的并集口径 fail-fast（行为差异见 IN2-03 报告）。
			return fmt.Errorf("未配置 ComfyUI 地址")
		}
		r = rr
	case ai.ImageBackendTypeXAI:
		a.cfg.ImageBackend = ai.ImageBackendTypeXAI
		a.cfg.ImageModel = "grok-imagine-image-quality" // 角色剧照默认高质量模型
		r = resolvedImageBackend{Kind: ai.ImageBackendTypeXAI}
	case ai.ImageBackendTypeHerdsman, ai.ImageBackendTypeOllama, ai.ImageBackendTypeGLM:
		rr, rerr := resolveImageBackend(backend, a.cfg, a.engineMgr)
		if rerr != nil {
			var re *imageBackendResolveError
			if errors.As(rerr, &re) {
				switch re.Reason {
				case reasonEngineDisabled:
					return fmt.Errorf("%s 引擎未启用，请先在模型中心启用", imageEngineDisplayName(re.Backend))
				case reasonGLMKeyMissing:
					return fmt.Errorf("GLM API Key 未配置，请先在模型中心 GLM 卡片保存 Key（open.bigmodel.cn 获取）")
				}
			}
			return rerr
		}
		r = rr
		a.cfg.ImageBackend = backend
		if imageModel != "" {
			a.cfg.ImageModel = imageModel
		}
	default:
		return fmt.Errorf("不支持的后端: %s（支持 xai / comfyui / herdsman / ollama / glm）", backend)
	}
	a.clientRef().SetImageBackend(r.Backend, r.Kind)

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
		backend = ai.ImageBackendTypeXAI
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
	case ai.ImageBackendTypeComfyUI:
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
	case ai.ImageBackendTypeXAI:
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

// GetComfyUILoras 返回 ComfyUI 当前可用的 LoRA 列表（绘梦 LoRA 多选动态加载）。
// 构造统一走 resolveImageBackend（IN2-03 收敛：app 层最后一个手写 ComfyUI 构造点）；
// comfyui 只会因地址缺失失败，文案保留收敛前口径。
func (a *mediaState) GetComfyUILoras() ([]string, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	r, err := resolveImageBackend(ai.ImageBackendTypeComfyUI, a.cfg, a.engineMgr)
	if err != nil {
		return nil, fmt.Errorf("ComfyUI 地址未配置")
	}
	cb, ok := r.Backend.(*ai.ComfyUIBackend)
	if !ok {
		return nil, fmt.Errorf("ComfyUI 后端类型异常: %T", r.Backend)
	}
	return cb.ListLoras(ctx)
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
