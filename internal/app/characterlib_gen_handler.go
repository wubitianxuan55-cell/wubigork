package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/util"
)

// libFillKeys 随机补全允许写入的字段（仅当当前为空时填充）。
var libFillKeys = []string{
	"roleType", "gender", "age", "personality", "appearance", "figure",
	"background", "motivation", "arc", "status", "notes",
	"behaviorRules", "emotionLogic",
}

// libRandomKeys 所有可随机生成的字段（"all" 时全部重生成）。
var libRandomKeys = append(append([]string{}, libFillKeys...),
	"tags", "dialogueSamples", "voiceGuide", "behaviorRules", "emotionLogic", "dims")

// libFillLabels 补全字段的中文标签（用于把已有设定注入 prompt，保持一致性）。
var libFillLabels = map[string]string{
	"roleType": "定位", "gender": "性别", "age": "年龄", "personality": "性格",
	"appearance": "外貌", "figure": "身材", "background": "背景",
	"motivation": "动机", "arc": "角色弧线", "status": "状态", "notes": "备注",
	"behaviorRules": "行为规则", "emotionLogic": "情感逻辑",
}

// libRandomLabels 随机字段的中文标签（all / 单字段随机时用于 prompt）。
var libRandomLabels = map[string]string{
	"roleType": "定位", "gender": "性别", "age": "年龄", "personality": "性格",
	"appearance": "外貌", "figure": "身材", "background": "背景",
	"motivation": "动机", "arc": "角色弧线", "status": "状态", "notes": "备注",
	"tags": "标签", "dialogueSamples": "对话样本", "voiceGuide": "口吻指南",
	"behaviorRules": "行为规则", "emotionLogic": "情感逻辑",
	"dims": "五维人格",
}

// dimsKeys 五维人格的字段键（T/I/S/O/R，与 whisper.PersonalityDims 一致）。
var dimsKeys = []string{"T", "I", "S", "O", "R"}

// dimsLabels 五维人格的中文说明（注入 prompt）。
var dimsLabels = "T 温柔、I 主动、S 顺从、O 独特、R 矜持"

// dimsIsDefault 判断五维人格是否仍是编辑器默认（全 50），视为“未设定”。
// 角色卡补齐时把默认五维人格当作空缺一并生成（见 CharacterGenerateFill）。
func dimsIsDefault(cur map[string]interface{}) bool {
	d, ok := cur["dims"].(map[string]interface{})
	if !ok {
		return true
	}
	for _, k := range dimsKeys {
		v, ok := d[k].(float64)
		if !ok || v != 50 {
			return false
		}
	}
	return true
}

// parseDims 解析 AI 输出的五维人格：支持对象 {"T":85,...} 或字符串
// "T=85,I=40,S=20,O=70,R=60" / "85/40/20/70/60" / "85,40,20,70,60"。
func parseDims(v interface{}) (map[string]interface{}, bool) {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(dimsKeys))
		for _, k := range dimsKeys {
			f, ok := numOf(t[k])
			if !ok || f < 0 || f > 100 {
				return nil, false
			}
			out[k] = f
		}
		return out, true
	case string:
		return parseDimsString(t)
	}
	return nil, false
}

// parseDimsString 解析字符串形式的五维人格（宽松分隔符）。
func parseDimsString(s string) (map[string]interface{}, bool) {
	vals := make([]float64, 0, len(dimsKeys))
	// 键值对形式：T=85,I=40,...（键名大小写不敏感）
	if strings.ContainsAny(s, "=：") {
		kv := make(map[string]float64, len(dimsKeys))
		for _, part := range strings.FieldsFunc(s, func(r rune) bool {
			return r == ',' || r == '，' || r == ' ' || r == ';' || r == '；'
		}) {
			key, val, ok := strings.Cut(part, "=")
			if !ok {
				if key, val, ok = strings.Cut(part, "："); !ok {
					return nil, false
				}
			}
			f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
			if err != nil {
				return nil, false
			}
			kv[strings.ToUpper(strings.TrimSpace(key))] = f
		}
		out := make(map[string]interface{}, len(dimsKeys))
		for _, k := range dimsKeys {
			f, ok := kv[k]
			if !ok || f < 0 || f > 100 {
				return nil, false
			}
			out[k] = f
		}
		return out, true
	}
	// 纯数值形式：85/40/20/70/60 或 85,40,20,70,60
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == '/' || r == ',' || r == '，' || r == ' ' || r == '-' || r == '、'
	}) {
		f, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return nil, false
		}
		vals = append(vals, f)
	}
	if len(vals) != len(dimsKeys) {
		return nil, false
	}
	out := make(map[string]interface{}, len(dimsKeys))
	for i, k := range dimsKeys {
		if vals[i] < 0 || vals[i] > 100 {
			return nil, false
		}
		out[k] = vals[i]
	}
	return out, true
}

