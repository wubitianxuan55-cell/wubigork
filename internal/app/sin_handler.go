package app

// ── 原罪（sin）板块后端：闲庭 play 空间的图文混杂故事创作 ──
//
// 设计口径（用户拍板：闲庭新增独立板块、复用办公组件、以对话为主、可生成
// 图文混杂的故事，取名「原罪」）：
//   - 会话存储复用 internal/chat 统一 Store（话题 mode=sin）——与聊天板块
//     同表不同域，两端各自过滤，互不串台，零新增存储格式；
//   - 文本生成复用 ai.Client 既有流式通道（ChatStreamChunks）+ 功能级路由
//     routeModel("sin")：可在模型中心把原罪单独绑到一个模型，与聊天/小说
//     互不干扰；
//   - 插图复用绘梦图像后端（mediaState.GenerateFreeImage 同一条生成链，
//     play 护栏 image_safe_mode 照常生效），产物按 space=play /
//     source_board=sin 登记进图像域台账（画室素材库可按来源筛选）。
//
// 事件：sin-stream:<runID>（delta / reasoning / done / error），与聊天板块
// chat-stream:<runID> 同形，前端订阅方式一致。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/modelengine"
)

// sinIllustrationsExtraKey 助手消息 extra 内插图映射的键（cue 索引 → 本地路径）。
const sinIllustrationsExtraKey = "illustrations"

// sinRunSeq 流式 runID 进程内序号：同一毫秒并发/连发不撞名（v4.219 同坑）。
var sinRunSeq atomic.Uint64

// ── 在途故事流登记（取消支持，v4.258）──
//
// 每个话题同时最多一个在途流（前端一次一轮），登记句柄供 SinCancel 精确取消：
// 取消 = ctx 中止（底层 HTTP 请求断开）+ 标记 cancelled —— 收尾按「保留已生成的
// 部分」落库（用户消息 + 部分正文 + extra.cancelled），不静默丢内容。
type sinRun struct {
	topicID   string
	cancel    context.CancelFunc
	cancelled atomic.Bool
}

var (
	sinRunsMu sync.Mutex
	sinRuns   = map[string]*sinRun{}
)

func registerSinRun(topicID string) *sinRun {
	run := &sinRun{topicID: topicID}
	sinRunsMu.Lock()
	sinRuns[topicID] = run
	sinRunsMu.Unlock()
	return run
}

func unregisterSinRun(topicID string, run *sinRun) {
	sinRunsMu.Lock()
	if sinRuns[topicID] == run {
		delete(sinRuns, topicID)
	}
	sinRunsMu.Unlock()
}

func takeSinRun(topicID string) *sinRun {
	sinRunsMu.Lock()
	defer sinRunsMu.Unlock()
	return sinRuns[topicID]
}

// SinCancel 取消某故事当前在途的生成（Composer「停止」按钮）：
//   - 中止底层流式请求；
//   - 已生成的部分照常落库（extra.cancelled=true），用户消息不丢；
//   - 无在途流时返回 error（前端按「没有正在生成的故事」如实提示）。
func (a *App) SinCancel(topicID string) error {
	if err := a.sinTopicGuard(topicID); err != nil {
		return err
	}
	run := takeSinRun(topicID)
	if run == nil {
		return fmt.Errorf("没有正在生成的故事")
	}
	run.cancelled.Store(true)
	if run.cancel != nil {
		run.cancel()
	}
	return nil
}

// ── 话题与消息 ────────────────────────────────────────────────

// SinTopicsList 列出全部原罪故事（mode=sin；聊天板块话题不混入）。
func (a *App) SinTopicsList() ([]chat.Topic, error) {
	if a.chatStore == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	all, err := a.chatStore.ListTopics()
	if err != nil {
		slog.Error("原罪故事列表读取失败", "error", err)
		return nil, err
	}
	out := make([]chat.Topic, 0, len(all))
	for _, t := range all {
		if t.Mode == sinTopicMode {
			out = append(out, t)
		}
	}
	return out, nil
}

// SinTopicCreate 新建故事话题（title 空 → 「新故事」）。
func (a *App) SinTopicCreate(title string) (chat.Topic, error) {
	if a.chatStore == nil {
		return chat.Topic{}, fmt.Errorf("chat store 未初始化")
	}
	if strings.TrimSpace(title) == "" {
		title = "新故事"
	}
	id := fmt.Sprintf("sin_%d_%d", time.Now().UnixMilli(), sinRunSeq.Add(1))
	if err := a.chatStore.CreateTopic(id, title, sinTopicMode); err != nil {
		return chat.Topic{}, err
	}
	return a.chatStore.GetTopic(id)
}

