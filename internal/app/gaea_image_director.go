package app

// gaea_image_director.go — 绘梦导演 Agent（11.6 并行池清欠；roadmap 绘梦#1）。
//
// 原判据：解析需求→自动拼装 ComfyUI 工作流→自评→带参重试。四段全部踩在
// 既有缝上：解析=routeModel LLM×util.RetryJSON（办公功能级路由同款）；拼装/
// 提交=generateImageInternal 生成链（绘梦/原罪同一条链，产物归属/落盘/台账
// /进度全继承）；自评=vision.RecognizeImage（视觉后端 config 驱动）；重试=
// 有界循环（默认 2 轮，钳 1..3），调整面=种子重掷+把自评 issues/fix 追加进
// 提示词（诚实口径：不改模型不换后端，模型在解析轮一次选定）。
//
// 诚实边界：自评不可用（无本地文件/视觉后端拒绝/输出不可解析）时如实记轨迹
// 并接受当前图（宁可诚实放行也不假装评过）；全部轮次失败时返回完整轨迹不
// 吞原因。循环核心 runImageDirectorC 对 plan/generate/eval 三缝依赖注入——
// 轨迹/重试/接受语义全部钉在纯测试上，App 层只做真实接线。

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/vision"
	"github.com/gaea/gaea/internal/util"
)

// ImageDirectorRequest 导演需求（自然语言）。
type ImageDirectorRequest struct {
	Need string `json:"need"`
	// Model 起始模型（前端当前选择；空=候选集第一个）。解析轮 LLM 可在候选集
	// 内改选，越界一律回落本值。
	Model string `json:"model"`
	// Aspect 构图比："1:1"/"16:9"/"9:16"/"4:3"/"3:4"，空=1:1。
	Aspect string `json:"aspect"`
	// MaxAttempts 生成-自评轮数上限，钳 1..3，0=2。
	MaxAttempts int `json:"maxAttempts"`
	// Seed 起始种子；0=随机。
	Seed int `json:"seed"`
}

// ImageDirectorPlan 解析轮产物（LLM 结构化）。
type ImageDirectorPlan struct {
	Model    string   `json:"model"`
	Prompt   string   `json:"prompt"`
	Negative string   `json:"negative"`
	Style    string   `json:"style"`
	Loras    []string `json:"loras"`
	Reason   string   `json:"reason"`
}

// ImageDirectorVerdict 自评轮产物（视觉模型结构化）。
type ImageDirectorVerdict struct {
	Pass   bool     `json:"pass"`
	Score  int      `json:"score"`
	Issues []string `json:"issues"`
	Fix    string   `json:"fix"`
}

// ImageDirectorStep 轨迹一步（人可读）。
type ImageDirectorStep struct {
	Stage   string `json:"stage"` // plan|generate|eval|adjust|done
	Attempt int    `json:"attempt"`
	Detail  string `json:"detail"`
}

// ImageDirectorResult 导演回执：最终图 + 计划 + 末轮自评 + 全程轨迹。
type ImageDirectorResult struct {
	Images   []imageItem           `json:"images"`
	Plan     *ImageDirectorPlan    `json:"plan"`
	Verdict  *ImageDirectorVerdict `json:"verdict"`
	Steps    []ImageDirectorStep   `json:"steps"`
	Attempts int                   `json:"attempts"`
	Accepted bool                  `json:"accepted"`
}

// imageDirectorAspects 构图比 → 尺寸（parseSize 钳 64..2048 口径内的常用档）。
var imageDirectorAspects = map[string]string{
	"1:1":  "1024x1024",
	"4:3":  "1152x864",
	"3:4":  "864x1152",
	"16:9": "1280x720",
	"9:16": "720x1280",
}

// imageDirectorDeps 导演循环的三缝（真实接线见 GaeaImageDirector）。
type imageDirectorDeps struct {
	// plan 需求 → 结构化计划；plan=nil = 解析失败（step.Detail 人可读原因）。
	plan func(ctx context.Context, need, fallbackModel string, models []string, size string) (*ImageDirectorPlan, ImageDirectorStep)
	// generate 单轮生成；errMsg 非空 = 本轮失败原因。
	generate func(model, prompt, negative, style string, loras []string, seed int, size string) ([]imageItem, string)
	// eval 自评；verdict=nil = 自评不可用（step.Detail 人可读原因）。
	eval func(need string, img imageItem) (*ImageDirectorVerdict, ImageDirectorStep)
	// randSeed 重掷种子（注入保测试确定）。
	randSeed func() int
}

