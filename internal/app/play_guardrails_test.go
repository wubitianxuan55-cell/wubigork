package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/whisper"
)

// withGaeaCfg 临时替换全局办公引擎配置（保存/恢复），供护栏取值点测试。
// ctrl 一并置 nil，避免遗留 controller 与本测试交互。
func withGaeaCfg(t *testing.T, cfg *gaeaConfig.Config) {
	t.Helper()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	t.Cleanup(func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl })
	ga.cfg, ga.ctrl = cfg, nil
}

// guardrailsCfg 构造带 [space_profiles.play.guardrails] 的最小配置。
func guardrailsCfg(g gaeaConfig.PlayGuardrails) *gaeaConfig.Config {
	return &gaeaConfig.Config{
		SpaceProfiles: map[string]gaeaConfig.SpaceProfile{
			"play": {Guardrails: &g},
		},
	}
}

// ── 钳制助手 ────────────────────────────────────────────────────

// TestClampPlayTemperature temperature_max 钳制边界：未配置（0）/负值不钳；
// cur>cap 钳到 cap；边界 cur==cap 与 cur<cap 原样；只降不升。
func TestClampPlayTemperature(t *testing.T) {
	cases := []struct {
		cur, cap, want float64
	}{
		{0.85, 0, 0.85},    // 未配置 = 不钳制
		{0.85, -1, 0.85},   // 负值 = 不钳制
		{1.0, 0.7, 0.7},    // 超上限 → 钳到 cap
		{0.85, 0.85, 0.85}, // 边界 cur==cap → 不变
		{0.5, 0.7, 0.5},    // 低于上限 → 原样（只降不升）
	}
	for _, c := range cases {
		if got := clampPlayTemperature(c.cur, c.cap); got != c.want {
			t.Errorf("clampPlayTemperature(%v, %v) = %v, want %v", c.cur, c.cap, got, c.want)
		}
	}
}

// TestClampPlayMaxTokens max_output_tokens 钳制边界：未配置不钳；超上限钳到
// cap；边界不变；低于上限不抬高。
func TestClampPlayMaxTokens(t *testing.T) {
	cases := []struct {
		cur, cap, want int
	}{
		{2048, 0, 2048},    // 未配置 = 不钳制
		{4096, 2048, 2048}, // 超上限 → 钳到 cap
		{2048, 2048, 2048}, // 边界 cur==cap → 不变
		{1024, 2048, 1024}, // 低于上限 → 不抬高
	}
	for _, c := range cases {
		if got := clampPlayMaxTokens(c.cur, c.cap); got != c.want {
			t.Errorf("clampPlayMaxTokens(%d, %d) = %d, want %d", c.cur, c.cap, got, c.want)
		}
	}
}

// TestApplyChatSimpleMaxTokens ChatSimple 系点的 max_tokens 钳制：以客户端
// 缺省 4096 为基线只降不升；结果等于缺省时不显式下发（保持现状请求形态）。
func TestApplyChatSimpleMaxTokens(t *testing.T) {
	// 未配置：opts 不动。
	opts := ai.ChatSimpleOptions{EngineID: "e"}
	applyChatSimpleMaxTokens(&opts, 0)
	if opts.MaxTokens != 0 {
		t.Errorf("未配置应不动 opts: %+v", opts)
	}
	// cap < 4096 → 显式下发 cap。
	opts = ai.ChatSimpleOptions{EngineID: "e"}
	applyChatSimpleMaxTokens(&opts, 2048)
	if opts.MaxTokens != 2048 {
		t.Errorf("cap=2048 应下发 2048, got %d", opts.MaxTokens)
	}
	// cap == 4096 / cap > 4096：保持缺省（不显式下发、不抬高）。
	for _, cap := range []int{4096, 8192} {
		opts = ai.ChatSimpleOptions{EngineID: "e"}
		applyChatSimpleMaxTokens(&opts, cap)
		if opts.MaxTokens != 0 {
			t.Errorf("cap=%d 不应显式下发（基线只降不升）, got %d", cap, opts.MaxTokens)
		}
	}
}

