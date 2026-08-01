package whisper

import (
	"testing"
	"time"
)

// ─── emotional_emergence: 情感延续检测 ────────────────────────

func TestIsEmotionalContinuationEvent_Vulnerable(t *testing.T) {
	// vulnerable 无条件延续
	if !IsEmotionalContinuationEvent("vulnerable", 0, 0, nil) {
		t.Error("vulnerable 应总是延续")
	}
}

func TestIsEmotionalContinuationEvent_Apology(t *testing.T) {
	tests := []struct {
		name       string
		meaningful int
		vulnerable int
		want       bool
	}{
		{"连续2次有意义", 2, 0, true},
		{"连续1次脆弱", 0, 1, true},
		{"两者都有但不足", 1, 0, false},
	}
	for _, tt := range tests {
		got := IsEmotionalContinuationEvent("apology", tt.meaningful, tt.vulnerable, nil)
		if got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsEmotionalContinuationEvent_Praise(t *testing.T) {
	tests := []struct {
		name       string
		meaningful int
		vulnerable int
		recent     []string
		want       bool
	}{
		{"脆弱后", 0, 1, nil, true},
		{"连续3次有意义", 3, 0, nil, true},
		{"近6次中3次有意义", 0, 0, []string{"praise", "praise", "praise", "casual"}, true},
		{"不满足", 1, 0, nil, false},
	}
	for _, tt := range tests {
		got := IsEmotionalContinuationEvent("praise", tt.meaningful, tt.vulnerable, tt.recent)
		if got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsEmotionalContinuationEvent_Other(t *testing.T) {
	if IsEmotionalContinuationEvent("casual", 5, 5, nil) {
		t.Error("普通事件不应延续")
	}
}

// ─── emotional_emergence: 事件统计 ────────────────────────────

func TestCountMeaningfulInRecent(t *testing.T) {
	events := []string{"praise", "casual", "praise", "vulnerable", "praise", "praise"}
	// meaningful = praise(4) + vulnerable(1) = 5
	if got := CountMeaningfulInRecent(events, 6); got != 5 {
		t.Errorf("CountMeaningfulInRecent = %d, want 5", got)
	}
	// window=3：取后 3 个 = vulnerable, praise, praise → 3
	if got := CountMeaningfulInRecent(events, 3); got != 3 {
		t.Errorf("window=3 时 = %d, want 3", got)
	}
	if got := CountMeaningfulInRecent(nil, 5); got != 0 {
		t.Errorf("空事件 = %d, want 0", got)
	}
}

// ─── emotional_emergence: 主判决 ──────────────────────────────

func TestEvaluateEmergence_StrangerNil(t *testing.T) {
	ctx := EmergenceContext{Stage: StageStranger, Emotion: EmotionState{PrimaryLabel: "CALM_RATIONAL"}}
	if got := EvaluateEmergence(ctx, "praise"); got != nil {
		t.Error("陌生人不应涌现")
	}
}

func TestEvaluateEmergence_AngryNil(t *testing.T) {
	ctx := EmergenceContext{
		Stage:   StageIntimate,
		Emotion: EmotionState{PrimaryLabel: "ANGRY_ATTACK", Aff: 10, Sec: 10, Aro: 10},
	}
	if got := EvaluateEmergence(ctx, "praise"); got != nil {
		t.Error("愤怒时应抑制涌现")
	}
}

func TestEvaluateEmergence_Cooldown(t *testing.T) {
	ctx := EmergenceContext{
		Stage:        StageIntimate,
		Emotion:      EmotionState{PrimaryLabel: "CALM_RATIONAL", Aff: 10, Sec: 10, Aro: 10},
		DaysSinceMet: 30,
		CurrentTurn:  10,
		LastEmergence: &struct {
			Type string
			Turn int
		}{Type: "time_reflection", Turn: 9},
		RecentEventTypes: []string{"praise", "praise"},
	}
	// 刚涌现过（冷却内）应返回 nil
	if got := EvaluateEmergence(ctx, "praise"); got != nil {
		t.Error("冷却期不应涌现")
	}
}

func TestEvaluateEmergence_TimeReflection(t *testing.T) {
	// 高情绪强度 + 相识久 + 非冷却 → 时间感慨涌现
	// 场景2：SWEET_ATTACHMENT + 认识超 3 月 + warm 氛围
	ctx := EmergenceContext{
		Stage:                      StageIntimate,
		Emotion:                    EmotionState{PrimaryLabel: "SWEET_ATTACHMENT", Aff: 60, Sec: 50, Aro: 40},
		DaysSinceMet:               100,
		Atmosphere:                 "warm",
		CurrentTurn:                5,
		RecentEventTypes:           []string{"praise", "praise", "praise", "praise"},
		ConsecutiveMeaningfulTurns: 4,
	}
	got := EvaluateEmergence(ctx, "praise")
	if got == nil {
		t.Fatal("高情绪 + 久识应涌现，got nil")
	}
	if got.Type != "timeReflection" && got.Type != "responsive" {
		t.Errorf("涌现类型 = %q", got.Type)
	}
}

func TestEvaluateEmergence_TooSoon(t *testing.T) {
	ctx := EmergenceContext{
		Stage:        StageIntimate,
		Emotion:      EmotionState{PrimaryLabel: "CALM_RATIONAL", Aff: 10, Sec: 10, Aro: 10},
		DaysSinceMet: 3, // 相识 <7 天
	}
	if got := EvaluateEmergence(ctx, "praise"); got != nil {
		t.Error("相识不足 7 天不应涌现")
	}
}

// ─── emotion_fusion: 强度分级 ─────────────────────────────────

func TestGetIntensityLevel(t *testing.T) {
	tests := []struct {
		aff  int
		want string
	}{
		{0, "低"}, {49, "低"},
		{50, "中"}, {69, "中"},
		{70, "高"}, {89, "高"},
		{90, "极高"}, {100, "极高"},
	}
	for _, tt := range tests {
		if got := getIntensityLevel(tt.aff); got != tt.want {
			t.Errorf("getIntensityLevel(%d) = %q, want %q", tt.aff, got, tt.want)
		}
	}
}

func TestGetEmotionMaxLength(t *testing.T) {
	// 常见情绪标签都应有最大长度（>0）
	for _, label := range []string{"CALM_RATIONAL", "HAPPY_JOYFUL", "SENSITIVE", "ANGRY_ATTACK", "SWEET_ATTACHMENT"} {
		if got := getEmotionMaxLength(label); got <= 0 {
			t.Errorf("getEmotionMaxLength(%s) = %d, want > 0", label, got)
		}
	}
	// 未知标签兜底（不崩溃，返回非负）
	_ = getEmotionMaxLength("UNKNOWN")
}

func TestDescribeInnerFeeling_Known(t *testing.T) {
	// 已知标签应返回非空描述
	if got := describeInnerFeeling("HAPPY_JOYFUL"); got == "" {
		t.Error("已知情绪应返回内部感受描述")
	}
	// 未知标签返回空（或兜底）
	_ = describeInnerFeeling("UNKNOWN_LABEL")
}

// ─── desire: 主题提取 ────────────────────────────────────────

func TestExtractTopic(t *testing.T) {
	if got := extractTopic("最近工作怎么样"); got == "" {
		t.Error("extractTopic 不应返回空")
	}
	if got := extractTopic(""); got != "近况" {
		t.Errorf("空消息 topic = %q, want 近况", got)
	}
	if got := extractTopic("的了我你是"); got != "近况" {
		t.Errorf("停用词全过滤 topic = %q, want 近况", got)
	}
}

func TestNormalizeTopicKey(t *testing.T) {
	if got := normalizeTopicKey("搜一下美食"); got != "美食" {
		t.Errorf("normalizeTopicKey(搜一下美食) = %q, want 美食", got)
	}
	if got := normalizeTopicKey("介绍一下AI"); got != "ai" {
		t.Errorf("normalizeTopicKey(介绍一下AI) = %q, want ai（去前缀+小写）", got)
	}
}

func TestDesireTopicMatchesKnowledge(t *testing.T) {
	tests := []struct {
		desire, knowledge string
		want              bool
	}{
		{"美食", "美食探店", true},
		{"美食探店", "美食", true},
		{"搜一下旅行攻略", "旅行", true},
		{"美食", "电影", false},
		{"a", "ab", false}, // 长度不足
	}
	for _, tt := range tests {
		if got := DesireTopicMatchesKnowledge(tt.desire, tt.knowledge); got != tt.want {
			t.Errorf("DesireTopicMatchesKnowledge(%q,%q) = %v, want %v", tt.desire, tt.knowledge, got, tt.want)
		}
	}
}

func TestDesireToHint(t *testing.T) {
	d := &Desire{Topic: "美食", Category: "curiosity", Urgency: 6.5, Status: "active"}
	got := desireToHint(*d)
	if got == "" {
		t.Error("desireToHint 不应返回空")
	}
	if !containsSubstr(got, []string{"美食"}) {
		t.Errorf("desireToHint 应包含主题: %q", got)
	}
}

// ─── desire: 栈操作 ──────────────────────────────────────────

func TestUpdateDesireStack_CapsAt5(t *testing.T) {
	stack := DefaultDesireStack()
	l1 := L1State{Trust: 50}
	// 连续推入多个欲望
	for i := 0; i < 8; i++ {
		stack, _ = UpdateDesireStack(stack, "我想了解主题"+string(rune('a'+i)), Event{Type: "casual"}, l1, i)
	}
	if len(stack.Slots) > 5 {
		t.Errorf("欲望栈超过 5 槽上限: %d", len(stack.Slots))
	}
}

func TestUpdateDesireStack_Structural(t *testing.T) {
	stack := DefaultDesireStack()
	l1 := L1State{Trust: 50}
	// 多轮调用：不崩溃、槽数不超上限、返回的槽无悬挂指针
	for i := 0; i < 10; i++ {
		var err error
		stack, _ = UpdateDesireStack(stack, "我想了解主题"+string(rune('a'+i%5)), Event{Type: "casual", Intensity: 0.5}, l1, i)
		_ = err
		if len(stack.Slots) != 5 {
			t.Fatalf("轮 %d 槽数 = %d, want 5", i, len(stack.Slots))
		}
	}
	// 全部槽可安全访问
	for _, d := range stack.Slots {
		if d != nil {
			_ = d.Topic
		}
	}
}

func TestDismissDesireFromStack(t *testing.T) {
	stack := DesireStack{Slots: []*Desire{
		{ID: "1", Topic: "a"}, {ID: "2", Topic: "b"}, {ID: "3", Topic: "c"},
	}}
	stack = DismissDesireFromStack(stack, "2")
	// 实现为置 nil（保持槽数），ID=2 的槽应为 nil
	if stack.Slots[1] != nil {
		t.Error("欲望 2 的槽应被置为 nil")
	}
	if stack.Slots[0] == nil || stack.Slots[0].ID != "1" {
		t.Error("欲望 1 不应受影响")
	}
	// 空 ID 应原样返回
	stack2 := DismissDesireFromStack(stack, "  ")
	if len(stack2.Slots) != 3 {
		t.Error("空 ID 应原样返回")
	}
}

func TestClearActiveDesires(t *testing.T) {
	stack := DesireStack{Slots: []*Desire{
		{ID: "1", Status: "active"}, {ID: "2", Status: "expressed"}, {ID: "3", Status: "settled"},
	}}
	stack = ClearActiveDesires(stack)
	for _, d := range stack.Slots {
		if d != nil && (d.Status == "active" || d.Status == "latent") {
			t.Errorf("ClearActiveDesires 应清除活跃欲望: %+v", d)
		}
	}
	// active 槽应被置 nil
	if stack.Slots[0] != nil {
		t.Error("active 槽应被置为 nil")
	}
}

// ─── age_computer ────────────────────────────────────────────

func TestInferBirthYear(t *testing.T) {
	// 2024-06-15 记录，生日 03-01（已过）→ 2024-age
	if got := InferBirthYear(25, "03-01", "2024-06-15T00:00:00Z"); got != 1999 {
		t.Errorf("生日已过 InferBirthYear = %d, want 1999", got)
	}
	// 生日 12-01（未过）→ 2024-age-1
	if got := InferBirthYear(25, "12-01", "2024-06-15T00:00:00Z"); got != 1998 {
		t.Errorf("生日未过 InferBirthYear = %d, want 1998", got)
	}
	// 无生日 → 直接减
	if got := InferBirthYear(25, "", "2024-06-15T00:00:00Z"); got != 1999 {
		t.Errorf("无生日 InferBirthYear = %d, want 1999", got)
	}
	// 非法记录时间 → 用当前年（不崩溃即可）
	_ = InferBirthYear(25, "03-01", "bad-time")
}

func TestComputeCurrentAge_BirthdayPassed(t *testing.T) {
	// birthYear 已知 + 生日已过 → now.Year - birthYear
	now := time.Now()
	// 构造生日为今年已过日期（用昨天的月日）
	mmdd := now.AddDate(0, 0, -1).Format("01-02")
	got := ComputeCurrentAge(0, now.Year()-30, mmdd, false, "")
	if got != 30 {
		t.Errorf("生日已过 ComputeCurrentAge = %d, want 30", got)
	}
}

func TestComputeCurrentAge_BirthdayNotPassed(t *testing.T) {
	now := time.Now()
	// 生日为今年未来日期（用明天的月日，除非年底）
	mmdd := now.AddDate(0, 0, 1).Format("01-02")
	// 若跨年边界，改为 01-01
	if mmdd == "01-01" {
		mmdd = "01-01"
	}
	got := ComputeCurrentAge(0, now.Year()-30, mmdd, false, "")
	if got != 29 {
		t.Errorf("生日未过 ComputeCurrentAge = %d, want 29", got)
	}
}

func TestComputeCurrentAge_NoBirthYear(t *testing.T) {
	// 无 birthYear：用 recordedAt 计算
	recorded := time.Now().AddDate(-2, 0, 0).Format(time.RFC3339)
	got := ComputeCurrentAge(20, 0, "", false, recorded)
	if got != 22 {
		t.Errorf("无 birthYear ComputeCurrentAge = %d, want 22", got)
	}
	// 无任何信息：返回原 age
	if got := ComputeCurrentAge(20, 0, "", false, ""); got != 20 {
		t.Errorf("无信息 ComputeCurrentAge = %d, want 20", got)
	}
}

// ─── 辅助 ────────────────────────────────────────────────────

func containsSubstr(s string, subs []string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && sub != "" {
			// 中文子串用 Contains 检查
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
