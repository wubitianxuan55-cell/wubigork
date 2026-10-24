package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestQwen21EditWorkflowSingleAndMultiReference(t *testing.T) {
	b := &ComfyUIBackend{}
	for _, n := range []int{1, 4, 10} {
		t.Run(fmt.Sprintf("%d_images", n), func(t *testing.T) {
			images := make([]string, n)
			for i := range images {
				images[i] = fmt.Sprintf("image-%d.png", i+1)
			}
			wf := b.buildQwen21EditWorkflow("把<image1>的外套改成红色", "", images, 42, nil)
			for i := 1; i <= n; i++ {
				key := fmt.Sprintf("images.image_%d", i)
				if got := nodeInput(t, wf, "7", key); got == nil {
					t.Errorf("missing encoder image key %s", key)
				}
				if _, ok := wf[fmt.Sprintf("1%d", i)]; !ok {
					t.Errorf("missing LoadImage node for image %d", i)
				}
				if got := wf[fmt.Sprintf("1%d", i)].(map[string]interface{})["class_type"]; got != "LoadImage" {
					t.Errorf("load node %d overwritten: %v", i, got)
				}
			}
			if got := nodeInput(t, wf, "7", "images.image_11"); got != nil {
				t.Errorf("unexpected eleventh image input: %v", got)
			}
			if _, ok := wf["500"]; ok {
				t.Error("prompt enhancer must stay disabled (no TextGenerate node)")
			}
			if got := nodeInput(t, wf, "4", "unet_name"); got != qwen21EditUNET {
				t.Errorf("unet = %v", got)
			}
			if got := nodeInput(t, wf, "5", "clip_name"); got != "qwen3vl_8b_int8_convrot.safetensors" {
				t.Errorf("clip = %v", got)
			}
			if got := nodeInput(t, wf, "6", "vae_name"); got != "qwen_image_2.1_vae_bf16.safetensors" {
				t.Errorf("vae = %v", got)
			}
			if got := nodeInput(t, wf, "42", "cfg"); got != 8.0 {
				t.Errorf("cfg = %v, want 8", got)
			}
			if got := nodeInput(t, wf, "44", "sigmas"); got != qwen21Sigmas {
				t.Errorf("sigmas = %v", got)
			}
			for _, key := range []string{"guider", "latent_image", "noise", "sampler", "sigmas"} {
				if got := nodeInput(t, wf, "10", key); got == nil {
					t.Errorf("sampler missing %s", key)
				}
			}
			if nodeInput(t, wf, "42", "positive").([]interface{})[0] != "7" ||
				nodeInput(t, wf, "42", "negative").([]interface{})[0] != "7" ||
				nodeInput(t, wf, "10", "latent_image").([]interface{})[0] != "9" {
				t.Error("Turbo Edit must use encoder conditioning + EmptyLatentImage through CFGGuider")
			}
		})
	}
}

func TestQwen21EditRejectsEleventhReference(t *testing.T) {
	b := NewComfyUIBackend("http://127.0.0.1:1")
	refs := make([]string, 11)
	for i := range refs {
		refs[i] = editDataURL()
	}
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model: Qwen21EditModel, Prompt: "编辑", Mode: "txt2img",
		RefImages: refs, RefMethod: "qedit",
	})
	if err == nil || !strings.Contains(err.Error(), "最多 10 张") {
		t.Fatalf("11 refs = %v, want explicit limit", err)
	}
}

func TestQwen21EditMissingModelHint(t *testing.T) {
	got := qwen21EditMissingModelHint(`value not in list: qwen_image_2.1_turbo_int8_convrot.safetensors`)
	for _, want := range []string{
		"qwen_image_2.1_turbo_int8_convrot.safetensors",
		"qwen3vl_8b_int8_convrot.safetensors",
		"qwen_image_2.1_vae_bf16.safetensors",
		"提示词增强器默认关闭",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("hint missing %q:\n%s", want, got)
		}
	}
	if qwen21EditMissingModelHint("network timeout") != "" {
		t.Error("ordinary network errors must not be reported as missing weights")
	}
	if qwen21EditMissingModelHint("dependency_cycle at TextEncodeQwenImage21") != "" {
		t.Error("workflow errors must not be reported as missing weights")
	}
	if qwen21EditMissingModelHint("value_not_in_list: missing_lora.safetensors") != "" {
		t.Error("missing LoRA must not be reported as missing base weights")
	}
}

func TestComfyUIEditRequiresQwen21Edit(t *testing.T) {
	b := NewComfyUIBackend("http://127.0.0.1:1")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model: "krea2", Prompt: "改", Mode: "edit", InitImage: editDataURL(),
	})
	if err == nil || !strings.Contains(err.Error(), Qwen21EditModel) {
		t.Fatalf("旧编辑模型必须显式拒绝并指向 %s，得到: %v", Qwen21EditModel, err)
	}
}

func TestQwen21EditRejectsMaskInsteadOfLegacyFallback(t *testing.T) {
	b := NewComfyUIBackend("http://127.0.0.1:1")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model: Qwen21EditModel, Prompt: "局部改", Mode: "edit",
		InitImage: realGrayPNGDataURL(t, 2, 1, false),
		Mask:      realGrayPNGDataURL(t, 2, 1, true),
	})
	if err == nil || !strings.Contains(err.Error(), "暂不支持蒙版") {
		t.Fatalf("2.1 Turbo Edit 不支持蒙版时必须诚实拒绝，不能回退旧模型，得到: %v", err)
	}
}
