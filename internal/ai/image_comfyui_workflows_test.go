package ai

// AP7-03 工作流节点表 golden：krea2 / z-image 两族 txt2img/img2img 收口到
// comfyQwenProfiles + buildQwenProfileTxt2Img/Img2Img 后，用序列化 JSON 逐字节
// 钉死节点表（键序=encoding/json 的 map 排序，稳定）。profile 任一字段漂移
// （权重名/CLIP type/采样器/shift/latent 类型/步数钳制）在此现形。
// Warmup 分发等价（AP7-03 行为面）也在本文件钉住。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// marshalWorkflow 序列化工作流（map 键序稳定，直接整串比对）。
func marshalWorkflow(t *testing.T, wf map[string]interface{}) string {
	t.Helper()
	b, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("序列化工作流失败: %v", err)
	}
	return string(b)
}

func TestGoldenKreaTxt2ImgWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildKreaWorkflow("测试 prompt", 1024, 1024, 42, 8, nil)
	want := `{"10":{"class_type":"KSampler","inputs":{"cfg":1,"denoise":1,"latent_image":["9",0],"model":["4",0],"negative":["13",0],"positive":["7",0],"sampler_name":"euler","scheduler":"simple","seed":42,"steps":8}},"11":{"class_type":"VAEDecode","inputs":{"samples":["10",0],"vae":["6",0]}},"12":{"class_type":"SaveImage","inputs":{"filename_prefix":"gaea","images":["11",0]}},"13":{"class_type":"ConditioningZeroOut","inputs":{"conditioning":["8",0]}},"4":{"class_type":"UNETLoader","inputs":{"unet_name":"krea2_turbo_fp8_scaled.safetensors","weight_dtype":"default"}},"5":{"class_type":"CLIPLoader","inputs":{"clip_name":"qwen3vl_4b_fp8_scaled.safetensors","type":"krea2"}},"6":{"class_type":"VAELoader","inputs":{"vae_name":"qwen_image_vae.safetensors"}},"7":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":"测试 prompt"}},"8":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":""}},"9":{"class_type":"EmptyLatentImage","inputs":{"batch_size":1,"height":1024,"width":1024}}}`
	if got := marshalWorkflow(t, wf); got != want {
		t.Errorf("krea2 txt2img 节点表漂移:\n got %s\nwant %s", got, want)
	}
	// 历史不对称：krea2 txt2img 步数透传不钳制（z-image 才有 [1,20] 钳制）
	if got := nodeInput(t, b.buildKreaWorkflow("p", 64, 64, 0, 25, nil), "10", "steps"); got != 25 {
		t.Errorf("krea2 txt2img 步数应透传，got %v", got)
	}
}

func TestGoldenZImageTxt2ImgWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildZImageWorkflow("测试 prompt", 1024, 1024, 42, 8, nil)
	want := `{"10":{"class_type":"KSampler","inputs":{"cfg":1,"denoise":1,"latent_image":["9",0],"model":["14",0],"negative":["13",0],"positive":["7",0],"sampler_name":"res_multistep","scheduler":"simple","seed":42,"steps":8}},"11":{"class_type":"VAEDecode","inputs":{"samples":["10",0],"vae":["6",0]}},"12":{"class_type":"SaveImage","inputs":{"filename_prefix":"gaea","images":["11",0]}},"13":{"class_type":"ConditioningZeroOut","inputs":{"conditioning":["8",0]}},"14":{"class_type":"ModelSamplingAuraFlow","inputs":{"model":["4",0],"shift":3}},"4":{"class_type":"UNETLoader","inputs":{"unet_name":"z_image_turbo_bf16_完整版_效果最好.safetensors","weight_dtype":"default"}},"5":{"class_type":"CLIPLoader","inputs":{"clip_name":"z-image\\qwen_3_4b.safetensors","type":"lumina2"}},"6":{"class_type":"VAELoader","inputs":{"vae_name":"z-image-qwen.safetensors"}},"7":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":"测试 prompt"}},"8":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":""}},"9":{"class_type":"EmptySD3LatentImage","inputs":{"batch_size":1,"height":1024,"width":1024}}}`
	if got := marshalWorkflow(t, wf); got != want {
		t.Errorf("z-image txt2img 节点表漂移:\n got %s\nwant %s", got, want)
	}
	// z-image 步数钳制（历史行为）：<=0 兜底 8、>20 收 20
	if got := nodeInput(t, b.buildZImageWorkflow("p", 64, 64, 0, 0, nil), "10", "steps"); got != 8 {
		t.Errorf("z-image txt2img steps<=0 应兜底 8，got %v", got)
	}
	if got := nodeInput(t, b.buildZImageWorkflow("p", 64, 64, 0, 25, nil), "10", "steps"); got != 20 {
		t.Errorf("z-image txt2img steps>20 应收 20，got %v", got)
	}
}

