package app

// gaea_session_title.go — 会话自动命名（2026-10，dsh session-title 蒸馏取道，
// docs/gaea-dsh-021-core-distill-2026-10.md P1-A）：首个成功回合结束后，用
// 办公本地路由（routeOfficeLocal，数据不出本机）给会话起一个短标题，写入
// 既有 .titles.json 注册表（与 GaeaRenameSession 同点，前端列表零改动零新
// 绑定）。上游三语义对齐：
//   ① latest-wins：注册表值即真相，自动命名只是最早的写入者之一；
//   ② 用户改名钉住：只在无标题时生成——用户命名后自动永不再写（上游
//      "explicit user rename pins" 的等价实现，无需来源字段）；
//   ③ 失败自愈：生成失败当轮不重试，触发条件是「无标题 + 回合成功」，
//      下一回合自然再试（天然退避，无定时器）。
// 与上游的刻意偏离：不持久化 provider/source 归因（注册表无来源字段，刻
// 意不扩 schema）；不做随对话演进重生成（v1 一次定名，演进重生成留观察池）。
// 触发点在 gaeaBuildController 的 TurnDone 槽（与自动做梦同位）。

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// gaeaAutoTitleMaxRunes 是自动标题的长度上限：会话列表单行展示宽度内的
// 安全值（中文 12 字上下），超长由模型侧约束+本地截断双保险。
const gaeaAutoTitleMaxRunes = 16

// gaeaAutoTitleState 是按会话路径的单飞登记：同一会话同一时刻至多一次
// 生成（TurnDone 与面板刷新可能密集触发）。
var gaeaAutoTitleState = struct {
	sync.Mutex
	running map[string]bool
}{running: map[string]bool{}}

// maybeAutoTitleAfterTurn 在回合成功结束后考虑自动命名（TurnDone 槽调用）。
// 全部早退条件在起 goroutine 前判完：无控制器/无客户端/无会话/已有标题/
// 已在生成中——都不付出任何成本。
func (a *App) maybeAutoTitleAfterTurn() {
	c := gaeaCtrl()
	if c == nil || a.clientRef() == nil {
		return
	}
	path := c.SessionPath()
	if strings.TrimSpace(path) == "" {
		return
	}
	dir := sessionDirForPath(path)
	if dir == "" {
		return
	}
	// 已有标题（用户命名或此前自动生成）→ 钉住不再生成（上游 ②）。
	if titles := loadSessionTitlesAll(dir); strings.TrimSpace(titles[filepath.Base(path)]) != "" {
		return
	}
	gaeaAutoTitleState.Lock()
	if gaeaAutoTitleState.running[path] {
		gaeaAutoTitleState.Unlock()
		return
	}
	gaeaAutoTitleState.running[path] = true
	gaeaAutoTitleState.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.gaeaBackgroundPanicNotice("会话自动命名", r, map[string]interface{}{"session": filepath.Base(path)})
			}
			gaeaAutoTitleState.Lock()
			delete(gaeaAutoTitleState.running, path)
			gaeaAutoTitleState.Unlock()
		}()
		title, err := a.generateSessionTitle(path)
		if err != nil {
			// 诚实静默：起名失败对用户不可感知，下一回合成功后自愈重试。
			slog.Debug("会话自动命名失败（下回合重试）", "session", filepath.Base(path), "error", err)
			return
		}
		if err := setSessionTitle(dir, path, title); err != nil {
			slog.Warn("会话自动命名写入失败", "error", err)
		}
	}()
}

// generateSessionTitle 读会话首条用户消息，经办公本地路由生成短标题。
// 只用首条 user_message（对齐上游 title 的 messageSeqs 只取 human 消息）。
func (a *App) generateSessionTitle(sessionPath string) (string, error) {
	entries, err := session.ReadEntriesFor(sessionPath)
	if err != nil {
		return "", fmt.Errorf("read session log: %w", err)
	}
	first := session.DeriveTitle(entries)
	if strings.TrimSpace(first) == "" {
		return "", fmt.Errorf("no user message to title from")
	}
	engID, model, _ := a.routeOfficeLocal("office")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	out, err := a.clientRef().ChatSimpleStreamWithOptions(ctx, model,
		"你为工作会话生成极短标题。只输出标题本身：不超过12个字、不带引号句号等标点、不换行、不加「会话」「对话」等后缀。",
		"根据这段用户请求生成会话标题：\n"+first,
		ai.ChatSimpleOptions{EngineID: engID, Feature: "office", Temperature: 0.2, MaxTokens: 40, TimeoutMinutes: 1},
	)
	if err != nil {
		return "", err
	}
	return sanitizeSessionTitle(out), nil
}

// sanitizeSessionTitle 把模型输出修整成可入库标题：压平换行、剥引点、截到
// 上限；空结果返回空串（调用方按失败处理，下回合自愈）。
func sanitizeSessionTitle(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " "))
	s = strings.Trim(s, "\"'「」『』《》`* 每日对话助手默认·。．.！!？?；;，,、")
	r := []rune(s)
	if len(r) > gaeaAutoTitleMaxRunes {
		s = string(r[:gaeaAutoTitleMaxRunes-1]) + "…"
	}
	return strings.TrimSpace(s)
}