// SinTopicRename 重命名故事（仅 sin 话题；非本板块话题拒改）。
func (a *App) SinTopicRename(id, title string) error {
	if err := a.sinTopicGuard(id); err != nil {
		return err
	}
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("标题不能为空")
	}
	return a.chatStore.RenameTopic(id, title)
}

// SinTopicDelete 删除故事（消息级联删除；仅 sin 话题）。
func (a *App) SinTopicDelete(id string) error {
	if err := a.sinTopicGuard(id); err != nil {
		return err
	}
	return a.chatStore.DeleteTopic(id)
}

// SinTopicClear 清空故事消息（保留话题本身；仅 sin 话题）。
func (a *App) SinTopicClear(id string) error {
	if err := a.sinTopicGuard(id); err != nil {
		return err
	}
	return a.chatStore.ClearMessages(id)
}

// SinMessages 读取故事全部消息（仅 sin 话题）。
func (a *App) SinMessages(topicID string) ([]chat.Message, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return nil, err
	}
	return a.chatStore.ListMessages(topicID)
}

// sinTopicGuard 话题归属校验：话题必须存在且 mode=sin（fail-closed，
// 防止前端误传聊天话题 id 后把其他板块的会话改写/删掉）。
func (a *App) sinTopicGuard(id string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("故事 ID 不能为空")
	}
	t, err := a.chatStore.GetTopic(id)
	if err != nil {
		return err
	}
	if t.Mode != sinTopicMode {
		return fmt.Errorf("不是原罪故事: %s", id)
	}
	return nil
}

// ── 文本生成 ──────────────────────────────────────────────────

// SinStream 故事续写流式入口：立即返回 runID，前端订阅
// "sin-stream:<runID>"（delta/reasoning/done/error）。
//
// 模型：功能级绑定 sin（模型中心「原罪」）→ 全局激活 → 兜底，与聊天板块
// 同一条 routeModel 降级链；未绑定即为全局模型。
func (a *App) SinStream(topicID, message string) (string, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return "", err
	}
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("故事指令不能为空")
	}
	if a.client == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	eng, model, source := a.routeModel("sin")
	if model == "" {
		return "", fmt.Errorf("未找到可用模型（可能处于离线模式且无本地模型）")
	}
	runID := fmt.Sprintf("ss_%d_%d", time.Now().UnixMilli(), sinRunSeq.Add(1))
	run := registerSinRun(topicID)
	go a.runSinStream(runID, topicID, message, eng, model, source, run)
	return runID, nil
}

