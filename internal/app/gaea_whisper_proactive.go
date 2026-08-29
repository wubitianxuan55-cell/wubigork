package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/whisper"
)

// ─── 主动关心定时推送（v4.3c，play 空间会客厅）────────────────────
//
// 信号链：就绪（有历史回合）→ 频控（AttentionManager ≤N 条/小时）→
// 作息尊重（MatchHabits 命中 dnd 免打扰习惯）→ 生日祝福（DetectSpecialDatesV2
// 命中生日且当天未祝福）→ 门控 + 合成器（EvaluateProactiveGate +
// ComposeProactiveMessage，与 GaeaWhisperProactiveNow 同款参数装配）。
// 合成成功经 `gaea-whisper-proactive` 事件推前端。play 红线：只读评估 +
// 事件发射，不落盘任何 work 目录。

// proactivePushCfg 主动关心定时推送配置（内存态，重启即失；不建表、不改 gaea.toml）。
type proactivePushCfg struct {
	Enabled        bool // 定时推送总开关（默认开）
	LimitPerHour   int  // 每小时主动消息上限（喂给 AttentionManager，默认 3）
	IntervalMin    int  // 评估间隔分钟（默认 30，允许 10–120）
	QuietStartHour int  // 免打扰时窗开始小时 0-23；-1=未启用（默认）
	QuietEndHour   int  // 免打扰时窗结束小时 0-23；-1=未启用（默认）
}

func defaultProactivePushCfg() proactivePushCfg {
	return proactivePushCfg{
		Enabled:        true,
		LimitPerHour:   3,
		IntervalMin:    30,
		QuietStartHour: -1,
		QuietEndHour:   -1,
	}
}

// getProactiveCfg 返回当前配置副本（惰性初始化默认值，兼容测试手工构造）。
func (a *whisperState) getProactiveCfg() proactivePushCfg {
	a.proactiveMu.Lock()
	defer a.proactiveMu.Unlock()
	if !a.proactiveCfgInit {
		a.proactiveCfg = defaultProactivePushCfg()
		a.proactiveCfgInit = true
	}
	return a.proactiveCfg
}

// attentionFor 取（惰性创建）指定人格的频控管理器，新建时套用当前上限。
func (a *whisperState) attentionFor(personalityID string) *whisper.AttentionManager {
	limit := a.getProactiveCfg().LimitPerHour
	a.attentionMu.Lock()
	defer a.attentionMu.Unlock()
	if a.attentionManagers == nil {
		a.attentionManagers = map[string]*whisper.AttentionManager{}
	}
	am, ok := a.attentionManagers[personalityID]
	if !ok {
		am = whisper.NewAttentionManager()
		am.SetProactiveLimit(limit)
		a.attentionManagers[personalityID] = am
	}
	return am
}

// birthdayGreetedToday 该人格当天是否已发过生日祝福（生日祝福每天仅首条）。
func (a *whisperState) birthdayGreetedToday(personalityID string, now time.Time) bool {
	day := now.Format("2006-01-02")
	a.birthdayGreetedMu.Lock()
	defer a.birthdayGreetedMu.Unlock()
	return a.birthdayGreeted[personalityID] == day
}

// markBirthdayGreeted 记录该人格当天已发生日祝福。
func (a *whisperState) markBirthdayGreeted(personalityID string, now time.Time) {
	a.birthdayGreetedMu.Lock()
	defer a.birthdayGreetedMu.Unlock()
	if a.birthdayGreeted == nil {
		a.birthdayGreeted = map[string]string{}
	}
	a.birthdayGreeted[personalityID] = now.Format("2006-01-02")
}

// inQuietWindow 判断 now 是否落在免打扰时窗内（start<=end 同天内，否则跨天
// 如 22-6）。任一值为 -1 视为未启用。
func inQuietWindow(now time.Time, startHour, endHour int) bool {
	if startHour < 0 || endHour < 0 || startHour > 23 || endHour > 23 {
		return false
	}
	h := now.Hour()
	if startHour <= endHour {
		return h >= startHour && h <= endHour
	}
	return h >= startHour || h <= endHour
}

// ─── 定时推送循环 ──────────────────────────────────────────────

// startProactiveTicker 启动主动关心定时推送循环：先评估一轮（幂等，会话按需
// 创建、启动时通常为空），之后按配置间隔循环。间隔每次从配置读取，改配置
// 下一轮生效。幂等：首次调用启动，Shutdown 时停止。
func (a *whisperState) startProactiveTicker() {
	a.proactiveOnce.Do(func() {
		stop := make(chan struct{})
		a.proactiveStop = stop
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("proactive ticker panic recovered", "panic", r)
				}
			}()
			a.tickProactive(time.Now())
			for {
				interval := time.Duration(a.getProactiveCfg().IntervalMin) * time.Minute
				timer := time.NewTimer(interval)
				select {
				case <-stop:
					timer.Stop()
					return
				case <-timer.C:
					a.tickProactive(time.Now())
				}
			}
		}()
	})
}

