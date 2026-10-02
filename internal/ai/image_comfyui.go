package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"image"
	"io"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ComfyUIBackend 通过 ComfyUI REST API 调用本地 Flux / Z-Image-Turbo 模型
type ComfyUIBackend struct {
	baseURL    string
	httpClient *http.Client // 可注入（测试用 httptest.Server 客户端替换），默认 30 分钟超时

	// 轮询间隔（测试可缩短；生产保持 2 秒）
	pollInterval time.Duration

	// 生成等待上限（测试可缩短；生产 30 分钟）——v4.404.1 由 15 分钟放宽：
	// 实录 krea2 img2img 在 iGPU 共享内存被换出后冷载 20.5 分钟（19:32:54 提交
	// 19:53:25 出图），15 分钟固定上限先到误报失败。
	genTimeout time.Duration

	// 本地取消标记（T6-4.1）：ComfyUI 无删除排队任务的 API，取消后拒绝新提交
	mu              sync.Mutex
	cancelled       bool
	currentPromptID string // 最近一次提交的任务 ID（诊断）
}

// NewComfyUIBackend 创建 ComfyUI 后端
func NewComfyUIBackend(baseURL string) *ComfyUIBackend {
	return &ComfyUIBackend{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		httpClient:   netclient.NewSimpleClient(30 * time.Minute), // CPU 模式可能很慢
		pollInterval: 2 * time.Second,
		genTimeout:   30 * time.Minute,
	}
}

// init 自注册：ComfyUI 后端经注册表提供（kind = ImageBackendKindComfyUI）。
func init() {
	RegisterImageBackend(ImageBackendKindComfyUI, func(cfg ImageBackendConfig) (ImageBackend, error) {
		if strings.TrimSpace(cfg.BaseURL) == "" {
			return nil, fmt.Errorf("ai: comfyui image backend requires base_url")
		}
		return NewComfyUIBackend(cfg.BaseURL), nil
	})
}

// Interrupt 中断 ComfyUI 当前正在执行的任务（POST /interrupt），并置位本地取消标记。
//
// ComfyUI 限制说明：ComfyUI 没有「删除排队任务」的 API——/queue 仅能查询
// 排队/运行中的任务，无法删除。因此取消采用「本地取消标记 + /interrupt 当前任务」：
//   - 置位 cancelled：GenerateImage 入口检测到后直接拒绝新提交（等价于删除排队项）；
//   - POST /interrupt：中断当前正在执行的采样任务（/interrupt 不接收 prompt_id，
//     中断的是当前执行中的任务）。
//
// 幂等：重复调用无害（ComfyUI 无任务时 /interrupt 返回 200），置位是幂等操作。
func (b *ComfyUIBackend) Interrupt(ctx context.Context) error {
	b.mu.Lock()
	b.cancelled = true
	b.mu.Unlock()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/interrupt", nil)
	if err != nil {
		return err
	}
	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ComfyUI /interrupt 失败 (%s): %w", b.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ComfyUI /interrupt HTTP %d: %s", resp.StatusCode, trimStr(string(body), 200))
	}
	return nil
}

// ResetCancel 清除本地取消标记（新一轮生成开始时由上层调用）。
func (b *ComfyUIBackend) ResetCancel() {
	b.mu.Lock()
	b.cancelled = false
	b.mu.Unlock()
}

// isCancelled 返回本地取消标记是否已置位。
func (b *ComfyUIBackend) isCancelled() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cancelled
}

