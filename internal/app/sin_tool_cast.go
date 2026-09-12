package app

// ── 原罪工具：角色卡查询（sin_cast）──
//
// 只读：本故事已选角色卡（角色库是跨板块共享资产层，见 sin_store.go 头注）。
// 角色块每轮已注入提示词，本工具是「核对细节」的补充入口——长篇写到后面时
// 回查某个角色的既定设定，避免凭记忆编造。
//
// 渲染收在纯函数 sinCastToolText 里（不依赖 App），便于单测与漂移锁定。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/characterlib"
)

func init() {
	registerSinTool(sinToolCast, func(c sinToolContext) sinTool {
		return sinCastTool{app: c.app, topicID: c.topicID}
	})
}

// sinCastTool 角色卡查询工具实例（名字与注册表常量 sinToolCast 区分）。
type sinCastTool struct {
	app     *App
	topicID string
}

func (sinCastTool) Name() string { return sinToolCast }

func (sinCastTool) Description() string {
	return "查询本故事已选角色卡的设定（只读）。本故事角色已经随提示词带入，只有需要核对某个角色的外观/性格/背景/口吻等细节时" +
		"（避免长篇写到后面自相矛盾）才调用；不带 name 列出全部角色，带 name 查一个。只读不改，不得据此改写角色设定或改名。"
}

func (sinCastTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "name":{"type":"string","description":"可选。角色名；不传则列出本故事全部角色。"}
}}`)
}

func (sinCastTool) ReadOnly() bool { return true }

func (t sinCastTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name string `json:"name"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
	}
	var cast []*characterlib.Character
	if t.app != nil {
		cast = t.app.sinCastCharacters(t.topicID)
	}
	return sinCastToolText(cast, p.Name), nil
}

// sinCastToolText 渲染角色卡查询结果：
//   - 未选角色：如实说明（不是错误——故事可以没有角色卡）；
//   - 不带名字：全部角色紧凑列示（只写有值的字段，不编造）；
//   - 带名字：精确优先，其次去空白、不区分大小写的包含匹配；
//   - 没命中：如实说明并回列可查的名字（让模型自己纠正，而不是猜一个）。
func sinCastToolText(cast []*characterlib.Character, want string) string {
	chars := make([]*characterlib.Character, 0, len(cast))
	for _, c := range cast {
		if c != nil && strings.TrimSpace(c.Name) != "" {
			chars = append(chars, c)
		}
	}
	if len(chars) == 0 {
		return "本故事还没有选择角色卡。"
	}
	want = strings.TrimSpace(want)
	if want == "" {
		var b strings.Builder
		fmt.Fprintf(&b, "本故事角色卡（%d 个，只读）：\n", len(chars))
		for _, c := range chars {
			b.WriteString(sinCastBrief(c))
		}
		return strings.TrimRight(b.String(), "\n")
	}
	if c := sinFindCastByName(chars, want); c != nil {
		var b strings.Builder
		b.WriteString("角色卡：" + c.Name + "\n")
		writeCastField(&b, "性别", c.Gender)
		writeCastField(&b, "年龄", c.Age)
		writeCastField(&b, "定位", c.RoleType)
		writeCastField(&b, "外观", c.Appearance)
		writeCastField(&b, "身材", c.Figure)
		writeCastField(&b, "性格", c.Personality)
		writeCastField(&b, "背景", c.Background)
		writeCastField(&b, "动机", c.Motivation)
		writeCastField(&b, "状态", c.Status)
		writeCastField(&b, "口吻", c.VoiceGuide)
		if len(c.DialogueSamples) > 0 {
			samples := c.DialogueSamples
			if len(samples) > 3 {
				samples = samples[:3]
			}
			writeCastField(&b, "说话样例", strings.Join(samples, " / "))
		}
		return strings.TrimRight(b.String(), "\n")
	}
	names := make([]string, 0, len(chars))
	for _, c := range chars {
		names = append(names, c.Name)
	}
	return fmt.Sprintf("本故事角色里没有叫「%s」的角色。可查的角色：%s", want, strings.Join(names, "、"))
}

// sinCastBrief 单行摘要（不带名字时列示用；字段过长按 120 rune 截断）。
func sinCastBrief(c *characterlib.Character) string {
	var b strings.Builder
	b.WriteString("- " + c.Name)
	var bits []string
	if v := strings.TrimSpace(c.Gender); v != "" {
		bits = append(bits, "性别 "+v)
	}
	if v := strings.TrimSpace(c.Age); v != "" {
		bits = append(bits, "年龄 "+v)
	}
	if v := strings.TrimSpace(c.RoleType); v != "" {
		bits = append(bits, "定位 "+v)
	}
	if len(bits) > 0 {
		b.WriteString("（" + strings.Join(bits, "，") + "）")
	}
	b.WriteString("\n")
	writeCastField(&b, "外观", truncateRunes(strings.TrimSpace(c.Appearance), 120))
	writeCastField(&b, "性格", truncateRunes(strings.TrimSpace(c.Personality), 120))
	writeCastField(&b, "口吻", truncateRunes(strings.TrimSpace(c.VoiceGuide), 80))
	return b.String()
}

// sinFindCastByName 名字匹配：精确优先，其次去空白后包含（大小写不敏感）。
func sinFindCastByName(cast []*characterlib.Character, want string) *characterlib.Character {
	for _, c := range cast {
		if c.Name == want {
			return c
		}
	}
	norm := strings.ToLower(strings.Join(strings.Fields(want), ""))
	if norm == "" {
		return nil
	}
	for _, c := range cast {
		got := strings.ToLower(strings.Join(strings.Fields(c.Name), ""))
		if got == "" {
			continue
		}
		if strings.Contains(got, norm) || strings.Contains(norm, got) {
			return c
		}
	}
	return nil
}
