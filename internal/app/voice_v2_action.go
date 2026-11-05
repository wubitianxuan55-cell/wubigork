// voice_v2_action.go — P2 确认执行：语音动作预览台账。
//
// P1.9 出预览卡（dry-run），本文件给「确认后执行」：preview_id + intent_hash +
// TTL + 幂等（roadmap P2 定稿口径）。台账存预览时的 text/action/target/reply，
// 执行时按台账重构 Intent 直派执行层——执行的就是预览的那个动作（不重解析，
// 无 intent-drift）。生图/发送文件/读屏留到单独权限设计之后（roadmap 原文），
// 白名单外交付「可预览不可执行」。
package app

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/intent"
)

const (
	// voiceV2PreviewTTL 预览有效期：语音流里确认是紧跟预览的即时动作，2 分钟
	// 足够；过期必须重新说话产生新预览（防止「三小时前说的一句话」被翻旧账执行）。
	voiceV2PreviewTTL = 2 * time.Minute
	// voiceV2PreviewMaxAlive 台账容量兜底（防泄漏）；超限先清过期，仍超丢最旧。
	voiceV2PreviewMaxAlive = 16
)

type voiceV2ActionPreview struct {
	text      string
	reply     string
	action    string
	target    string
	hash      string
	expiresAt time.Time
	executed  bool
}

var (
	voiceV2PreviewMu     sync.Mutex
	voiceV2PreviewLedger = map[string]*voiceV2ActionPreview{}
	voiceV2PreviewSeq    atomic.Int64
)

// voiceV2PreviewID 可桩点（测试固定 id）。
var voiceV2PreviewID = func() string {
	return fmt.Sprintf("vap_%d_%d", time.Now().UnixMilli(), voiceV2PreviewSeq.Add(1))
}

// voiceV2ExecutableActions 语音确认执行白名单（P2 roadmap：生图/发送文件/
// 读屏留到单独权限设计之后——白名单外「可预览不可执行」）。白名单再窄不做，
// 执行层复用 intent_router 既有能力（与命令面板同权限面）。
var voiceV2ExecutableActions = map[string]bool{
	string(intent.ActionNavigate): true,
	string(intent.ActionStatus):   true,
	string(intent.ActionReminder): true,
	string(intent.ActionSaveTask): true,
}

// voiceV2T0Actions M5 刀一（2026-11-05 拍板 §7-1 采纳）：只读/可逆导航免确认
// 自动执行——预览时直跑并返回 executed，不再入台账等确认（减摩擦；T1
// reminder/save_task 维持预览+确认卡）。T0 动作每轮至多一次（用户说话触发，
// 无模型循环发起面，频控挂刀二模型工具通道）。
var voiceV2T0Actions = map[string]bool{
	string(intent.ActionNavigate): true,
	string(intent.ActionStatus):   true,
}

func voiceV2PreviewHash(action, target, reply string) string {
	sum := sha256.Sum256([]byte(action + "|" + target + "|" + reply))
	return fmt.Sprintf("%x", sum[:8])
}

// pruneVoiceV2PreviewLocked 清过期；仍超上限丢最旧（插入前调用，调用方持锁）。
func pruneVoiceV2PreviewLocked() {
	now := time.Now()
	for id, p := range voiceV2PreviewLedger {
		if now.After(p.expiresAt) {
			delete(voiceV2PreviewLedger, id)
		}
	}
	for len(voiceV2PreviewLedger) >= voiceV2PreviewMaxAlive {
		oldestID, oldestAt := "", time.Now().Add(time.Minute)
		for id, p := range voiceV2PreviewLedger {
			if p.expiresAt.Before(oldestAt) {
				oldestID, oldestAt = id, p.expiresAt
			}
		}
		if oldestID == "" {
			return
		}
		delete(voiceV2PreviewLedger, oldestID)
	}
}

