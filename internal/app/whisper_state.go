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
	// v4.8.3：自引用声明——回调闭包内产物图片卡片需要 Server 引用；闭包捕获
	// 变量本身，运行时（Start 之后）必已赋值（赋值先行于 Start 内部 goroutine
	// 创建，happens-before 成立）。
	var srv *weixin.Server
	srv = weixin.New(cfg, func(userMsg, fromUser string) (string, error) {
		// v4.4 任务化路由（第一档）：提醒类请求就地处理（解析→落盘→确认），
		// 不进聊天管道。
		if reply, handled := w.tryWxReminder(userMsg, ast.ID); handled {
			return reply, nil
		}
		// v4.6.1 指令中枢 S4.5：微信消息接统一路由——提醒特例之外，navigate /
		// generate_image / edit_image（v4.9 对话式改图，带助手上下文取入站图
		// 缓存）/ status / reminder（语音同款能力面）全部命中即执行，回复经
		// 同一回推通道；未命中才走原轻语聊天。产物文件卡片（CardPath）经
		// SendFileCard 回推。
		if w.app != nil {
			if res := w.app.routeIntentWithResultForAssistant(userMsg, ast.ID); res.Handled {
				if res.CardPath != "" {
					// v4.8.3 真协议图片卡片：SendFileCard 内部完成 getuploadurl
					// → CDN 密文上传 → image_item 卡片 + caption 补发（任何
					// 失败降级文本卡片），返回 nil 即「已送出」——回空串让
					// handle 跳过重复推送；仅当连降级都失败才回文本路径由
					// 外层 Push 兜底。
					if sendErr := srv.SendFileCard(res.CardPath, res.Reply); sendErr == nil {
						return "", nil
					}
					return res.Reply + "（产物：" + res.CardPath + "）", nil
				}
				return res.Reply, nil
			}
		}
		// 助手名注入（如"峨嵋"）收编进 whisperChatAsAssistant：赋值移入
		// WhisperChat 的 LockTurn 持锁窗口——同人格多助手共享同一 orchestrator，
		// 原锁外直写 orch.AssistantName 在并发回调下互相覆盖且构成数据竞争。
		result, err := w.whisperChatAsAssistant(userMsg, ast.PersonalityID, ast.Name, false)
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
	// v4.8.3：识别器换 visionOCRText——多模态 Qwen 主模型优先（手写体强），
	// PaddleOCR 三级链降为兜底。
	if w.app != nil {
		srv.MediaRecognizer = weixin.OCRMediaRecognizer(w.app.visionOCRText)
	}
	// v4.9 对话式改图：入站图片旁路缓存——识别链路解密落盘后把文件复制一份
	// 进 wx_edit_cache 自持缓存（TTL 10 分钟，同助手只留最新一张），改图意图
	// 执行层（execEditImage）按 ast.ID 取用。缓存失败只记日志，不影响识别与
	// 聊天主流程；现有 MediaRecognizer 注入与回调链顺序不变。
	srv.OnInboundImage = func(fromUser, localPath string) {
		if _, err := wxEditImageCache(w.whisperDataRoot).Set(ast.ID, localPath); err != nil {
			slog.Warn("[assistant] 入站图片改图缓存失败", "assistant", ast.ID, "err", err)
		}
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
		// v4.9：助手停用即清它的改图缓存（自持副本文件一并删除）；删除助手的
		// 全量清理走 wxEditImageCache(...).PurgeAll()。
		wxEditImageCache(w.whisperDataRoot).Delete(id)
	}
}
