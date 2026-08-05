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
	"net/http"
	"sort"
	"strings"
	"time"
)

// ComfyUIBackend 通过 ComfyUI REST API 调用本地 Flux / Z-Image-Turbo 模型
type ComfyUIBackend struct {
	baseURL    string
	httpClient *http.Client
}

// NewComfyUIBackend 创建 ComfyUI 后端
func NewComfyUIBackend(baseURL string) *ComfyUIBackend {
	return &ComfyUIBackend{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: netclient.NewSimpleClient(30 * time.Minute), // CPU 模式可能很慢
	}
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
//   A. [["file1.safetensors", ...]]                      （仅一层包装）
//   B. ["LORAS", ["file1.safetensors", ...]]             （标准 ComfyUI）
//   C. ["LORAS", {"file1.safetensors": {...}}]           （对象映射）
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
			for k := range m {
				names = append(names, k)
			}
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("lora_name 列表为空: %s", trimStr(string(raw), 120))
	}
	return names, nil
}

// GenerateImage 通过 ComfyUI 生成图片
func (b *ComfyUIBackend) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	// 解析尺寸
	width, height := 1024, 1024
	if req.Size != "" {
		parts := strings.Split(req.Size, "x")
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &width)
			fmt.Sscanf(parts[1], "%d", &height)
		}
	}

	seed := req.Seed
	if seed == 0 {
		seed = rand.Intn(1 << 31)
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
	switch {
	case strings.HasPrefix(req.Model, "krea2"):
		steps := 8
		workflow = b.buildKreaWorkflow(req.Prompt, width, height, seed, steps, loras)
	case req.Model == "z-image-turbo":
		steps := 8
		unetModel := "z_image_turbo_bf16_完整版_效果最好.safetensors"
		workflow = b.buildZImageWorkflow(req.Prompt, req.Negative, width, height, seed, steps, unetModel, loras)
	default:
		// 默认走 Krea2 Turbo
		slog.Info("ComfyUI 默认使用 Krea2 Turbo", "model", req.Model)
		steps := 8
		workflow = b.buildKreaWorkflow(req.Prompt, width, height, seed, steps, loras)
	}

	// 1. 提交任务
	promptID, err := b.queuePrompt(ctx, workflow)
	if err != nil {
		return nil, fmt.Errorf("ComfyUI 提交失败: %w", err)
	}
	slog.Info("ComfyUI 任务已提交", "promptID", promptID, "size", fmt.Sprintf("%dx%d", width, height))

	// 2. 轮询等待完成
	imageData, err := b.waitForResult(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("ComfyUI 生成失败: %w", err)
	}

	return &ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data: []ImageData{
			{B64JSON: imageData},
		},
	}, nil
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

// waitForResult 轮询等待 ComfyUI 生成完成，返回 base64 图片
func (b *ComfyUIBackend) waitForResult(ctx context.Context, promptID string) (string, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("ComfyUI 生成超时 (10分钟)")
		case <-ticker.C:
			images, done, err := b.checkHistory(promptID)
			if err != nil {
				if done {
					return "", err
				}
				slog.Warn("ComfyUI 轮询失败", "error", err)
				continue
			}
			if done {
				if len(images) == 0 {
					return "", fmt.Errorf("ComfyUI 完成但无输出图片")
				}
				// 下载第一张图片并返回 base64
				return b.downloadImage(ctx, images[0])
			}
		}
	}
}

// checkHistory 查询任务状态，返回 (图片文件名列表, 是否完成, 错误)
func (b *ComfyUIBackend) checkHistory(promptID string) ([]string, bool, error) {
	resp, err := b.httpClient.Get(b.baseURL + "/history/" + promptID)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("读取 ComfyUI history 失败: %w", err)
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
			return nil, true, fmt.Errorf("%s", errMsg)
		}
	}

	outputs, ok := entryMap["outputs"].(map[string]interface{})
	if !ok {
		return nil, true, nil
	}

	// 遍历输出节点找图片
	var imageFiles []string
	for _, output := range outputs {
		outputMap, ok := output.(map[string]interface{})
		if !ok {
			continue
		}
		imgs, ok := outputMap["images"].([]interface{})
		if !ok {
			continue
		}
		for _, img := range imgs {
			imgMap, ok := img.(map[string]interface{})
			if !ok {
				continue
			}
			if fn, ok := imgMap["filename"].(string); ok {
				imageFiles = append(imageFiles, fn)
			}
		}
	}

	return imageFiles, true, nil
}

// downloadImage 从 ComfyUI 下载图片并返回 base64 data URL
func (b *ComfyUIBackend) downloadImage(ctx context.Context, filename string) (string, error) {
	url := fmt.Sprintf("%s/view?filename=%s&subfolder=&type=output", b.baseURL, filename)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载图片失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	mimeType := "image/png"
	if resp.Header.Get("Content-Type") != "" {
		mimeType = resp.Header.Get("Content-Type")
	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