// runImageDirectorC 导演循环核心（纯函数）：解析 → [生成 → 自评 → (调整重试)]^n。
// 全程轨迹落 Steps；发射事件归调用方（App 层在缝内自发）。
func runImageDirectorC(ctx context.Context, req ImageDirectorRequest, models []string, deps imageDirectorDeps) (ImageDirectorResult, error) {
	res := ImageDirectorResult{Steps: []ImageDirectorStep{}}
	need := strings.TrimSpace(req.Need)
	if need == "" {
		return res, fmt.Errorf("需求描述为空：告诉导演要画什么")
	}
	maxAttempts := req.MaxAttempts
	if maxAttempts == 0 || maxAttempts < 1 || maxAttempts > 3 {
		maxAttempts = 2
	}
	size := imageDirectorAspects[req.Aspect]
	if size == "" {
		size = imageDirectorAspects["1:1"]
	}
	fallbackModel := req.Model
	if fallbackModel == "" && len(models) > 0 {
		fallbackModel = models[0]
	}
	curSeed := req.Seed
	if curSeed == 0 {
		curSeed = deps.randSeed()
	}

	// ── 解析轮 ──
	plan, step := deps.plan(ctx, need, fallbackModel, models, size)
	res.Steps = append(res.Steps, step)
	if plan == nil {
		return res, fmt.Errorf("%s", step.Detail)
	}
	res.Plan = plan

	// ── 生成-自评有界循环 ──
	// 模型解析轮一次选定（plan.Model 已归一），轮间不改。
	curPrompt := plan.Prompt
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		res.Attempts = attempt
		res.Steps = append(res.Steps, ImageDirectorStep{Stage: "generate", Attempt: attempt,
			Detail: fmt.Sprintf("生成中：%s（%s，seed %d）", plan.Model, size, curSeed)})

		images, errMsg := deps.generate(plan.Model, curPrompt, plan.Negative, plan.Style, plan.Loras, curSeed, size)
		if errMsg != "" {
			res.Steps = append(res.Steps, ImageDirectorStep{Stage: "generate", Attempt: attempt, Detail: errMsg})
			if attempt < maxAttempts {
				curSeed = deps.randSeed()
				res.Steps = append(res.Steps, ImageDirectorStep{Stage: "adjust", Attempt: attempt, Detail: "生成失败，换种子重试"})
				continue
			}
			return res, fmt.Errorf("%s", errMsg)
		}
		if len(images) == 0 {
			res.Steps = append(res.Steps, ImageDirectorStep{Stage: "generate", Attempt: attempt, Detail: "生成返回空结果"})
			if attempt < maxAttempts {
				curSeed = deps.randSeed()
				continue
			}
			return res, fmt.Errorf("图片生成失败：全部轮次空结果")
		}
		res.Images = images

		// ── 自评轮 ──
		verdict, vstep := deps.eval(need, images[0])
		res.Steps = append(res.Steps, vstep)
		if verdict == nil {
			// 自评不可用：诚实放行当前图（原因已在轨迹）。
			res.Accepted = true
			res.Steps = append(res.Steps, ImageDirectorStep{Stage: "done", Attempt: attempt, Detail: "自评不可用，接受当前图"})
			return res, nil
		}
		res.Verdict = verdict
		if verdict.Pass {
			res.Accepted = true
			res.Steps = append(res.Steps, ImageDirectorStep{Stage: "done", Attempt: attempt,
				Detail: fmt.Sprintf("第 %d 轮自评通过（%d 分）", attempt, verdict.Score)})
			return res, nil
		}
		if attempt < maxAttempts {
			curPrompt = plan.Prompt + imageDirectorAdjustment(verdict)
			curSeed = deps.randSeed()
			res.Steps = append(res.Steps, ImageDirectorStep{Stage: "adjust", Attempt: attempt, Detail: "自评未过，按调整建议重试"})
			continue
		}
	}
	res.Steps = append(res.Steps, ImageDirectorStep{Stage: "done", Attempt: maxAttempts,
		Detail: fmt.Sprintf("已用尽 %d 轮自评未过，返回末轮结果（诚实口径：未达标）", maxAttempts)})
	return res, nil
}

