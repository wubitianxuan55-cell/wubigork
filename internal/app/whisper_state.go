package app

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

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
		// v4.41.1 文件消息直通聊天：注入行携带提取正文（≤6000 字），过提醒/
		// 意图路由会被正文里的「打开/看看 + 板块别名」碎片误触发（真机实证：
		// 评审报告正文含「编程」被劫持回「打开编程」）——文件内容是聊天上下文
		// 不是指令，跳过两段路由，追加「确认收件+询问需求」引导后直进轻语。
		if isWxFileInjectMsg(userMsg) {
			userMsg = applyWxFileGuidance(userMsg)
			result, err := w.whisperChatAsAssistant(userMsg, ast.PersonalityID, ast.Name, false)
			if err != nil {
				return "", err
			}
			reply, _ := result["reply"].(string)
			if reply == "" {
				reply = "（思考中…）"
			}
			return reply, nil
		}
		// v4.4 任务化路由（第一档）：提醒类请求就地处理（解析→落盘→确认），
		// 不进聊天管道。
		if reply, handled := w.tryWxReminder(userMsg, ast.ID); handled {
			return reply, nil
		}
		// v4.42 微信智能体（LLM 工具调用派发，主路径）：模型自己决定调哪个
		// 板块能力（navigate/生图/改图/提醒/发文件/状态/读屏），本地执行后结果
		// 回微信——「重新整理后发给我」类幻觉从根上消灭（模型有工具可调，不
		// 再嘴上假装）。能力门不满足（模型目录无 tools 能力位/离线模式）或
		// agent 执行出错才降级到下方关键词意图路由（快路径语义保留）；产物
		// 文件卡片经同一 SendFileCard 链回推（末张 caption 带最终回复，回空串
		// 防重复推送的既有口径不变）。
		if w.app != nil {
			if wxAgentAvailable(w.app) {
				reply, cards, agentErr := w.runWxAgentTurn(ast.ID, ast.PersonalityID, ast.Name, userMsg)
				if agentErr == nil {
					if len(cards) > 0 {
						if reply == "" {
							reply = "（已生成产物）"
						}
						var failed []string
						for i, card := range cards {
							caption := ""
							if i == len(cards)-1 {
								caption = reply
							}
							// SendFileCard 内部完成 CDN 上传 + 卡片推送（任何失败
							// 内部降级文本卡片）；仅当连降级都失败才记入 failed。
							if sendErr := srv.SendFileCard(card, caption); sendErr != nil {
								failed = append(failed, card)
							}
						}
						if len(failed) == len(cards) {
							// 全部失败：整体降级文本（路径交外层 Push 兜底）
							return reply + "（产物：" + strings.Join(failed, "；") + "）", nil
						}
						if len(failed) > 0 {
							return "（以下产物回推失败，请到桌面端查看：" + strings.Join(failed, "；") + "）", nil
						}
						return "", nil // 卡片全部送出（末张 caption 已带回复）
					}
					if reply == "" {
						reply = "（思考中…）"
					}
					return reply, nil
				}
				// agent 出错：落到下方 routeIntent 兜底（再未命中走轻语聊天）
			}
			// v4.6.1 指令中枢 S4.5（v4.42 起降级为兜底）：微信消息接统一路由
			// ——提醒特例之外，navigate / generate_image / edit_image（v4.9
			// 对话式改图，带助手上下文取入站图缓存）/ status / reminder（语音
			// 同款能力面）全部命中即执行，回复经同一回推通道；未命中才走原
			// 轻语聊天。产物文件卡片（CardPath）经 SendFileCard 回推。
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
		// v4.41.2 反幻觉护栏：未被产物推送意图接住的「发文件给我」类请求，
		// 提示模型如实说明能力边界（真机实证：聊天管道曾声称「已整理好发你」
		// 而实际什么都没发）。
		userMsg = applyWxSendHonestyGuidance(userMsg)
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
	// v4.41 微信文件收发：入站文件下载成功后回调（契约 weixin.InboundFileHandler，
	// 下载/解密由通道线实装）——复制自持进 wx_files + 内容提取一行注入，模型由此
	// 直接「看见」文件内容作答（nil/panic/空串由 clawbot 回退占位行）。wx_files
	// 持久化是有意的：stopAssistantWx 不清理——「把文件发我」的回推
	// （ActionSendLatestFile → CardPath → SendFileCard 文件卡）与桌面端打开都
	// 消费自持副本，防洪靠 wxFileStore 的数量/总量双阈值滚动清理。
	srv.FileHandler = newWxFileHandler(w.whisperDataRoot)
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
