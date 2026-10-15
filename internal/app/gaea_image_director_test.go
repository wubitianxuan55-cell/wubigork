package app

// 绘梦导演循环核心测试：轨迹/重试/接受语义全钉在纯函数 runImageDirectorC 上
// （plan/generate/eval 三缝脚本化）；归一化/调整段/候选集小单测随后。

import (
	"context"
	"strings"
	"testing"
)

func dirPassVerdict() *ImageDirectorVerdict {
	return &ImageDirectorVerdict{Pass: true, Score: 88}
}

// okPlan 返回一个固定计划缝。
func okPlan(model string) func(context.Context, string, string, []string, string) (*ImageDirectorPlan, ImageDirectorStep) {
	return func(context.Context, string, string, []string, string) (*ImageDirectorPlan, ImageDirectorStep) {
		return &ImageDirectorPlan{Model: model, Prompt: "雨夜天台，冷色调"}, ImageDirectorStep{Stage: "plan", Detail: "计划就绪"}
	}
}

func TestImageDirector_FirstPass(t *testing.T) {
	var genPrompt string
	deps := imageDirectorDeps{
		plan: okPlan("krea2"),
		generate: func(model, prompt, negative, style string, loras []string, seed int, size string) ([]imageItem, string) {
			genPrompt = prompt
			return []imageItem{{FilePath: "C:/img/a.png", Seed: seed}}, ""
		},
		eval: func(string, imageItem) (*ImageDirectorVerdict, ImageDirectorStep) {
			return dirPassVerdict(), ImageDirectorStep{Stage: "eval", Detail: "88 分"}
		},
		randSeed: func() int { return 7 },
	}
	res, err := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "雨夜天台", MaxAttempts: 2}, []string{"krea2", "flux"}, deps)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accepted || res.Attempts != 1 {
		t.Errorf("accepted/attempts = %v/%d, want true/1", res.Accepted, res.Attempts)
	}
	if genPrompt != "雨夜天台，冷色调" {
		t.Errorf("首轮 prompt 应为计划原文，got %q", genPrompt)
	}
	if res.Verdict == nil || res.Verdict.Score != 88 {
		t.Errorf("verdict 未回传")
	}
	stages := stageList(res.Steps)
	if strings.Join(stages, ",") != "plan,generate,eval,done" {
		t.Errorf("轨迹 = %v", stages)
	}
}

func TestImageDirector_RetryWithAdjustmentThenPass(t *testing.T) {
	calls := 0
	var secondPrompt string
	var secondSeed int
	deps := imageDirectorDeps{
		plan: okPlan("krea2"),
		generate: func(model, prompt, negative, style string, loras []string, seed int, size string) ([]imageItem, string) {
			calls++
			if calls == 1 {
				return []imageItem{{FilePath: "a.png", Seed: seed}}, ""
			}
			secondPrompt = prompt
			secondSeed = seed
			return []imageItem{{FilePath: "b.png", Seed: seed}}, ""
		},
		eval: func(string, imageItem) (*ImageDirectorVerdict, ImageDirectorStep) {
			calls++
			if calls <= 2 { // 首轮 eval
				return &ImageDirectorVerdict{Pass: false, Score: 40, Issues: []string{"人脸崩坏", "手部畸形"}, Fix: "减少手部特写"},
					ImageDirectorStep{Stage: "eval", Detail: "40 分"}
			}
			return dirPassVerdict(), ImageDirectorStep{Stage: "eval", Detail: "90 分"}
		},
		randSeed: func() int { return 42 },
	}
	res, err := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "角色立绘", MaxAttempts: 2, Seed: 1}, []string{"krea2"}, deps)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accepted || res.Attempts != 2 {
		t.Errorf("accepted/attempts = %v/%d, want true/2", res.Accepted, res.Attempts)
	}
	// 调整段：计划原文 + issues + fix。
	if !strings.Contains(secondPrompt, "雨夜天台") && !strings.Contains(secondPrompt, "角色立绘") {
		t.Errorf("二轮 prompt 丢失计划原文：%q", secondPrompt)
	}
	if !strings.Contains(secondPrompt, "注意避免：人脸崩坏；手部畸形") || !strings.Contains(secondPrompt, "调整：减少手部特写") {
		t.Errorf("二轮 prompt 未含调整段：%q", secondPrompt)
	}
	// 种子重掷（首轮 seed=1 显式，二轮=注入的 42）。
	if secondSeed != 42 {
		t.Errorf("二轮种子 = %d, want 42（重掷）", secondSeed)
	}
	if stages := stageList(res.Steps); strings.Join(stages, ",") != "plan,generate,eval,adjust,generate,eval,done" {
		t.Errorf("轨迹 = %v", stages)
	}
}

