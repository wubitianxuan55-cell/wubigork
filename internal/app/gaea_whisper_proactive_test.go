package app

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
)

// registerProactiveOrch 注册一个已就绪（有历史回合）的轻语会话到进程级会话表，
// 并保证测试结束清理，避免污染其他用例。
func registerProactiveOrch(t *testing.T, a *whisperState, personalityID string, seed func(*whisper.Orchestrator)) *whisper.Orchestrator {
	t.Helper()
	sessionID := "whisper_" + personalityID
	preset := whisper.GetPreset(personalityID)
	if preset == nil {
		// 测试用自定义人格：保持 Preset.ID == personalityID（评估按 Preset.ID
		// 记频控/生日标记，与生产 getOrCreateOrch 语义一致）。
		preset = &whisper.PersonalityPreset{
			ID: personalityID, Label: personalityID,
			Dims: whisper.PersonalityDims{T: 50, I: 50, S: 50, O: 50, R: 50},
		}
	}
	orch := whisper.NewOrchestrator(sessionID, *preset)
	orch.DataRoot = a.whisperDataRoot
	if seed != nil {
		seed(orch)
	}
	whisperSessionsMu.Lock()
	whisperSessions[sessionID] = orch
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		whisperSessionsMu.Lock()
		delete(whisperSessions, sessionID)
		whisperSessionsMu.Unlock()
	})
	return orch
}

// seedHighAffIntimate 高亲和 + 亲密关系 + 已离线 2h：门控通过，
// 合成器应产出 playful_nudge（非深夜、aff>55、非陌生人）。
func seedHighAffIntimate(orch *whisper.Orchestrator, now time.Time) {
	st := &orch.State
	st.Counters.TotalTurns = 5
	st.LastActive = now.Add(-2 * time.Hour)
	st.Emotion = whisper.EmotionState{Aff: 80, Sec: 70, Aro: 20, Dom: 30, PrimaryLabel: "HAPPY_JOY"}
	st.Relationship = whisper.L1State{Stage: whisper.StageIntimate, Trust: 80, Rifts: 0, SharedEventsCount: 10}
}

func proactiveNow() time.Time {
	// 固定 12:00 白天：非 late_night，生日检测/时窗判定均可注入。
	return time.Date(2026, 8, 30, 12, 0, 0, 0, time.Local)
}

// ─── 评估：门控合成路径 ───────────────────────────────────────

func TestProactiveEvaluate_GateComposeSends(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	orch := registerProactiveOrch(t, a.whisperState, "proT1", func(o *whisper.Orchestrator) {
		seedHighAffIntimate(o, now)
	})

	res := a.evaluateProactiveForSession(orch, now)
	if res == nil || !res.ShouldSend {
		t.Fatalf("高亲和亲密会话应推送，got %+v", res)
	}
	if res.MessageType != whisper.ProactivePlayful {
		t.Fatalf("期望 playful_nudge，got %q", res.MessageType)
	}
	if res.PromptHint == "" {
		t.Fatal("promptHint 不应为空")
	}
}

func TestProactiveEvaluate_NotReadySkip(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	orch := registerProactiveOrch(t, a.whisperState, "proT2", func(o *whisper.Orchestrator) {
		seedHighAffIntimate(o, now)
		o.State.Counters.TotalTurns = 0 // 从未互动 → 不就绪
	})

	if res := a.evaluateProactiveForSession(orch, now); res != nil {
		t.Fatalf("未就绪会话不应推送，got %+v", res)
	}
}

// ─── 评估：频控跳过 ───────────────────────────────────────────

func TestProactiveEvaluate_BudgetExceededSkip(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	orch := registerProactiveOrch(t, a.whisperState, "proT3", func(o *whisper.Orchestrator) {
		seedHighAffIntimate(o, now)
	})
	// 默认上限 3 条/小时：已发 3 条 → 超预算
	am := a.attentionFor("proT3")
	for i := 0; i < 3; i++ {
		am.RecordProactive(now)
	}
	if !am.IsBudgetExceeded(now) {
		t.Fatal("前置：预算应超限")
	}
	if res := a.evaluateProactiveForSession(orch, now); res != nil {
		t.Fatalf("超预算应跳过，got %+v", res)
	}
}

func TestProactiveTicker_BudgetSkipRecordsNothing(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	registerProactiveOrch(t, a.whisperState, "proT4", func(o *whisper.Orchestrator) {
		seedHighAffIntimate(o, now)
	})
	am := a.attentionFor("proT4")
	for i := 0; i < 3; i++ {
		am.RecordProactive(now)
	}
	if pushed := a.tickProactive(now); pushed != 0 {
		t.Fatalf("超预算 tick 不应推送，got %d", pushed)
	}
	// 跳过不额外记账（预算记录只发生在真正发送后）
	if got := len(am.State().LastProactiveAt); got != 3 {
		t.Fatalf("跳过不应新增记账，got %d", got)
	}
}