// ListLoras 返回 ComfyUI 当前可用的 LoRA 名称列表（models/loras 下相对路径，含子目录）。
// 通过 object_info 获取 LoraLoaderModelOnly / LoraLoader 节点的 lora_name 可选值，
// 避免前端硬编码文件名与本地 models/loras 不一致导致提交 400。
func (b *ComfyUIBackend) ListLoras(ctx context.Context) ([]string, error) {
	for _, nodeType := range []string{"LoraLoaderModelOnly", "LoraLoader"} {
		names, err := b.listLorasForNode(ctx, nodeType)
		if err == nil {
			sort.Strings(names)
			return names, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// 节点类型不存在（404 / 空响应）时尝试下一种
	}
	return nil, fmt.Errorf("ComfyUI 未提供 LoRA 列表（object_info 中找不到 LoraLoader 节点）")
}

// listLorasForNode 查询单个节点类型的 lora_name 可选值。
func (b *ComfyUIBackend) listLorasForNode(ctx context.Context, nodeType string) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/object_info/"+nodeType, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("连接 ComfyUI 失败 (%s): %w", b.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ComfyUI HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info map[string]struct {
		Input struct {
			Required map[string]json.RawMessage `json:"required"`
		} `json:"input"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("解析 object_info 失败: %w", err)
	}
	node, ok := info[nodeType]
	if !ok {
		return nil, fmt.Errorf("object_info 缺少节点 %s", nodeType)
	}
	raw, ok := node.Input.Required["lora_name"]
	if !ok {
		return nil, fmt.Errorf("节点 %s 缺少 lora_name 输入", nodeType)
	}

	names, err := parseLoraNames(raw)
	if err != nil {
		return nil, err
	}
	return names, nil
}

// parseLoraNames 解析 object_info 中 lora_name 可选值，兼容多种版本结构：
//
//	A. [["file1.safetensors", ...]]                      （仅一层包装）
//	B. ["LORAS", ["file1.safetensors", ...]]             （标准 ComfyUI）
//	C. ["LORAS", {"file1.safetensors": {...}}]           （对象映射）
func parseLoraNames(raw json.RawMessage) ([]string, error) {
	var pair []json.RawMessage
	if err := json.Unmarshal(raw, &pair); err != nil {
		return nil, fmt.Errorf("lora_name 结构异常: %s", trimStr(string(raw), 120))
	}
	var names []string
	for _, item := range pair {
		var list []string
		if err := json.Unmarshal(item, &list); err == nil {
			names = append(names, list...)
			continue
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(item, &m); err == nil {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys) // map 迭代顺序随机：排序保证 LoRA 列表顺序稳定（E01/C03 flaky）
			names = append(names, keys...)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("lora_name 列表为空: %s", trimStr(string(raw), 120))
	}
	return names, nil
}

// GenerateImage 通过 ComfyUI 生成图片 / 图生图 / 文生视频
func (b *ComfyUIBackend) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	// T6-4.1 本地取消标记：取消后拒绝新提交（ComfyUI 无删除排队任务 API）
	if b.isCancelled() {
		return nil, fmt.Errorf("生成已取消，请重新发起生成")
	}
	defer b.clearCurrentPromptID()

	// 解析尺寸（T6-4.4：Sscanf 改为严格解析 + 64–2048 钳制，非法输入返回中文错误）
	width, height := 1024, 1024
	if req.Size != "" {
		w, h, err := parseSize(req.Size)
		if err != nil {
			return nil, err
		}
		width, height = w, h
	}

	seed := req.Seed
	if seed == 0 {
		seed = rand.Intn(1 << 31)
	}
	mode := req.Mode
	if mode == "" {
		mode = "txt2img"
	}
	// T2 参考槽：img2img 近似（v0 行为）/ qedit 参考编辑（阶段三刀 A）。
	// qedit：参考图由 edit 分支上传（三图槽），此处只路由不改 InitImage。
	if len(req.RefImages) > 0 {
		nm, err := comfyResolveRefMode(mode, req.RefMethod)
		if err != nil {
			return nil, err
		}
		mode = nm
		if mode == "qedit" {
			if req.Mask != "" {
				return nil, fmt.Errorf("参考编辑（qedit）不支持蒙版——蒙版仅与指令编辑组合")
			}
			if len(req.RefImages) > 3 {
				return nil, fmt.Errorf("Qwen 参考编辑最多 3 张参考图（image1..3），当前 %d 张", len(req.RefImages))
			}
		} else if req.InitImage == "" {
			req.InitImage = req.RefImages[0]
		}
	}

	// 指令编辑本地档（阶段二刀 A）：Qwen-Image-Edit 2511 官方模板蒸馏——
	// 原图上 → FluxKontextImageScale 重标 → 语义编辑（非图生图整幅重绘）。
	// 阶段二刀 B：Mask 非空时走蒙版局部重绘（白=重绘区）——SetLatentNoiseMask
	// 限制重采样区域，蒙版与缩放图同尺寸对齐（kontextScaleSize）。
	// 阶段三刀 A：qedit 参考编辑共用本链——参考图 1..3 进 image1..3 槽
	//（TextEncodeQwenImageEditPlus 三图参考，人物一致性正路）。
	var editImages []string
	var editMaskName string
	editTargetW, editTargetH := 0, 0
	if mode == "edit" {
		if req.InitImage == "" {
			return nil, fmt.Errorf("指令编辑需要原图（data URL）")
		}
		name, err := b.uploadImage(ctx, req.InitImage)
		if err != nil {
			return nil, err
		}
		editImages = []string{name}
		if req.Mask != "" {
			imgBytes, _, err := decodeDataURLBytes(req.InitImage)
			if err != nil {
				return nil, fmt.Errorf("蒙版编辑需要可解析的原图尺寸: %w", err)
			}
			cfg, _, err := image.DecodeConfig(bytes.NewReader(imgBytes))
			if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
				return nil, fmt.Errorf("蒙版编辑需要可解析的原图尺寸（decode 失败）")
			}
			editTargetW, editTargetH = kontextScaleSize(cfg.Width, cfg.Height)
			maskName, err := b.uploadImage(ctx, req.Mask)
			if err != nil {
				return nil, err
			}
			editMaskName = maskName
		}
	}
	if mode == "qedit" {
		if len(req.RefImages) == 0 {
			return nil, fmt.Errorf("参考编辑（qedit）需要至少 1 张参考图")
		}
		for _, ref := range req.RefImages {
			name, err := b.uploadImage(ctx, ref)
			if err != nil {
				return nil, err
			}
			editImages = append(editImages, name)
		}
	}

	// 解析 LoRA 列表
	var loras []string
	if req.Lora != "" {
		for _, l := range strings.Split(req.Lora, ",") {
			l = strings.TrimSpace(l)
			if l != "" {
				loras = append(loras, l)
			}
		}
	}

	var workflow map[string]interface{}
	kind := "image"
	switch mode {
	case "edit", "qedit":
		// 指令编辑/参考编辑本地档：单编辑族 Qwen-Image-Edit 2511（req.Model 是
		// 生图模型名，不代表编辑引擎——app 层已把元数据如实改写 qwen-image-edit）。
		workflow = b.buildQwenImageEditWorkflow(req.Prompt, seed, editImages, editMaskName, editTargetW, editTargetH)
	case "img2img":
		// 图生图：上传参考图 → LoadImage + VAEEncode → 低 denoise 重绘
		if req.InitImage == "" {
			return nil, fmt.Errorf("图生图需要提供参考图")
		}
		imageName, err := b.uploadImage(ctx, req.InitImage)
		if err != nil {
			return nil, err
		}
		denoise := req.Denoise
		if denoise <= 0 || denoise > 1 {
			denoise = 0.65
		}
		switch {
		case req.Model == "z-image-turbo":
			workflow = b.buildZImageImg2ImgWorkflow(req.Prompt, req.Negative, width, height, seed, 8, loras, imageName, denoise)
		case req.Model == "krea2" || strings.HasPrefix(req.Model, "krea2"):
			workflow = b.buildKreaImg2ImgWorkflow(req.Prompt, req.Negative, width, height, seed, 8, loras, imageName, denoise)
		default:
			// T6-4.2：禁止静默降级——flux 等未实现图生图流程的模型直接报错
			return nil, fmt.Errorf("模型 %s 暂不支持图生图（支持 krea2 / z-image-turbo）", req.Model)
		}
	case "t2v":
		// 文生视频：LTX-Video 工作流（输出 SaveAnimatedWEBP 动画）
		if req.Size == "" {
			width, height = 768, 512
		}
		frames := req.Frames
		if frames <= 0 {
			frames = 97
		}
		fps := req.FPS
		if fps <= 0 {
			fps = 8
		}
		workflow = b.buildLTXVideoWorkflow(req.Prompt, req.Negative, width, height, seed, frames, fps, req.Model)
		kind = "video"
	default:
		// 文生图（默认）：模型 → 工作流显式映射表（T6-4.2），未知模型返回中文错误
		builder, ok := lookupTxt2imgBuilder(req.Model)
		if !ok {
			return nil, fmt.Errorf("不支持的模型: %s（ComfyUI 支持 krea2 / z-image-turbo / flux）", req.Model)
		}
		workflow = builder(b, req.Prompt, req.Negative, width, height, seed, 8, loras)
	}

	// 1. 提交任务
	promptID, err := b.queuePrompt(ctx, workflow)
	if err != nil {
		if mode == "edit" {
			return nil, fmt.Errorf("ComfyUI 提交失败（指令编辑本地档）: %w%s", err, qwenEditMissingModelHint(err.Error()))
		}
		return nil, fmt.Errorf("ComfyUI 提交失败: %w", err)
	}
	b.mu.Lock()
	b.currentPromptID = promptID
	b.mu.Unlock()
	slog.Info("ComfyUI 任务已提交", "promptID", promptID, "size", fmt.Sprintf("%dx%d", width, height))

	// 节点 id → class_type 映射（WebSocket progress_state 只给节点 id，用于展示当前节点）
	nodeClasses := make(map[string]string)
	for id, n := range workflow {
		if nm, ok := n.(map[string]interface{}); ok {
			if ct, ok := nm["class_type"].(string); ok {
				nodeClasses[id] = ct
			}
		}
	}

	// 2. 轮询等待完成
	imageData, outKind, err := b.waitForResult(ctx, promptID, req, nodeClasses)
	if err != nil {
		return nil, fmt.Errorf("ComfyUI 生成失败: %w", err)
	}
	if outKind != "" {
		kind = outKind
	}

	return &ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data: []ImageData{
			{B64JSON: imageData, Kind: kind},
		},
	}, nil
}

// parseSize 解析 "宽x高" 尺寸（T6-4.4）：
//   - 格式非法 / 非数字 → 返回中文错误（不再静默回退 1024）
//   - 数值钳制到 64–2048（超上限压缩、低于下限抬高）
func parseSize(size string) (int, int, error) {
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("尺寸格式无效: %q（应为 宽x高，如 1024x1024）", size)
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("尺寸格式无效: %q（宽度不是数字）", size)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("尺寸格式无效: %q（高度不是数字）", size)
	}
	clamp := func(v int) int {
		if v < 64 {
			return 64
		}
		if v > 2048 {
			return 2048
		}
		return v
	}
	return clamp(w), clamp(h), nil
}

// txt2imgWorkflowBuilder 文生图工作流构建器（模型 → 工作流显式映射）。
// steps 由调用方给定：真实生成为 8，Warmup 空跑取 1（同一构建器保证
// 预热加载的就是下次真实生成要用的模型文件）。
type txt2imgWorkflowBuilder func(b *ComfyUIBackend, prompt, negative string, width, height, seed, steps int, loras []string) map[string]interface{}

// txt2imgWorkflows 文生图模型 → 工作流映射表（T6-4.2 名实相符）。
// 新增模型必须在此登记，未登记的模型在 GenerateImage 中直接返回中文错误，
// 禁止静默降级到其他模型。
var txt2imgWorkflows = map[string]txt2imgWorkflowBuilder{
	"krea2": func(b *ComfyUIBackend, prompt, negative string, width, height, seed, steps int, loras []string) map[string]interface{} {
		return b.buildKreaWorkflow(prompt, width, height, seed, steps, loras)
	},
	"z-image-turbo": func(b *ComfyUIBackend, prompt, negative string, width, height, seed, steps int, loras []string) map[string]interface{} {
		return b.buildZImageWorkflow(prompt, width, height, seed, steps, loras)
	},
	"flux": func(b *ComfyUIBackend, prompt, negative string, width, height, seed, steps int, loras []string) map[string]interface{} {
		return b.buildFluxWorkflow(prompt, width, height, seed, loras)
	},
}

// lookupTxt2imgBuilder 按模型名查找文生图工作流构建器。
// krea2 系列（krea2-*）兼容历史前缀匹配；其余模型必须精确命中白名单。
func lookupTxt2imgBuilder(model string) (txt2imgWorkflowBuilder, bool) {
	if b, ok := txt2imgWorkflows[model]; ok {
		return b, true
	}
	if strings.HasPrefix(model, "krea2") {
		return txt2imgWorkflows["krea2"], true
	}
	return nil, false
}

// ComfyUIWarmupSupported 报告模型是否可预热（有无登记的工作流构建器）。
// 供 App 层在发起预热前做零成本闸（未知模型静默跳过，不打扰用户）。
func ComfyUIWarmupSupported(model string) bool {
	_, ok := lookupTxt2imgBuilder(model)
	return ok
}

// Warmup 提交一次极小空跑（64×64、1 步）把目标模型的 UNET/CLIP/VAE 预加载
// 进显存——CU1（蒸馏 unsloth §六-2）：惰性加载是「首图几十秒」体感的根源，
// 绘梦页首入时后台预热把「惰性首次」变「提前完成」。产出图直接丢弃。
// 复用 txt2imgWorkflows 登记的同一构建器（lookupTxt2imgBuilder 分发，步数取 1）
// 保证预热加载的就是下次真实生成要用的模型文件；空跑算力开销可忽略，
// 加载才是目的。与真实生成共用 ComfyUI 队列：预热排队中用户提交真任务时，
// 真任务只是排在极小空跑后面，天然串行无冲突。
func (b *ComfyUIBackend) Warmup(ctx context.Context, model string) error {
	if model == "" {
		model = "krea2"
	}
	builder, ok := lookupTxt2imgBuilder(model)
	if !ok {
		return fmt.Errorf("模型 %s 未登记预热工作流", model)
	}
	workflow := builder(b, "(warmup)", "", 64, 64, 0, 1, nil)
	promptID, err := b.queuePrompt(ctx, workflow)
	if err != nil {
		return fmt.Errorf("预热任务提交失败: %w", err)
	}
	// 等空跑完成=模型确已进显存；产物丢弃（64×64 下载开销可忽略）。
	// 进度回调 nil：预热静默，不向前端推进度。
	_, _, err = b.waitForResult(ctx, promptID, &ImageGenerationRequest{Model: model}, nil)
	if err != nil {
		return fmt.Errorf("预热任务未完成: %w", err)
	}
	slog.Info("ComfyUI 预热完成", "model", model)
	return nil
}

// buildFluxWorkflow 构建 FLUX.1-schnell 工作流（官方 ComfyUI 模板）：
// UNETLoader(flux1-schnell) + DualCLIPLoader(type=flux, T5+CLIP-L) + VAELoader(ae)
// → EmptySD3LatentImage → KSampler(cfg=1.0, euler/simple, 4 步) → VAEDecode → SaveImage。
//
// 注意：Flux 官方模板不使用负面提示词（负面 CLIPTextEncode 用空文本）；
// 模型文件 flux1-schnell.safetensors 需放在 ComfyUI models/unet 或 models/diffusion_models。
// uploadImage 将 base64 data URL 参考图上传到 ComfyUI /upload/image，返回文件名

func (b *ComfyUIBackend) uploadImage(ctx context.Context, dataURL string) (string, error) {
	commaIdx := strings.Index(dataURL, ",")
	if commaIdx < 0 {
		return "", fmt.Errorf("参考图 data URL 无效")
	}
	raw, err := base64.StdEncoding.DecodeString(dataURL[commaIdx+1:])
	if err != nil {
		return "", fmt.Errorf("参考图解码失败: %w", err)
	}
	ext := "png"
	switch {
	case strings.HasPrefix(dataURL, "data:image/jpeg"), strings.HasPrefix(dataURL, "data:image/jpg"):
		ext = "jpg"
	case strings.HasPrefix(dataURL, "data:image/webp"):
		ext = "webp"
	case strings.HasPrefix(dataURL, "data:image/gif"):
		ext = "gif"
	}
	filename := fmt.Sprintf("gaea_init_%d.%s", time.Now().UnixNano(), ext)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("image", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(raw); err != nil {
		return "", err
	}
	_ = mw.WriteField("overwrite", "true")
	_ = mw.Close()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/upload/image", &buf)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("上传参考图失败 (%s): %w", b.baseURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("上传参考图失败 HTTP %d: %s", resp.StatusCode, trimStr(string(body), 300))
	}
	var res struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &res); err != nil || res.Name == "" {
		return "", fmt.Errorf("上传参考图响应异常: %s", trimStr(string(body), 200))
	}
	return res.Name, nil
}

func (b *ComfyUIBackend) queuePrompt(ctx context.Context, workflow map[string]interface{}) (string, error) {
	body := map[string]interface{}{
		"prompt": workflow,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/prompt", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("连接 ComfyUI 失败 (%s): %w", b.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 ComfyUI 响应失败: %w", err)
	}
	if resp.StatusCode != 200 {
		errMsg := trimStr(string(respBody), 500)
		extra := ""
		if strings.Contains(errMsg, "value_not_in_list") {
			extra = "\n💡 提交的模型/LoRA 不在 ComfyUI 列表中：请在绘梦页重新选择 LoRA（列表已与本地 ComfyUI 同步），或确认 ComfyUI models 目录包含所选文件"
		} else if strings.Contains(errMsg, "ZImagePowerNodes") {
			extra = "\n💡 请安装 ComfyUI 插件: ZImagePowerNodes\n   cd custom_nodes && git clone https://github.com/martin-rizzo/ComfyUI-ZImagePowerNodes.git"
		} else if strings.Contains(errMsg, "UnetLoaderGGUF") || strings.Contains(errMsg, "CLIPLoaderGGUF") {
			extra = "\n💡 请安装 ComfyUI 插件: ComfyUI-GGUF\n   cd custom_nodes && git clone https://github.com/city96/ComfyUI-GGUF.git"
		} else if strings.Contains(errMsg, "missing_node_type") {
			extra = "\n💡 工作流使用了自定义节点，请确认已安装所需插件（ComfyUI-GGUF + ZImagePowerNodes）"
		}
		return "", fmt.Errorf("ComfyUI HTTP %d: %s%s", resp.StatusCode, errMsg, extra)
	}

	var result struct {
		PromptID string `json:"prompt_id"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析 ComfyUI 响应失败: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("ComfyUI 错误: %s", result.Error)
	}
	return result.PromptID, nil
}