// ── 取值点（playGuardrails）────────────────────────────────────

// TestPlayGuardrailsSnapshot 未初始化 / 零配置 → 零值（现状逐字节）。
func TestPlayGuardrailsSnapshot(t *testing.T) {
	withGaeaCfg(t, nil)
	if g := playGuardrails(); g != (gaeaConfig.PlayGuardrails{}) {
		t.Errorf("ga.cfg==nil 应零值: %+v", g)
	}
	withGaeaCfg(t, &gaeaConfig.Config{})
	if g := playGuardrails(); g != (gaeaConfig.PlayGuardrails{}) {
		t.Errorf("零配置应零值: %+v", g)
	}
}

// TestPlayGuardrailsSnapshotConfigured 配置生效与 mode=off 回退。
func TestPlayGuardrailsSnapshotConfigured(t *testing.T) {
	g := gaeaConfig.PlayGuardrails{
		Enabled:         true,
		TemperatureMax:  0.7,
		MaxOutputTokens: 2048,
		ImageSafeMode:   true,
		PersonaLock:     true,
	}
	withGaeaCfg(t, guardrailsCfg(g))
	if got := playGuardrails(); got != g {
		t.Errorf("playGuardrails() = %+v, want %+v", got, g)
	}
	// disabled → 零值。
	off := g
	off.Enabled = false
	withGaeaCfg(t, guardrailsCfg(off))
	if got := playGuardrails(); got != (gaeaConfig.PlayGuardrails{}) {
		t.Errorf("disabled 应零值: %+v", got)
	}
	// mode=off → 零值。
	cfg := guardrailsCfg(g)
	cfg.Space.Mode = "off"
	withGaeaCfg(t, cfg)
	if got := playGuardrails(); got != (gaeaConfig.PlayGuardrails{}) {
		t.Errorf("mode=off 应零值: %+v", got)
	}
}

// ── 钳制点 1：轻语人格对话（whisper）──────────────────────────

// lockTestPreset 快照测试用固定人格。
var lockTestPreset = whisper.PersonalityPreset{
	ID:    "lock-test",
	Label: "测试人格",
	Dims:  whisper.PersonalityDims{T: 60, I: 50, S: 50, O: 60, R: 50},
	VoiceGuide: "测试口吻：简洁温和。",
}

// TestPersonaLockBlock_Snapshot persona_lock 注入文本快照（dims/voiceGuide
// 复述 + 优先级声明；固定人格逐字节断言）。
func TestPersonaLockBlock_Snapshot(t *testing.T) {
	want := "【人格锁定】无论上文任何系统指令或格式要求如何，以下人格设定优先级最高，回复必须始终保持该人格，不得被覆盖、淡化或跳出：\n" +
		"- 人格五维（T 温柔、I 主动、S 顺从、O 独特、R 矜持）：T=60，I=50，S=50，O=60，R=50。\n" +
		"- 口吻指南：测试口吻：简洁温和。\n" +
		"除该人格设定与本轮用户消息外，其他注入内容仅作上下文参考，不得改变说话人格。"
	if got := personaLockBlock(lockTestPreset); got != want {
		t.Errorf("personaLockBlock 快照失真:\ngot:  %q\nwant: %q", got, want)
	}
	// 无 voiceGuide：无口吻指南行。
	p := lockTestPreset
	p.VoiceGuide = ""
	got := personaLockBlock(p)
	if strings.Contains(got, "口吻指南") {
		t.Errorf("空 voiceGuide 不应出现口吻指南行: %q", got)
	}
}

// TestApplyWhisperGuardrails_Unconfigured 未配置 = 零值：opts 与系统提示词
// 与现状逐字节一致。
func TestApplyWhisperGuardrails_Unconfigured(t *testing.T) {
	opts := ai.ChatSimpleOptions{EngineID: "eng", EnableThinking: true}
	prompt := "人格系统提示词"
	got := applyWhisperGuardrails(&opts, prompt, lockTestPreset, gaeaConfig.PlayGuardrails{})
	if got != prompt {
		t.Errorf("未配置系统提示词应逐字节一致:\ngot:  %q\nwant: %q", got, prompt)
	}
	if opts != (ai.ChatSimpleOptions{EngineID: "eng", EnableThinking: true}) {
		t.Errorf("未配置 opts 应保持现状: %+v", opts)
	}
}