// ─── 评估：作息尊重（dnd 习惯）───────────────────────────────

func TestProactiveEvaluate_HabitDndSkip(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	orch := registerProactiveOrch(t, a.whisperState, "proT5", func(o *whisper.Orchestrator) {
		seedHighAffIntimate(o, now)
		// 命中当前小时的 dnd 免打扰习惯（每天）
		o.HabitsStore.Upsert(whisper.UserHabit{
			Type: "dnd", Scope: "short_term",
			HourStart: now.Hour(), HourEnd: now.Hour(),
			Confidence: 0.9, Source: "detected",
		})
	})

	if res := a.evaluateProactiveForSession(orch, now); res != nil {
		t.Fatalf("作息 dnd 时段应跳过，got %+v", res)
	}
}

func TestProactiveEvaluate_NonDndHabitDoesNotSkip(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	orch := registerProactiveOrch(t, a.whisperState, "proT6", func(o *whisper.Orchestrator) {
		seedHighAffIntimate(o, now)
		// 健康提醒习惯：不阻止主动消息
		o.HabitsStore.Upsert(whisper.UserHabit{
			Type: "health_reminder", Scope: "long_term",
			HourStart: now.Hour(), HourEnd: now.Hour(),
			Confidence: 0.9, Source: "detected",
		})
	})

	if res := a.evaluateProactiveForSession(orch, now); res == nil || !res.ShouldSend {
		t.Fatalf("非 dnd 习惯不应跳过，got %+v", res)
	}
}

// ─── 评估：生日祝福 ───────────────────────────────────────────

// seedBirthdayFact 灌入当天生日的记忆事实（AgeMeta.BirthdayMMDD）。
func seedBirthdayFact(orch *whisper.Orchestrator, now time.Time) {
	todayMMDD := fmt.Sprintf("%02d-%02d", int(now.Month()), int(now.Day()))
	orch.FactStore.Add(whisper.MemoryFact{
		ID: "bf-1", Domain: "user_profile", Subcategory: "BASIC_PROFILE",
		Subject: "用户生日", Summary: "用户的生日是今天", Weight: 1, Confidence: 0.9,
		Status: "active", AgeMeta: &whisper.AgeMeta{BirthdayMMDD: todayMMDD},
	})
}

func TestProactiveEvaluate_BirthdayTriggers(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	orch := registerProactiveOrch(t, a.whisperState, "proT7", func(o *whisper.Orchestrator) {
		// 低亲和（门控 silent）+ 生日事实 → 生日祝福应绕过门控
		o.State.Counters.TotalTurns = 5
		o.State.LastActive = now.Add(-2 * time.Hour)
		o.State.Emotion = whisper.EmotionState{Aff: 10, Sec: 20, Aro: 5, Dom: 5, PrimaryLabel: "CALM_RATIONAL"}
		o.State.Relationship = whisper.L1State{Stage: whisper.StageStranger, Trust: 10, Rifts: 0}
		seedBirthdayFact(o, now)
	})

	res := a.evaluateProactiveForSession(orch, now)
	if res == nil || !res.ShouldSend {
		t.Fatalf("生日当天应触发生日祝福，got %+v", res)
	}
	if res.MessageType != whisper.ProactiveBirthday {
		t.Fatalf("期望 birthday 类型，got %q", res.MessageType)
	}
	if res.PromptHint == "" {
		t.Fatal("生日祝福 promptHint 不应为空")
	}
}

func TestProactiveEvaluate_BirthdayOncePerDay(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	orch := registerProactiveOrch(t, a.whisperState, "proT8", func(o *whisper.Orchestrator) {
		o.State.Counters.TotalTurns = 5
		o.State.LastActive = now.Add(-2 * time.Hour)
		o.State.Emotion = whisper.EmotionState{Aff: 10, Sec: 20, Aro: 5, Dom: 5, PrimaryLabel: "CALM_RATIONAL"}
		o.State.Relationship = whisper.L1State{Stage: whisper.StageStranger, Trust: 10, Rifts: 0}
		seedBirthdayFact(o, now)
	})

	if res := a.evaluateProactiveForSession(orch, now); res == nil || res.MessageType != whisper.ProactiveBirthday {
		t.Fatalf("首条应为生日祝福，got %+v", res)
	}
	// 标记当天已祝福 → 第二次不再走生日分支；低亲和门控 silent → 不推送
	a.markBirthdayGreeted("proT8", now)
	if res := a.evaluateProactiveForSession(orch, now); res != nil {
		t.Fatalf("同一天不应重复生日祝福，got %+v", res)
	}
}

