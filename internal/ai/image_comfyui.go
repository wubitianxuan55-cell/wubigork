package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
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
			unetModel := "z_image_turbo_bf16_完整版_效果最好.safetensors"
			workflow = b.buildZImageImg2ImgWorkflow(req.Prompt, req.Negative, width, height, seed, 8, unetModel, loras, imageName, denoise)
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
		workflow = builder(b, req.Prompt, req.Negative, width, height, seed, loras)
	}

	// 1. 提交任务
	promptID, err := b.queuePrompt(ctx, workflow)
	if err != nil {
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
type txt2imgWorkflowBuilder func(b *ComfyUIBackend, prompt, negative string, width, height, seed int, loras []string) map[string]interface{}

// txt2imgWorkflows 文生图模型 → 工作流映射表（T6-4.2 名实相符）。
// 新增模型必须在此登记，未登记的模型在 GenerateImage 中直接返回中文错误，
// 禁止静默降级到其他模型。
var txt2imgWorkflows = map[string]txt2imgWorkflowBuilder{
	"krea2": func(b *ComfyUIBackend, prompt, negative string, width, height, seed int, loras []string) map[string]interface{} {
		return b.buildKreaWorkflow(prompt, width, height, seed, 8, loras)
	},
	"z-image-turbo": func(b *ComfyUIBackend, prompt, negative string, width, height, seed int, loras []string) map[string]interface{} {
		return b.buildZImageWorkflow(prompt, negative, width, height, seed, 8, "z_image_turbo_bf16_完整版_效果最好.safetensors", loras)
	},
	"flux": func(b *ComfyUIBackend, prompt, negative string, width, height, seed int, loras []string) map[string]interface{} {
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

// buildFluxWorkflow 构建 FLUX.1-schnell 工作流（官方 ComfyUI 模板）：
// UNETLoader(flux1-schnell) + DualCLIPLoader(type=flux, T5+CLIP-L) + VAELoader(ae)
// → EmptySD3LatentImage → KSampler(cfg=1.0, euler/simple, 4 步) → VAEDecode → SaveImage。
//
// 注意：Flux 官方模板不使用负面提示词（负面 CLIPTextEncode 用空文本）；
// 模型文件 flux1-schnell.safetensors 需放在 ComfyUI models/unet 或 models/diffusion_models。
func (b *ComfyUIBackend) buildFluxWorkflow(prompt string, width, height, seed int, loras []string) map[string]interface{} {
	wf := map[string]interface{}{
		"4": map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": "flux1-schnell.safetensors", "weight_dtype": "default"}},
		"5": map[string]interface{}{"class_type": "DualCLIPLoader", "inputs": map[string]interface{}{"clip_name1": "t5xxl_fp8_e4m3fn.safetensors", "clip_name2": "clip_l.safetensors", "type": "flux"}},
		"6": map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": "ae.safetensors"}},
		"7": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"5", 0}}},
		"8": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": "", "clip": []interface{}{"5", 0}}},
		"9": map[string]interface{}{"class_type": "EmptySD3LatentImage", "inputs": map[string]interface{}{"width": width, "height": height, "batch_size": 1}},
	}
	modelSourceID := injectLoraNodes(wf, "4", loras)
	wf["10"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
		"seed": seed, "steps": 4, "cfg": 1.0, "sampler_name": "euler", "scheduler": "simple", "denoise": 1.0,
		"model": []interface{}{modelSourceID, 0}, "positive": []interface{}{"7", 0}, "negative": []interface{}{"8", 0}, "latent_image": []interface{}{"9", 0},
	}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// clearCurrentPromptID 清空当前任务 ID（GenerateImage 结束时调用）。
func (b *ComfyUIBackend) clearCurrentPromptID() {
	b.mu.Lock()
	b.currentPromptID = ""
	b.mu.Unlock()
}

// injectLoraNodes 在工作流中注入 LoraLoaderModelOnly 节点链
// 返回最后一个节点的 ID（UNETLoader 或最后一个 LoRA 节点）
// loras 为空时直接返回 originalModelNodeID
func injectLoraNodes(workflow map[string]interface{}, originalModelNodeID string, loras []string) string {
	if len(loras) == 0 {
		return originalModelNodeID
	}
	currentNodeID := originalModelNodeID
	for i, loraPath := range loras {
		nodeID := fmt.Sprintf("%d", 20+i)
		workflow[nodeID] = map[string]interface{}{
			"class_type": "LoraLoaderModelOnly",
			"inputs": map[string]interface{}{
				"model":          []interface{}{currentNodeID, 0},
				"lora_name":      loraPath,
				"strength_model": 1.0,
			},
		}
		currentNodeID = nodeID
	}
	return currentNodeID
}

// buildZImageWorkflow 构建 Z-Image-Turbo 工作流（官方 Comfy-Org 模板）
// CLIPLoader lumina2, SD3Latent, AuraFlow shift=3, ConditioningZeroOut, CFG 1.0, res_multistep/simple
func (b *ComfyUIBackend) buildZImageWorkflow(prompt string, negative string, width int, height int, seed int, steps int, unetModel string, loras []string) map[string]interface{} {
	if steps <= 0 {
		steps = 8
	}
	if steps > 20 {
		steps = 20
	}
	wf := map[string]interface{}{
		"4":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": unetModel, "weight_dtype": "default"}},
		"5":  map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": "z-image\\qwen_3_4b.safetensors", "type": "lumina2"}},
		"6":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": "z-image-qwen.safetensors"}},
		"7":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"5", 0}}},
		"8":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": "", "clip": []interface{}{"5", 0}}},
		"9":  map[string]interface{}{"class_type": "EmptySD3LatentImage", "inputs": map[string]interface{}{"width": width, "height": height, "batch_size": 1}},
		"13": map[string]interface{}{"class_type": "ConditioningZeroOut", "inputs": map[string]interface{}{"conditioning": []interface{}{"8", 0}}},
	}
	modelSourceID := injectLoraNodes(wf, "4", loras)
	wf["14"] = map[string]interface{}{"class_type": "ModelSamplingAuraFlow", "inputs": map[string]interface{}{"model": []interface{}{modelSourceID, 0}, "shift": 3}}
	wf["10"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
		"seed": seed, "steps": steps, "cfg": 1.0, "sampler_name": "res_multistep", "scheduler": "simple", "denoise": 1.0,
		"model": []interface{}{"14", 0}, "positive": []interface{}{"7", 0}, "negative": []interface{}{"13", 0}, "latent_image": []interface{}{"9", 0},
	}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// buildKreaWorkflow 构建 Krea2 Turbo 工作流（官方 Comfy-Org 模板）
