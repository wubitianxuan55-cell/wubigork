package ai

// image_comfyui_workflows.go — ComfyUI 工作流构建族（P5 破防专项拆自
// image_comfyui.go：各模型 txt2img/img2img/edit/video 的工作流拼装、LoRA 注入、
// Kontext 分辨率表——搬移零功能变更）。

import (
	"fmt"
	"math"
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

// Qwen-Image-Edit 2511 官方模板文件名（comfyui-workflow-templates
// image_qwen_image_edit_2511 蒸馏，2026-09-22 实读）。单点维护——后续
// 配置化（config 键覆盖）在此扩展；VAE 与 krea2 共用。
const (
	qwenEditUNET = "qwen_image_edit_2511_fp8mixed.safetensors"
	qwenEditCLIP = "qwen_2.5_vl_7b_fp8_scaled.safetensors"
	qwenEditVAE  = "qwen_image_vae.safetensors"
)

// buildQwenImageEditWorkflow 构建 Qwen-Image-Edit 2511 指令编辑工作流
// （官方模板 API 图还原，取道不取器）：
//
//	LoadImage → FluxKontextImageScale（按原图宽高比重标，不套 1024×1024）
//	  ├→ TextEncodeQwenImageEditPlus(正) image1=缩放图 + vae
//	  ├→ TextEncodeQwenImageEditPlus(负) image1=缩放图 + vae
//	  └→ VAEEncode(缩放图) → KSampler.latent_image
//	UNETLoader → ModelSamplingAuraFlow(shift 3.1) → CFGNorm(1) → KSampler
//	KSampler(steps 20 / cfg 4.0 / euler / simple / denoise 1.0) → VAEDecode → SaveImage
//
// 蒙版局部重绘（阶段二刀 B，maskName 非空时）：
//
//	LoadImage(mask 灰度：白=重绘) → ImageScale(lanczos, targetW/H) → ImageToMask(red)
//	  → SetLatentNoiseMask(VAEEncode) → KSampler.latent_image
//
//	蒙版与缩放图同尺寸（targetW/H=kontextScaleSize(原图)，与 FluxKontextImageScale
//	的目标一致）——SetLatentNoiseMask 只做裁剪不 resize，尺寸不对齐会错位；
//	TextEncode 的 image1 保持全图（语义参考需全图上下文，重绘区域由 noise_mask 限制）。
//
// 官方 Note「Comfy」列参数（20 步 CFG 4.0）；denoise 恒 1.0=语义编辑非整幅重绘；
// FluxKontextMultiReferenceLatentMethod 官方明示「用 Comfy 官方权重不需要」不接；
// Lightning 4 步 LoRA 为可选加速件，v1 不接（观察池）。
func (b *ComfyUIBackend) buildQwenImageEditWorkflow(prompt string, seed int, images []string, maskName string, maskTargetW, maskTargetH int) map[string]interface{} {
	wf := map[string]interface{}{
		"4":   map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": qwenEditUNET, "weight_dtype": "default"}},
		"5":   map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": qwenEditCLIP, "type": "qwen_image", "device": "default"}},
		"6":   map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": qwenEditVAE}},
		"145": map[string]interface{}{"class_type": "ModelSamplingAuraFlow", "inputs": map[string]interface{}{"model": []interface{}{"4", 0}, "shift": 3.1}},
		"152": map[string]interface{}{"class_type": "CFGNorm", "inputs": map[string]interface{}{"model": []interface{}{"145", 0}, "strength": 1.0, "pre_cfg": false}},
		"1":   map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": images[0]}},
		// FluxKontextImageScale：把原图重标到 Kontext 系最优分辨率（保宽高比）
		"160": map[string]interface{}{"class_type": "FluxKontextImageScale", "inputs": map[string]interface{}{"image": []interface{}{"1", 0}}},
		"7":   map[string]interface{}{"class_type": "TextEncodeQwenImageEditPlus", "inputs": map[string]interface{}{"clip": []interface{}{"5", 0}, "vae": []interface{}{"6", 0}, "image1": []interface{}{"160", 0}, "prompt": prompt}},
		"8":   map[string]interface{}{"class_type": "TextEncodeQwenImageEditPlus", "inputs": map[string]interface{}{"clip": []interface{}{"5", 0}, "vae": []interface{}{"6", 0}, "image1": []interface{}{"160", 0}, "prompt": ""}},
	}
	// 阶段三刀 A：参考槽 image2/3（qedit 多图参考；正/负 TextEncode 都接全部
	// 参考槽——官方模板口径）。参考图各自 LoadImage+FluxKontextImageScale
	//（与 image1 同款重标，编辑族输入尺寸规约一致）。
	for i, name := range images[1:] {
		loadID := fmt.Sprintf("2%d", i+1) // "21"/"22"
		scaleID := fmt.Sprintf("17%d", i+1)
		wf[loadID] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": name}}
		wf[scaleID] = map[string]interface{}{"class_type": "FluxKontextImageScale", "inputs": map[string]interface{}{"image": []interface{}{loadID, 0}}}
		for _, encID := range []string{"7", "8"} {
			wf[encID].(map[string]interface{})["inputs"].(map[string]interface{})[fmt.Sprintf("image%d", i+2)] = []interface{}{scaleID, 0}
		}
	}
	wf["15"] = map[string]interface{}{"class_type": "VAEEncode", "inputs": map[string]interface{}{"pixels": []interface{}{"160", 0}, "vae": []interface{}{"6", 0}}}
	// 蒙版链：latent 源改接 SetLatentNoiseMask 输出（"163"），无蒙版时直连 VAEEncode
	latentNode := []interface{}{"15", 0}
	if maskName != "" {
		wf["2"] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": maskName}}
		wf["161"] = map[string]interface{}{"class_type": "ImageScale", "inputs": map[string]interface{}{
			"image": []interface{}{"2", 0}, "upscale_method": "lanczos",
			"width": maskTargetW, "height": maskTargetH, "crop": "disabled",
		}}
		wf["162"] = map[string]interface{}{"class_type": "ImageToMask", "inputs": map[string]interface{}{"image": []interface{}{"161", 0}, "channel": "red"}}
		wf["163"] = map[string]interface{}{"class_type": "SetLatentNoiseMask", "inputs": map[string]interface{}{"samples": []interface{}{"15", 0}, "mask": []interface{}{"162", 0}}}
		latentNode = []interface{}{"163", 0}
	}
	wf["10"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
		"seed": seed, "steps": 20, "cfg": 4.0, "sampler_name": "euler", "scheduler": "simple", "denoise": 1.0,
		"model": []interface{}{"152", 0}, "positive": []interface{}{"7", 0}, "negative": []interface{}{"8", 0}, "latent_image": latentNode,
	}}
	wf["11"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"10", 0}, "vae": []interface{}{"6", 0}}}
	wf["12"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "gaea", "images": []interface{}{"11", 0}}}
	return wf
}

