// voice_v2_tools.go — M5 刀二：语音对话的模型工具调用通道（2026-11-05 拍板落地）。
//
// 设计稿 §4：模型在语音对话中发起 function call → 语音 LLM 腿截获（识别而非
// 剥除，e211ff9e 反面）→ dry-run 评估（Tier 判级）→ T0 直执行回执；T1 登记
// P2 台账 → 前端预览卡 → 确认执行（复用 P2 台账全部防线：preview_id/hash/
// TTL/幂等）。首批两枚=status+save_task（§7-3 拍板）；T2/T3 不在清单=工具面
// 即白名单（无「拒绝并改口」分支——首批不含外发/隐私工具）。
//
// 频控（设计稿 §4）：同 action 每分钟 ≤3——防模型循环发起（tianxuan 线循环
// 调用教训）。只挂模型触发面；用户说话触发的 P2 预览不经此路径。
package app

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/intent"
)

// voiceV2ToolSchemas 首批工具定义（挂语音 LLM 请求；模型按需发起）。
var voiceV2ToolSchemas = []ai.ChatToolSchema{
	{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        "get_status",
			Description: "查询当前可用的模型引擎状态（哪些引擎在跑、什么模型）。只读，立即返回。",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
		},
	},
	{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        "save_task",
			Description: "把一个任务/待办保存到任务收件箱。调用后系统会向用户出示确认卡，用户确认后才真正保存。",
			Parameters: json.RawMessage(
				`{"type":"object","properties":{"title":{"type":"string","description":"任务标题，一句话，不超过 120 字"}},"required":["title"]}`,
			),
		},
	},
}

// voiceV2ToolPreviewEvent 模型工具 T1 预览凭据下发事件（前端出确认卡；
// navigate 事件同款通道）。
const voiceV2ToolPreviewEvent = "gaea-voice-tool-preview"

// ─────────── 频控（滑动窗：同 action 60s ≤3） ───────────

var (
	voiceV2ToolRateMu    sync.Mutex
	voiceV2ToolRateHits  = map[string][]time.Time{}
	voiceV2ToolRateLimit = 3
	voiceV2ToolRateWin   = time.Minute
	// voiceV2ToolRateNow 可桩点（测试固定时钟）。
	voiceV2ToolRateNow = time.Now
)

func voiceV2ToolAllow(action string) bool {
	voiceV2ToolRateMu.Lock()
	defer voiceV2ToolRateMu.Unlock()
	now := voiceV2ToolRateNow()
	hits := voiceV2ToolRateHits[action]
	fresh := hits[:0]
	for _, t := range hits {
		if now.Sub(t) < voiceV2ToolRateWin {
			fresh = append(fresh, t)
		}
	}
	if len(fresh) >= voiceV2ToolRateLimit {
		voiceV2ToolRateHits[action] = fresh
		return false
	}
	voiceV2ToolRateHits[action] = append(fresh, now)
	return true
}

// resetVoiceV2ToolRate 测试隔离（包级全局防跨例泄漏）。
func resetVoiceV2ToolRate() {
	voiceV2ToolRateMu.Lock()
	voiceV2ToolRateHits = map[string][]time.Time{}
	voiceV2ToolRateMu.Unlock()
}

// ─────────── 分派（工具调用 → 执行/台账 → 回执话术） ───────────

// dispatchVoiceV2Tool 模型发起的一次工具调用。返回 (ackText, previewPayload)。
// ackText=回给模型的面向用户的回执话术（管线经 onDelta 下发=TTS+transcript 同源）；
// previewPayload 非空=T1 已登记台账，需经事件下发前端出确认卡。
// execVoiceV2Action 为共享执行层（T0 直执行；T1 由本函数先登记台账，确认后
// 走 VoiceV2ExecuteAction 同一执行面）。
func (a *mediaState) dispatchVoiceV2Tool(call ai.ChatToolCall) (string, map[string]interface{}) {
	var args struct {
		Title string `json:"title"`
	}
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	switch call.Function.Name {
	case "get_status":
		if !voiceV2ToolAllow("get_status") {
			return "这个查询刚才太频繁了，先歇一歇再说。", nil
		}
		reply, ok := execVoiceV2Action(a.app, string(intent.ActionStatus),
			&intent.Intent{Action: intent.ActionStatus, Target: "model"})
		if !ok {
			return "抱歉，状态查询没跑成，请稍后再试。", nil
		}
		return reply, nil
	case "save_task":
		title := sanitizeVoiceTaskTitle(args.Title)
		if title == "" {
			return "想帮你记任务，但没听清内容是什么，再说一次任务名称好吗？", nil
		}
		if !voiceV2ToolAllow("save_task") {
			return "保存任务的请求刚才太频繁了，先歇一歇。", nil
		}
		reply := "我来帮你保存任务「" + title + "」，请在确认卡上点执行。"
		id, hash := a.registerVoiceV2ToolPreview(intent.ActionSaveTask, title, reply)
		return reply, map[string]interface{}{
			"previewId": id, "intentHash": hash,
			"action": string(intent.ActionSaveTask), "reply": reply,
		}
	default:
		// 工具面即白名单：清单外名字不该出现在请求里（注入面已限定），
		// 兜底诚实拒绝——不编造执行成功。
		return "这个操作我还没有开通，暂时帮不了。", nil
	}
}

// sanitizeVoiceTaskTitle 标题清洗（去空白；120 rune 上限与 reSaveTask 同口径）。
func sanitizeVoiceTaskTitle(s string) string {
	runes := []rune(s)
	start, end := 0, len(runes)
	for start < end && (runes[start] == ' ' || runes[start] == '\t') {
		start++
	}
	for end > start && (runes[end-1] == ' ' || runes[end-1] == '\t') {
		end--
	}
	out := string(runes[start:end])
	if n := len([]rune(out)); n > 120 {
		out = string([]rune(out)[:120])
	}
	return out
}

// registerVoiceV2ToolPreview 模型工具 T1 入 P2 台账（与 VoiceV2PreviewIntent
// 同一台账/同一 TTL/同一幂等面——执行的就是登记的那个动作，无 drift）。
func (a *mediaState) registerVoiceV2ToolPreview(action intent.Action, target, reply string) (string, string) {
	hash := voiceV2PreviewHash(string(action), target, reply)
	id := voiceV2PreviewID()
	voiceV2PreviewMu.Lock()
	pruneVoiceV2PreviewLocked()
	voiceV2PreviewLedger[id] = &voiceV2ActionPreview{
		text:      reply,
		reply:     reply,
		action:    string(action),
		target:    target,
		hash:      hash,
		expiresAt: time.Now().Add(voiceV2PreviewTTL),
	}
	voiceV2PreviewMu.Unlock()
	return id, hash
}