// comfyOutputFile ComfyUI 输出文件（图片或视频）
type comfyOutputFile struct {
	filename   string
	subfolder  string
	outputType string
	kind       string // image | video
	format     string
}

// waitForResult 轮询等待 ComfyUI 生成完成，返回 base64 data URL 与输出类型。
// nodeClasses 用于把 WebSocket 进度消息里的节点 id 映射为 class_type 展示名。
func (b *ComfyUIBackend) waitForResult(ctx context.Context, promptID string, req *ImageGenerationRequest, nodeClasses map[string]string) (string, string, error) {
	ticker := time.NewTicker(b.pollInterval)
	defer ticker.Stop()

	// WebSocket 实时进度（尽力而为：连接失败静默，不影响历史轮询主路径）
	if req.ProgressCallback != nil {
		pollCtx, pollCancel := context.WithCancel(ctx)
		defer pollCancel()
		go b.pollComfyProgress(pollCtx, promptID, nodeClasses, func(status string, elapsed int, percent int, node string) {
			req.ProgressCallback(status, elapsed, percent, node)
		})
	}

	timeout := time.After(b.genTimeout)
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-timeout:
			// 超时前最后一搏：服务端可能刚完成（模型冷载尾差），收割产物不浪费
			return b.harvestOnTimeout(ctx, promptID, time.Since(start))
		case <-ticker.C:
			if req.ProgressCallback != nil {
				// percent=-1 / node=""：真实进度由 ws 回调推送，这里只保底刷新 elapsed
				req.ProgressCallback("running", int(time.Since(start).Seconds()), -1, "")
			}
			dataURL, kind, done, err := b.collectResult(ctx, promptID)
			if err != nil {
				// T6-4.1：取消后轮询即刻退出（checkHistory 携带 ctx）
				if ctx.Err() != nil {
					return "", "", ctx.Err()
				}
				if done {
					// done 时的 kind 保持原样返回（下载失败路径与历史返回形状一致）
					return "", kind, err
				}
				slog.Warn("ComfyUI 轮询失败", "error", err)
				continue
			}
			if done {
				return dataURL, kind, nil
			}
		}
	}
}