// numOf 从任意 JSON 数值类型提取 float64。
func numOf(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case int:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	}
	return 0, false
}

// CharacterGenerateFill 一键随机补全角色库角色：只填充空缺字段，保留已有内容。
// 不依赖小说项目（无项目时世界观置为“暂无”），返回合并后的角色 JSON。
func (a *App) CharacterGenerateFill(chJSON string) (string, error) {
	return a.characterGenerate(chJSON, "fill", nil)
}

// CharacterGenerateRandom 按字段随机再生成角色设定。
// fields 语义：
//
//	空字符串 / "fill" → 只补空缺字段（与 CharacterGenerateFill 一致）
//	"all" → 全部字段重新随机（含性格，姓名不变）
//	其他 → 逗号分隔的字段 key，如 "personality,appearance" 仅随机这些字段
func (a *App) CharacterGenerateRandom(chJSON, fields string) (string, error) {
	targets := parseRandomFields(fields)
	if len(targets) == 0 {
		// 空 / "fill" / 非法字段列表：回落为补齐空缺，避免误触发全量随机
		return a.characterGenerate(chJSON, "fill", nil)
	}
	return a.characterGenerate(chJSON, "random", targets)
}

// CharacterFillAll 批量补齐全局角色库所有可见角色的空缺字段。
// 复用单角色补全逻辑（character-generate-single + fill 模式）：
// 只填充空缺、保留已有内容，基于已有设定（性格等）推断；
// 逐角色处理并广播 character-fill-progress 进度事件。
func (a *App) CharacterFillAll() (map[string]interface{}, error) {
	if a.charLib == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	if a.client == nil || a.eng == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	all, _, err := a.charLib.List("", "", false, 100000, 0)
	if err != nil {
		return nil, fmt.Errorf("读取角色库失败: %w", err)
	}
	if len(all) == 0 {
		return map[string]interface{}{
			"total": 0, "filled": 0, "skipped": 0, "failed": 0, "failNames": []string{},
		}, nil
	}

	filled, skipped, failed := 0, 0, 0
	var failNames []string
	for i, c := range all {
		a.emit("character-fill-progress", map[string]interface{}{
			"current": i + 1,
			"total":   len(all),
			"name":    c.Name,
		})
		if !hasMissingFillFields(c) {
			skipped++
			continue
		}
		merged, err := a.characterGenerate(string(util.MustMarshal(c)), "fill", nil)
		if err != nil {
			failed++
			failNames = append(failNames, c.Name)
			continue
		}
		var next characterlib.Character
		if err := json.Unmarshal([]byte(merged), &next); err != nil {
			failed++
			failNames = append(failNames, c.Name)
			continue
		}
		// 保护身份与底层字段不被 AI 结果覆盖
		next.ID = c.ID
		next.Kind = c.Kind
		next.CreatedAt = c.CreatedAt
		next.AssistantID = c.AssistantID
		next.ChatEnabled = c.ChatEnabled
		next.Hidden = c.Hidden
		if err := a.charLib.Upsert(&next); err != nil {
			failed++
			failNames = append(failNames, c.Name)
			continue
		}
		filled++
	}
	return map[string]interface{}{
		"total":     len(all),
		"filled":    filled,
		"skipped":   skipped,
		"failed":    failed,
		"failNames": failNames,
	}, nil
}