// GaeaImageDirector 绘梦导演：解析需求 → 生成 → 视觉自评 → 带参重试。
// 长调用（生成秒级~分钟级），进度经 image-director-step 事件与 ComfyUI 任务
// 进度双通道可见；返回完整轨迹。
func (a *mediaState) GaeaImageDirector(req ImageDirectorRequest) (ImageDirectorResult, error) {
	models := a.imageDirectorCandidates(req.Model)
	emit := func(stage string, attempt int, detail string) {
		a.emit("image-director-step", map[string]interface{}{"stage": stage, "attempt": attempt, "detail": detail})
	}
	deps := imageDirectorDeps{
		plan: func(ctx context.Context, need, fallbackModel string, ms []string, size string) (*ImageDirectorPlan, ImageDirectorStep) {
			p, s := a.imageDirectorPlan(ctx, need, fallbackModel, ms, size)
			emit("plan", 0, s.Detail)
			return p, s
		},
		generate: func(model, prompt, negative, style string, loras []string, seed int, size string) ([]imageItem, string) {
			out, err := a.generateImageInternal(imageGenInternal{
				prompt: prompt, negative: negative, size: size, style: style,
				model: model, seed: seed, n: 1, lora: strings.Join(loras, ","),
				sourceBoard: "imagegen",
			})
			if err != nil {
				return nil, "生成链错误：" + err.Error()
			}
			if msg, ok := out["error"].(string); ok {
				return nil, msg
			}
			images, _ := out["images"].([]imageItem)
			return images, ""
		},
		eval: func(need string, img imageItem) (*ImageDirectorVerdict, ImageDirectorStep) {
			v, s := a.imageDirectorEval(a.ctx, need, img)
			emit("eval", 0, s.Detail)
			return v, s
		},
		randSeed: func() int { return rand.Intn(1 << 31) },
	}
	res, err := runImageDirectorC(a.ctx, req, models, deps)
	for _, s := range res.Steps {
		if s.Stage == "done" {
			emit("done", s.Attempt, s.Detail)
		}
	}
	return res, err
}