// comfyResolveRefMode 解析参考槽的一致性方法（阶段三刀 A 起）：
//   - ""/"img2img" → 图生图近似（v0 行为不动）；
//   - "qedit" → Qwen-Image-Edit 参考编辑（阶段三刀 A）：txt2img+参考 → 编辑
//     引擎多图参考槽（TextEncodeQwenImageEditPlus image1..3），人物一致性正路
//     （Qwen 架构族无 IP-Adapter 生态）；
//   - "ipadapter"/"pulid" → 诚实拒绝并改口指向 qedit（路线修订：Qwen 架构族
//     无此生态，不再「排期中」）。
func comfyResolveRefMode(mode, refMethod string) (string, error) {
	if mode == "img2img" {
		return mode, nil
	}
	switch refMethod {
	case "", "img2img":
		return "img2img", nil
	case "qedit":
		return "qedit", nil
	case "ipadapter", "pulid":
		return mode, fmt.Errorf("一致性方法 %s 不适用于当前模型族（krea2/z-image 为 Qwen-Image 架构，无 IP-Adapter 生态）；人物一致请用 qedit（Qwen 参考编辑）或 img2img 近似", refMethod)
	default:
		return mode, fmt.Errorf("未知一致性方法: %s", refMethod)
	}
}

// collectResult 收割一次任务状态并取回产物（AP7-10：正常轮询与超时收割
// 共用的「checkHistory + downloadFile」——原来两处各写一遍，改一处漏一处）。
// 返回 (dataURL, kind, done, err)：
//   - done=false          → 任务未完成（err 为轮询失败原因，可为 nil）
//   - done=true, err=nil  → dataURL/kind 为已下载产物
//   - done=true, err!=nil → 执行错误 / 完成但无输出文件 / 下载失败
//     （下载失败时 kind 仍带回首文件类型，与历史返回形状一致）
func (b *ComfyUIBackend) collectResult(ctx context.Context, promptID string) (string, string, bool, error) {
	files, done, err := b.checkHistory(ctx, promptID)
	if err != nil {
		return "", "", done, err
	}
	if !done {
		return "", "", false, nil
	}
	if len(files) == 0 {
		return "", "", true, fmt.Errorf("ComfyUI 完成但无输出文件")
	}
	dataURL, derr := b.downloadFile(ctx, files[0])
	if derr != nil {
		return "", files[0].kind, true, derr
	}
	return dataURL, files[0].kind, true, nil
}

