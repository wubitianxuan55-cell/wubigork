package ai

// image_comfyui_workflows.go — ComfyUI 工作流构建族（P5 破防专项拆自
// image_comfyui.go：各模型 txt2img/img2img/edit/video 的工作流拼装与 LoRA 注入）。

import (
	"fmt"
	"strings"
)

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

// comfyQwenProfile krea2 / z-image-turbo 两族工作流的单点差异表（AP7-03）。
// 两族节点拓扑逐节点相同：UNET → [LoRA 链] → [AuraFlow] → KSampler →
// VAEDecode → SaveImage，负向走 ConditioningZeroOut；逐字段差异只有
// 权重名 / CLIP type / 空 latent 类型 / 采样器 / AuraFlow shift / 步数钳制——
// 原来四份节点表两两复制（txt2img/img2img × krea2/z-image），改一处漏一处。
type comfyQwenProfile struct {
	unetName     string // UNETLoader.unet_name
	clipName     string // CLIPLoader.clip_name
	clipType     string // CLIPLoader.type
	vaeName      string // VAELoader.vae_name
	latentClass  string // txt2img 空 latent 节点类（EmptyLatentImage / EmptySD3LatentImage）
	samplerName  string // KSampler.sampler_name
	auraShift    int    // >0 时接 ModelSamplingAuraFlow(shift)；0 不接（krea2 不需要）
	defaultSteps int    // txt2img steps<=0 兜底步数（仅 clampSteps 族生效）
	clampSteps   bool   // txt2img 步数钳制（<=0 兜底、>20 收 20）——z-image 历史行为，krea2 透传
}

var comfyQwenProfiles = map[string]comfyQwenProfile{
	// Krea2 Turbo（官方 Comfy-Org 模板）：CLIPLoader krea2, EmptyLatentImage,
	// 无 AuraFlow, CFG 1.0, euler/simple。VAE 用 qwen_image_vae（用 ae 会出灰紫图）。
	"krea2": {
		unetName:     "krea2_turbo_fp8_scaled.safetensors",
		clipName:     "qwen3vl_4b_fp8_scaled.safetensors",
		clipType:     "krea2",
		vaeName:      "qwen_image_vae.safetensors",
		latentClass:  "EmptyLatentImage",
		samplerName:  "euler",
		defaultSteps: 8,
	},
	// Z-Image-Turbo（官方 Comfy-Org 模板）：CLIPLoader lumina2, SD3Latent,
	// AuraFlow shift=3, CFG 1.0, res_multistep/simple。
	"z-image-turbo": {
		unetName:     "z_image_turbo_bf16_完整版_效果最好.safetensors",
		clipName:     "z-image\\qwen_3_4b.safetensors",
		clipType:     "lumina2",
		vaeName:      "z-image-qwen.safetensors",
		latentClass:  "EmptySD3LatentImage",
		samplerName:  "res_multistep",
		auraShift:    3,
		defaultSteps: 8,
		clampSteps:   true,
	},
}

// buildQwenProfileTxt2Img krea2/z-image 族共享 txt2img 节点表（AP7-03 收口）。
// 负面词 txt2img 恒空文本（历史行为：KSampler 负向接 ConditioningZeroOut，
// 传入的 negative 参数不进工作流）。
func (b *ComfyUIBackend) buildQwenProfileTxt2Img(p comfyQwenProfile, prompt string, width, height, seed, steps int, loras []string) map[string]interface{} {
	if p.clampSteps {
		if steps <= 0 {
			steps = p.defaultSteps
		}
		if steps > 20 {
			steps = 20
		}
	}
	wf := map[string]interface{}{
		"4":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": p.unetName, "weight_dtype": "default"}},
		"5":  map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": p.clipName, "type": p.clipType}},
		"6":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": p.vaeName}},
		"7":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"5", 0}}},
		"8":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": "", "clip": []interface{}{"5", 0}}},
		"9":  map[string]interface{}{"class_type": p.latentClass, "inputs": map[string]interface{}{"width": width, "height": height, "batch_size": 1}},
		"13": map[string]interface{}{"class_type": "ConditioningZeroOut", "inputs": map[string]interface{}{"conditioning": []interface{}{"8", 0}}},
	}
	modelSourceID := injectLoraNodes(wf, "4", loras)
	modelInput := []interface{}{modelSourceID, 0}
	if p.auraShift > 0 {
		wf["14"] = map[string]interface{}{"class_type": "ModelSamplingAuraFlow", "inputs": map[string]interface{}{"model": []interface{}{modelSourceID, 0}, "shift": p.auraShift}}
		modelInput = []interface{}{"14", 0}
	}
	wf["10"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
		"seed": seed, "steps": steps, "cfg": 1.0, "sampler_name": p.samplerName, "scheduler": "simple", "denoise": 1.0,
		"model": modelInput, "positive": []interface{}{"7", 0}, "negative": []interface{}{"13", 0}, "latent_image": []interface{}{"9", 0},
	}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// buildQwenProfileImg2Img krea2/z-image 族共享 img2img 节点表（AP7-03 收口）：