func TestImageDirector_ExhaustedHonestFail(t *testing.T) {
	deps := imageDirectorDeps{
		plan: okPlan("krea2"),
		generate: func(model, prompt, negative, style string, loras []string, seed int, size string) ([]imageItem, string) {
			return []imageItem{{FilePath: "x.png"}}, ""
		},
		eval: func(string, imageItem) (*ImageDirectorVerdict, ImageDirectorStep) {
			return &ImageDirectorVerdict{Pass: false, Score: 30}, ImageDirectorStep{Stage: "eval", Detail: "30 分"}
		},
		randSeed: func() int { return 9 },
	}
	res, err := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "something", MaxAttempts: 2}, []string{"krea2"}, deps)
	if err != nil {
		t.Fatalf("用尽轮次应返回末轮结果而非 error：%v", err)
	}
	if res.Accepted {
		t.Errorf("自评未过不得标 accepted")
	}
	last := res.Steps[len(res.Steps)-1]
	if last.Stage != "done" || !strings.Contains(last.Detail, "诚实口径") {
		t.Errorf("末步应显式「未达标」： %+v", last)
	}
}

func TestImageDirector_EvalUnavailableAcceptsHonestly(t *testing.T) {
	deps := imageDirectorDeps{
		plan: okPlan("krea2"),
		generate: func(model, prompt, negative, style string, loras []string, seed int, size string) ([]imageItem, string) {
			return []imageItem{{}}, "" // 无本地文件路径
		},
		eval: func(string, imageItem) (*ImageDirectorVerdict, ImageDirectorStep) {
			return nil, ImageDirectorStep{Stage: "eval", Detail: "自评跳过：本图无本地文件路径"}
		},
		randSeed: func() int { return 3 },
	}
	res, err := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "x"}, []string{"krea2"}, deps)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accepted || res.Verdict != nil {
		t.Errorf("自评不可用应诚实放行（accepted=true, verdict=nil），got %v/%v", res.Accepted, res.Verdict)
	}
}

func TestImageDirector_PlanFailureSurfaces(t *testing.T) {
	deps := imageDirectorDeps{
		plan: func(context.Context, string, string, []string, string) (*ImageDirectorPlan, ImageDirectorStep) {
			return nil, ImageDirectorStep{Stage: "plan", Detail: "未找到可用 LLM"}
		},
		generate: func(string, string, string, string, []string, int, string) ([]imageItem, string) {
			t.Fatal("解析失败不得进入生成")
			return nil, ""
		},
		eval:     func(string, imageItem) (*ImageDirectorVerdict, ImageDirectorStep) { return nil, ImageDirectorStep{} },
		randSeed: func() int { return 1 },
	}
	_, err := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "x"}, []string{"krea2"}, deps)
	if err == nil || !strings.Contains(err.Error(), "未找到可用 LLM") {
		t.Errorf("解析失败原因应透出，got %v", err)
	}
}

func TestImageDirector_GenFailureRetriesThenErrors(t *testing.T) {
	calls := 0
	deps := imageDirectorDeps{
		plan: okPlan("krea2"),
		generate: func(string, string, string, string, []string, int, string) ([]imageItem, string) {
			calls++
			return nil, "ComfyUI 未运行"
		},
		eval: func(string, imageItem) (*ImageDirectorVerdict, ImageDirectorStep) {
			t.Fatal("生成失败不得进入自评")
			return nil, ImageDirectorStep{}
		},
		randSeed: func() int { return 5 },
	}
	_, err := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "x", MaxAttempts: 2}, []string{"krea2"}, deps)
	if err == nil || !strings.Contains(err.Error(), "ComfyUI 未运行") {
		t.Errorf("末轮失败原因应透出，got %v", err)
	}
	if calls != 2 {
		t.Errorf("生成调用 = %d, want 2（重试一轮）", calls)
	}
	if stages := stageList(mustSteps(t, deps)); !strings.Contains(strings.Join(stages, ","), "adjust") {
		t.Errorf("失败重试轨迹缺 adjust")
	}
}

