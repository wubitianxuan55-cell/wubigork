package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

// 工作流节点断言辅助：读取 map 里嵌套的 inputs 字段
func nodeInput(t *testing.T, wf map[string]interface{}, nodeID, key string) interface{} {
	t.Helper()
	n, ok := wf[nodeID].(map[string]interface{})
	if !ok {
		t.Fatalf("节点 %s 缺失或类型异常", nodeID)
	}
	inputs, ok := n["inputs"].(map[string]interface{})
	if !ok {
		t.Fatalf("节点 %s 无 inputs", nodeID)
	}
	return inputs[key]
}

func TestBuildKreaWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildKreaWorkflow("测试 prompt", 1024, 1024, 42, 8, nil)

	// 模型三要素（记忆：Krea2 打通三要素缺一不可）
	unet := nodeInput(t, wf, "4", "unet_name")
	if unet != "krea2_turbo_fp8_scaled.safetensors" {
		t.Errorf("UNET = %v, want krea2_turbo_fp8_scaled.safetensors", unet)
	}
	clipType := nodeInput(t, wf, "5", "type")
	if clipType != "krea2" {
		t.Errorf("CLIP type = %v, want krea2", clipType)
	}
	clipName := nodeInput(t, wf, "5", "clip_name")
	if clipName != "qwen3vl_4b_fp8_scaled.safetensors" {
		t.Errorf("CLIP = %v, want qwen3vl_4b_fp8_scaled.safetensors", clipName)
	}
	vae := nodeInput(t, wf, "6", "vae_name")
	if vae != "qwen_image_vae.safetensors" {
		t.Errorf("VAE = %v, want qwen_image_vae.safetensors（用 ae.safetensors 会出灰紫图）", vae)
	}

	// 关键节点：EmptyLatentImage（非 SD3！）+ ConditioningZeroOut + CFG=1.0
	if wf["9"].(map[string]interface{})["class_type"] != "EmptyLatentImage" {
		t.Errorf("节点9 = %v, want EmptyLatentImage（Krea2 不能用 EmptySD3LatentImage）", wf["9"].(map[string]interface{})["class_type"])
	}
	if wf["13"].(map[string]interface{})["class_type"] != "ConditioningZeroOut" {
		t.Errorf("节点13 = %v, want ConditioningZeroOut", wf["13"].(map[string]interface{})["class_type"])
	}
	if wf["10"].(map[string]interface{})["class_type"] != "KSampler" {
		t.Fatalf("节点10 不是 KSampler")
	}
	if cfg := nodeInput(t, wf, "10", "cfg"); cfg != 1.0 {
		t.Errorf("KSampler cfg = %v, want 1.0", cfg)
	}
	if steps := nodeInput(t, wf, "10", "steps"); steps != 8 {
		t.Errorf("KSampler steps = %v, want 8", steps)
	}
	// Krea2 不需要 ModelSamplingAuraFlow
	if _, exists := wf["14"]; exists {
		t.Errorf("Krea2 工作流不应有 ModelSamplingAuraFlow 节点（记忆：Krea2 不需要）")
	}
}

func TestBuildZImageWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildZImageWorkflow("测试 prompt", "负面", 1024, 1024, 42, 8, "z_image_turbo_bf16_完整版_效果最好.safetensors", nil)

	// ZIT 关键差异：EmptySD3LatentImage + ModelSamplingAuraFlow(shift=3) + CLIP lumina2
	if wf["9"].(map[string]interface{})["class_type"] != "EmptySD3LatentImage" {
		t.Errorf("节点9 = %v, want EmptySD3LatentImage", wf["9"].(map[string]interface{})["class_type"])
	}
	if wf["14"].(map[string]interface{})["class_type"] != "ModelSamplingAuraFlow" {
		t.Errorf("节点14 = %v, want ModelSamplingAuraFlow", wf["14"].(map[string]interface{})["class_type"])
	}
	if shift := nodeInput(t, wf, "14", "shift"); shift != 3 {
		t.Errorf("AuraFlow shift = %v, want 3", shift)
	}
	if clipType := nodeInput(t, wf, "5", "type"); clipType != "lumina2" {
		t.Errorf("CLIP type = %v, want lumina2", clipType)
	}
	if sampler := nodeInput(t, wf, "10", "sampler_name"); sampler != "res_multistep" {
		t.Errorf("sampler = %v, want res_multistep", sampler)
	}
	if wf["13"].(map[string]interface{})["class_type"] != "ConditioningZeroOut" {
		t.Errorf("节点13 = %v, want ConditioningZeroOut", wf["13"].(map[string]interface{})["class_type"])
	}
}