// ─── tick 整轮：发送 + 记账 + 事件 payload ────────────────────

func TestProactiveTicker_SendsAndRecords(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	registerProactiveOrch(t, a.whisperState, "proT9", func(o *whisper.Orchestrator) {
		seedHighAffIntimate(o, now)
	})

	if pushed := a.tickProactive(now); pushed != 1 {
		t.Fatalf("应推送 1 条，got %d", pushed)
	}
	// 发送后频控记账（每会话独立预算）
	am := a.attentionFor("proT9")
	if got := len(am.State().LastProactiveAt); got != 1 {
		t.Fatalf("发送后应记录 1 条预算，got %d", got)
	}
	// 预算未满（默认 3 条/小时，当前仅 1 条）时第二轮仍可推送——补满到上限后
	// 第三轮才被频控拦住
	if pushed := a.tickProactive(now); pushed != 1 {
		t.Fatalf("预算未满应可继续推送，got %d", pushed)
	}
	am.RecordProactive(now) // 补足 3 条
	if !am.IsBudgetExceeded(now) {
		t.Fatal("前置：预算应超限")
	}
	if pushed := a.tickProactive(now); pushed != 0 {
		t.Fatalf("预算满后应被频控拦住，got %d", pushed)
	}
}

func TestProactiveTicker_BirthdayPushedOnce(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	registerProactiveOrch(t, a.whisperState, "proT10", func(o *whisper.Orchestrator) {
		o.State.Counters.TotalTurns = 5
		o.State.LastActive = now.Add(-2 * time.Hour)
		o.State.Emotion = whisper.EmotionState{Aff: 10, Sec: 20, Aro: 5, Dom: 5, PrimaryLabel: "CALM_RATIONAL"}
		o.State.Relationship = whisper.L1State{Stage: whisper.StageStranger, Trust: 10, Rifts: 0}
		seedBirthdayFact(o, now)
	})

	if pushed := a.tickProactive(now); pushed != 1 {
		t.Fatalf("生日当天应推送 1 条，got %d", pushed)
	}
	if !a.birthdayGreetedToday("proT10", now) {
		t.Fatal("发送后应标记当天已祝福")
	}
	if pushed := a.tickProactive(now); pushed != 0 {
		t.Fatalf("同一天第二次 tick 不应再推送，got %d", pushed)
	}
}

func TestProactiveEventPayload(t *testing.T) {
	now := proactiveNow()
	res := &whisper.ProactiveComposeResult{
		ShouldSend:  true,
		MessageType: whisper.ProactiveBirthday,
		PromptHint:  "生日快乐！",
	}
	p := proactiveEventPayload("proPayload", res, now)
	if p["personalityID"] != "proPayload" || p["messageType"] != "birthday" || p["promptHint"] != "生日快乐！" {
		t.Fatalf("payload 字段不符: %+v", p)
	}
	if p["space"] != "play" {
		t.Fatalf("payload 应带 space=play（subscribeForSpace 过滤），got %v", p["space"])
	}
	if _, ok := p["sentAt"]; !ok {
		t.Fatal("payload 应带 sentAt")
	}
}

// ─── tick 整轮：开关 / 免打扰时窗 ─────────────────────────────

func TestProactiveTicker_DisabledAndQuietWindow(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := proactiveNow()
	registerProactiveOrch(t, a.whisperState, "proT11", func(o *whisper.Orchestrator) {
		seedHighAffIntimate(o, now)
	})

	// 总开关关闭
	if err := a.GaeaWhisperSetProactiveConfig(`{"enabled": false}`); err != nil {
		t.Fatalf("关闭开关: %v", err)
	}
	if pushed := a.tickProactive(now); pushed != 0 {
		t.Fatalf("关闭开关后不应推送，got %d", pushed)
	}

	// 恢复开关 + 免打扰时窗覆盖当前小时
	if err := a.GaeaWhisperSetProactiveConfig(`{"enabled": true}`); err != nil {
		t.Fatalf("恢复开关: %v", err)
	}
	h := now.Hour()
	if err := a.GaeaWhisperSetProactiveConfig(fmt.Sprintf(`{"quietStartHour": %d, "quietEndHour": %d}`, h, h)); err != nil {
		t.Fatalf("设置时窗: %v", err)
	}
	if pushed := a.tickProactive(now); pushed != 0 {
		t.Fatalf("免打扰时窗内不应推送，got %d", pushed)
	}
}