// CLIPLoader krea2, EmptyLatentImage, 无 AuraFlow, CFG 1.0, euler/simple, 8步
// 模型: UNET=krea2_turbo_fp8_scaled, CLIP=qwen3vl_4b_fp8_scaled, VAE=qwen_image_vae
func (b *ComfyUIBackend) buildKreaWorkflow(prompt string, width, height, seed, steps int, loras []string) map[string]interface{} {
	wf := map[string]interface{}{
		"4":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": "krea2_turbo_fp8_scaled.safetensors", "weight_dtype": "default"}},
		"5":  map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": "qwen3vl_4b_fp8_scaled.safetensors", "type": "krea2"}},
		"6":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": "qwen_image_vae.safetensors"}},
		"7":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"5", 0}}},
		"8":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": "", "clip": []interface{}{"5", 0}}},
		"9":  map[string]interface{}{"class_type": "EmptyLatentImage", "inputs": map[string]interface{}{"width": width, "height": height, "batch_size": 1}},
		"13": map[string]interface{}{"class_type": "ConditioningZeroOut", "inputs": map[string]interface{}{"conditioning": []interface{}{"8", 0}}},
	}
	// LoRA 注入（如果有）
	modelSourceID := injectLoraNodes(wf, "4", loras)
	wf["10"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{"seed": seed, "steps": steps, "cfg": 1.0, "sampler_name": "euler", "scheduler": "simple", "denoise": 1.0, "model": []interface{}{modelSourceID, 0}, "positive": []interface{}{"7", 0}, "negative": []interface{}{"13", 0}, "latent_image": []interface{}{"9", 0}}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// buildKreaImg2ImgWorkflow 构建 Krea2 Turbo 图生图工作流：
// LoadImage(参考图) → VAEEncode → KSampler(低 denoise) → VAEDecode → SaveImage
func (b *ComfyUIBackend) buildKreaImg2ImgWorkflow(prompt, negative string, width, height, seed, steps int, loras []string, imageName string, denoise float64) map[string]interface{} {
	wf := map[string]interface{}{
		"4":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": "krea2_turbo_fp8_scaled.safetensors", "weight_dtype": "default"}},
		"5":  map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": "qwen3vl_4b_fp8_scaled.safetensors", "type": "krea2"}},
		"6":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": "qwen_image_vae.safetensors"}},
		"7":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"5", 0}}},
		"8":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": negative, "clip": []interface{}{"5", 0}}},
		"13": map[string]interface{}{"class_type": "ConditioningZeroOut", "inputs": map[string]interface{}{"conditioning": []interface{}{"8", 0}}},
		"1":  map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": imageName}},
		"15": map[string]interface{}{"class_type": "VAEEncode", "inputs": map[string]interface{}{"pixels": []interface{}{"1", 0}, "vae": []interface{}{"6", 0}}},
	}
	modelSourceID := injectLoraNodes(wf, "4", loras)
	wf["10"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
		"seed": seed, "steps": steps, "cfg": 1.0, "sampler_name": "euler", "scheduler": "simple", "denoise": denoise,
		"model": []interface{}{modelSourceID, 0}, "positive": []interface{}{"7", 0}, "negative": []interface{}{"13", 0}, "latent_image": []interface{}{"15", 0},
	}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// buildZImageImg2ImgWorkflow 构建 Z-Image-Turbo 图生图工作流