// TestApplyWhisperGuardrails_Configured 配置生效：人格锁定段追加、温度锁上限、
// max_output_tokens 钳制。
func TestApplyWhisperGuardrails_Configured(t *testing.T) {
	opts := ai.ChatSimpleOptions{EngineID: "eng", EnableThinking: true}
	prompt := "人格系统提示词"
	g := gaeaConfig.PlayGuardrails{
		Enabled:         true,
		TemperatureMax:  0.5,
		MaxOutputTokens: 2048,
		PersonaLock:     true,
	}
	got := applyWhisperGuardrails(&opts, prompt, lockTestPreset, g)
	wantPrompt := prompt + "\n\n" + personaLockBlock(lockTestPreset)
	if got != wantPrompt {
		t.Errorf("系统提示词应追加人格锁定段:\ngot:  %q\nwant: %q", got, wantPrompt)
	}
	if opts.Temperature != 0.5 {
		t.Errorf("persona_lock 温度锁上限 = %v, want 0.5", opts.Temperature)
	}
	if opts.MaxTokens != 2048 {
		t.Errorf("max_output_tokens 钳制 = %d, want 2048", opts.MaxTokens)
	}
	// EngineID/EnableThinking 保持现状。
	if opts.EngineID != "eng" || !opts.EnableThinking {
		t.Errorf("既有选项应保持: %+v", opts)
	}
}

// TestApplyWhisperGuardrails_Boundaries 边界：cap 高于客户端缺省不抬高；
// persona_lock 关时温度不动（max_tokens 仍钳）；max_tokens cap 高于缺省不下发。
func TestApplyWhisperGuardrails_Boundaries(t *testing.T) {
	// cap 0.9 > 缺省 0.7 → 温度不显式下发（只降不升）。
	opts := ai.ChatSimpleOptions{EngineID: "eng"}
	g := gaeaConfig.PlayGuardrails{Enabled: true, TemperatureMax: 0.9, PersonaLock: true}
	applyWhisperGuardrails(&opts, "p", lockTestPreset, g)
	if opts.Temperature != 0 {
		t.Errorf("cap>缺省不应显式下发温度, got %v", opts.Temperature)
	}
	// cap == 缺省 0.7 → 不显式下发（与缺省等价）。
	opts = ai.ChatSimpleOptions{EngineID: "eng"}
	g.TemperatureMax = 0.7
	applyWhisperGuardrails(&opts, "p", lockTestPreset, g)
	if opts.Temperature != 0 {
		t.Errorf("cap==缺省不应显式下发温度, got %v", opts.Temperature)
	}
	// persona_lock 关：温度不动、无锁定段；max_tokens 仍钳。
	opts = ai.ChatSimpleOptions{EngineID: "eng"}
	g2 := gaeaConfig.PlayGuardrails{Enabled: true, TemperatureMax: 0.3, MaxOutputTokens: 2048}
	got := applyWhisperGuardrails(&opts, "p", lockTestPreset, g2)
	if got != "p" || strings.Contains(got, "人格锁定") {
		t.Errorf("persona_lock 关不应追加锁定段: %q", got)
	}
	if opts.Temperature != 0 {
		t.Errorf("persona_lock 关温度应不动, got %v", opts.Temperature)
	}
	if opts.MaxTokens != 2048 {
		t.Errorf("max_output_tokens 仍应钳制, got %d", opts.MaxTokens)
	}
	// max_tokens cap 8192 > 缺省 4096 → 不显式下发。
	opts = ai.ChatSimpleOptions{EngineID: "eng"}
	g3 := gaeaConfig.PlayGuardrails{Enabled: true, MaxOutputTokens: 8192}
	applyWhisperGuardrails(&opts, "p", lockTestPreset, g3)
	if opts.MaxTokens != 0 {
		t.Errorf("cap>缺省不应显式下发 max_tokens, got %d", opts.MaxTokens)
	}
}