func (a *App) runSinStream(runID, topicID, userMessage, eng, model, source string, run *sinRun) {
	defer unregisterSinRun(topicID, run)
	defer func() {
		if r := recover(); r != nil {
			slog.Error("sin stream panic", "panic", r, "runID", runID)
			a.emit("sin-stream:"+runID, map[string]interface{}{"type": "error", "error": "流式生成异常"})
		}
	}()

	// 前情装配：历史消息在写入本轮交换之前读取（本轮用户消息由 userMessage
	// 直接带入提示，避免重复）。
	history, err := a.chatStore.ListMessages(topicID)
	if err != nil {
		slog.Warn("原罪前情读取失败，按无历史继续", "topicID", topicID, "error", err)
		history = nil
	}
	// 角色库内容（本故事已选角色）注入：写作锚点。硬隔离上不违背——角色库是
	// 跨板块共享资产层，不是办公数据面（见 sin_store.go 头注）。
	cast := a.sinCastCharacters(topicID)
	userPrompt := buildSinUserPrompt(history, userMessage, cast)

	// play 护栏（配置未启用 = 零值 = 不钳制，与既有四个直连生成点同语义）。
	g := playGuardrails()
	opts := ai.ChatSimpleOptions{
		EngineID:       eng,
		Temperature:    clampPlayTemperature(sinTemperature, g.TemperatureMax),
		MaxTokens:      clampPlayMaxTokens(sinMaxTokens, g.MaxOutputTokens),
		TimeoutMinutes: 10,
	}

	chunks, cancel, err := a.client.ChatStreamChunks(a.ctx, model, sinSystemPrompt(), userPrompt, opts)
	if err != nil {
		a.emit("sin-stream:"+runID, map[string]interface{}{"type": "error", "error": err.Error()})
		return
	}
	defer cancel()
	// 取消句柄交给登记项：SinCancel 调用它即断开底层请求（chunks 通道随之收尾）
	run.cancel = cancel
	if run.cancelled.Load() {
		cancel() // 极窄竞态：注册与取消同帧 → 立即中止
	}

	var reply, reasoning strings.Builder
	var usage *ai.ChatUsage
	for chunk := range chunks {
		if run.cancelled.Load() {
			break
		}
		if chunk.Error != "" {
			// 取消引发的读错误按「已停止」收尾（不当失败报）
			if run.cancelled.Load() {
				break
			}
			a.emit("sin-stream:"+runID, map[string]interface{}{"type": "error", "error": chunk.Error})
			return
		}
		if chunk.Done {
			usage = chunk.Usage
			break
		}
		if chunk.Content != "" {
			reply.WriteString(chunk.Content)
			a.emit("sin-stream:"+runID, map[string]interface{}{"type": "delta", "content": chunk.Content})
		}
		if chunk.Reasoning != "" {
			reasoning.WriteString(chunk.Reasoning)
			a.emit("sin-stream:"+runID, map[string]interface{}{"type": "reasoning", "content": chunk.Reasoning})
		}
	}

	replyStr := reply.String()
	reasoningStr := reasoning.String()
	cancelled := run.cancelled.Load()
	extra := ""
	extraMap := map[string]interface{}{"reasoning": reasoningStr}
	if cancelled {
		extraMap["cancelled"] = true
	}
	if b, err := json.Marshal(extraMap); err == nil {
		extra = string(b)
	}
	// 落库失败必须透传（消息已生成但未持久化 = 用户可见的失败，不静默吞错）。
	if err := a.appendChatExchange(topicID, userMessage, replyStr, extra); err != nil {
		slog.Error("原罪故事落库失败", "runID", runID, "topicID", topicID, "error", err)
		a.emit("sin-stream:"+runID, map[string]interface{}{
			"type":  "error",
			"error": "回复生成完成但故事保存失败: " + err.Error(),
		})
		return
	}

	// 落库后取回本轮助手消息 id：插图回写（SinIllustrate）按 id 定位。
	// 单写者（本话题同轮次）下取最后一条 assistant 即本轮回复。
	messageID := int64(0)
	if msgs, err := a.chatStore.ListMessages(topicID); err == nil {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == "assistant" {
				messageID = msgs[i].ID
				break
			}
		}
	}

	costCNY := 0.0
	if usage != nil {
		usdCny := 0.0
		if a.cfg != nil {
			usdCny = a.cfg.UsdCnyRate
		}
		costCNY = modelengine.EstimateCostCNY(eng, model, usage.PromptTokens, usage.CompletionTokens, usdCny)
	}
	a.emit("sin-stream:"+runID, map[string]interface{}{
		"type":       "done",
		"reply":      replyStr,
		"reasoning":  reasoningStr,
		"topicID":    topicID,
		"message_id": messageID,
		"cancelled":  cancelled,
		"answered_by": map[string]interface{}{
			"engine": eng, "model": model, "source": source, "cost_cny": costCNY,
		},
	})
}

// ── 插图 ──────────────────────────────────────────────────────