func (b *ComfyUIBackend) buildZImageImg2ImgWorkflow(prompt, negative string, width, height, seed, steps int, unetModel string, loras []string, imageName string, denoise float64) map[string]interface{} {
	wf := map[string]interface{}{
		"4":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": unetModel, "weight_dtype": "default"}},
		"5":  map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": "z-image\\qwen_3_4b.safetensors", "type": "lumina2"}},
		"6":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": "z-image-qwen.safetensors"}},
		"7":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"5", 0}}},
		"8":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": negative, "clip": []interface{}{"5", 0}}},
		"13": map[string]interface{}{"class_type": "ConditioningZeroOut", "inputs": map[string]interface{}{"conditioning": []interface{}{"8", 0}}},
		"1":  map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": imageName}},
		"15": map[string]interface{}{"class_type": "VAEEncode", "inputs": map[string]interface{}{"pixels": []interface{}{"1", 0}, "vae": []interface{}{"6", 0}}},
	}
	modelSourceID := injectLoraNodes(wf, "4", loras)
	wf["14"] = map[string]interface{}{"class_type": "ModelSamplingAuraFlow", "inputs": map[string]interface{}{"model": []interface{}{modelSourceID, 0}, "shift": 3}}
	wf["10"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
		"seed": seed, "steps": steps, "cfg": 1.0, "sampler_name": "res_multistep", "scheduler": "simple", "denoise": denoise,
		"model": []interface{}{"14", 0}, "positive": []interface{}{"7", 0}, "negative": []interface{}{"13", 0}, "latent_image": []interface{}{"15", 0},
	}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// buildLTXVideoWorkflow 构建 LTX-Video 文生视频工作流（ComfyUI ≥0.30 节点组）：
