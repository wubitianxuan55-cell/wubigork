package app

import (
	"context"
	"fmt"
)

// ── Ghost Text 内联补全 ─────────────────────────────────────
// ── Cmd+K 命令编辑 ──────────────────────────────────────────

// CmdKEdit 根据自然语言指令编辑选中文本。
// 小说编辑器内调用 → 走 novel 功能绑定路由（P1 已知限制修复：
// 此前用全局 cfg.Model + 活跃引擎，忽略功能绑定）。
func (a *App) CmdKEdit(selectedText string, instruction string, styleProfile string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}

	featEng, featModel, _ := a.routeModel("novel")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background() // 测试/异常路径防御：client 需要非 nil ctx
	}
	edited, err := a.client.CmdKEdit(ctx, featEng, featModel, selectedText, instruction, styleProfile)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"edited": edited,
	}, nil
}
