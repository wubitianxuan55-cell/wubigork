package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/util"
)

// libFillKeys 随机补全允许写入的字段（仅当当前为空时填充）。
var libFillKeys = []string{
	"roleType", "gender", "age", "personality", "appearance", "figure",
	"background", "motivation", "arc", "status", "notes",
}

// libRandomKeys 所有可随机生成的字段（"all" 时全部重生成）。
var libRandomKeys = append(append([]string{}, libFillKeys...),
	"tags", "dialogueSamples", "voiceGuide", "behaviorRules", "emotionLogic")

// libFillLabels 补全字段的中文标签（用于把已有设定注入 prompt，保持一致性）。
var libFillLabels = map[string]string{
	"roleType": "定位", "gender": "性别", "age": "年龄", "personality": "性格",
	"appearance": "外貌", "figure": "身材", "background": "背景",
	"motivation": "动机", "arc": "角色弧线", "status": "状态", "notes": "备注",
}

// libRandomLabels 随机字段的中文标签（all / 单字段随机时用于 prompt）。
var libRandomLabels = map[string]string{
	"roleType": "定位", "gender": "性别", "age": "年龄", "personality": "性格",
	"appearance": "外貌", "figure": "身材", "background": "背景",
	"motivation": "动机", "arc": "角色弧线", "status": "状态", "notes": "备注",
	"tags": "标签", "dialogueSamples": "对话样本", "voiceGuide": "口吻指南",
	"behaviorRules": "行为规则", "emotionLogic": "情感逻辑",
}

// CharacterGenerateFill 一键随机补全角色库角色：只填充空缺字段，保留已有内容。
// 不依赖小说项目（无项目时世界观置为“暂无”），返回合并后的角色 JSON。
func (a *App) CharacterGenerateFill(chJSON string) (string, error) {
	return a.characterGenerate(chJSON, "fill", nil)
}

// CharacterGenerateRandom 按字段随机再生成角色设定。
// fields 语义：
//   空字符串 / "fill" → 只补空缺字段（与 CharacterGenerateFill 一致）
//   "all" → 全部字段重新随机（含性格，姓名不变）
//   其他 → 逗号分隔的字段 key，如 "personality,appearance" 仅随机这些字段
func (a *App) CharacterGenerateRandom(chJSON, fields string) (string, error) {
	targets := parseRandomFields(fields)
	if len(targets) == 0 {
		// 空 / "fill" / 非法字段列表：回落为补齐空缺，避免误触发全量随机
		return a.characterGenerate(chJSON, "fill", nil)
	}
	return a.characterGenerate(chJSON, "random", targets)
}

// parseRandomFields 解析随机列表；"all" 返回全部可随机字段。
func parseRandomFields(fields string) []string {
	f := strings.TrimSpace(fields)
	if f == "" || f == "fill" {
		return nil
	}
	if f == "all" {
		return append([]string{}, libRandomKeys...)
	}
	seen := make(map[string]bool)
	var out []string
	for _, k := range strings.FieldsFunc(f, func(r rune) bool {
		return r == ',' || r == '，' || r == ' '
	}) {
		if _, ok := libRandomLabels[k]; ok && !seen[k] {
			out = append(out, k)
			seen[k] = true
		}
	}
	return out
}