// CheckpointLoaderSimple + CLIPLoader(ltxv) → CLIPTextEncode ×2 → LTXVConditioning
// → EmptyLTXVLatentVideo + LTXVScheduler + KSamplerSelect → SamplerCustom
// → VAEDecode → CreateVideo → SaveVideo
//
// 注意：ComfyUI 0.30+ 移除了旧版 LTXVLoader/LTXVSampler/SaveAnimatedWEBP 组合，
// 官方模板改用上述节点组（LTXVLoader/LTXVSampler 会以 missing_node_type 报错）。
func (b *ComfyUIBackend) buildLTXVideoWorkflow(prompt, negative string, width, height, seed, frames, fps int, model string) map[string]interface{} {
	ckpt := strings.TrimSpace(model)
	if ckpt == "" {
		ckpt = "ltx-video-2b-v0.9.safetensors"
	}
	if frames < 16 {
		frames = 16
	}
	frames = frames / 8 * 8 // LTX 潜空间帧数需为 8 的倍数
	if fps <= 0 {
		fps = 8
	}
	wf := map[string]interface{}{
		"1": map[string]interface{}{"class_type": "CheckpointLoaderSimple", "inputs": map[string]interface{}{"ckpt_name": ckpt}},
		"2": map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": "t5xxl_fp16.safetensors", "type": "ltxv"}},
		"3": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"2", 0}}},
		"4": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": negative, "clip": []interface{}{"2", 0}}},
		"5": map[string]interface{}{"class_type": "LTXVConditioning", "inputs": map[string]interface{}{"positive": []interface{}{"3", 0}, "negative": []interface{}{"4", 0}, "frame_rate": 25}},
		"6": map[string]interface{}{"class_type": "EmptyLTXVLatentVideo", "inputs": map[string]interface{}{"width": width, "height": height, "length": frames, "batch_size": 1}},
		"7": map[string]interface{}{"class_type": "LTXVScheduler", "inputs": map[string]interface{}{"steps": 30, "max_shift": 2.05, "base_shift": 0.95, "stretch": true, "terminal": 0.1}},
		"8": map[string]interface{}{"class_type": "KSamplerSelect", "inputs": map[string]interface{}{"sampler_name": "euler"}},
		"9": map[string]interface{}{"class_type": "SamplerCustom", "inputs": map[string]interface{}{
			"model": []interface{}{"1", 0}, "add_noise": true, "noise_seed": seed, "cfg": 2.0,
			"positive": []interface{}{"5", 0}, "negative": []interface{}{"5", 1},
			"sampler": []interface{}{"8", 0}, "sigmas": []interface{}{"7", 0}, "latent_image": []interface{}{"6", 0},
		}},
		"10": map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"9", 0}, "vae": []interface{}{"1", 2}}},
		"11": map[string]interface{}{"class_type": "CreateVideo", "inputs": map[string]interface{}{"images": []interface{}{"10", 0}, "fps": fps}},
		"12": map[string]interface{}{"class_type": "SaveVideo", "inputs": map[string]interface{}{"video": []interface{}{"11", 0}, "filename_prefix": "gaea", "format": "auto", "codec": "auto"}},
	}
	return wf
}

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

	timeout := time.After(15 * time.Minute)
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-timeout:
			return "", "", fmt.Errorf("ComfyUI 生成超时 (15分钟)")
		case <-ticker.C:
			if req.ProgressCallback != nil {
				// percent=-1 / node=""：真实进度由 ws 回调推送，这里只保底刷新 elapsed
				req.ProgressCallback("running", int(time.Since(start).Seconds()), -1, "")
			}
			files, done, err := b.checkHistory(ctx, promptID)
			if err != nil {
				// T6-4.1：取消后轮询即刻退出（checkHistory 携带 ctx）
				if ctx.Err() != nil {
					return "", "", ctx.Err()
				}
				if done {
					return "", "", err
				}
				slog.Warn("ComfyUI 轮询失败", "error", err)
				continue
			}
			if done {
				if len(files) == 0 {
					return "", "", fmt.Errorf("ComfyUI 完成但无输出文件")
				}
				dataURL, err := b.downloadFile(ctx, files[0])
				return dataURL, files[0].kind, err
			}
		}
	}
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
	defer conn.Close()

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
						Value float64 `json:"value"`
						Max   float64 `json:"max"`
						State string  `json:"state"`
						NodeID string  `json:"node_id"`
					} `json:"nodes"`
				}
				if json.Unmarshal(ev.Data, &d) != nil {
					continue
				}
				if d.ID != "" && d.ID != promptID {
					continue
				}
				for id, n := range d.Nodes {
					if n.State == "running" && n.Max > 0 {
						cb("running", int(time.Since(start).Seconds()), clampPercent(n.Value/n.Max), nodeClasses[id])
						break
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