// VoiceV2PreviewIntent 最终转写 → 意图 dry-run → T0（navigate/status）免确认
// 直接执行（M5 刀一，executed=true/false+reason；前端出回执不再出确认卡）；
// T1 白名单内登记台账返回 preview_id/intent_hash/TTL（确认后执行）；白名单外
// 「可预览不可执行」；未命中 handled=false（前端按「未识别为动作」静默）。
// dryRun 恒不打 LLM。
func (a *App) VoiceV2PreviewIntent(userText string) map[string]interface{} {
	res := a.routeIntentMode(userText, true)
	if !res.Handled {
		return map[string]interface{}{"handled": false}
	}
	base := map[string]interface{}{
		"handled": true,
		"reply":   res.Reply,
		"action":  res.Action,
		"target":  res.Target,
	}
	if !voiceV2ExecutableActions[res.Action] {
		base["executable"] = false
		return base
	}
	if voiceV2T0Actions[res.Action] {
		it := &intent.Intent{Action: intent.Action(res.Action), Target: res.Target, Text: userText}
		reply, ok := execVoiceV2Action(a, res.Action, it)
		base["executed"] = ok
		if ok {
			base["reply"] = reply
		} else {
			base["reason"] = "exec-failed"
		}
		return base
	}
	id := voiceV2PreviewID()
	hash := voiceV2PreviewHash(res.Action, res.Target, res.Reply)
	voiceV2PreviewMu.Lock()
	pruneVoiceV2PreviewLocked()
	voiceV2PreviewLedger[id] = &voiceV2ActionPreview{
		text:      userText,
		reply:     res.Reply,
		action:    res.Action,
		target:    res.Target,
		hash:      hash,
		expiresAt: time.Now().Add(voiceV2PreviewTTL),
	}
	voiceV2PreviewMu.Unlock()
	base["executable"] = true
	base["previewId"] = id
	base["intentHash"] = hash
	base["ttlMs"] = voiceV2PreviewTTL.Milliseconds()
	return base
}

// VoiceV2ExecuteAction 确认执行：幂等（executed 标记领走即置位，重复调用=
// already-executed）、TTL、hash 校验、白名单再校验（纵深），执行按台账重构
// Intent 直派既有执行层——只执行被预览的那个动作，最多一次。
func (a *App) VoiceV2ExecuteAction(previewID, intentHash string) map[string]interface{} {
	voiceV2PreviewMu.Lock()
	p := voiceV2PreviewLedger[previewID]
	if p == nil {
		voiceV2PreviewMu.Unlock()
		return map[string]interface{}{"executed": false, "reason": "no-preview"}
	}
	if p.executed {
		voiceV2PreviewMu.Unlock()
		return map[string]interface{}{"executed": false, "reason": "already-executed"}
	}
	if time.Now().After(p.expiresAt) {
		delete(voiceV2PreviewLedger, previewID)
		voiceV2PreviewMu.Unlock()
		return map[string]interface{}{"executed": false, "reason": "expired"}
	}
	if intentHash == "" || intentHash != p.hash {
		voiceV2PreviewMu.Unlock()
		return map[string]interface{}{"executed": false, "reason": "hash-mismatch"}
	}
	if !voiceV2ExecutableActions[p.action] {
		voiceV2PreviewMu.Unlock()
		return map[string]interface{}{"executed": false, "reason": "action-not-allowed"}
	}
	p.executed = true
	preview := *p
	voiceV2PreviewMu.Unlock()

	it := &intent.Intent{Action: intent.Action(preview.action), Target: preview.target, Text: preview.text}
	reply, ok := execVoiceV2Action(a, preview.action, it)
	if !ok {
		return map[string]interface{}{"executed": false, "reason": "exec-failed"}
	}
	return map[string]interface{}{"executed": true, "reply": reply, "action": preview.action}
}

// execVoiceV2Action 动作直派（PreviewIntent T0 直执行与 ExecuteAction 台账
// 确认共用；白名单外=不执行）。执行层复用 intent_router 既有能力。
// 包级可桩点：单测禁连真实执行面（GetModelMonitor/addWxReminder 等副作用，
// App 测试语义管线地雷），编排逻辑（T0 直执行/T1 台账）与执行解耦。
var execVoiceV2Action = func(a *App, action string, it *intent.Intent) (string, bool) {
	switch intent.Action(action) {
	case intent.ActionNavigate:
		return a.execNavigate(it)
	case intent.ActionStatus:
		return a.execStatus(it)
	case intent.ActionReminder:
		return a.execReminder(it)
	case intent.ActionSaveTask:
		return a.execSaveTask(it, "")
	default:
		return "", false
	}
}