// characterGenerate 随机生成的核心实现：fill 模式只填空缺字段；
// random 模式根据 targets 随机指定字段（targets 为空时按全部字段处理）。
func (a *App) characterGenerate(chJSON, mode string, targets []string) (string, error) {
	var cur map[string]interface{}
	if err := json.Unmarshal([]byte(chJSON), &cur); err != nil {
		return "", fmt.Errorf("解析角色数据失败: %w", err)
	}
	name, _ := cur["name"].(string)
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("角色名称不能为空")
	}
	if a.client == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	if a.eng == nil {
		return "", fmt.Errorf("prompt 引擎未初始化")
	}
	tmpl := a.eng.Get("character-generate-single")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 character-generate-single 模板文件")
	}

	// 世界观与题材：有项目则取项目，无项目不阻填（角色库全局可用）。
	genre := ""
	worldview := "（暂无世界观）"
	if pm := a.getPM(); pm != nil {
		if pm.Meta != nil && pm.Meta.Genre != "" {
			genre = pm.Meta.Genre
		}
		if wf, err := pm.ReadWorldviewFile(); err == nil && wf != nil {
			worldview = wf.ToMarkdown()
		}
	}

	story := "题材: " + genre

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"name":         name,
		"worldview":    worldview,
		"story_thread": story,
	})

	existing := summarizeExisting(cur)
	if mode == "fill" {
		if existing != "" {
			userPrompt += "\n\n【已有设定】" + existing
		}
		userPrompt += "\n\n【补全要求】角色已有信息必须保持不变，仅补齐空缺字段；输出仍为完整 JSON。"
	} else {
		if existing != "" {
			userPrompt += "\n\n【当前设定（仅作上下文参考，可完全推翻）】" + existing
		}
		if len(targets) == 0 || len(targets) == len(libRandomKeys) {
			userPrompt += "\n\n【随机要求】请为角色重新设计一套完整的新设定（含性格、外貌、背景等全部字段），可完全推翻当前设定；姓名必须与输入完全一致、保持不变；输出完整 JSON。"
		} else {
			userPrompt += "\n\n【随机要求】仅重新随机设计以下字段：" + randomTargetLabels(targets) + "。其余字段必须与当前设定完全一致、保持不变；姓名不变；输出完整 JSON。"
		}
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// 角色库使用独立功能绑定（未绑定时回退全局激活引擎）
	eng, model := a.cfg.GetFeatureModel("characterlib")
	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		EngineID:    eng,
		Temperature: 0.85,
		MaxTokens:   2048,
	})
	if err != nil {
		return "", fmt.Errorf("AI 补全失败: %w", err)
	}

	var gen map[string]interface{}
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &gen); err != nil {
		return "", fmt.Errorf("解析 AI 结果失败: %w", err)
	}
	if mode == "fill" {
		mergeFill(cur, gen)
	} else {
		mergeRandom(cur, gen, targets)
	}
	return string(util.MustMarshal(cur)), nil
}

// randomTargetLabels 字段 key 列表 → 中文标签列表（如 "personality,appearance" → “性格、外貌”）。
func randomTargetLabels(targets []string) string {
	var labels []string
	for _, k := range targets {
		if l, ok := libRandomLabels[k]; ok {
			labels = append(labels, l)
		}
	}
	return strings.Join(labels, "、")
}

// mergeRandom 将生成结果写入目标字段（非空才覆盖），其余字段保持不变。
func mergeRandom(cur, gen map[string]interface{}, targets []string) {
	targetSet := make(map[string]bool, len(targets))
	for _, k := range targets {
		targetSet[k] = true
	}
	if len(targetSet) == 0 {
		for _, k := range libRandomKeys {
			targetSet[k] = true
		}
	}
	for _, k := range libRandomKeys {
		if !targetSet[k] {
			continue
		}
		genVal, ok := gen[k]
		if !ok && k == "roleType" {
			genVal, ok = gen["role_type"]
		}
		if !ok {
			continue
		}
		genVal = normalizeGenValue(k, genVal)
		if isEmptyGen(genVal) {
			continue
		}
		cur[k] = genVal
	}
}