func TestGoldenKreaImg2ImgWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildKreaImg2ImgWorkflow("测试 prompt", "坏手", 1024, 1024, 42, 8, nil, "ref.png", 0.65)
	want := `{"1":{"class_type":"LoadImage","inputs":{"image":"ref.png"}},"10":{"class_type":"KSampler","inputs":{"cfg":1,"denoise":0.65,"latent_image":["15",0],"model":["4",0],"negative":["13",0],"positive":["7",0],"sampler_name":"euler","scheduler":"simple","seed":42,"steps":8}},"11":{"class_type":"VAEDecode","inputs":{"samples":["10",0],"vae":["6",0]}},"12":{"class_type":"SaveImage","inputs":{"filename_prefix":"gaea","images":["11",0]}},"13":{"class_type":"ConditioningZeroOut","inputs":{"conditioning":["8",0]}},"15":{"class_type":"VAEEncode","inputs":{"pixels":["1",0],"vae":["6",0]}},"4":{"class_type":"UNETLoader","inputs":{"unet_name":"krea2_turbo_fp8_scaled.safetensors","weight_dtype":"default"}},"5":{"class_type":"CLIPLoader","inputs":{"clip_name":"qwen3vl_4b_fp8_scaled.safetensors","type":"krea2"}},"6":{"class_type":"VAELoader","inputs":{"vae_name":"qwen_image_vae.safetensors"}},"7":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":"测试 prompt"}},"8":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":"坏手"}}}`
	if got := marshalWorkflow(t, wf); got != want {
		t.Errorf("krea2 img2img 节点表漂移:\n got %s\nwant %s", got, want)
	}
}

func TestGoldenZImageImg2ImgWorkflow(t *testing.T) {
	b := &ComfyUIBackend{}
	wf := b.buildZImageImg2ImgWorkflow("测试 prompt", "坏手", 1024, 1024, 42, 8, nil, "ref.png", 0.65)
	want := `{"1":{"class_type":"LoadImage","inputs":{"image":"ref.png"}},"10":{"class_type":"KSampler","inputs":{"cfg":1,"denoise":0.65,"latent_image":["15",0],"model":["14",0],"negative":["13",0],"positive":["7",0],"sampler_name":"res_multistep","scheduler":"simple","seed":42,"steps":8}},"11":{"class_type":"VAEDecode","inputs":{"samples":["10",0],"vae":["6",0]}},"12":{"class_type":"SaveImage","inputs":{"filename_prefix":"gaea","images":["11",0]}},"13":{"class_type":"ConditioningZeroOut","inputs":{"conditioning":["8",0]}},"14":{"class_type":"ModelSamplingAuraFlow","inputs":{"model":["4",0],"shift":3}},"15":{"class_type":"VAEEncode","inputs":{"pixels":["1",0],"vae":["6",0]}},"4":{"class_type":"UNETLoader","inputs":{"unet_name":"z_image_turbo_bf16_完整版_效果最好.safetensors","weight_dtype":"default"}},"5":{"class_type":"CLIPLoader","inputs":{"clip_name":"z-image\\qwen_3_4b.safetensors","type":"lumina2"}},"6":{"class_type":"VAELoader","inputs":{"vae_name":"z-image-qwen.safetensors"}},"7":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":"测试 prompt"}},"8":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":"坏手"}}}`
	if got := marshalWorkflow(t, wf); got != want {
		t.Errorf("z-image img2img 节点表漂移:\n got %s\nwant %s", got, want)
	}
}

func TestGoldenZImageTxt2ImgWithLora(t *testing.T) {
	// LoRA 链插在 UNET 与 AuraFlow 之间：4 → 20(LoRA) → 14(AuraFlow) → KSampler
	b := &ComfyUIBackend{}
	wf := b.buildZImageWorkflow("p", 512, 512, 7, 8, []string{"zimage\\detail.safetensors"})
	want := `{"10":{"class_type":"KSampler","inputs":{"cfg":1,"denoise":1,"latent_image":["9",0],"model":["14",0],"negative":["13",0],"positive":["7",0],"sampler_name":"res_multistep","scheduler":"simple","seed":7,"steps":8}},"11":{"class_type":"VAEDecode","inputs":{"samples":["10",0],"vae":["6",0]}},"12":{"class_type":"SaveImage","inputs":{"filename_prefix":"gaea","images":["11",0]}},"13":{"class_type":"ConditioningZeroOut","inputs":{"conditioning":["8",0]}},"14":{"class_type":"ModelSamplingAuraFlow","inputs":{"model":["20",0],"shift":3}},"20":{"class_type":"LoraLoaderModelOnly","inputs":{"lora_name":"zimage\\detail.safetensors","model":["4",0],"strength_model":1}},"4":{"class_type":"UNETLoader","inputs":{"unet_name":"z_image_turbo_bf16_完整版_效果最好.safetensors","weight_dtype":"default"}},"5":{"class_type":"CLIPLoader","inputs":{"clip_name":"z-image\\qwen_3_4b.safetensors","type":"lumina2"}},"6":{"class_type":"VAELoader","inputs":{"vae_name":"z-image-qwen.safetensors"}},"7":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":"p"}},"8":{"class_type":"CLIPTextEncode","inputs":{"clip":["5",0],"text":""}},"9":{"class_type":"EmptySD3LatentImage","inputs":{"batch_size":1,"height":512,"width":512}}}`
	if got := marshalWorkflow(t, wf); got != want {
		t.Errorf("z-image txt2img+LoRA 节点表漂移:\n got %s\nwant %s", got, want)
	}
}