// harvestOnTimeout 超时前最后一搏：再查一次历史，服务端已完成则收割产物
// （v4.404.1——服务端已做的工作不因客户端上限白等浪费）；否则如实报超时。
func (b *ComfyUIBackend) harvestOnTimeout(ctx context.Context, promptID string, waited time.Duration) (string, string, error) {
	dataURL, kind, done, err := b.collectResult(ctx, promptID)
	if err == nil && done {
		slog.Info("ComfyUI 生成超时前收割到已完成产物", "promptID", promptID, "waited", waited.String())
		return dataURL, kind, nil
	}
	return "", "", fmt.Errorf("ComfyUI 生成超时 (%d分钟)", int(b.genTimeout.Minutes()))
}

// pollComfyProgress 订阅 ComfyUI /ws 实时进度（尽力而为，失败静默）。
// 兼容两种消息：新版 progress_state（nodes{id:{value,max,state}}，节点名经 nodeClasses 映射）
// 与旧版 progress（data.value/max/node，node 即 class_type）。
func (b *ComfyUIBackend) pollComfyProgress(ctx context.Context, promptID string, nodeClasses map[string]string, cb func(status string, elapsed int, percent int, node string)) {
	u, err := url.Parse(b.baseURL)
	if err != nil {
		return
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/ws"
	q := u.Query()
	q.Set("clientId", strconv.FormatInt(rand.Int63(), 10))
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	start := time.Now()
	readErr := make(chan error, 1)
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				readErr <- err
				return
			}
			var ev struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if json.Unmarshal(msg, &ev) != nil {
				continue
			}
			switch ev.Type {
			case "status": // 队列深度（v4.405）：前有任务时让「排队可见」而不是干等
				var d struct {
					Status struct {
						ExecInfo struct {
							QueueRemaining int `json:"queue_remaining"`
						} `json:"exec_info"`
					} `json:"status"`
				}
				if json.Unmarshal(ev.Data, &d) != nil {
					continue
				}
				if d.Status.ExecInfo.QueueRemaining > 1 {
					cb("queued", int(time.Since(start).Seconds()), -1, "queue")
				}
			case "executing": // 节点开始执行（v4.405）：载入模型等无百分比阶段也可见
				var d struct {
					Node interface{} `json:"node"`
					ID   string      `json:"prompt_id"`
				}
				if json.Unmarshal(ev.Data, &d) != nil {
					continue
				}
				if d.ID != "" && d.ID != promptID {
					continue
				}
				node, ok := d.Node.(string)
				if !ok || node == "" || node == "null" { // null=整单结束，历史轮询接管
					continue
				}
				if ct, ok := nodeClasses[node]; ok {
					node = ct
				}
				cb("running", int(time.Since(start).Seconds()), -1, node)
			case "progress": // node 字段可能是节点 id（新版）或 class_type（旧版）
				var d struct {
					Value float64 `json:"value"`
					Max   float64 `json:"max"`
					Node  string  `json:"node"`
					ID    string  `json:"prompt_id"`
				}
				if json.Unmarshal(ev.Data, &d) != nil || d.Max <= 0 {
					continue
				}
				if d.ID != "" && d.ID != promptID {
					continue
				}
				node := d.Node
				if ct, ok := nodeClasses[node]; ok {
					node = ct
				}
				cb("running", int(time.Since(start).Seconds()), clampPercent(d.Value/d.Max), node)
			case "progress_state": // 新版 ComfyUI：nodes 按节点 id 上报
				var d struct {
					ID    string `json:"prompt_id"`
					Nodes map[string]struct {
						Value  float64 `json:"value"`
						Max    float64 `json:"max"`
						State  string  `json:"state"`
						NodeID string  `json:"node_id"`
					} `json:"nodes"`
				}
				if json.Unmarshal(ev.Data, &d) != nil {
					continue
				}
				if d.ID != "" && d.ID != promptID {
					continue
				}
				// 先找带进度的运行节点；没有则退而求其次报「正在执行某节点」——
				// 载入模型阶段 value/max 恒 0，不能因此整段不可见（v4.405）
				var runningID string
				var runningNoMax bool
				for id, n := range d.Nodes {
					if n.State != "running" {
						continue
					}
					if n.Max > 0 {
						runningID = id
						runningNoMax = false
						break
					}
					if runningID == "" {
						runningID = id
						runningNoMax = true
					}
				}
				if runningID != "" {
					if runningNoMax {
						cb("running", int(time.Since(start).Seconds()), -1, nodeClasses[runningID])
					} else {
						cb("running", int(time.Since(start).Seconds()), clampPercent(d.Nodes[runningID].Value/d.Nodes[runningID].Max), nodeClasses[runningID])
					}
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case <-readErr:
	}
}

// clampPercent 把 0-1 比例收敛到 0-100 整数百分比。
func clampPercent(f float64) int {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 100
	}
	return int(f * 100)
}

