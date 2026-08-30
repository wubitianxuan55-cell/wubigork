package app

import (
	"fmt"
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

	// 确保核心 AI 助手 gaea 始终存在（角色中心必须有 gaea；
	// 旧数据无 gaea 时补建，已有则跳过）
	if _, ok := w.assistantMgr.FindByPersonality("gaea"); !ok {
		coreAst := assistant.Assistant{
			ID:            "gaea",
			Name:          "gaea",
			PersonalityID: "gaea",
			Enabled:       true,
		}
		if token := os.Getenv("WXCLAW_TOKEN"); token != "" {
			coreAst.WxToken = token
		}
		if err := w.assistantMgr.Add(coreAst); err != nil {
			slog.Warn("[assistant] 创建核心助手 gaea 失败", "err", err)
		} else {
			slog.Info("[assistant] 核心 AI 助手 gaea 已就绪")
		}
	}

	for _, ast := range w.assistantMgr.Enabled() {
		w.startAssistantWx(ast)
	}

	// v4.4：微信「离线代办」——加载持久化提醒 + 启动到点回推循环。
	w.loadReminders()
	w.startReminderTicker()
}

func (w *whisperState) startAssistantWx(ast assistant.Assistant) {
	cfg := weixin.Config{
		ILinkURL:      "https://ilinkai.weixin.qq.com",
		BotToken:      ast.WxToken,
		BotID:         ast.WxBotID,
		AssistantID:   ast.ID,
		PersonalityID: ast.PersonalityID,
	}
	srv := weixin.New(cfg, func(userMsg, fromUser string) (string, error) {
		// v4.4 任务化路由（第一档）：提醒类请求就地处理（解析→落盘→确认），
		// 不进聊天管道。
		if reply, handled := w.tryWxReminder(userMsg, ast.ID); handled {
			return reply, nil
		}
		// v4.6.1 指令中枢 S4.5：微信消息接统一路由——提醒特例之外，navigate /
		// generate_image / status / reminder（语音同款能力面）全部命中即执行，
		// 回复经同一回推通道；未命中才走原轻语聊天。产物文件卡片（CardPath）
		// 待 iLink 上传端点探明后接线（当前回复文本已带产物去向）。
		if w.app != nil {
			if res := w.app.routeIntentWithResult(userMsg); res.Handled {
				if res.CardPath != "" {
					return res.Reply + "（产物：" + res.CardPath + "）", nil
				}
				return res.Reply, nil
			}
		}
		// 注入助手自定义名字（如"峨嵋"），系统提示词用该名字而非默认"gaea"
		if orch := w.getOrCreateOrch(ast.PersonalityID); orch != nil && ast.Name != "" {
			orch.AssistantName = ast.Name
		}
		result, err := w.WhisperChatWithSearch(userMsg, ast.PersonalityID, false, false)
		if err != nil {
			return "", err
		}
		reply, _ := result["reply"].(string)
		if reply == "" {
			reply = "（思考中…）"
		}
		return reply, nil
	})
	// v4.8 子项 b：图片消息→OCR 识别一行注入（url→下载→OCR→清理）。
	if w.app != nil {
		srv.MediaRecognizer = weixin.OCRMediaRecognizer(w.app.GaeaOCRText)
	}
	// 会话过期钩子（T6-9.1）：errcode=-14 时 Server 触发回调并停止轮询——
	// 这里 emit 前端 notice 事件，让用户看到提示并重新扫码绑定；
	// 状态已由 Server.SessionExpired() 透出（WhisperWeixinStatus 的 wxSessionExpired）
	srv.OnSessionExpired = func() {
		name := ast.Name
		if name == "" {
			name = ast.ID
		}
		slog.Warn("[assistant] 微信会话过期，请重新扫码绑定", "assistant", ast.ID)
		w.emit("gaea-event", map[string]interface{}{
			"kind":  "notice",
			"level": "warn",
			"text":  fmt.Sprintf("微信助手 %s 会话过期，请重新扫码绑定", name),
		})
	}
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