// TestWarmupDispatchMatchesTxt2imgRegistry AP7-03 行为面：Warmup 改经
// lookupTxt2imgBuilder 分发后，与历史手写 switch 逐 case 等价——提交的空跑
// 工作流 == 登记构建器在 (prompt="(warmup)", 64×64, seed 0, steps 1) 的产物。
func TestWarmupDispatchMatchesTxt2imgRegistry(t *testing.T) {
	var gotWorkflow map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/prompt":
			var body struct {
				Prompt map[string]interface{} `json:"prompt"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotWorkflow = body.Prompt
			w.Write([]byte(`{"prompt_id":"w"}`))
		case strings.HasPrefix(r.URL.Path, "/history/"):
			w.Write([]byte(`{"w":{"status":{"status_str":"success"},"outputs":{"12":{"images":[{"filename":"w.png","subfolder":"","type":"output"}]}}}}`))
		case r.URL.Path == "/view":
			w.Write([]byte("PNG"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.pollInterval = 20 * time.Millisecond // 测试提速（生产 2s）

	// krea2 精确名与 krea2-* 前缀都走 krea2 工作流（历史 HasPrefix 语义）
	for _, model := range []string{"krea2", "krea2-xl", "z-image-turbo", "flux"} {
		gotWorkflow = nil
		if err := b.Warmup(context.Background(), model); err != nil {
			t.Fatalf("Warmup(%s): %v", model, err)
		}
		builder, ok := lookupTxt2imgBuilder(model)
		if !ok {
			t.Fatalf("模型 %s 应已登记", model)
		}
		want := builder(b, "(warmup)", "", 64, 64, 0, 1, nil)
		gotJSON, wantJSON := marshalWorkflow(t, gotWorkflow), marshalWorkflow(t, want)
		if gotJSON != wantJSON {
			t.Errorf("Warmup(%s) 工作流与登记构建器 steps=1 不一致:\n got %s\nwant %s", model, gotJSON, wantJSON)
		}
	}

	// 逐 case 关键字段（不依赖 lookup 自身，防同源循环论证）。
	// gotWorkflow 经 JSON 解码，数字均为 float64。
	gotWorkflow = nil
	if err := b.Warmup(context.Background(), "z-image-turbo"); err != nil {
		t.Fatalf("Warmup(z-image-turbo): %v", err)
	}
	if got := nodeInput(t, gotWorkflow, "4", "unet_name"); got != "z_image_turbo_bf16_完整版_效果最好.safetensors" {
		t.Errorf("z-image 预热应加载 turbo 权重，got %v", got)
	}
	if got := nodeInput(t, gotWorkflow, "10", "steps"); got != float64(1) {
		t.Errorf("z-image 预热应 1 步，got %v", got)
	}
	if got := nodeInput(t, gotWorkflow, "14", "shift"); got != float64(3) {
		t.Errorf("z-image 预热应带 AuraFlow shift=3，got %v", got)
	}

	gotWorkflow = nil
	if err := b.Warmup(context.Background(), "flux"); err != nil {
		t.Fatalf("Warmup(flux): %v", err)
	}
	// flux 无 steps 参数（内部固定 4 步），历史预热即 4 步
	if got := nodeInput(t, gotWorkflow, "10", "steps"); got != float64(4) {
		t.Errorf("flux 预热应固定 4 步，got %v", got)
	}
	if got := nodeInput(t, gotWorkflow, "5", "type"); got != "flux" {
		t.Errorf("flux 预热 CLIP type 应为 flux，got %v", got)
	}

	// 空模型回落 krea2（历史行为）
	gotWorkflow = nil
	if err := b.Warmup(context.Background(), ""); err != nil {
		t.Fatalf("Warmup(\"\"), %v", err)
	}
	if got := nodeInput(t, gotWorkflow, "4", "unet_name"); got != "krea2_turbo_fp8_scaled.safetensors" {
		t.Errorf("空模型应回落 krea2 权重，got %v", got)
	}

	// 未登记模型：与历史 switch 相同的诚实拒绝（错误文案不变）
	if err := b.Warmup(context.Background(), "ghost-model"); err == nil || !strings.Contains(err.Error(), "未登记预热工作流") {
		t.Errorf("未知模型应拒绝，got %v", err)
	}
	// z-image 前缀不兼容（历史 switch 只认精确名，lookup 同）
	if err := b.Warmup(context.Background(), "z-image-turbo-x"); err == nil || !strings.Contains(err.Error(), "未登记预热工作流") {
		t.Errorf("z-image 非精确名应拒绝，got %v", err)
	}
}