// tickProactive 一轮评估：快照全部轻语会话 → 逐个评估 → 合成成功即推事件并
// 记录频控/生日标记。now 可注入（测试）。返回推送条数（测试断言用）。
func (a *whisperState) tickProactive(now time.Time) (pushed int) {
	cfg := a.getProactiveCfg()
	if !cfg.Enabled {
		return 0
	}
	if inQuietWindow(now, cfg.QuietStartHour, cfg.QuietEndHour) {
		slog.Debug("proactive: 免打扰时窗内跳过", "start", cfg.QuietStartHour, "end", cfg.QuietEndHour)
		return 0
	}

	whisperSessionsMu.RLock()
	orchs := make([]*whisper.Orchestrator, 0, len(whisperSessions))
	for _, o := range whisperSessions {
		orchs = append(orchs, o)
	}
	whisperSessionsMu.RUnlock()

	for _, orch := range orchs {
		res := a.evaluateProactiveForSession(orch, now)
		if res == nil || !res.ShouldSend {
			continue
		}
		personalityID := orch.Preset.ID
		a.emit("gaea-whisper-proactive", proactiveEventPayload(personalityID, res, now))
		a.attentionFor(personalityID).RecordProactive(now)
		if res.MessageType == whisper.ProactiveBirthday {
			a.markBirthdayGreeted(personalityID, now)
		}
		pushed++
		slog.Info("proactive: 主动消息已推送", "personalityID", personalityID, "messageType", res.MessageType)
	}
	return pushed
}

// proactiveEventPayload 主动消息事件 payload（personalityID/messageType/promptHint；
// 附 space=play 供前端 subscribeForSpace 过滤，sentAt 便于前端去抖展示）。
func proactiveEventPayload(personalityID string, res *whisper.ProactiveComposeResult, now time.Time) map[string]interface{} {
	return map[string]interface{}{
		"personalityID": personalityID,
		"messageType":   string(res.MessageType),
		"promptHint":    res.PromptHint,
		"space":         "play",
		"sentAt":        now.Format(time.RFC3339),
	}
}

// evaluateProactiveForSession 评估单个会话本轮是否推送主动消息（v4.3c）。
// 返回 nil = 不推送；非 nil = 推送内容。play 空间：只读评估，零落盘。
func (a *whisperState) evaluateProactiveForSession(orch *whisper.Orchestrator, now time.Time) *whisper.ProactiveComposeResult {
	if orch == nil {
		return nil
	}
	orch.LockTurn()
	defer orch.UnlockTurn()

	st := orch.State
	// 就绪判定：有历史回合才视为已就绪（不对从未互动的会话主动开口）。
	if st.Counters.TotalTurns <= 0 {
		return nil
	}
	personalityID := orch.Preset.ID

	// ① 频控：超预算跳过（预算记录在发送成功后，见 tickProactive）。
	if a.attentionFor(personalityID).IsBudgetExceeded(now) {
		return nil
	}

	// ② 作息尊重：命中 dnd 习惯（用户明示免打扰/睡眠/忙碌时段）→ 跳过。
	if habits := orch.HabitsStore.MatchHabits(now); len(habits) > 0 {
		for _, h := range habits {
			if h.Type == "dnd" {
				return nil
			}
		}
	}

	// ③ 生日祝福：DetectSpecialDatesV2 命中生日且当天未祝福 → 生日消息类型。
	if !a.birthdayGreetedToday(personalityID, now) {
		for _, sd := range a.detectSpecialDates(orch, now) {
			if sd.Type == "birthday" {
				return &whisper.ProactiveComposeResult{
					ShouldSend:  true,
					MessageType: whisper.ProactiveBirthday,
					PromptHint:  buildBirthdayHint(sd.Subject, personalityID),
				}
			}
		}
	}

	// ④ 门控 + 合成器（与 GaeaWhisperProactiveNow 同款参数装配）。
	timeOfDay := timeOfDayFor(now)
	gate := whisper.EvaluateProactiveGate(
		st.Emotion.Aff, st.Emotion.Aro, st.Emotion.Sec,
		st.Relationship.Trust, st.Relationship.Rifts,
		st.Relationship.Stage, timeOfDay, orch.AdultMode)
	gapHours := 0.0
	if !st.LastActive.IsZero() {
		gapHours = now.Sub(st.LastActive).Hours()
	}
	res := whisper.ComposeProactiveMessage(
		gate, st.Emotion.Aff, st.Emotion.Sec,
		st.Relationship.Trust, st.Relationship.Stage,
		timeOfDay, gapHours, false, personalityID)
	if res == nil || !res.ShouldSend {
		return nil
	}
	return res
}

