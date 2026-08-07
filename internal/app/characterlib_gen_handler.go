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

// libFillKeys 随机补齐允许写入的字段（仅当当前为空时填充）。
var libFillKeys = []string{
	"roleType", "gender", "age", "personality", "appearance", "figure",
	"background", "motivation", "arc", "status", "notes",
}

// libFillLabels 补齐字段的中文标签（用于把已有设定注入 prompt，保持一致性）。
var libFillLabels = map[string]string{
	"roleType": "定位", "gender": "性别", "age": "年龄", "personality": "性格",
	"appearance": "外貌", "figure": "身材", "background": "背景",
	"motivation": "动机", "arc": "角色弧线", "status": "状态", "notes": "备注",
}

// CharacterGenerateFill 一键随机补齐角色库角色：只填充空缺字段，保留已有内容。
// 不依赖小说项目（无项目时世界观置为“暂无”），返回合并后的角色 JSON。
func (a *App) CharacterGenerateFill(chJSON string) (string, error) {
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

	// 世界观与题材：有项目则取项目，无项目不阻塞（角色库全局可用）
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
	if existing := summarizeExisting(cur); existing != "" {
		story += "\n已有设定（必须保留不变）: " + existing
	}
	story += "\n请为角色补齐空缺字段，输出完整 JSON。"

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"name":         name,
		"worldview":    worldview,
		"story_thread": story,
	})
	userPrompt += "\n\n【补齐要求】角色已有信息必须保留不变，仅补齐空缺字段；输出仍为完整 JSON。"

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	eng, model := a.cfg.GetFeatureModel("novel")
	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		EngineID:    eng,
		Temperature: 0.85,
		MaxTokens:   2048,
	})
	if err != nil {
		return "", fmt.Errorf("AI 补齐失败: %w", err)
	}

	var gen map[string]interface{}
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &gen); err != nil {
		return "", fmt.Errorf("解析 AI 结果失败: %w", err)
	}
	mergeFill(cur, gen)
	return string(util.MustMarshal(cur)), nil
}

// CharacterGeneratePortrait 为角色库角色生成剧照：按角色字段构建智能 prompt，
// 复用图片生成管线（ComfyUI / xAI / Herdsman / Ollama），返回图片 data URL 或远程 URL。
// 前端拿到后写入 portraitUrl 再随角色保存。
func (a *App) CharacterGeneratePortrait(chJSON, model string) (string, error) {
	var c characterlib.Character
	if err := json.Unmarshal([]byte(chJSON), &c); err != nil {
		return "", fmt.Errorf("解析角色数据失败: %w", err)
	}
	if strings.TrimSpace(c.Name) == "" {
		return "", fmt.Errorf("角色名称不能为空")
	}

	negative := "文字, 水印, 签名, 低质量, 模糊, 肢体变形, 多余手指, 多眼多嘴"
	res, err := a.GenerateFreeImage(buildPortraitPrompt(c), negative, "1024x1024", "", model, 0, 1, "")
	if err != nil {
		return "", fmt.Errorf("剧照生成失败: %w", err)
	}
	if errMsg, ok := res["error"].(string); ok && errMsg != "" {
		return "", fmt.Errorf("剧照生成失败：%s", errMsg)
	}
	imgs, ok := res["images"].([]imageItem)
	if !ok || len(imgs) == 0 || imgs[0].Image == "" {
		return "", fmt.Errorf("剧照生成返回为空")
	}
	return imgs[0].Image, nil
}

// mergeFill 用生成结果填充当前为空的可写字段（role_type → roleType 归一化）。
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
	// tags：仅当当前为空且生成结果非空时填充
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
	parts := []string{"角色立绘：" + c.Name}
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