func TestImageDirector_EmptyNeedAndClamps(t *testing.T) {
	deps := imageDirectorDeps{randSeed: func() int { return 1 }}
	if _, err := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "  "}, []string{"krea2"}, deps); err == nil {
		t.Errorf("空需求应报错")
	}
	// MaxAttempts=5 → 钳 2：generate 计数到 2 即停（eval 恒未过强制重试）。
	n := 0
	deps.plan = okPlan("krea2")
	deps.generate = func(string, string, string, string, []string, int, string) ([]imageItem, string) {
		n++
		return []imageItem{{FilePath: "x.png"}}, ""
	}
	deps.eval = func(string, imageItem) (*ImageDirectorVerdict, ImageDirectorStep) {
		return &ImageDirectorVerdict{Pass: false, Score: 10}, ImageDirectorStep{Stage: "eval", Detail: "10 分"}
	}
	res, err := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "x", MaxAttempts: 5}, []string{"krea2"}, deps)
	if err != nil || res.Attempts != 2 {
		t.Errorf("钳制失效：attempts=%d err=%v", res.Attempts, err)
	}
	// Aspect 映射：16:9 → 1280x720。
	var gotSize string
	deps.generate = func(_ string, _ string, _ string, _ string, _ []string, _ int, size string) ([]imageItem, string) {
		gotSize = size
		return []imageItem{{}}, ""
	}
	_, _ = runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "x", Aspect: "16:9", MaxAttempts: 1}, []string{"krea2"}, deps)
	if gotSize != "1280x720" {
		t.Errorf("16:9 尺寸 = %q, want 1280x720", gotSize)
	}
	// 未知 Aspect → 1:1。
	_, _ = runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "x", Aspect: "7:3", MaxAttempts: 1}, []string{"krea2"}, deps)
	if gotSize != "1024x1024" {
		t.Errorf("未知比例应回落 1:1，got %q", gotSize)
	}
}

func TestNormalizeImageDirectorPlan(t *testing.T) {
	p := ImageDirectorPlan{Model: "hallucinated-model", Prompt: "  星空  ", Negative: " 模糊 ", Style: " 水彩 ", Loras: []string{" a.safetensors ", "", "b.safetensors"}}
	if err := normalizeImageDirectorPlan(&p, "krea2", []string{"krea2", "flux"}); err != nil {
		t.Fatal(err)
	}
	if p.Model != "krea2" {
		t.Errorf("越界模型应回落起始模型，got %q", p.Model)
	}
	if p.Prompt != "星空" || p.Negative != "模糊" || p.Style != "水彩" {
		t.Errorf("字段裁剪失效：%+v", p)
	}
	if len(p.Loras) != 2 || p.Loras[0] != "a.safetensors" {
		t.Errorf("LoRA 裁剪失效：%v", p.Loras)
	}
	empty := ImageDirectorPlan{Prompt: "   "}
	if err := normalizeImageDirectorPlan(&empty, "krea2", nil); err == nil {
		t.Errorf("空 prompt 应拒")
	}
}

func TestImageDirectorAdjustment(t *testing.T) {
	v := &ImageDirectorVerdict{Issues: []string{"构图偏"}, Fix: " 居中构图 "}
	got := imageDirectorAdjustment(v)
	if !strings.Contains(got, "注意避免：构图偏") || !strings.Contains(got, "调整：居中构图") {
		t.Errorf("调整段 = %q", got)
	}
	if imageDirectorAdjustment(&ImageDirectorVerdict{}) != "" {
		t.Errorf("空 verdict 调整段应为空串")
	}
}

func TestIsLikelyImageModelID(t *testing.T) {
	yes := []string{"grok-imagine-image", "flux-dev", "Krea2-Turbo", "z-image-turbo", "sdxl-base"}
	no := []string{"qwen3-8b", "whisper-base", ""}
	for _, id := range yes {
		if !isLikelyImageModelID(strings.ToLower(id)) {
			t.Errorf("%q 应判为图片模型", id)
		}
	}
	for _, id := range no {
		if isLikelyImageModelID(id) {
			t.Errorf("%q 不应判为图片模型", id)
		}
	}
}

func stageList(steps []ImageDirectorStep) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		out = append(out, s.Stage)
	}
	return out
}

func mustSteps(t *testing.T, deps imageDirectorDeps) []ImageDirectorStep {
	t.Helper()
	res, _ := runImageDirectorC(context.Background(), ImageDirectorRequest{Need: "x", MaxAttempts: 2}, []string{"krea2"}, deps)
	return res.Steps
}
