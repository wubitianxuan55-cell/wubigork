package app

import (
	"encoding/json"
	"fmt"

	"github.com/wubigork/wubigork/internal/util"
)

// ── 脑暴 ────────────────────────────────────────────────────

// BrainstormIdea AI 生成的创意点子
type BrainstormIdea struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Pitch    string   `json:"pitch"`
	Conflict string   `json:"conflict"`
	Audience string   `json:"audience"`
	Tags     []string `json:"tags"`
}

// BrainstormIdeas 根据题材生成 6 个核心小说点子
func (a *App) BrainstormIdeas(genre string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未就绪，请先登录")
	}

	tmpl := a.eng.Get("brainstorm-ideas")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 brainstorm-ideas 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"genre_and_prompt": fmt.Sprintf("题材: %s\n请生成 6 个不同角度的小说核心点子。", genre),
	})

	reply, err := a.client.ChatSimpleStream(a.ctx, a.cfg.Model, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI 脑暴请求失败: %w", err)
	}

	jsonStr := util.ExtractJSON(reply)
	var result struct {
		Ideas []BrainstormIdea `json:"ideas"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		runes := []rune(reply)
		return nil, fmt.Errorf("解析脑暴结果失败: %w\n原始回复前200字: %s", err, string(runes[:util.Min(len(runes), 200)]))
	}

	if len(result.Ideas) == 0 {
		return nil, fmt.Errorf("AI 未生成任何点子")
	}

	return map[string]interface{}{
		"ideas": result.Ideas,
	}, nil
}