// SinIllustrate 为故事中的一处插图标记生成画面，并把产物路径回写到该轮
// 助手消息的 extra.illustrations[cue]（重开故事直接按映射渲染，不重新生成）。
//
// 参数：topicID 故事 id；messageID 目标助手消息 id（<=0 表示不落库，仅返回
// 图片）；cue 前端解析出的标记索引/键；prompt 画面提示词；size 留空按后端默认。
func (a *App) SinIllustrate(topicID string, messageID int64, cue string, prompt string, size string) (map[string]interface{}, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("缺少插图提示词")
	}
	if a.mediaState == nil || a.mediaState.client == nil {
		return nil, fmt.Errorf("图像后端未初始化")
	}
	// ── 角色一致性（v4.258）：先做文本锚点（所有后端受益），再按后端能力决定
	//    是否加 T2 图像参考槽（ComfyUI krea2/z-image-turbo · Herdsman 图生图）。
	cast := a.sinCastCharacters(topicID)
	picked := sinPickRefCharacters(cast, prompt)
	promptUsed, anchorNames := sinAugmentPromptWithCast(prompt, picked)
	refModel := a.mediaState.cfg.ImageModel
	refMode, refMethod, refOK, refReason := sinRefPlan(a.mediaState.cfg.ImageBackend, refModel)
	refs := sinResolveRefImages(picked)
	switch {
	case !refOK:
		if len(picked) == 0 {
			refReason = "" // 没点名任何角色：不算「不支持」，只是这次用不上
		}
	case len(refs.images) == 0:
		// 后端支持参考槽，但这一轮拿不到任何可用参考图（没选角色兜底 / 角色库
		// 只有远端 URL / 本地文件缺失）——如实跳过并给原因，不硬塞。
		refOK = false
		if len(picked) > 0 {
			refReason = "该角色没有可用的参考图/立绘（角色库可生成剧照后重试）"
		} else {
			refReason = ""
		}
	}
	if !refOK {
		// 没有可用参考图时必须退回纯文本：留下 mode=img2img 会让后端因缺参考图
		// 整单报错（真机症状「marshal image request: 图生图需要提供参考图」），
		// 插图不该因为用不上参考槽而失败。
		refMode, refMethod = "", ""
		refs.images, refs.names, refs.charID = nil, nil, ""
	}

	// 硬隔离（v4.257）：产物落原罪自有目录（<用户配置目录>/gaea/sin/art），
	// 台账来源记为 sin——不写办公工作区、不依赖办公 ImageSaveDir 配置，
	// 且生成链内部只登记一条（调用方不再补第二条）。
	gen := imageGenInternal{
		prompt: strings.TrimSpace(promptUsed), size: size,
		sourceBoard: sinSourceBoard, saveDir: sinArtDir(),
		mode: refMode, refImages: refs.images, refMethod: refMethod, denoise: sinRefDenoise,
	}
	if len(refs.names) == 1 {
		gen.characterID = refs.charID // 单角色锚定：台账按「真正带参考图」的角色回溯
	}
	refFallback := false
	res, err := a.mediaState.generateImageInternal(gen)
	genErr := func() string {
		if msg, ok := res["error"].(string); ok {
			return msg
		}
		return ""
	}()
	if len(refs.images) > 0 && (err != nil || genErr != "") {
		// 参考图这条路失败不该让插图整体失败：退回纯文本重试一次（如实标记 fallback）
		slog.Warn("原罪参考槽生成失败，退回纯文本重试", "error", err, "resError", genErr)
		refFallback = true
		refs.images, refs.names, refs.charID = nil, nil, ""
		gen.mode, gen.refImages, gen.refMethod, gen.characterID = "", nil, "", ""
		res, err = a.mediaState.generateImageInternal(gen)
	}
	if err != nil {
		return nil, err
	}
	if msg, ok := res["error"].(string); ok && msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}
	images, _ := res["images"].([]imageItem)
	if len(images) == 0 {
		return nil, fmt.Errorf("未生成图片")
	}
	item := images[0]
	path := item.FilePath
	// 兜底落盘：后端在极少数情况下不回 data URL（例如仅返回远端 URL 且下载
	// 失败）时 FilePath 为空——这里再写一次原罪自有目录，保证插图有稳定本地
	// 路径可供 Markdown 导出与重开渲染。与生成链内保存同目录，不产生副本。
	if path == "" && item.Image != "" {
		path = a.mediaState.saveMediaToDisk(item.Image, prompt, sinArtDir())
	}
	if path == "" {
		return nil, fmt.Errorf("插图未落盘（后端未返回可保存的图片数据）")
	}
	// 台账登记已在生成链内部完成（recordImageHubGeneratedFor：
	// space=play / source_board=sin）——这里不再补登记，避免画室同一张图重复两条。

	persisted := false
	if messageID > 0 && cue != "" {
		if err := a.sinAttachIllustration(messageID, cue, path); err != nil {
			slog.Warn("原罪插图回写失败（图片已生成）", "messageID", messageID, "error", err)
		} else {
			persisted = true
		}
	}
	return map[string]interface{}{
		"path":           path,
		"cue":            cue,
		"message_id":     messageID,
		"model":          item.Model,
		"seed":           item.Seed,
		"persisted":      persisted,
		"prompt_used":    promptUsed,
		"ref_used":       len(refs.images) > 0,
		"ref_characters": refs.names,
		"ref_reason":     refReason,
		"ref_fallback":   refFallback,
		"anchor_added":   anchorNames,
	}, nil
}