func TestParseLoraNames(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "单层包装（本机 ComfyUI 结构）",
			raw:  `[["flux1\\a.safetensors", "zimage\\b.safetensors"]]`,
			want: []string{"flux1\\a.safetensors", "zimage\\b.safetensors"},
		},
		{
			name: "标准 ComfyUI",
			raw:  `["LORAS", ["flux1\\a.safetensors", "zimage\\b.safetensors"]]`,
			want: []string{"flux1\\a.safetensors", "zimage\\b.safetensors"},
		},
		{
			name: "对象映射结构",
			raw:  `["LORAS", {"flux1\\a.safetensors": {"weight": 1}, "zimage\\b.safetensors": {}}]`,
			want: []string{"flux1\\a.safetensors", "zimage\\b.safetensors"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLoraNames(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("parseLoraNames: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d]=%q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestBuildKreaWorkflow_LoRAInjection(t *testing.T) {
	b := &ComfyUIBackend{}
	loras := []string{"zimage\\z-image-细节增强v2.safetensors", "zimage\\z-Image-3D卡通_V1.safetensors"}
	wf := b.buildKreaWorkflow("prompt", 1024, 1024, 1, 8, loras)

	// 两个 LoRA 节点 20、21，KSampler 的 model 指向最后一个 LoRA 节点（链尾 21）
	if wf["20"].(map[string]interface{})["class_type"] != "LoraLoaderModelOnly" {
		t.Errorf("节点20 = %v, want LoraLoaderModelOnly", wf["20"].(map[string]interface{})["class_type"])
	}
	if wf["21"].(map[string]interface{})["class_type"] != "LoraLoaderModelOnly" {
		t.Errorf("节点21 = %v, want LoraLoaderModelOnly", wf["21"].(map[string]interface{})["class_type"])
	}
	if got := nodeInput(t, wf, "10", "model"); got.([]interface{})[0] != "21" {
		t.Errorf("KSampler model 指向 %v, want 21（应指向 LoRA 链尾）", got)
	}
	// 链首 LoRA 的 model 应指向 UNETLoader(4)
	if got := nodeInput(t, wf, "20", "model"); got.([]interface{})[0] != "4" {
		t.Errorf("节点20 model 指向 %v, want 4", got)
	}
}

func TestBuildKreaImg2ImgWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildKreaImg2ImgWorkflow("测试 prompt", "坏手", 1024, 1024, 42, 8, nil, "ref.png", 0.65)

	if wf["1"].(map[string]interface{})["class_type"] != "LoadImage" {
		t.Errorf("节点1 = %v, want LoadImage", wf["1"].(map[string]interface{})["class_type"])
	}
	if got := nodeInput(t, wf, "1", "image"); got != "ref.png" {
		t.Errorf("LoadImage 图片 = %v, want ref.png", got)
	}
	if got := nodeInput(t, wf, "15", "pixels"); got.([]interface{})[0] != "1" {
		t.Errorf("VAEEncode pixels 指向 %v, want 1（LoadImage 输出）", got)
	}
	if got := nodeInput(t, wf, "10", "latent_image"); got.([]interface{})[0] != "15" {
		t.Errorf("KSampler latent 指向 %v, want 15（VAEEncode 输出）", got)
	}
	if got := nodeInput(t, wf, "10", "denoise"); got.(float64) != 0.65 {
		t.Errorf("KSampler denoise = %v, want 0.65", got)
	}
}

func TestParseSize(t *testing.T) {
	cases := []struct {
		name      string
		size      string
		wantW     int
		wantH     int
		wantError bool
	}{
		{name: "正常尺寸", size: "1024x1024", wantW: 1024, wantH: 1024},
		{name: "含空格", size: " 768 x 512 ", wantW: 768, wantH: 512},
		{name: "低于下限钳制到 64", size: "32x32", wantW: 64, wantH: 64},
		{name: "超上限钳制到 2048", size: "4096x100", wantW: 2048, wantH: 100},
		{name: "单边钳制", size: "100x99999", wantW: 100, wantH: 2048},
		{name: "非数字", size: "abc", wantError: true},
		{name: "缺少高度", size: "1024", wantError: true},
		{name: "宽度非数字", size: "ax512", wantError: true},
		{name: "高度非数字", size: "512xb", wantError: true},
		{name: "多余分段", size: "1024x1024x1024", wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, h, err := parseSize(tc.size)
			if tc.wantError {
				if err == nil {
					t.Fatalf("parseSize(%q) 应报错, got %dx%d", tc.size, w, h)
				}
				if !strings.Contains(err.Error(), "尺寸格式无效") {
					t.Errorf("错误应为中文尺寸提示: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSize(%q): %v", tc.size, err)
			}
			if w != tc.wantW || h != tc.wantH {
				t.Errorf("parseSize(%q) = %dx%d, want %dx%d", tc.size, w, h, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestBuildFluxWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildFluxWorkflow("测试 prompt", 1024, 1024, 42, nil)

	// Flux 三要素：UNETLoader(flux1-schnell) + DualCLIPLoader(type=flux) + VAELoader(ae)
	unet := nodeInput(t, wf, "4", "unet_name")
	if unet != "flux1-schnell.safetensors" {
		t.Errorf("UNET = %v, want flux1-schnell.safetensors", unet)
	}
	if wf["5"].(map[string]interface{})["class_type"] != "DualCLIPLoader" {
		t.Errorf("节点5 = %v, want DualCLIPLoader", wf["5"].(map[string]interface{})["class_type"])
	}
	if clipType := nodeInput(t, wf, "5", "type"); clipType != "flux" {
		t.Errorf("CLIP type = %v, want flux", clipType)
	}
	if vae := nodeInput(t, wf, "6", "vae_name"); vae != "ae.safetensors" {
		t.Errorf("VAE = %v, want ae.safetensors", vae)
	}
	// Flux 潜空间与采样：EmptySD3LatentImage + KSampler(cfg=1.0, euler/simple, 4 步)
	if wf["9"].(map[string]interface{})["class_type"] != "EmptySD3LatentImage" {
		t.Errorf("节点9 = %v, want EmptySD3LatentImage", wf["9"].(map[string]interface{})["class_type"])
	}
	if steps := nodeInput(t, wf, "10", "steps"); steps != 4 {
		t.Errorf("KSampler steps = %v, want 4（schnell 4 步）", steps)
	}
	if sampler := nodeInput(t, wf, "10", "sampler_name"); sampler != "euler" {
		t.Errorf("sampler = %v, want euler", sampler)
	}
	// Flux 官方模板不用 ConditioningZeroOut（负面用空文本）
	if _, exists := wf["13"]; exists {
		t.Errorf("Flux 工作流不应有 ConditioningZeroOut 节点")
	}
}

func TestBuildLTXVideoWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildLTXVideoWorkflow("测试 prompt", "低质量", 768, 512, 42, 97, 8, "")

	if wf["1"].(map[string]interface{})["class_type"] != "CheckpointLoaderSimple" {
		t.Errorf("节点1 = %v, want CheckpointLoaderSimple", wf["1"].(map[string]interface{})["class_type"])
	}
	if wf["12"].(map[string]interface{})["class_type"] != "SaveVideo" {
		t.Errorf("节点12 = %v, want SaveVideo", wf["12"].(map[string]interface{})["class_type"])
	}
	if got := nodeInput(t, wf, "6", "length"); got.(int) != 96 {
		t.Errorf("视频帧数 = %v, want 96（97 归一化为 8 的倍数）", got)
	}
	if got := nodeInput(t, wf, "6", "width"); got.(int) != 768 {
		t.Errorf("视频宽度 = %v, want 768", got)
	}
	if got := nodeInput(t, wf, "11", "fps"); got.(int) != 8 {
		t.Errorf("视频 fps = %v, want 8", got)
	}
	// SamplerCustom 接线：model=Checkpoint[0]，positive/negative=LTXVConditioning，
	// sampler=KSamplerSelect，sigmas=LTXVScheduler，latent=EmptyLTXVLatentVideo
	if got := nodeInput(t, wf, "9", "model"); !equalLink(got, "1", 0) {
		t.Errorf("SamplerCustom.model = %v, want [1,0]", got)
	}
	if got := nodeInput(t, wf, "9", "positive"); !equalLink(got, "5", 0) {
		t.Errorf("SamplerCustom.positive = %v, want [5,0]", got)
	}
	if got := nodeInput(t, wf, "9", "negative"); !equalLink(got, "5", 1) {
		t.Errorf("SamplerCustom.negative = %v, want [5,1]", got)
	}
	if got := nodeInput(t, wf, "9", "sigmas"); !equalLink(got, "7", 0) {
		t.Errorf("SamplerCustom.sigmas = %v, want [7,0]", got)
	}
	if got := nodeInput(t, wf, "12", "video"); !equalLink(got, "11", 0) {
		t.Errorf("SaveVideo.video = %v, want [11,0]", got)
	}
}

// equalLink 断言工作流链接（[]interface{}{节点ID, 输出槽}）。
func equalLink(got interface{}, wantNode string, wantSlot int) bool {
	arr, ok := got.([]interface{})
	if !ok || len(arr) != 2 {
		return false
	}
	return arr[0] == wantNode && arr[1] == wantSlot
}

func TestComfyExecutionHint(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want string
	}{
		{"rms_rope", "ComfyUI 执行错误: rms_rope(): incompatible function arguments", "requirements.txt"},
		{"fp8_layout", "ComfyUI 执行错误: 'NoneType' object has no attribute 'Params'", "requirements.txt"},
		{"kitchen_import", "ComfyUI 执行错误: cannot import name 'AsymW4A8Int8Layout' from 'comfy_kitchen.tensor'", "requirements.txt"},
		{"value_not_in_list", "ComfyUI 执行错误: value_not_in_list: model not found", "绘梦页重新选择 LoRA"},
		{"unknown", "ComfyUI 执行错误: 显存不足", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := comfyExecutionHint(tc.msg)
			if tc.want == "" {
				if got != "" {
					t.Fatalf("comfyExecutionHint(%q) = %q, want 空", tc.msg, got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("comfyExecutionHint(%q) = %q, want 包含 %q", tc.msg, got, tc.want)
			}
		})
	}
}