// hasMissingFillFields 判断角色是否存在可补齐的空缺字段。
func hasMissingFillFields(c characterlib.Character) bool {
	for _, v := range []string{
		c.RoleType, c.Gender, c.Age, c.Personality, c.Appearance,
		c.Figure, c.Background, c.Motivation, c.Arc, c.Status, c.Notes,
		c.BehaviorRules, c.EmotionLogic,
	} {
		if strings.TrimSpace(v) == "" {
			return true
		}
	}
	if len(c.Tags) == 0 {
		return true
	}
	// 五维人格全为默认 50 视为未设定，批量补齐时一并生成。
	d := c.Dims
	return d.T == 50 && d.I == 50 && d.S == 50 && d.O == 50 && d.R == 50
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
		if dimsIsDefault(cur) {
			userPrompt += "\n\n【五维人格】当前角色的五维人格尚未设定（默认 50/50/50/50/50）。" +
				"请按补全后的性格给出五维人格（0-100 整数）：" + dimsLabels + "。" +
				"在 JSON 的 dims 字段输出对象，如 {\"dims\":{\"T\":85,\"I\":40,\"S\":20,\"O\":70,\"R\":60}}。"
		}
	} else {
		if existing != "" {
			userPrompt += "\n\n【当前设定（仅作上下文参考，可完全推翻）】" + existing
		}
		if len(targets) == 0 || len(targets) == len(libRandomKeys) {
			userPrompt += "\n\n【随机要求】请为角色重新设计一套完整的新设定（含性格、外貌、背景、五维人格等全部字段），" +
				"可完全推翻当前设定；姓名必须与输入完全一致、保持不变；输出完整 JSON。" +
				"其中 dims 为五维人格（0-100 整数）：" + dimsLabels + "，必须与重新设计的性格保持一致。" +
				"示例：{\"dims\":{\"T\":30,\"I\":80,\"S\":70,\"O\":40,\"R\":50}}。"
		} else {
			userPrompt += "\n\n【随机要求】仅重新随机设计以下字段：" + randomTargetLabels(targets) + "。" +
				"其余字段必须与当前设定完全一致、保持不变；姓名不变；输出完整 JSON。"
			if containsKey(targets, "dims") {
				userPrompt += "其中 dims 为五维人格（0-100 整数）：" + dimsLabels + "，与当前性格保持一致。" +
					"示例：{\"dims\":{\"T\":30,\"I\":80,\"S\":70,\"O\":40,\"R\":50}}。"
			}
		}
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// 角色库使用独立功能绑定（未绑定时回退全局激活引擎）
	eng, model := a.cfg.GetFeatureModel("characterlib")
	// S1.5-B play 内容护栏：0.85/2048 为该点现状基线，temperature_max/
	// max_output_tokens 只降不升（cap 未配置或 >= 基线时不钳，零值 = 现状
	// 逐字节）。
	g := playGuardrails()
	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		EngineID:    eng,
		Temperature: clampPlayTemperature(0.85, g.TemperatureMax),
		MaxTokens:   clampPlayMaxTokens(2048, g.MaxOutputTokens),
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

// containsKey 判断 targets 是否包含指定 key。
func containsKey(targets []string, key string) bool {
	for _, k := range targets {
		if k == key {
			return true
		}
	}
	return false
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
		if k == "dims" {
			if genDims, ok := parseDims(gen["dims"]); ok {
				cur["dims"] = genDims
			}
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

// mergeFill 用生成结果填充当前为空的字段（兼容 AI 输出的 snake_case 字段名）。
func mergeFill(cur, gen map[string]interface{}) {
	for _, k := range libFillKeys {
		genVal, ok := gen[k].(string)
		if !ok {
			if snake, has := libSnakeFallback[k]; has {
				genVal, ok = gen[snake].(string)
			}
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
	// 五维人格：仅当当前为默认（全 50）且生成结果有效时填充。
	if dimsIsDefault(cur) {
		if genDims, ok := parseDims(gen["dims"]); ok {
			cur["dims"] = genDims
		}
	}
}

// libSnakeFallback AI 输出字段名 → 库内 camelCase 键的映射（模板输出常带下划线）。
var libSnakeFallback = map[string]string{
	"roleType":      "role_type",
	"behaviorRules": "behavior_rules",
	"emotionLogic":  "emotion_logic",
	"voiceGuide":    "voice_guide",
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
	if d, ok := cur["dims"].(map[string]interface{}); ok && !dimsIsDefault(cur) {
		parts = append(parts, fmt.Sprintf("五维人格: T%v I%v S%v O%v R%v",
			d["T"], d["I"], d["S"], d["O"], d["R"]))
	}
	return strings.Join(parts, "；")
}

// buildPortraitPrompt 按角色字段构建剧照 prompt（跳过空字段）。
func buildPortraitPrompt(c characterlib.Character) string {
	parts := []string{ai.PortraitStylePrefix + "角色立绘，" + c.Name}
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