// checkHistory 查询任务状态，返回 (输出文件列表, 是否完成, 错误)。
// 携带 ctx（T6-4.1）：取消后请求即刻失败，轮询立即退出。
func (b *ComfyUIBackend) checkHistory(ctx context.Context, promptID string) ([]comfyOutputFile, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/history/"+promptID, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("读取 ComfyUI history 失败: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, false, fmt.Errorf("ComfyUI history HTTP %d: %s", resp.StatusCode, trimStr(string(body), 300))
	}

	var history map[string]interface{}
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, false, err
	}

	entry, ok := history[promptID]
	if !ok {
		return nil, false, nil // 还没完成
	}

	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		return nil, true, nil
	}

	// 检测执行错误
	if status, ok := entryMap["status"].(map[string]interface{}); ok {
		if statusStr, _ := status["status_str"].(string); statusStr == "error" {
			// 提取错误消息
			errMsg := "ComfyUI 执行错误"
			if msgs, ok := status["messages"].([]interface{}); ok {
				for _, m := range msgs {
					if msgArr, ok := m.([]interface{}); ok && len(msgArr) >= 2 {
						if msgType, _ := msgArr[0].(string); msgType == "execution_error" {
							if details, ok := msgArr[1].(map[string]interface{}); ok {
								if em, _ := details["exception_message"].(string); em != "" {
									errMsg = errMsg + ": " + strings.TrimSpace(em)
								}
							}
						}
					}
				}
			}
			return nil, true, fmt.Errorf("%s%s", errMsg, comfyExecutionHint(errMsg))
		}
	}

	outputs, ok := entryMap["outputs"].(map[string]interface{})
	if !ok {
		return nil, true, nil
	}

	// 遍历输出节点，收集 images / gifs / videos
	var files []comfyOutputFile
	for _, output := range outputs {
		outputMap, ok := output.(map[string]interface{})
		if !ok {
			continue
		}
		for _, key := range []string{"images", "gifs", "videos"} {
			items, ok := outputMap[key].([]interface{})
			if !ok {
				continue
			}
			for _, item := range items {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				fn, _ := itemMap["filename"].(string)
				if fn == "" {
					continue
				}
				f := comfyOutputFile{
					filename:   fn,
					subfolder:  strOr(itemMap["subfolder"]),
					outputType: strOr(itemMap["type"]),
					format:     strOr(itemMap["format"]),
					kind:       "image",
				}
				if f.outputType == "" {
					f.outputType = "output"
				}
				if key == "videos" {
					f.kind = "video"
				}
				files = append(files, f)
			}
		}
	}

	return files, true, nil
}