// ── 钳制点 2：小说章节（create_chapter）────────────────────────

// TestApplyChapterGuardrails 章节透传点：未配置=逐字节现状；配置=钳制生效；
// temperature_max 边界（==cap 不变 / >cap 不抬高）；temperature<=0 不注入。
func TestApplyChapterGuardrails(t *testing.T) {
	newReq := func() *ai.ChatRequest {
		return &ai.ChatRequest{Model: "m", EngineID: "e"}
	}
	// 未配置：temperature 原样透传、max_tokens 不设置。
	req := newReq()
	applyChapterGuardrails(req, 1.0, gaeaConfig.PlayGuardrails{})
	if req.Temperature != 1.0 || req.MaxTokens != 0 {
		t.Errorf("未配置应现状: temp=%v max=%d", req.Temperature, req.MaxTokens)
	}
	// 配置：钳制生效。
	req = newReq()
	applyChapterGuardrails(req, 1.0, gaeaConfig.PlayGuardrails{Enabled: true, TemperatureMax: 0.7, MaxOutputTokens: 2048})
	if req.Temperature != 0.7 || req.MaxTokens != 2048 {
		t.Errorf("配置应钳制: temp=%v max=%d, want 0.7/2048", req.Temperature, req.MaxTokens)
	}
	// 边界：temperature == cap → 不变；cap > temperature → 不抬高。
	req = newReq()
	applyChapterGuardrails(req, 0.7, gaeaConfig.PlayGuardrails{Enabled: true, TemperatureMax: 0.7})
	if req.Temperature != 0.7 {
		t.Errorf("边界 temp==cap 应不变, got %v", req.Temperature)
	}
	req = newReq()
	applyChapterGuardrails(req, 0.5, gaeaConfig.PlayGuardrails{Enabled: true, TemperatureMax: 0.9})
	if req.Temperature != 0.5 {
		t.Errorf("cap>temperature 不应抬高, got %v", req.Temperature)
	}
	// temperature<=0（该点未设置温度）：不注入；max_tokens 仍下发上限。
	req = newReq()
	applyChapterGuardrails(req, 0, gaeaConfig.PlayGuardrails{Enabled: true, TemperatureMax: 0.7, MaxOutputTokens: 2048})
	if req.Temperature != 0 {
		t.Errorf("temperature<=0 不应注入温度, got %v", req.Temperature)
	}
	if req.MaxTokens != 2048 {
		t.Errorf("直连 ChatRequest 应显式下发上限, got %d", req.MaxTokens)
	}
}

// ── 钳制点 3：剧情支线（plot_branch，ChatSimple 基线钳制）────────
// （走 applyChatSimpleMaxTokens，已在 TestApplyChatSimpleMaxTokens 覆盖；
// 此处固化支线两点的调用形态：未配置不设置、配置钳制。）

func TestPlotBranchOptionsShape(t *testing.T) {
	// 未配置：支线两点 opts == {EngineID}（现状逐字节）。
	opts := ai.ChatSimpleOptions{EngineID: "novel-eng"}
	applyChatSimpleMaxTokens(&opts, gaeaConfig.PlayGuardrails{}.MaxOutputTokens)
	if opts != (ai.ChatSimpleOptions{EngineID: "novel-eng"}) {
		t.Errorf("未配置支线 opts 应现状: %+v", opts)
	}
	// 配置：显式钳制上限。
	opts = ai.ChatSimpleOptions{EngineID: "novel-eng"}
	applyChatSimpleMaxTokens(&opts, 2048)
	if opts.MaxTokens != 2048 {
		t.Errorf("配置支线应钳制, got %d", opts.MaxTokens)
	}
}

// ── 钳制点 5：生图（image_safe_mode）──────────────────────────