// LoadImage(参考图) → VAEEncode → KSampler(低 denoise) → VAEDecode → SaveImage。
func (b *ComfyUIBackend) buildQwenProfileImg2Img(p comfyQwenProfile, prompt, negative string, width, height, seed, steps int, loras []string, imageName string, denoise float64) map[string]interface{} {
	wf := map[string]interface{}{
		"4":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": p.unetName, "weight_dtype": "default"}},
		"5":  map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": p.clipName, "type": p.clipType}},
		"6":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": p.vaeName}},
		"7":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"5", 0}}},
		"8":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": negative, "clip": []interface{}{"5", 0}}},
		"13": map[string]interface{}{"class_type": "ConditioningZeroOut", "inputs": map[string]interface{}{"conditioning": []interface{}{"8", 0}}},
		"1":  map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": imageName}},
		"15": map[string]interface{}{"class_type": "VAEEncode", "inputs": map[string]interface{}{"pixels": []interface{}{"1", 0}, "vae": []interface{}{"6", 0}}},
	}
	modelSourceID := injectLoraNodes(wf, "4", loras)
	modelInput := []interface{}{modelSourceID, 0}
	if p.auraShift > 0 {
		wf["14"] = map[string]interface{}{"class_type": "ModelSamplingAuraFlow", "inputs": map[string]interface{}{"model": []interface{}{modelSourceID, 0}, "shift": p.auraShift}}
		modelInput = []interface{}{"14", 0}
	}
	wf["10"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
		"seed": seed, "steps": steps, "cfg": 1.0, "sampler_name": p.samplerName, "scheduler": "simple", "denoise": denoise,
		"model": modelInput, "positive": []interface{}{"7", 0}, "negative": []interface{}{"13", 0}, "latent_image": []interface{}{"15", 0},
	}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// buildZImageWorkflow 构建 Z-Image-Turbo 工作流（官方 Comfy-Org 模板）：
// 差异字段见 comfyQwenProfiles["z-image-turbo"]。
func (b *ComfyUIBackend) buildZImageWorkflow(prompt string, width int, height int, seed int, steps int, loras []string) map[string]interface{} {
	return b.buildQwenProfileTxt2Img(comfyQwenProfiles["z-image-turbo"], prompt, width, height, seed, steps, loras)
}

// buildKreaWorkflow 构建 Krea2 Turbo 工作流（官方 Comfy-Org 模板）：
// 差异字段见 comfyQwenProfiles["krea2"]。
func (b *ComfyUIBackend) buildKreaWorkflow(prompt string, width, height, seed, steps int, loras []string) map[string]interface{} {
	return b.buildQwenProfileTxt2Img(comfyQwenProfiles["krea2"], prompt, width, height, seed, steps, loras)
}

// buildKreaImg2ImgWorkflow 构建 Krea2 Turbo 图生图工作流。
func (b *ComfyUIBackend) buildKreaImg2ImgWorkflow(prompt, negative string, width, height, seed, steps int, loras []string, imageName string, denoise float64) map[string]interface{} {
	return b.buildQwenProfileImg2Img(comfyQwenProfiles["krea2"], prompt, negative, width, height, seed, steps, loras, imageName, denoise)
}

// buildZImageImg2ImgWorkflow 构建 Z-Image-Turbo 图生图工作流。
func (b *ComfyUIBackend) buildZImageImg2ImgWorkflow(prompt, negative string, width, height, seed, steps int, loras []string, imageName string, denoise float64) map[string]interface{} {
	return b.buildQwenProfileImg2Img(comfyQwenProfiles["z-image-turbo"], prompt, negative, width, height, seed, steps, loras, imageName, denoise)
}

// Qwen-Image-2.1-Turbo 官方模板文件名（comfyui-workflow-templates
// image_qwen_image_2_1_turbo_t2i 蒸馏，2026-10-16 实读；权重本机
// ComfyUI-Shared 共享目录已就位）。
const (
	qwen21UNET = "qwen_image_2.1_turbo_int8_convrot.safetensors"
	qwen21CLIP = "qwen3vl_8b_int8_convrot.safetensors"
	qwen21VAE  = "qwen_image_2.1_vae_bf16.safetensors"
)