// preferredKontextResolutions ComfyUI FluxKontextImageScale 的分辨率预设表
// （本机 ComfyUI 0.36 comfy_extras/nodes_flux.py:105 实读，17 档）。
var preferredKontextResolutions = [][2]int{
	{672, 1568}, {688, 1504}, {720, 1456}, {752, 1392}, {800, 1328},
	{832, 1248}, {880, 1184}, {944, 1104}, {1024, 1024}, {1104, 944},
	{1184, 880}, {1248, 832}, {1328, 800}, {1392, 752}, {1456, 720},
	{1504, 688}, {1568, 672},
}

// kontextScaleSize 复刻 FluxKontextImageScale 的尺寸选择：宽高比最接近的
// 预设档（平手取先出现者，与 Python min 语义一致）；退化输入兜底 1024²。
// 蒙版链用它把灰度蒙版对齐到缩放图的目标尺寸。
func kontextScaleSize(w, h int) (int, int) {
	if w <= 0 || h <= 0 {
		return 1024, 1024
	}
	aspect := float64(w) / float64(h)
	bestW, bestH := 1024, 1024
	bestDiff := math.MaxFloat64
	for _, r := range preferredKontextResolutions {
		if d := math.Abs(aspect - float64(r[0])/float64(r[1])); d < bestDiff {
			bestDiff, bestW, bestH = d, r[0], r[1]
		}
	}
	return bestW, bestH
}

// qwenEditMissingModelHint 指令编辑提交失败时的缺权重提示：ComfyUI 对不在列表的
// 模型回 value_not_in_list——gaea 从不自动下载模型，这里给出三件文件名 + HF 链接
// + 存放目录（可操作）；错误不匹配时返回空串零影响。
func qwenEditMissingModelHint(errMsg string) string {
	if !strings.Contains(errMsg, "value_not_in_list") &&
		!strings.Contains(errMsg, "qwen_image_edit") &&
		!strings.Contains(errMsg, "qwen_2_5_vl") &&
		!strings.Contains(errMsg, "missing") {
		return ""
	}
	return "\n💡 指令编辑本地档（Qwen-Image-Edit 2511）需要以下模型文件（放入 ComfyUI 对应目录后重试）：\n" +
		"   models/diffusion_models/" + qwenEditUNET + "\n" +
		"     ← https://huggingface.co/Comfy-Org/Qwen-Image-Edit_ComfyUI（split_files/diffusion_models；bf16 版同名替换亦可）\n" +
		"   models/text_encoders/" + qwenEditCLIP + "\n" +
		"     ← https://huggingface.co/Comfy-Org/HunyuanVideo_1.5_repackaged（split_files/text_encoders）\n" +
		"   models/vae/" + qwenEditVAE + "（与 krea2 共用，通常已存在）"
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