// detectSpecialDates 装配 DetectSpecialDatesV2 的四类数据源：相识日/ackem 生日
// （状态指针）/ 生日事实（FactStore 中带 AgeMeta.BirthdayMMDD 的记忆）/
// 时间锚点（State.TemporalAnchors）。
func (a *whisperState) detectSpecialDates(orch *whisper.Orchestrator, now time.Time) []whisper.SpecialDateV2 {
	firstMet := ""
	if orch.State.FirstMetDate != nil {
		firstMet = orch.State.FirstMetDate.Format("2006-01-02")
	}
	ackemBD := ""
	if orch.State.AckemBirthday != nil {
		ackemBD = orch.State.AckemBirthday.Format("2006-01-02")
	}
	var birthdays []whisper.BirthdayEntryV2
	for _, f := range orch.FactStore.ListAll() {
		if f.AgeMeta != nil && f.AgeMeta.BirthdayMMDD != "" {
			birthdays = append(birthdays, whisper.BirthdayEntryV2{
				Subject:      f.Subject,
				BirthdayMMDD: f.AgeMeta.BirthdayMMDD,
			})
		}
	}
	var anchors []whisper.AnchorEntryV2
	for _, an := range orch.State.TemporalAnchors {
		anchors = append(anchors, whisper.AnchorEntryV2{
			AnchorDate:         an.AnchorDate,
			AnchorType:         string(an.AnchorType),
			LinkedFactIDs:      strings.Join(an.LinkedFactIDs, ","),
			EmotionalIntensity: an.EmotionalIntensity,
		})
	}
	return whisper.DetectSpecialDatesV2(now, firstMet, ackemBD, birthdays, anchors)
}

// buildBirthdayHint 生日祝福提示词（人格感知，风格对齐合成器各分支 hint）。
func buildBirthdayHint(subject, personalityID string) string {
	name := subject
	if name == "" {
		name = "ta"
	}
	base := "今天是" + name + "的生日。用温暖真诚的语气送上生日祝福，可以自然提起你们之间的一些共同回忆，但不要长篇大论。"
	switch personalityID {
	case "tsundere":
		return base + " 保持傲娇——明明是关心却装作不以为然。"
	case "yandere":
		return base + " 带着一点独占感的祝福。"
	case "kuudere":
		return base + " 话不多，但字字有温度。"
	case "genki":
		return base + " 元气满满地为ta庆生！"
	}
	return base
}

// ─── 频控配置绑定（v4.3c，play 空间）──────────────────────────

// GaeaWhisperProactiveConfig 返回主动关心定时推送当前配置：开关/每小时上限/
// 评估间隔/免打扰时窗。
func (a *whisperState) GaeaWhisperProactiveConfig() (map[string]interface{}, error) {
	cfg := a.getProactiveCfg()
	return map[string]interface{}{
		"enabled":        cfg.Enabled,
		"limitPerHour":   cfg.LimitPerHour,
		"intervalMin":    cfg.IntervalMin,
		"quietStartHour": cfg.QuietStartHour,
		"quietEndHour":   cfg.QuietEndHour,
	}, nil
}

// GaeaWhisperSetProactiveConfig 更新频控/时窗配置（cfgJSON 支持部分字段，缺省
// 保持原值）。校验：limitPerHour ≥ 1；intervalMin 10–120；时窗小时 -1 或 0–23。
// 上限变更即时应用到全部已创建会话的频控实例。
func (a *whisperState) GaeaWhisperSetProactiveConfig(cfgJSON string) error {
	var raw struct {
		Enabled        *bool `json:"enabled"`
		LimitPerHour   *int  `json:"limitPerHour"`
		IntervalMin    *int  `json:"intervalMin"`
		QuietStartHour *int  `json:"quietStartHour"`
		QuietEndHour   *int  `json:"quietEndHour"`
	}
	if cfgJSON != "" {
		if err := json.Unmarshal([]byte(cfgJSON), &raw); err != nil {
			return fmt.Errorf("解析主动关心配置失败: %w", err)
		}
	}

	next := a.getProactiveCfg()
	if raw.Enabled != nil {
		next.Enabled = *raw.Enabled
	}
	if raw.LimitPerHour != nil {
		if *raw.LimitPerHour < 1 {
			return fmt.Errorf("limitPerHour 必须 ≥ 1（关闭请用 enabled=false）")
		}
		next.LimitPerHour = *raw.LimitPerHour
	}
	if raw.IntervalMin != nil {
		if *raw.IntervalMin < 10 || *raw.IntervalMin > 120 {
			return fmt.Errorf("intervalMin 必须在 10–120 分钟")
		}
		next.IntervalMin = *raw.IntervalMin
	}
	for _, v := range []*int{raw.QuietStartHour, raw.QuietEndHour} {
		if v != nil && (*v < -1 || *v > 23) {
			return fmt.Errorf("时窗小时必须为 -1（未启用）或 0–23")
		}
	}
	if raw.QuietStartHour != nil {
		next.QuietStartHour = *raw.QuietStartHour
	}
	if raw.QuietEndHour != nil {
		next.QuietEndHour = *raw.QuietEndHour
	}

	a.proactiveMu.Lock()
	a.proactiveCfg = next
	a.proactiveCfgInit = true
	a.proactiveMu.Unlock()

	// 上限变更即时应用到全部已有会话频控。取锁顺序注意：attentionFor 与
	// SetProactiveConfig 都先读 cfg（proactiveMu 用完即放）再取 attentionMu，
	// 两把锁从不同时持有，无死锁风险。
	a.attentionMu.Lock()
	for _, am := range a.attentionManagers {
		am.SetProactiveLimit(next.LimitPerHour)
	}
	a.attentionMu.Unlock()
	return nil
}