// TestApplyImageSafeMode 安全段注入：未启用原样；启用追加后缀；空提示词原样。
func TestApplyImageSafeMode(t *testing.T) {
	if got := applyImageSafeMode("一只猫", false); got != "一只猫" {
		t.Errorf("未启用应原样: %q", got)
	}
	if got := applyImageSafeMode("", true); got != "" {
		t.Errorf("空提示词应原样: %q", got)
	}
	got := applyImageSafeMode("一只猫", true)
	if got != "一只猫"+imageSafePromptSuffix {
		t.Errorf("启用应追加安全段: %q", got)
	}
	// 尾部空白先规整再拼接（不产生连续空行）。
	got = applyImageSafeMode("一只猫  \n", true)
	if got != "一只猫"+imageSafePromptSuffix {
		t.Errorf("尾部空白应规整: %q", got)
	}
}

// capturingPromptBackend 捕获提交的 ImageGenerationRequest（安全段断言用）。
type capturingPromptBackend struct {
	req *ai.ImageGenerationRequest
}

func (b *capturingPromptBackend) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	b.req = req
	return &ai.ImageGenerationResponse{Data: []ai.ImageData{{B64JSON: pngDataURLApp("guardrail")}}}, nil
}

// TestGenerateFreeImage_SafeMode 绘梦直连点：未配置提示词逐字节原样；安全
// 模式提交前注入安全段，历史元数据 Prompt 仍为用户原始提示词。
func TestGenerateFreeImage_SafeMode(t *testing.T) {
	run := func(t *testing.T, safe bool) (submitted, recorded string) {
		t.Helper()
		dir := t.TempDir()
		fake := &capturingPromptBackend{}
		c := &ai.Client{}
		c.SetImageBackend(fake, "comfyui")
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "comfyui", ImageSaveDir: dir}, client: c}}
		withGaeaCfg(t, guardrailsCfg(gaeaConfig.PlayGuardrails{Enabled: true, ImageSafeMode: safe}))
		res, err := ms.GenerateFreeImage("一只猫", "", "512x512", "", "krea2", 42, 1, "")
		if err != nil {
			t.Fatalf("GenerateFreeImage: %v", err)
		}
		if errMsg, _ := res["error"].(string); errMsg != "" {
			t.Fatalf("GenerateFreeImage 错误: %s", errMsg)
		}
		images := res["images"].([]imageItem)
		if fake.req == nil {
			t.Fatal("请求未被捕获")
		}
		return fake.req.Prompt, images[0].Prompt
	}

	// 未配置（零值）：提示词原样（现状逐字节）。
	submitted, recorded := run(t, false)
	if submitted != "一只猫" || recorded != "一只猫" {
		t.Errorf("未配置应原样: submitted=%q recorded=%q", submitted, recorded)
	}
	// 安全模式：提交注入安全段；历史记录保持原始提示词。
	submitted, recorded = run(t, true)
	if submitted != "一只猫"+imageSafePromptSuffix {
		t.Errorf("安全模式应注入安全段: %q", submitted)
	}
	if recorded != "一只猫" {
		t.Errorf("历史 Prompt 应保持原始提示词: %q", recorded)
	}
}

// TestImageGenTool_SafeMode image_gen 工具：未配置提示词原样；安全模式注入。
func TestImageGenTool_SafeMode(t *testing.T) {
	run := func(t *testing.T, safe bool) string {
		t.Helper()
		t.Chdir(t.TempDir())
		fake := &capturingPromptBackend{}
		c := &ai.Client{}
		c.SetImageBackend(fake, "comfyui")
		a := &App{core: &core{
			cfg:    &config.Config{ImageBackend: "comfyui", ImageModel: "test-model"},
			client: c,
		}}
		withGaeaCfg(t, guardrailsCfg(gaeaConfig.PlayGuardrails{Enabled: true, ImageSafeMode: safe}))
		tool := imageGenTool{a: a}
		if _, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"cat"}`)); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if fake.req == nil {
			t.Fatal("请求未被捕获")
		}
		return fake.req.Prompt
	}
	if got := run(t, false); got != "cat" {
		t.Errorf("未配置应原样: %q", got)
	}
	if got := run(t, true); got != "cat"+imageSafePromptSuffix {
		t.Errorf("安全模式应注入安全段: %q", got)
	}
}