// qwen21Sigmas 官方 turbo 档手写 σ 表（ManualSigmas，9 值 = 8 步）——
// turbo 权重按该日程蒸馏，步数不开放（同 flux 固定 4 步的先例）。
const qwen21Sigmas = "1.0, 0.978453, 0.95418, 0.926626, 0.89508, 0.845148, 0.704534, 0.414568, 0"

const (
	Qwen21EditModel = "qwen-image-2.1-edit"
	qwen21EditUNET  = qwen21UNET
)

// buildQwen21EditWorkflow reconstructs the official
// image_qwen_image_2_1_turbo_image_edit API graph. image_1 is the edit target; extra
// images are references. The prompt enhancer subgraph is intentionally absent
// (official template default: refine_prompt=false), so no extra PE weight or
// extra generation pass is required.
func (b *ComfyUIBackend) buildQwen21EditWorkflow(prompt, negative string, images []string, seed int, loras []string) map[string]interface{} {
	wf := map[string]interface{}{
		"4": map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": qwen21EditUNET, "weight_dtype": "default"}},
		"5": map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": qwen21CLIP, "type": "qwen_image", "device": "default"}},
		"6": map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": qwen21VAE}},
	}
	for i, name := range images {
		loadID := fmt.Sprintf("1%d", i+1)
		wf[loadID] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": name}}
	}
	encInputs := map[string]interface{}{
		"clip": []interface{}{"5", 0}, "vae": []interface{}{"6", 0},
		"prompt": prompt, "negative_prompt": negative, "resolution": 1024,
	}
	for i := range images {
		encInputs[fmt.Sprintf("images.image_%d", i+1)] = []interface{}{fmt.Sprintf("1%d", i+1), 0}
	}
	wf["7"] = map[string]interface{}{"class_type": "TextEncodeQwenImage21", "inputs": encInputs}
	modelSource := injectLoraNodes(wf, "4", loras)
	wf["40"] = map[string]interface{}{"class_type": "QwenImage21Cache", "inputs": map[string]interface{}{
		"model": []interface{}{modelSource, 0}, "device": "auto", "dtype": "default",
	}}
	wf["9"] = map[string]interface{}{"class_type": "EmptyLatentImage", "inputs": map[string]interface{}{"width": 1024, "height": 1024, "batch_size": 1}}
	wf["41"] = map[string]interface{}{"class_type": "RandomNoise", "inputs": map[string]interface{}{"noise_seed": seed, "control": "randomize"}}
	wf["42"] = map[string]interface{}{"class_type": "CFGGuider", "inputs": map[string]interface{}{
		"model": []interface{}{"40", 0}, "positive": []interface{}{"7", 0}, "negative": []interface{}{"7", 1}, "cfg": 8.0,
	}}
	wf["43"] = map[string]interface{}{"class_type": "KSamplerSelect", "inputs": map[string]interface{}{"sampler_name": "euler"}}
	wf["44"] = map[string]interface{}{"class_type": "ManualSigmas", "inputs": map[string]interface{}{"sigmas": qwen21Sigmas}}
	wf["10"] = map[string]interface{}{"class_type": "SamplerCustomAdvanced", "inputs": map[string]interface{}{
		"noise": []interface{}{"41", 0}, "guider": []interface{}{"42", 0}, "sampler": []interface{}{"43", 0},
		"sigmas": []interface{}{"44", 0}, "latent_image": []interface{}{"9", 0},
	}}
	wf["90"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["91"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"90", 0}}}
	return wf
}

