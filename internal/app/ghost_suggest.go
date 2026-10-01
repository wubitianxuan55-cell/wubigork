package app

// ghost_suggest.go — 场景编辑器内联续写（v4.444 GhostText 接线的后端半边）。
//
// 历史：前端 GhostText 组件（v4.425）只有 UI 壳——无请求触发、无后端发射方、
// 挂载点 enabled=false，v4.429「受控值疑点待真机」实际无从测起（死代码）。
// 本刀给后半边：单发（非流式）续写建议——1~2 句的补全走 SSE 流是过度设计，
// 直接调用即可，也没有 ghost-stream 多实例串扰问题。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/maturecraft"
)

// ghostContextRunes 续写上下文窗口：取光标前文本的尾部即可，长了慢且偏题。
const ghostContextRunes = 800

// NovelGhostSuggest 场景编辑器内联续写建议：给光标前文本，返回 1~2 句接续。
// 单发非流式；空建议返回错误（前端不弹框）。
func (a *writingState) NovelGhostSuggest(textBefore string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("AI client not ready")
	}
	pm := a.getPM()
	if pm == nil {
		return "", fmt.Errorf("请先打开项目")
	}
	runes := []rune(strings.TrimSpace(textBefore))
	if len(runes) < 10 {
		return "", fmt.Errorf("上下文不足（至少 10 字再请求续写）")
	}
	if len(runes) > ghostContextRunes {
		runes = runes[len(runes)-ghostContextRunes:]
	}

	system := "你是小说续写引擎。只输出接续文字：一到两句（不超过 60 字），" +
		"贴合前文的叙事声音、节奏与人称，不解释、不重复前文、不加引号、不输出任何前缀或说明。" +
		"前文在句中则顺着句子续，在句末则开新句。"
	// v4.444：成人向一行口径（完整工艺区段对 60 字补全是浪费，只保「不降级不回避」）。
	if lv := maturecraft.NormalizeLevel(pm.Meta.Mature); lv != "" {
		system += "本书为成人向作品（尺度：" + maturecraft.LevelName(lv) + "），续写按本书尺度正常书写，不回避、不降级为纯爱。"
	}
	user := "【前文】\n" + string(runes) + "\n【接续】"

	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return "", fmt.Errorf("未找到可用模型（可能离线）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.7, MaxTokens: 160, TimeoutMinutes: 1,
	})
	if err != nil {
		return "", fmt.Errorf("续写失败: %w", err)
	}
	out := cleanGhostReply(reply)
	if out == "" {
		return "", fmt.Errorf("无建议")
	}
	return out, nil
}

// cleanGhostReply 清理续写输出：去首尾空白与包裹引号、截掉模型常见的解释前缀。
func cleanGhostReply(s string) string {
	out := strings.TrimSpace(s)
	out = strings.Trim(out, "\"“”「」『』")
	for _, prefix := range []string{"接续：", "续写：", "接续:", "续写:"} {
		out = strings.TrimPrefix(out, prefix)
	}
	return strings.TrimSpace(out)
}