// sinAttachIllustration 把 (cue → path) 并入指定消息 extra.illustrations。
// 已有映射保留（多张插图逐张追加），其他 extra 字段（reasoning 等）不动。
func (a *App) sinAttachIllustration(messageID int64, cue, path string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	msg, err := a.chatStore.GetMessage(messageID)
	if err != nil {
		return err
	}
	extra := map[string]interface{}{}
	if raw := strings.TrimSpace(msg.Extra); raw != "" {
		_ = json.Unmarshal([]byte(raw), &extra) // 坏 JSON 按空 extra 继续（辅助映射容错）
	}
	arts, _ := extra[sinIllustrationsExtraKey].(map[string]interface{})
	if arts == nil {
		arts = map[string]interface{}{}
	}
	arts[cue] = path
	extra[sinIllustrationsExtraKey] = arts
	b, err := json.Marshal(extra)
	if err != nil {
		return err
	}
	return a.chatStore.UpdateMessageExtra(messageID, string(b))
}

// ── 导出 ──────────────────────────────────────────────────────

// SinExportMarkdown 把故事导出为图文混杂的 Markdown：正文原样保留，插图
// 标记替换为 Markdown 图片（已生成插图用本地路径，未生成的保留画面描述
// 作为占位引用），供前端经「另存为」落盘成 .md/.txt 成品。
func (a *App) SinExportMarkdown(topicID string) (string, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return "", err
	}
	topic, err := a.chatStore.GetTopic(topicID)
	if err != nil {
		return "", err
	}
	msgs, err := a.chatStore.ListMessages(topicID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# " + topic.Title + "\n\n")
	b.WriteString("> 原罪 · 图文故事（`" + topic.CreatedAt + "` 起）\n\n")
	for _, m := range msgs {
		arts := sinIllustrationMap(m.Extra)
		if m.Role == "user" {
			b.WriteString("**我：**\n\n" + strings.TrimSpace(m.Content) + "\n\n")
			continue
		}
		b.WriteString(renderSinStoryMarkdown(m.Content, arts))
		b.WriteString("\n---\n\n")
	}
	return b.String(), nil
}

// sinIllustrationMap 从消息 extra 解析插图映射（坏 JSON/缺字段 → 空映射）。
func sinIllustrationMap(extra string) map[string]string {
	out := map[string]string{}
	raw := strings.TrimSpace(extra)
	if raw == "" {
		return out
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return out
	}
	arts, _ := m[sinIllustrationsExtraKey].(map[string]interface{})
	for k, v := range arts {
		if s, ok := v.(string); ok && s != "" {
			out[k] = s
		}
	}
	return out
}

// renderSinStoryMarkdown 把一条助手正文渲染为 Markdown：插图标记 →
// ![画面描述](路径)。未生成插图的标记渲染为斜体占位（不吞掉画面描述）。
func renderSinStoryMarkdown(content string, arts map[string]string) string {
	var b strings.Builder
	rest := content
	idx := 0
	for {
		start := strings.Index(rest, sinIllustrationCueOpen)
		if start < 0 {
			b.WriteString(rest)
			break
		}
		end := strings.Index(rest[start+len(sinIllustrationCueOpen):], sinIllustrationCueClose)
		if end < 0 {
			b.WriteString(rest)
			break
		}
		prompt := strings.TrimSpace(rest[start+len(sinIllustrationCueOpen) : start+len(sinIllustrationCueOpen)+end])
		b.WriteString(rest[:start])
		if path, ok := arts[sinCueKey(idx)]; ok && path != "" {
			b.WriteString("![" + prompt + "](" + path + ")")
		} else if path, ok := arts[prompt]; ok && path != "" {
			b.WriteString("![" + prompt + "](" + path + ")")
		} else {
			b.WriteString("_（插图未生成：" + prompt + "）_")
		}
		rest = rest[start+len(sinIllustrationCueOpen)+end+len(sinIllustrationCueClose):]
		idx++
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return ""
	}
	return text + "\n\n"
}

// sinCueKey 插图标记的稳定键：同一条消息内按出现次序编号（0 起）。
// 键必须与前端解析规则一致（前端 lib.sinCueKey 同语义），否则导出/重开时
// 插图映射对不上，会退化成「插图未生成」占位。
func sinCueKey(idx int) string {
	return strconv.Itoa(idx)
}