// buildQwen21Workflow 构建 Qwen-Image-2.1-Turbo 文生图工作流（官方模板
// API 图还原，取道不取器）：
//
//	UNETLoader → [LoRA 链] → QwenImage21Cache(auto/default) → CFGGuider(cfg 1.0)
//	CLIPLoader(qwen_image) → TextEncodeQwenImage21(prompt + negative_prompt
//	  同节点双编码，输出 0=正 / 1=负 conditioning)
//	EmptyLatentImage + RandomNoise + KSamplerSelect(euler) + ManualSigmas
//	  → SamplerCustomAdvanced → VAEDecode → SaveImage
//
// 官方模板的可选件不接：TextGenerate 提示词改写支路（PE 模型
// qwen3.5_9b_*_pe_t2i 另需 9.5GB，模板默认 switch=false 直通原文，gaea 的
// 提示词由上层 LLM 组织）。2.1 的编辑由 buildQwen21EditWorkflow 独立承接。
// TextEncodeQwenImage21.resolution 只作用于参考图重标，本工作流不传参考图，
// 保持模板默认 1024（latent 由 EmptyLatentImage 按请求宽高给出）。
func (b *ComfyUIBackend) buildQwen21Workflow(prompt, negative string, width, height, seed int, loras []string) map[string]interface{} {
	wf := map[string]interface{}{
		"4": map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": qwen21UNET, "weight_dtype": "default"}},
		"5": map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": qwen21CLIP, "type": "qwen_image", "device": "default"}},
		"6": map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": qwen21VAE}},
		"7": map[string]interface{}{"class_type": "TextEncodeQwenImage21", "inputs": map[string]interface{}{
			"clip": []interface{}{"5", 0}, "prompt": prompt, "negative_prompt": negative, "resolution": 1024,
		}},
		"9":  map[string]interface{}{"class_type": "EmptyLatentImage", "inputs": map[string]interface{}{"width": width, "height": height, "batch_size": 1}},
		"41": map[string]interface{}{"class_type": "RandomNoise", "inputs": map[string]interface{}{"noise_seed": seed, "control": "randomize"}},
		"43": map[string]interface{}{"class_type": "KSamplerSelect", "inputs": map[string]interface{}{"sampler_name": "euler"}},
		"44": map[string]interface{}{"class_type": "ManualSigmas", "inputs": map[string]interface{}{"sigmas": qwen21Sigmas}},
	}
	modelSourceID := injectLoraNodes(wf, "4", loras)
	wf["40"] = map[string]interface{}{"class_type": "QwenImage21Cache", "inputs": map[string]interface{}{
		"model": []interface{}{modelSourceID, 0}, "device": "auto", "dtype": "default",
	}}
	wf["42"] = map[string]interface{}{"class_type": "CFGGuider", "inputs": map[string]interface{}{
		"model": []interface{}{"40", 0}, "positive": []interface{}{"7", 0}, "negative": []interface{}{"7", 1}, "cfg": 1.0,
	}}
	wf["10"] = map[string]interface{}{"class_type": "SamplerCustomAdvanced", "inputs": map[string]interface{}{
		"noise": []interface{}{"41", 0}, "guider": []interface{}{"42", 0}, "sampler": []interface{}{"43", 0},
		"sigmas": []interface{}{"44", 0}, "latent_image": []interface{}{"9", 0},
	}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// qwen21MissingModelHint 文生图提交失败时的缺权重提示：gaea 从不自动下载
// 模型，这里给出三件文件名 + 存放目录（可操作）；错误不含本族权重名时
// 返回空串零影响（krea2/z-image 的 LoRA 缺失不误伤）。
func qwen21MissingModelHint(errMsg string) string {
	if !strings.Contains(errMsg, "qwen_image_2.1") && !strings.Contains(errMsg, "qwen3vl_8b") {
		return ""
	}
	return "\n💡 文生图本地档（Qwen-Image-2.1-Turbo）需要以下模型文件（放入 ComfyUI 对应目录后重试）：\n" +
		"   models/diffusion_models/" + qwen21UNET + "\n" +
		"   models/text_encoders/" + qwen21CLIP + "\n" +
		"   models/vae/" + qwen21VAE + "\n" +
		"   ← 可在 ComfyUI 模板管理器打开「Qwen Image 2.1 Turbo」模板一键下载，或到 ModelScope/HuggingFace 搜索同名权重"
}

func qwen21EditMissingModelHint(errMsg string) string {
	missingValue := strings.Contains(errMsg, "value_not_in_list") ||
		strings.Contains(strings.ToLower(errMsg), "value not in list")
	if !missingValue ||
		(!strings.Contains(errMsg, qwen21EditUNET) &&
			!strings.Contains(errMsg, qwen21CLIP) &&
			!strings.Contains(errMsg, qwen21VAE)) {
		return ""
	}
	return "\n💡 Qwen Image 2.1 Turbo Edit 需要以下模型文件（放入 ComfyUI 对应目录后重试）：\n" +
		"   models/diffusion_models/" + qwen21EditUNET + "\n" +
		"   models/text_encoders/" + qwen21CLIP + "\n" +
		"   models/vae/" + qwen21VAE + "\n" +
		"   ← 可在 ComfyUI 模板管理器打开「Image Edit (Qwen Image 2.1 Turbo)」查看官方下载地址；gaea 不会自动下载权重。\n" +
		"   提示词增强器默认关闭，不需要额外 PE 权重。"
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
