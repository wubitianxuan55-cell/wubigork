package ai

import (
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