// normalizeGenValue 将 AI 输出规范化到数据类型：标签/对话样本支持数组或分隔的字符串。
func normalizeGenValue(k string, v interface{}) interface{} {
	switch k {
	case "tags", "dialogueSamples":
		switch t := v.(type) {
		case []interface{}:
			return t
		case []string:
			out := make([]interface{}, 0, len(t))
			for _, s := range t {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
			return out
		case string:
			var out []interface{}
			for _, s := range strings.FieldsFunc(t, func(r rune) bool {
				return r == ',' || r == '，' || r == '、' || r == '\n' || r == '。' || r == ';'
			}) {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return v
}

// isEmptyGen 判断随机生成值是否为空（空字符串 / 空数组）。
func isEmptyGen(v interface{}) bool {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t) == ""
	case []interface{}:
		return len(t) == 0
	case []string:
		return len(t) == 0
	}
	return false
}

// CharacterGeneratePortrait 为角色库角色生成剧照：按角色字段构建智能 prompt。
// 使用独立的剧照后端/模型（未单独配置时跟随绘梦），不影响绘梦页当前选择。
// 返回图片 data URL 或远程 URL，前端拿到后写入 portraitUrl 再随角色保存。
func (a *App) CharacterGeneratePortrait(chJSON, model string) (string, error) {
	var c characterlib.Character
	if err := json.Unmarshal([]byte(chJSON), &c); err != nil {
		return "", fmt.Errorf("解析角色数据失败: %w", err)
	}
	if strings.TrimSpace(c.Name) == "" {
		return "", fmt.Errorf("角色名称不能为空")
	}

	backend := a.cfg.PortraitBackend // 空 = 跟随绘梦
	imgModel := a.cfg.PortraitModel
	if imgModel == "" {
		imgModel = a.cfg.ImageModel
	}
	if model != "" {
		imgModel = model // 显式传入的模型优先
	}

	client, err := a.buildPortraitClient()
	if err != nil {
		return "", err
	}
	negative := "文字, 水印, 签名, 低质量, 模糊, 肢体变形, 多余手指, 多眼多嘴"
	req := &ai.ImageGenerationRequest{
		Model:    imgModel,
		Prompt:   buildPortraitPrompt(c),
		Negative: negative,
		N:        1,
		Size:     "1024x1024",
	}
	if backend != "comfyui" {
		req.Size = ""
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := client.GenerateImage(ctx, req)
	if err != nil {
		return "", fmt.Errorf("剧照生成失败: %w", err)
	}
	if len(resp.Data) == 0 {
		return "", fmt.Errorf("剧照生成返回为空")
	}
	img := resp.Data[0].URL
	if img == "" {
		img = resp.Data[0].B64JSON
	}
	if img == "" {
		return "", fmt.Errorf("剧照生成返回为空")
	}
	return img, nil
}

// buildPortraitClient 为角色剧照构建独立图片客户端（不改变绘梦当前后端）。
// backend: comfyui / herdsman / ollama 走对应后端，xai 或空走 xAI 原生管线。
func (a *App) buildPortraitClient() (*ai.Client, error) {
	backend := a.cfg.PortraitBackend
	if backend == "" {
		backend = a.cfg.ImageBackend
	}
	if backend == "" {
		backend = "xai"
	}
	client := ai.NewClient(a.cfg)
	switch backend {
	case "comfyui":
		if a.cfg.ComfyUIURL == "" {
			return nil, fmt.Errorf("未配置 ComfyUI 地址")
		}
		client.SetImageBackend(ai.NewComfyUIBackend(a.cfg.ComfyUIURL), "comfyui")
	case "herdsman", "ollama":
		eng, ok := a.engineMgr.GetEngine(backend)
		if !ok || !eng.Enabled {
			return nil, fmt.Errorf("剧照引擎 %s 未启用，请先在模型中心启用", backend)
		}
		client.SetImageBackend(ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey), backend)
	default: // xai
		client.SetImageBackend(nil, "xai")
	}
	return client, nil
}

// mergeFill 用生成结果填充当前为空的字段（role_type → roleType 归一化）。
func mergeFill(cur, gen map[string]interface{}) {
	for _, k := range libFillKeys {
		genVal, ok := gen[k].(string)
		if !ok && k == "roleType" {
			genVal, ok = gen["role_type"].(string)
		}
		if !ok {
			continue
		}
		curVal, _ := cur[k].(string)
		if strings.TrimSpace(curVal) == "" && strings.TrimSpace(genVal) != "" {
			cur[k] = genVal
		}
	}
	// tags：仅当当前为空且生成结果非空时填充。
	curTags, _ := cur["tags"].([]interface{})
	if len(curTags) == 0 {
		if genTags, ok := gen["tags"].([]interface{}); ok && len(genTags) > 0 {
			cur["tags"] = genTags
		}
	}
}

// summarizeExisting 汇总当前非空字段（中文标签），供 AI 保持一致。
func summarizeExisting(cur map[string]interface{}) string {
	var parts []string
	for _, k := range libFillKeys {
		if v, ok := cur[k].(string); ok && strings.TrimSpace(v) != "" {
			parts = append(parts, libFillLabels[k]+": "+v)
		}
	}
	if tags, ok := cur["tags"].([]interface{}); ok && len(tags) > 0 {
		var ts []string
		for _, t := range tags {
			if s, ok := t.(string); ok && s != "" {
				ts = append(ts, s)
			}
		}
		if len(ts) > 0 {
			parts = append(parts, "标签: "+strings.Join(ts, "、"))
		}
	}
	return strings.Join(parts, "；")
}

// buildPortraitPrompt 按角色字段构建剧照 prompt（跳过空字段）。
func buildPortraitPrompt(c characterlib.Character) string {
	parts := []string{"角色立绘，" + c.Name}
	if c.Gender != "" {
		parts = append(parts, c.Gender)
	}
	if c.Age != "" {
		parts = append(parts, "年龄"+c.Age)
	}
	if c.Appearance != "" {
		parts = append(parts, "外貌："+c.Appearance)
	}
	if c.Figure != "" {
		parts = append(parts, "身材："+c.Figure)
	}
	if c.Personality != "" {
		parts = append(parts, "气质："+c.Personality)
	}
	if len(c.Tags) > 0 {
		parts = append(parts, "特征标签："+strings.Join(c.Tags, "、"))
	}
	parts = append(parts, "半身像，居中构图，干净简洁背景，高细节，唯美光影")
	return strings.Join(parts, "。")
}