func TestInQuietWindow(t *testing.T) {
	base := time.Date(2026, 8, 30, 12, 0, 0, 0, time.Local) // 12:00
	if inQuietWindow(base, -1, -1) {
		t.Error("未启用时窗不应命中")
	}
	if !inQuietWindow(base, 12, 12) {
		t.Error("同小时时窗应命中")
	}
	if inQuietWindow(base, 13, 14) {
		t.Error("时窗外不应命中")
	}
	night := time.Date(2026, 8, 30, 23, 30, 0, 0, time.Local) // 23:30
	if !inQuietWindow(night, 22, 6) {
		t.Error("跨天时窗（22-6）23:30 应命中")
	}
	morning := time.Date(2026, 8, 30, 5, 30, 0, 0, time.Local) // 05:30
	if !inQuietWindow(morning, 22, 6) {
		t.Error("跨天时窗（22-6）05:30 应命中")
	}
}

// ─── 配置读写 ─────────────────────────────────────────────────

func TestProactiveConfig_DefaultsAndRoundTrip(t *testing.T) {
	a := newChatServiceTestApp(t)

	cfg, err := a.GaeaWhisperProactiveConfig()
	if err != nil {
		t.Fatalf("GaeaWhisperProactiveConfig: %v", err)
	}
	if cfg["enabled"] != true || cfg["limitPerHour"] != 3 || cfg["intervalMin"] != 30 {
		t.Fatalf("默认配置不符: %+v", cfg)
	}
	if cfg["quietStartHour"] != -1 || cfg["quietEndHour"] != -1 {
		t.Fatalf("默认时窗应未启用: %+v", cfg)
	}

	// 部分字段更新：未提供的字段保持原值
	if err := a.GaeaWhisperSetProactiveConfig(`{"limitPerHour": 5, "intervalMin": 45, "quietStartHour": 22, "quietEndHour": 6}`); err != nil {
		t.Fatalf("SetProactiveConfig: %v", err)
	}
	cfg, _ = a.GaeaWhisperProactiveConfig()
	if cfg["limitPerHour"] != 5 || cfg["intervalMin"] != 45 || cfg["quietStartHour"] != 22 || cfg["quietEndHour"] != 6 {
		t.Fatalf("更新后配置不符: %+v", cfg)
	}
	if cfg["enabled"] != true {
		t.Fatalf("未提供 enabled 应保持原值: %+v", cfg)
	}

	// 上限应用到已存在频控实例
	am := a.attentionFor("proCfg1")
	if got := am.State().ProactiveMessagesPerHour; got != 5 {
		t.Fatalf("频控上限应更新为 5，got %d", got)
	}
}

func TestProactiveConfig_Validation(t *testing.T) {
	a := newChatServiceTestApp(t)
	cases := []struct {
		name string
		json string
	}{
		{"interval 过小", `{"intervalMin": 5}`},
		{"interval 过大", `{"intervalMin": 999}`},
		{"limit 为 0", `{"limitPerHour": 0}`},
		{"limit 为负", `{"limitPerHour": -1}`},
		{"时窗越界", `{"quietStartHour": 24}`},
		{"非法 JSON", `{bad`},
	}
	for _, tc := range cases {
		if err := a.GaeaWhisperSetProactiveConfig(tc.json); err == nil {
			t.Errorf("%s 应报错", tc.name)
		}
	}
	// 报错后配置不被破坏
	cfg, _ := a.GaeaWhisperProactiveConfig()
	if cfg["limitPerHour"] != 3 || cfg["intervalMin"] != 30 {
		t.Fatalf("校验失败不应改变配置: %+v", cfg)
	}
}

// ─── 绑定面完整性（新增绑定签名可用）─────────────────────────

func TestProactiveConfig_BindingJSONShape(t *testing.T) {
	a := newChatServiceTestApp(t)
	// 模拟前端传参形态：完整 JSON 往返
	full := `{"enabled":true,"limitPerHour":2,"intervalMin":60,"quietStartHour":23,"quietEndHour":7}`
	if err := a.GaeaWhisperSetProactiveConfig(full); err != nil {
		t.Fatalf("完整 JSON 更新: %v", err)
	}
	cfg, err := a.GaeaWhisperProactiveConfig()
	if err != nil {
		t.Fatalf("读取配置: %v", err)
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]interface{}
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round["limitPerHour"] != float64(2) || round["quietStartHour"] != float64(23) {
		t.Fatalf("JSON 往返不符: %s", string(b))
	}
}
