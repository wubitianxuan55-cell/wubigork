package app

import (
	"fmt"
)

// ── Ghost Text 内联补全 ─────────────────────────────────────
// ── Cmd+K 命令编辑 ──────────────────────────────────────────

// CmdKEdit 根据自然语言指令编辑选中文本
func (a *App) CmdKEdit(selectedText string, instruction string, styleProfile string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}

	edited, err := a.client.CmdKEdit(a.ctx, a.cfg.Model, selectedText, instruction, styleProfile)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"edited": edited,
	}, nil
}