// comfyExecutionHint 针对常见 ComfyUI 执行错误追加可操作的中文提示。
// 已知环境故障（T6-4.2 同类风格）：
//   - comfy-kitchen 版本与 ComfyUI 源码不匹配（rms_rope ABI 错误、fp8 布局缺失）
//     → 提示用户按 ComfyUI 官方指引更新 Python 依赖；
//   - 模型/LoRA 不在列表中 → 提示重新选择 LoRA / 检查 models 目录。
func comfyExecutionHint(errMsg string) string {
	switch {
	case strings.Contains(errMsg, "rms_rope"),
		strings.Contains(errMsg, "AsymW4A8Int8Layout"),
		strings.Contains(errMsg, "comfy_kitchen"),
		strings.Contains(errMsg, "comfy-kitchen"),
		strings.Contains(errMsg, "'NoneType' object has no attribute 'Params'"):
		return "\n💡 ComfyUI 依赖与代码版本不匹配（comfy-kitchen 过旧/损坏，fp8 与加速内核不可用）。请在 ComfyUI 安装目录运行: python -m pip install -r requirements.txt，然后重启 ComfyUI"
	case strings.Contains(errMsg, "value_not_in_list"):
		return "\n💡 提交的模型/LoRA 不在 ComfyUI 列表中：请在绘梦页重新选择 LoRA（列表已与本地 ComfyUI 同步），或确认 ComfyUI models 目录包含所选文件"
	default:
		return ""
	}
}

func strOr(v interface{}) string {
	s, _ := v.(string)
	return s
}

// downloadFile 从 ComfyUI 下载输出文件并返回 base64 data URL
func (b *ComfyUIBackend) downloadFile(ctx context.Context, f comfyOutputFile) (string, error) {
	url := fmt.Sprintf("%s/view?filename=%s&subfolder=%s&type=%s",
		b.baseURL, url.QueryEscape(f.filename), url.QueryEscape(f.subfolder), f.outputType)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载输出文件失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("下载输出文件失败 HTTP %d: %s", resp.StatusCode, trimStr(string(data), 300))
	}

	mimeType := "image/png"
	switch f.format {
	case "webp":
		mimeType = "image/webp"
	case "gif":
		mimeType = "image/gif"
	case "jpeg", "jpg":
		mimeType = "image/jpeg"
	case "mp4":
		mimeType = "video/mp4"
	case "webm":
		mimeType = "video/webm"
	case "mov":
		mimeType = "video/quicktime"
	default:
		if ct := resp.Header.Get("Content-Type"); ct != "" {
			mimeType = ct
		}
	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