// imageDirectorCandidates 导演可选模型集：comfyui=登记的文生图模型族（ai 包
// 单源）；其它后端=引擎模型清单里的图片模型，拿不到则退 [起始模型]。
func (a *mediaState) imageDirectorCandidates(fallback string) []string {
	backend := ""
	if a.cfg != nil {
		backend = a.cfg.ImageBackend
	}
	if backend == ai.ImageBackendTypeComfyUI {
		return ai.ComfyUITxt2imgModels()
	}
	if a.engineMgr != nil && backend != "" {
		if eng, ok := a.engineMgr.GetEngine(backend); ok {
			var out []string
			for _, m := range eng.Models {
				id := strings.TrimSpace(m.ID)
				if id != "" && isLikelyImageModelID(strings.ToLower(id)) {
					out = append(out, id)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	if fallback != "" {
		return []string{fallback}
	}
	return []string{"krea2"}
}

// isLikelyImageModelID 粗判模型 ID 是否图片族（与前端 classifyModel 口径对齐
// 的保守版：认 imagine/image/flux/krea/z-image/sdxl/sd3 词根）。
func isLikelyImageModelID(id string) bool {
	for _, kw := range []string{"imagine", "image", "flux", "krea", "z-image", "sdxl", "sd3", "dalle"} {
		if strings.Contains(id, kw) {
			return true
		}
	}
	return false
}

// imageDirectorPlan 解析轮：需求 → 结构化计划（RetryJSON 功能级路由）。
// 返回 nil plan 时 step.Detail 是人可读失败原因。
func (a *mediaState) imageDirectorPlan(ctx context.Context, need, fallbackModel string, models []string, size string) (*ImageDirectorPlan, ImageDirectorStep) {
	eng, model, _ := a.routeModel("imagegen")
	if eng == "" || model == "" {
		return nil, ImageDirectorStep{Stage: "plan", Detail: "未找到可用 LLM（模型中心未配置或离线），无法解析需求"}
	}
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.clientRef().ChatSimpleStreamWithOptions(ctx, model, sys, usr, ai.ChatSimpleOptions{
			EngineID:    eng,
			Feature:     "imagegen",
			Temperature: 0.3,
			MaxTokens:   1024,
		})
	}
	system := "你是绘梦导演：把用户的自然语言需求变成一次文生图的可执行计划。" +
		"只输出 JSON（```json 代码块包裹），字段：" +
		`{"model":"从候选模型里选一个","prompt":"优化后的画面提示词，具体到主体/构图/光线/色调，不超过 200 字","negative":"负面提示词，没有就空串","style":"风格后缀，可空","loras":["LoRA 文件名，没有就空数组"],"reason":"一句话选型理由"}` + "\n" +
		"候选模型：" + strings.Join(models, "、") + "。画面尺寸将是 " + size + "，构图按此比例设计。不要输出 JSON 以外的内容。"
	jsonStr, err := util.RetryJSON(ctx, caller, system, "需求："+need, 2)
	if err != nil {
		return nil, ImageDirectorStep{Stage: "plan", Detail: "需求解析失败：" + err.Error()}
	}
	var plan ImageDirectorPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, ImageDirectorStep{Stage: "plan", Detail: "解析计划 JSON 失败：" + err.Error()}
	}
	if err := normalizeImageDirectorPlan(&plan, fallbackModel, models); err != nil {
		return nil, ImageDirectorStep{Stage: "plan", Detail: err.Error()}
	}
	return &plan, ImageDirectorStep{Stage: "plan", Detail: fmt.Sprintf("计划就绪：%s（%s）", plan.Model, plan.Reason)}
}

// normalizeImageDirectorPlan 计划归一（纯函数）：字段裁剪 + 模型越界回落
// （LLM 幻觉防护——候选集外一律回起始模型）。prompt 空 = 不可执行。
func normalizeImageDirectorPlan(plan *ImageDirectorPlan, fallbackModel string, models []string) error {
	plan.Prompt = strings.TrimSpace(plan.Prompt)
	if plan.Prompt == "" {
		return fmt.Errorf("解析出的提示词为空")
	}
	if !containsString(models, plan.Model) {
		plan.Model = fallbackModel
	}
	plan.Negative = strings.TrimSpace(plan.Negative)
	plan.Style = strings.TrimSpace(plan.Style)
	plan.Reason = strings.TrimSpace(plan.Reason)
	var loras []string
	for _, l := range plan.Loras {
		if l = strings.TrimSpace(l); l != "" {
			loras = append(loras, l)
		}
	}
	plan.Loras = loras
	return nil
}

// imageDirectorEval 自评轮：视觉模型按需求打分。verdict 为 nil = 自评不可用
// （无本地文件/视觉失败/输出不可解析），原因记入 step。
func (a *mediaState) imageDirectorEval(ctx context.Context, need string, img imageItem) (*ImageDirectorVerdict, ImageDirectorStep) {
	path := img.FilePath
	if path == "" {
		return nil, ImageDirectorStep{Stage: "eval", Detail: "自评跳过：本图无本地文件路径"}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	prompt := "你是绘画评审。需求描述：" + need + "\n" +
		"请评审这张图是否满足需求，只输出 JSON：{\"pass\":布尔,\"score\":0到100整数,\"issues\":[\"不满足点的短句\"],\"fix\":\"下一轮生成的调整建议，一句话\"}。明显满足则 pass=true。"
	text, err := vision.RecognizeImage(ctx, path, prompt)
	if err != nil {
		return nil, ImageDirectorStep{Stage: "eval", Detail: "自评失败：" + err.Error()}
	}
	jsonStr := util.ExtractJSON(text)
	if jsonStr == "" {
		return nil, ImageDirectorStep{Stage: "eval", Detail: "自评输出不可解析（无 JSON），原文截断：" + util.Truncate(text, 120)}
	}
	var v ImageDirectorVerdict
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return nil, ImageDirectorStep{Stage: "eval", Detail: "自评 JSON 解析失败：" + err.Error()}
	}
	detail := fmt.Sprintf("自评 %d 分（pass=%v）", v.Score, v.Pass)
	if len(v.Issues) > 0 {
		detail += "：" + strings.Join(v.Issues, "；")
	}
	return &v, ImageDirectorStep{Stage: "eval", Detail: detail}
}

// imageDirectorAdjustment 把自评结论折算成下一轮的提示词追加段。
func imageDirectorAdjustment(v *ImageDirectorVerdict) string {
	var b strings.Builder
	if len(v.Issues) > 0 {
		b.WriteString("。注意避免：")
		b.WriteString(strings.Join(v.Issues, "；"))
	}
	if fix := strings.TrimSpace(v.Fix); fix != "" {
		b.WriteString("。调整：")
		b.WriteString(fix)
	}
	return b.String()
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
