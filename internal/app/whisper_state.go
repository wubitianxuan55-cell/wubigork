package app

import (
	"log/slog"
	"os"

	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/channels/weixin"
)

// ── 轻语域：微信通道管理（原 app.go 中的 initWeixin/startAssistantWx/stopAssistantWx）──

func (w *whisperState) initWeixin() {
	var err error
	w.assistantMgr, err = assistant.Load(w.whisperDataRoot)
	if err != nil {
		slog.Error("[assistant] 加载失败，重试", "err", err)
		if w.assistantMgr, err = assistant.Load(w.whisperDataRoot); err != nil {
			slog.Error("[assistant] 重试加载仍失败，使用空管理器", "err", err)
			w.assistantMgr = assistant.NewEmpty(w.whisperDataRoot)
		}
	}
	w.weixinServers = make(map[string]*weixin.Server)

	// 首次启动：创建默认助手
	if len(w.assistantMgr.List()) == 0 {
		defaultAst := assistant.Assistant{
			ID:            "default",
			Name:          "轻语",
			PersonalityID: "deredere",
			Enabled:       true,
		}
		if token := os.Getenv("WXCLAW_TOKEN"); token != "" {
			defaultAst.WxToken = token
		}
		w.assistantMgr.Add(defaultAst)
	}

	for _, ast := range w.assistantMgr.Enabled() {
		w.startAssistantWx(ast)
	}
}

func (w *whisperState) startAssistantWx(ast assistant.Assistant) {
	cfg := weixin.Config{
		ILinkURL:      "https://ilinkai.weixin.qq.com",
		BotToken:      ast.WxToken,
		AssistantID:   ast.ID,
		PersonalityID: ast.PersonalityID,
	}
	srv := weixin.New(cfg, func(userMsg, fromUser string) (string, error) {
		result, err := w.WhisperChatWithSearch(userMsg, ast.PersonalityID)
		if err != nil {
			return "", err
		}
		reply, _ := result["reply"].(string)
		if reply == "" {
			reply = "（思考中…）"
		}
		return reply, nil
	})
	if err := srv.Start(); err != nil {
		slog.Error("[assistant] 微信启动失败", "assistant", ast.ID, "err", err)
		return
	}
	w.weixinMu.Lock()
	w.weixinServers[ast.ID] = srv
	w.weixinMu.Unlock()
}

func (w *whisperState) stopAssistantWx(id string) {
	w.weixinMu.Lock()
	srv, ok := w.weixinServers[id]
	if ok {
		delete(w.weixinServers, id)
	}
	w.weixinMu.Unlock()
	if ok {
		srv.Stop()
	}
}
